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

// applyPoolTx é a transação da poça de PV temporários, SEM transporte.
//
// Ela nasceu na ALE-278 porque a mesma sequência — `BeginTx`, listar os efeitos,
// planejar pelo `sheet`, apagar/zerar os deslocados, gravar o novo, `Commit` —
// estava escrita DUAS vezes: aqui e dentro da cena da ficha, que a montava para
// a concessão de um poder. Duas transações sobre a mesma regra divergem na
// primeira vez que uma das duas ganhar um passo.
//
// A conta continua sendo do `sheet` (`PlanPoolSupremacy`): "se você receber PV
// temporários de mais de uma fonte, considere apenas o maior valor" (p256).
// Poça SUPERADA não é erro — é a regra dizendo que esta não vale —, e por isso
// ela volta com o plano e sem efeito.
func (sr sheetRules) applyPoolTx(
	ctx context.Context, id int64, source, catalogID, scope string, amount int, note string,
) (sheet.PoolPlan, sheet.EffectDTO, error) {
	mods := []map[string]any{{"target": map[string]any{"k": "tempHp"}, "amount": amount, "bonusType": "untyped", "note": note}}
	modJSON, _ := json.Marshal(mods)

	tx, err := sr.db.BeginTx(ctx, nil)
	if err != nil {
		return sheet.PoolPlan{}, sheet.EffectDTO{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := sr.queries.WithTx(tx)

	rows, err := q.ListActiveEffectsByCharacter(ctx, id)
	if err != nil {
		return sheet.PoolPlan{}, sheet.EffectDTO{}, fmt.Errorf("ler os efeitos do personagem %d: %w", id, err)
	}
	plan := sheet.PlanPoolSupremacy(sheet.ParseTempHpPools(rows), catalogID, scope, amount)
	if plan.Superseded {
		return plan, sheet.EffectDTO{}, tx.Commit()
	}
	for _, z := range plan.ZeroWrites {
		if err := q.UpdateEffectModifiers(ctx, sqlcgen.UpdateEffectModifiersParams{Modifiers: z.Modifiers, ID: z.EffectID}); err != nil {
			return plan, sheet.EffectDTO{}, fmt.Errorf("zerar a poça deslocada %d: %w", z.EffectID, err)
		}
	}
	for _, delID := range plan.DeleteIDs {
		if err := q.DeleteEffectByID(ctx, delID); err != nil {
			return plan, sheet.EffectDTO{}, fmt.Errorf("apagar a poça deslocada %d: %w", delID, err)
		}
	}
	eff, err := q.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: source, Catalogid: catalogID, Scope: scope, Modifiers: string(modJSON), Createdat: plataforma.NowISO(),
	})
	if err != nil {
		return plan, sheet.EffectDTO{}, err
	}
	if err := tx.Commit(); err != nil {
		return plan, sheet.EffectDTO{}, err
	}
	return plan, effectDTOFromUpsert(eff), nil
}

// applySpellBuffEffect is the spell-buff domain rule, transport-agnostic: the spell must
// carry a buff block; upsert its modifiers under (character, spell, scope). Used by the
// HTTP handler and — via the same core — the WS `apply-effect` gateway handler (B.6).
// Returns the effect + an HTTP-ish status the caller maps to its transport.
func (sr sheetRules) applySpellBuffEffect(ctx context.Context, charID int64, spellID string, scopeOverride *string) (sheet.EffectDTO, int, error) {
	spell, known := catalog.LookupSpell(spellID)
	if !known || spell.Buff == nil {
		return sheet.EffectDTO{}, http.StatusBadRequest, plataforma.NewFieldError(
			http.StatusBadRequest,
			fmt.Sprintf("Spell %q has no applicable buff", spellID),
			plataforma.FieldErrorMap{"spellId": {"Magia sem efeito aplicável"}},
		)
	}
	scope := derefStr(scopeOverride, spell.Buff.DefaultScope)
	eff, err := sr.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: charID, Source: "spell", Catalogid: spellID, Scope: scope, Modifiers: string(spell.Buff.Modifiers), Createdat: plataforma.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, http.StatusInternalServerError, errors.New("Could not apply buff")
	}
	return effectDTOFromUpsert(eff), http.StatusOK, nil
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

// powerTempHpAmount computes a temp-HP power's magnitude: character level + the attribute's
// computed total (mirrors tempHpModifier), reusing the already-loaded row.
func (sr sheetRules) powerTempHpAmount(r *http.Request, row sqlcgen.Character, attribute string) (int, bool) {
	if sr.catalogs == nil {
		return 0, false
	}
	sheet, err := sr.ComputeSheet(r.Context(), row)
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
