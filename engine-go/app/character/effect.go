package character

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ApplySpellBuff liga o efeito de uma magia de melhoria.
//
// A magia tem de CARREGAR um bloco de melhoria; o escopo vem dela e o chamador
// só o sobrescreve quando quer. A gravação é a mesma para as duas entradas que
// existiram — a rota JSON, que morreu, e a aba Efeitos da ficha —, e é isso que
// mantém uma resposta só para "esta magia deixa efeito?".
func (p Plays) ApplySpellBuff(
	ctx context.Context, characterID int64, spellID string, escopo *string,
) (sheet.EffectDTO, error) {
	spell, known := catalog.LookupSpell(spellID)
	if !known || spell.Buff == nil {
		return sheet.EffectDTO{}, fmt.Errorf("a magia %q não deixa efeito para aplicar", spellID)
	}
	scope := spell.Buff.DefaultScope
	if escopo != nil {
		scope = *escopo
	}
	eff, err := p.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: "spell", Catalogid: spellID, Scope: scope,
		Modifiers: string(spell.Buff.Modifiers), Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, fmt.Errorf("gravar o efeito de %q na ficha %d: %w", spellID, characterID, err)
	}
	return effectFrom(eff), nil
}

// ApplyTempHpPool liga uma reserva de PV temporários.
//
// Ela era uma TRANSAÇÃO de quatro passos — listar os efeitos, planejar um
// "vale-o-maior", apagar/zerar as poças deslocadas, gravar a nova — e virou um
// `UPSERT` só quando aquela regra deixou de existir (ALE-347). A **p106** manda
// SOMAR os pontos temporários, e o `domain/sheet/temp_hp.go` explica por que a
// ordem de drenagem é escolha nossa e o empilhamento é do motor.
//
// O `UPSERT` continua fazendo o trabalho que sobrou: reentrar na mesma Fúria
// REESCREVE a poça daquele poder em vez de abrir uma segunda, porque a chave é
// (personagem, catálogo, escopo). Fonte repetida não empilha; fontes
// diferentes, sim.
func (p Plays) ApplyTempHpPool(
	ctx context.Context, characterID int64, fonte, catalogID, escopo string, quanto int, nota string,
) (sheet.EffectDTO, error) {
	mods := []map[string]any{
		{"target": map[string]any{"k": "tempHp"}, "amount": quanto, "bonusType": "untyped", "note": nota},
	}
	modJSON, _ := json.Marshal(mods)

	eff, err := p.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: fonte, Catalogid: catalogID, Scope: escopo,
		Modifiers: string(modJSON), Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		return sheet.EffectDTO{}, fmt.Errorf("gravar a poça de %d PV temporários de %q: %w", quanto, catalogID, err)
	}
	return effectFrom(eff), nil
}

// TempHpAmount é quanto de PV temporário um poder concede: o NÍVEL do
// personagem mais o total computado do atributo-chave.
//
// Devolve `false` em vez de erro quando não dá para responder — motor ausente,
// ficha que não computa, atributo que o poder cita e a ficha não tem. Quem
// chama trata os três do mesmo jeito: não concede nada. Uma concessão de zero
// gravaria uma linha de efeito que a aba mostraria valendo nada.
func (p Plays) TempHpAmount(ctx context.Context, row sqlcgen.Character, atributo string) (int, bool) {
	if p.catalogs == nil {
		return 0, false
	}
	computada, err := sheet.LoadAndCompute(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return 0, false
	}
	attr, tem := computada.Attributes[atributo]
	if !tem {
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
