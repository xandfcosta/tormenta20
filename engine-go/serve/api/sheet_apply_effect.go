package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/wire"

	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// applyTempHpPool liga uma reserva de PV temporários, SEM transporte.
//
// Ela era uma TRANSAÇÃO de quatro passos — listar os efeitos, planejar o
// vale-o-maior, apagar/zerar as poças deslocadas, gravar a nova — e virou um
// `UPSERT` só quando a regra caiu (ALE-347). A p106 manda SOMAR os pontos
// temporários, e o `domain/sheet` explica por que a ordem de drenagem é nossa e
// o empilhamento é do motor.
//
// O `Upsert` continua fazendo o trabalho que sobrou: reentrar na mesma Fúria
// REESCREVE a poça da Alma de Bronze em vez de abrir uma segunda, porque a
// chave é (personagem, catálogo, escopo). Fonte repetida não empilha; fontes
// diferentes, sim.
func (sr sheetRules) applyTempHpPool(
	ctx context.Context, id int64, source, catalogID, scope string, amount int, note string,
) (sheet.EffectDTO, error) {
	mods := []map[string]any{{"target": map[string]any{"k": "tempHp"}, "amount": amount, "bonusType": "untyped", "note": note}}
	modJSON, _ := json.Marshal(mods)

	eff, err := sr.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: id, Source: source, Catalogid: catalogID, Scope: scope,
		Modifiers: string(modJSON), Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, fmt.Errorf("gravar a poça de %d PV temporários de %q: %w", amount, catalogID, err)
	}
	return effectDTOFromUpsert(eff), nil
}

// applySpellBuffEffect is the spell-buff domain rule, transport-agnostic: the spell must
// carry a buff block; upsert its modifiers under (character, spell, scope). Used by the
// HTTP handler and — via the same core — the WS `apply-effect` gateway handler (B.6).
// Returns the effect + an HTTP-ish status the caller maps to its transport.
func (sr sheetRules) applySpellBuffEffect(ctx context.Context, charID int64, spellID string, scopeOverride *string) (sheet.EffectDTO, int, error) {
	spell, known := catalog.LookupSpell(spellID)
	if !known || spell.Buff == nil {
		return sheet.EffectDTO{}, http.StatusBadRequest, wire.NewFieldError(
			http.StatusBadRequest,
			fmt.Sprintf("Spell %q has no applicable buff", spellID),
			wire.FieldErrorMap{"spellId": {"Magia sem efeito aplicável"}},
		)
	}
	scope := derefStr(scopeOverride, spell.Buff.DefaultScope)
	eff, err := sr.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: charID, Source: "spell", Catalogid: spellID, Scope: scope, Modifiers: string(spell.Buff.Modifiers), Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, http.StatusInternalServerError, errors.New("Could not apply buff")
	}
	return effectDTOFromUpsert(eff), http.StatusOK, nil
}

// powerTempHpAmount computes a temp-HP power's magnitude: character level + the attribute's
// computed total (mirrors tempHpModifier), reusing the already-loaded row.
func (sr sheetRules) powerTempHpAmount(ctx context.Context, row sqlcgen.Character, attribute string) (int, bool) {
	if sr.catalogs == nil {
		return 0, false
	}
	sheet, err := sr.ComputeSheet(ctx, row)
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

// derefStr mora com o ÚNICO chamador que tem: o compilador do Go não acusa
// função de pacote sem uso, então uma cópia solta num arquivo vizinho sobrevive
// em silêncio à morte de quem a chamava.
func derefStr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
