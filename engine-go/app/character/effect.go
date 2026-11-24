package character

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ApplySpellBuff grava o bônus de uma magia na ficha de quem o recebe, pela escrita do efeito da magia
// (`writeSpellEffect`; ver GLOSSARY, **efeito da magia**).
//
// A DURAÇÃO é a escolhida entre as que o bônus pode ter (`sheet.BuffDurations`): a do verbete, quando vem vazia, ou
// a que o aprimoramento deu à conjuração, como a Velocidade de cena. Quem aplica não sustenta nada — a manutenção e
// o limite de uma sustentada são de quem conjurou —, e por isso aqui não há recusa da segunda sustentada.
//
//	p.ApplySpellBuff(ctx, characterID, "velocidade", "cena")
func (p Plays) ApplySpellBuff(ctx context.Context, characterID int64, spellID, duration string) error {
	spell, known := book.SpellByID(spellID)
	if !known || spell.Buff == nil {
		return fmt.Errorf("a magia %q não deixa efeito para aplicar", spellID)
	}
	durations, err := sheet.BuffDurations(spell)
	if err != nil {
		return err
	}
	scope := durations[0]
	if duration != "" {
		asked, err := engine.ParseDuration(duration)
		if err != nil || !slices.Contains(durations, asked.Stored()) {
			labels := make([]string, 0, len(durations))
			for _, d := range durations {
				labels = append(labels, engine.DurationLabel(d))
			}
			return fmt.Errorf("%s não dura %q: o bônus dela dura %s", spell.Name, duration, strings.Join(labels, " ou "))
		}
		scope = asked.Stored()
	}
	if _, err := writeSpellEffect(ctx, p.queries, characterID, spellID, spellEffectWrite{
		Scope: scope, Modifiers: string(spell.Buff.Modifiers),
	}); err != nil {
		return err
	}
	return nil
}

// ApplyTempHpPool liga uma reserva de PV temporários.
//
// Ela era uma TRANSAÇÃO de quatro passos — listar os efeitos, planejar um
// "vale-o-maior", apagar/zerar as poças deslocadas, gravar a nova — e virou um
// `UPSERT` só quando aquela regra deixou de existir (ALE-347). A **p106** manda
// SOMAR os pontos temporários, e o `domain/sheet/temp_pools.go` explica por que a
// ordem de drenagem é escolha nossa e o empilhamento é do motor.
//
// O `UPSERT` continua fazendo o trabalho que sobrou: reentrar na mesma Fúria
// REESCREVE a poça daquele poder em vez de abrir uma segunda, porque a chave é
// (personagem, catálogo, escopo). Fonte repetida não empilha; fontes
// diferentes, sim.
func (p Plays) ApplyTempHpPool(
	ctx context.Context, characterID int64, source, catalogID, scope string, howMuch int, note string,
) (sheet.EffectDTO, error) {
	mods := []map[string]any{
		{"target": map[string]any{"k": "tempHp"}, "amount": howMuch, "bonusType": "untyped", "note": note},
	}
	modJSON, _ := json.Marshal(mods)

	eff, err := p.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: source, Catalogid: catalogID, Scope: scope,
		Modifiers: string(modJSON), Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, fmt.Errorf("gravar a poça de %d PV temporários de %q: %w", howMuch, catalogID, err)
	}
	return effectFrom(eff), nil
}

// ApplyTempPool liga a reserva que a concessão dá — de PV ou de PM, pelo `Kind` dela —
// pela mesma gravação da `ApplyTempHpPool`: fonte repetida reescreve, fontes diferentes somam.
//
//	p.ApplyTempPool(ctx, characterID, "power", "class.cavaleiro.estandarte", *spec.Grant, 3)
func (p Plays) ApplyTempPool(
	ctx context.Context, characterID int64, source, catalogID string, grant book.PowerGrant, howMuch int,
) (sheet.EffectDTO, error) {
	if grant.Kind != "temp-mp" {
		return p.ApplyTempHpPool(ctx, characterID, source, catalogID, grant.Scope, howMuch, "PV temporários")
	}
	mods := []map[string]any{
		{"target": map[string]any{"k": "tempMp"}, "amount": howMuch, "bonusType": "untyped", "note": "PM temporários"},
	}
	modJSON, _ := json.Marshal(mods)
	eff, err := p.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: source, Catalogid: catalogID, Scope: grant.Scope,
		Modifiers: string(modJSON), Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, fmt.Errorf("gravar a poça de %d PM temporários de %q: %w", howMuch, catalogID, err)
	}
	return effectFrom(eff), nil
}

// GrantAmount é quanto a concessão dá a esta ficha, pela conta do verbete
// (`PowerGrant.AmountFor`) com o atributo computado.
func (p Plays) GrantAmount(ctx context.Context, row sqlcgen.Character, grant book.PowerGrant) (int, bool) {
	if p.catalogs == nil {
		return 0, false
	}
	computed, err := sheet.LoadAndCompute(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return 0, false
	}
	attr, found := computed.Attributes[grant.Attribute]
	if !found {
		return 0, false
	}
	return grant.AmountFor(int(row.Level), attr.Total), true
}

// TempHpAmount é quanto de PV temporário um poder concede: o NÍVEL do
// personagem mais o total computado do atributo-chave.
//
// Devolve `false` em vez de erro quando não dá para responder — motor ausente,
// ficha que não computa, atributo que o poder cita e a ficha não tem. Quem
// chama trata os três do mesmo jeito: não concede nada. Uma concessão de zero
// gravaria uma linha de efeito que a aba mostraria valendo nada.
func (p Plays) TempHpAmount(ctx context.Context, row sqlcgen.Character, attribute string) (int, bool) {
	if p.catalogs == nil {
		return 0, false
	}
	computed, err := sheet.LoadAndCompute(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return 0, false
	}
	attr, found := computed.Attributes[attribute]
	if !found {
		return 0, false
	}
	return int(row.Level) + attr.Total, true
}

func effectFrom(e sqlcgen.UpsertActiveEffectRow) sheet.EffectDTO {
	return sheet.EffectDTO{
		ID: e.ID, CatalogID: e.Catalogid, Scope: e.Scope,
		Modifiers: e.Modifiers, CreatedAt: e.Createdat,
	}
}
