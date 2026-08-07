package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
)

const manualTempHpCatalogID = "manual-temp-hp"

// handleApplyEffect ports CharacterEffectsService.applyEffect: applies a manual
// temp-HP pool, a spell buff, or (deferred) a power grant. NOTE: the powerId path
// (activation-registry grants + server-side pool compute) is not yet ported.
func (s *Server) handleApplyEffect(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		SpellID      *string `json:"spellId"`
		PowerID      *string `json:"powerId"`
		ManualTempHp *int64  `json:"manualTempHp"`
		Scope        *string `json:"scope"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}

	switch {
	case body.ManualTempHp != nil:
		s.applyManualPool(w, r, id, *body.ManualTempHp, derefStr(body.Scope, "scene"))
	case body.PowerID != nil:
		s.applyPowerGrant(w, r, id, *body.PowerID, body.Scope)
	case body.SpellID == nil:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     "applyEffect requires spellId, powerId or manualTempHp",
			"fieldErrors": FieldErrorMap{"spellId": {"Informe uma magia, um poder ou PV temporários"}},
		})
	default:
		dto, status, err := s.applySpellBuffEffect(r.Context(), id, *body.SpellID, body.Scope)
		if err != nil {
			writeDomainError(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, dto)
	}
}

func (s *Server) applyManualPool(w http.ResponseWriter, r *http.Request, id, amount int64, scope string) {
	if amount < 0 || amount > 9999 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     fmt.Sprintf("manualTempHp must be an integer >= 0 — got %d", amount),
			"fieldErrors": FieldErrorMap{"manualTempHp": {"Informe um valor inteiro ≥ 0"}},
		})
		return
	}
	if amount == 0 {
		ids, err := s.queries.ListEffectIdsByCatalog(r.Context(), sqlcgen.ListEffectIdsByCatalogParams{Characterid: id, Catalogid: manualTempHpCatalogID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not clear pool")
			return
		}
		if err := s.queries.DeleteEffectsByCatalog(r.Context(), sqlcgen.DeleteEffectsByCatalogParams{Characterid: id, Catalogid: manualTempHpCatalogID}); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not clear pool")
			return
		}
		if ids == nil {
			ids = []int64{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"cleared": true, "removedEffectIds": ids})
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
		writeError(w, http.StatusInternalServerError, "Could not apply pool")
		return
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)

	rows, err := q.ListActiveEffectsByCharacter(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load effects")
		return
	}
	plan := planPoolSupremacy(parseTempHpPools(rows), catalogID, scope, amount)
	if plan.superseded {
		_ = tx.Commit()
		writeJSON(w, http.StatusOK, map[string]any{"superseded": true, "keptEffectId": plan.keptEffectID, "keptAmount": plan.keptAmount})
		return
	}
	for _, z := range plan.zeroWrites {
		if err := q.UpdateEffectModifiers(r.Context(), sqlcgen.UpdateEffectModifiersParams{Modifiers: z.modifiers, ID: z.effectID}); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not displace pool")
			return
		}
	}
	for _, delID := range plan.deleteIDs {
		if err := q.DeleteEffectByID(r.Context(), delID); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not displace pool")
			return
		}
	}
	eff, err := q.UpsertActiveEffect(r.Context(), sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: source, Catalogid: catalogID, Scope: scope, Modifiers: string(modJSON), Createdat: nowISO(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not apply pool")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not apply pool")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"effect":    effectDTOFromUpsert(eff),
		"displaced": plan.displaced,
	})
}

// applySpellBuffEffect is the spell-buff domain rule, transport-agnostic: the spell must
// carry a buff block; upsert its modifiers under (character, spell, scope). Used by the
// HTTP handler and — via the same core — the WS `apply-effect` gateway handler (B.6).
// Returns the effect + an HTTP-ish status the caller maps to its transport.
func (s *Server) applySpellBuffEffect(ctx context.Context, charID int64, spellID string, scopeOverride *string) (EffectDTO, int, error) {
	spell, known := catalog.LookupSpell(spellID)
	if !known || spell.Buff == nil {
		return EffectDTO{}, http.StatusBadRequest, &fieldError{
			status:  http.StatusBadRequest,
			message: fmt.Sprintf("Spell %q has no applicable buff", spellID),
			fields:  FieldErrorMap{"spellId": {"Magia sem efeito aplicável"}},
		}
	}
	scope := derefStr(scopeOverride, spell.Buff.DefaultScope)
	eff, err := s.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: charID, Source: "spell", Catalogid: spellID, Scope: scope, Modifiers: string(spell.Buff.Modifiers), Createdat: nowISO(),
	})
	if err != nil {
		return EffectDTO{}, http.StatusInternalServerError, errors.New("Could not apply buff")
	}
	return effectDTOFromUpsert(eff), http.StatusOK, nil
}

// applyPowerGrant ports the Nest powerId branch: resolve the power's activation grant and
// apply it — a temp-HP pool scaled by (level + attribute total), or a fixed active-effect.
// Unknown power / power without a grant → 400 (like the Nest registry lookup).
func (s *Server) applyPowerGrant(w http.ResponseWriter, r *http.Request, id int64, powerID string, scopeOverride *string) {
	spec, known := catalog.LookupActivation(powerID)
	if !known {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     fmt.Sprintf("Power %q not found in the activation registry", powerID),
			"fieldErrors": FieldErrorMap{"powerId": {"Poder desconhecido"}},
		})
		return
	}
	if spec.Grant == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     fmt.Sprintf("Power %q has no applicable grant", powerID),
			"fieldErrors": FieldErrorMap{"powerId": {"Poder sem efeito aplicável"}},
		})
		return
	}
	grant := spec.Grant
	scope := derefStr(scopeOverride, grant.Scope)
	if grant.Kind == "temp-hp" {
		amount, ok := s.powerTempHpAmount(r, id, grant.Attribute)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Could not compute power temp HP")
			return
		}
		s.applyPool(w, r, id, "power", powerID, scope, amount, "PV temporários")
		return
	}
	// active-effect: upsert the grant's modifiers as a power effect (source "power").
	mods := grant.Modifiers
	if len(mods) == 0 {
		mods = json.RawMessage("[]")
	}
	eff, err := s.queries.UpsertActiveEffect(r.Context(), sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: "power", Catalogid: powerID, Scope: scope, Modifiers: string(mods), Createdat: nowISO(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not apply power")
		return
	}
	writeJSON(w, http.StatusOK, effectDTOFromUpsert(eff))
}

// powerTempHpAmount computes a temp-HP power's magnitude: character level + the attribute's
// computed total (mirrors tempHpModifier). Needs the rules engine primed.
func (s *Server) powerTempHpAmount(r *http.Request, id int64, attribute string) (int, bool) {
	if s.catalogs == nil {
		return 0, false
	}
	row, err := s.queries.GetCharacter(r.Context(), id)
	if err != nil {
		return 0, false
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		return 0, false
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return 0, false
	}
	attr, ok := s.catalogs.ComputeSheetV2(ec, map[string]bool{}).Attributes[attribute]
	if !ok {
		return 0, false
	}
	return int(row.Level) + attr.Total, true
}

func effectDTOFromUpsert(e sqlcgen.UpsertActiveEffectRow) EffectDTO {
	return EffectDTO{ID: e.ID, CatalogID: e.Catalogid, Scope: e.Scope, Modifiers: e.Modifiers, CreatedAt: e.Createdat}
}
