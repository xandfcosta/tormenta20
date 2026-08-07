package api

import (
	"encoding/json"
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
		writeError(w, http.StatusNotImplemented, "power grants are not yet supported by the Go API")
	case body.SpellID == nil:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     "applyEffect requires spellId, powerId or manualTempHp",
			"fieldErrors": FieldErrorMap{"spellId": {"Informe uma magia, um poder ou PV temporários"}},
		})
	default:
		s.applySpellBuff(w, r, id, *body.SpellID, body.Scope)
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

// applySpellBuff ports applySpellBuff: the spell must carry a buff block, upsert
// its modifiers under (character, spell, scope).
func (s *Server) applySpellBuff(w http.ResponseWriter, r *http.Request, id int64, spellID string, scopeOverride *string) {
	spell, known := catalog.LookupSpell(spellID)
	if !known || spell.Buff == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     fmt.Sprintf("Spell %q has no applicable buff", spellID),
			"fieldErrors": FieldErrorMap{"spellId": {"Magia sem efeito aplicável"}},
		})
		return
	}
	scope := derefStr(scopeOverride, spell.Buff.DefaultScope)
	eff, err := s.queries.UpsertActiveEffect(r.Context(), sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: "spell", Catalogid: spellID, Scope: scope, Modifiers: string(spell.Buff.Modifiers), Createdat: nowISO(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not apply buff")
		return
	}
	writeJSON(w, http.StatusOK, effectDTOFromUpsert(eff))
}

func effectDTOFromUpsert(e sqlcgen.UpsertActiveEffectRow) EffectDTO {
	return EffectDTO{ID: e.ID, CatalogID: e.Catalogid, Scope: e.Scope, Modifiers: e.Modifiers, CreatedAt: e.Createdat}
}
