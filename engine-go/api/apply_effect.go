package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

const manualTempHpCatalogID = "manual-temp-hp"

// handleApplyEffect applies a manual
// temp-HP pool, a spell buff, or (deferred) a power grant. NOTE: the powerId path
// (activation-registry grants + server-side pool compute) is not yet ported.
func (s *Server) handleApplyEffect(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		SpellID      *string `json:"spellId"`
		PowerID      *string `json:"powerId"`
		ManualTempHp *int64  `json:"manualTempHp"`
		Scope        *string `json:"scope"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}

	switch {
	case body.ManualTempHp != nil:
		s.applyManualPool(w, r, row.ID, *body.ManualTempHp, derefStr(body.Scope, "scene"))
	case body.PowerID != nil:
		s.applyPowerGrant(w, r, row, *body.PowerID, body.Scope)
	case body.SpellID == nil:
		plataforma.WriteFieldError(w, http.StatusBadRequest, "applyEffect requires spellId, powerId or manualTempHp", plataforma.FieldErrorMap{"spellId": {"Informe uma magia, um poder ou PV temporários"}})
	default:
		dto, status, err := s.applySpellBuffEffect(r.Context(), row.ID, *body.SpellID, body.Scope)
		if err != nil {
			plataforma.WriteDomainError(w, status, err)
			return
		}
		plataforma.WriteJSON(w, http.StatusOK, dto)
	}
}

func (s *Server) applyManualPool(w http.ResponseWriter, r *http.Request, id, amount int64, scope string) {
	if amount < 0 || amount > 9999 {
		plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("manualTempHp must be an integer >= 0 — got %d", amount), plataforma.FieldErrorMap{"manualTempHp": {"Informe um valor inteiro ≥ 0"}})
		return
	}
	if amount == 0 {
		ids, err := s.queries.ListEffectIdsByCatalog(r.Context(), sqlcgen.ListEffectIdsByCatalogParams{Characterid: id, Catalogid: manualTempHpCatalogID})
		if err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not clear pool")
			return
		}
		if err := s.queries.DeleteEffectsByCatalog(r.Context(), sqlcgen.DeleteEffectsByCatalogParams{Characterid: id, Catalogid: manualTempHpCatalogID}); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not clear pool")
			return
		}
		if ids == nil {
			ids = []int64{}
		}
		plataforma.WriteJSON(w, http.StatusOK, map[string]any{"cleared": true, "removedEffectIds": ids})
		return
	}
	s.applyPool(w, r, id, "manual", manualTempHpCatalogID, scope, int(amount), "PV temporários (manual)")
}

// applyPool ports temp-hp.service applyPool: upsert a tempHp pool under vale-o-maior.
func (s *Server) applyPool(w http.ResponseWriter, r *http.Request, id int64, source, catalogID, scope string, amount int, note string) {
	mods := []map[string]any{{"target": map[string]any{"k": "tempHp"}, "amount": amount, "bonusType": "untyped", "note": note}}
	modJSON, _ := json.Marshal(mods)

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply pool")
		return
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)

	rows, err := q.ListActiveEffectsByCharacter(r.Context(), id)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load effects")
		return
	}
	plan := planPoolSupremacy(parseTempHpPools(rows), catalogID, scope, amount)
	if plan.superseded {
		_ = tx.Commit()
		plataforma.WriteJSON(w, http.StatusOK, map[string]any{"superseded": true, "keptEffectId": plan.keptEffectID, "keptAmount": plan.keptAmount})
		return
	}
	for _, z := range plan.zeroWrites {
		if err := q.UpdateEffectModifiers(r.Context(), sqlcgen.UpdateEffectModifiersParams{Modifiers: z.modifiers, ID: z.effectID}); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not displace pool")
			return
		}
	}
	for _, delID := range plan.deleteIDs {
		if err := q.DeleteEffectByID(r.Context(), delID); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not displace pool")
			return
		}
	}
	eff, err := q.UpsertActiveEffect(r.Context(), sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: source, Catalogid: catalogID, Scope: scope, Modifiers: string(modJSON), Createdat: plataforma.NowISO(),
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply pool")
		return
	}
	if err := tx.Commit(); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply pool")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{
		"effect":    effectDTOFromUpsert(eff),
		"displaced": plan.displaced,
	})
}

// applySpellBuffEffect is the spell-buff domain rule, transport-agnostic: the spell must
// carry a buff block; upsert its modifiers under (character, spell, scope). Used by the
// HTTP handler and — via the same core — the WS `apply-effect` gateway handler (B.6).
// Returns the effect + an HTTP-ish status the caller maps to its transport.
func (s *Server) applySpellBuffEffect(ctx context.Context, charID int64, spellID string, scopeOverride *string) (sheet.EffectDTO, int, error) {
	spell, known := catalog.LookupSpell(spellID)
	if !known || spell.Buff == nil {
		return sheet.EffectDTO{}, http.StatusBadRequest, plataforma.NewFieldError(
			http.StatusBadRequest,
			fmt.Sprintf("Spell %q has no applicable buff", spellID),
			plataforma.FieldErrorMap{"spellId": {"Magia sem efeito aplicável"}},
		)
	}
	scope := derefStr(scopeOverride, spell.Buff.DefaultScope)
	eff, err := s.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: charID, Source: "spell", Catalogid: spellID, Scope: scope, Modifiers: string(spell.Buff.Modifiers), Createdat: plataforma.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, http.StatusInternalServerError, errors.New("Could not apply buff")
	}
	return effectDTOFromUpsert(eff), http.StatusOK, nil
}

// applyPowerGrant is the powerId branch: resolve the power's activation grant and
// apply it — a temp-HP pool scaled by (level + attribute total), or a fixed active-effect.
// Unknown power / power without a grant → 400.
func (s *Server) applyPowerGrant(w http.ResponseWriter, r *http.Request, row sqlcgen.Character, powerID string, scopeOverride *string) {
	grant, ok := resolvePowerGrant(w, powerID)
	if !ok {
		return
	}
	scope := derefStr(scopeOverride, grant.Scope)
	if grant.Kind == "temp-hp" {
		amount, ok := s.powerTempHpAmount(r, row, grant.Attribute)
		if !ok {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not compute power temp HP")
			return
		}
		s.applyPool(w, r, row.ID, "power", powerID, scope, amount, "PV temporários")
		return
	}
	s.upsertPowerEffect(w, r, row.ID, powerID, scope, grant.Modifiers)
}

// resolvePowerGrant looks up a power's activation grant, writing the appropriate 400 and
// returning ok=false when the power is unknown or has no applicable grant.
func resolvePowerGrant(w http.ResponseWriter, powerID string) (*catalog.ActivationGrant, bool) {
	spec, known := catalog.LookupActivation(powerID)
	if !known {
		plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("Power %q not found in the activation registry", powerID), plataforma.FieldErrorMap{"powerId": {"Poder desconhecido"}})
		return nil, false
	}
	if spec.Grant == nil {
		plataforma.WriteFieldError(w, http.StatusBadRequest, fmt.Sprintf("Power %q has no applicable grant", powerID), plataforma.FieldErrorMap{"powerId": {"Poder sem efeito aplicável"}})
		return nil, false
	}
	return spec.Grant, true
}

// upsertPowerEffect upserts an active-effect power grant's modifiers (source "power").
func (s *Server) upsertPowerEffect(w http.ResponseWriter, r *http.Request, id int64, powerID, scope string, modifiers json.RawMessage) {
	mods := modifiers
	if len(mods) == 0 {
		mods = json.RawMessage("[]")
	}
	eff, err := s.queries.UpsertActiveEffect(r.Context(), sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: "power", Catalogid: powerID, Scope: scope, Modifiers: string(mods), Createdat: plataforma.NowISO(),
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not apply power")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, effectDTOFromUpsert(eff))
}

// powerTempHpAmount computes a temp-HP power's magnitude: character level + the attribute's
// computed total (mirrors tempHpModifier), reusing the already-loaded row.
func (s *Server) powerTempHpAmount(r *http.Request, row sqlcgen.Character, attribute string) (int, bool) {
	if s.catalogs == nil {
		return 0, false
	}
	sheet, err := s.ComputeSheet(r.Context(), row)
	if err != nil {
		return 0, false
	}
	attr, ok := sheet.Attributes[attribute]
	if !ok {
		return 0, false
	}
	return int(row.Level) + attr.Total, true
}

func effectDTOFromUpsert(e sqlcgen.UpsertActiveEffectRow) sheet.EffectDTO {
	return sheet.EffectDTO{ID: e.ID, CatalogID: e.Catalogid, Scope: e.Scope, Modifiers: e.Modifiers, CreatedAt: e.Createdat}
}
