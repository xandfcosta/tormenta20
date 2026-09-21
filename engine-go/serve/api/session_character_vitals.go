package api

import (
	"context"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O PV do rastreador É o PV da ficha, e ele atravessa uma PORTA.
//
// Este arquivo implementa `live.SheetVitals`. Os métodos moram deste lado porque
// a regra do dano — drenar os PV temporários antes dos reais — é da FICHA, e um
// pacote do regime ao vivo não pode conhecê-la. O regime declara o que precisa;
// quem entrega fica aqui.
//
// A LINHA DO PERSONAGEM é a fonte, e não um blob no estado da sessão. Com duas
// fontes a mesma tela mostra 52/95 na iniciativa e 57/95 no card do grupo, e o
// caminho do socket ignora PV TEMPORÁRIOS: bater 5 num personagem com Armadura
// Arcana cobraria dos PV reais, enquanto o mesmo 5 pela ficha drena o pool
// primeiro — duas regras para a mesma pancada. NPC continua vivendo só no
// rastreador: não há ficha atrás dele.

// sheetVitals é quem cumpre a porta. Guarda só o que precisa — as queries e um
// jeito de pedir o motor do momento — em vez de um `*Server` inteiro: uma porta
// que recebesse o servidor não seria porta, seria o acoplamento de antes com
// outro nome.
//
// O motor vem por FUNÇÃO e não por valor porque o `primeCatalogs` o troca depois
// do `NewServer`: um ponteiro copiado na montagem seria o motor de antes, que é
// a mesma armadilha que aquele método documenta para a cena da Mesa.
type sheetVitals struct {
	q        *sqlcgen.Queries
	catalogs func() *engine.Catalogs
}

// drainTempHpAndHit roda a ordem do livro — os poços temporários primeiro, o
// maior antes — e devolve o PV que sobra nos reais, já gravadas as drenagens.
//
// O PV com que ela decide é o DERIVADO, que chega pelo funil: com a coluna crua,
// um personagem cujo máximo mudou por catálogo apanharia a partir do PV de
// ontem.
func drainTempHpAndHit(
	ctx context.Context, q *sqlcgen.Queries, characterID int64, hpCurrent int64, amount int,
) (sheet.DamagePlan, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, characterID)
	if err != nil {
		return sheet.DamagePlan{}, err
	}
	plan := sheet.PlanDamage(sheet.ParseTempHpPools(effects), int(hpCurrent), amount)
	for _, u := range plan.Updates {
		if err := q.UpdateEffectModifiers(ctx, sqlcgen.UpdateEffectModifiersParams{
			Modifiers: u.Modifiers, ID: u.EffectID,
		}); err != nil {
			return sheet.DamagePlan{}, err
		}
	}
	for _, delID := range plan.DeleteIDs {
		if err := q.DeleteEffectByID(ctx, delID); err != nil {
			return sheet.DamagePlan{}, err
		}
	}
	return plan, nil
}

// ApplyDelta move o PV/PM de um personagem por um delta e grava, devolvendo os
// valores que a entrada do rastreador tem de espelhar.
//
// Dano (PV negativo) passa pela drenagem dos temporários; cura e PM são presos
// na faixa pelo funil.
func (v sheetVitals) ApplyDelta(
	ctx context.Context, charID int64, hpDelta, mpDelta *int64,
) (*int64, *int64, error) {
	return v.applyRule(ctx, charID, func(p sheet.Pools) (sheet.Pools, error) {
		if hpDelta != nil && *hpDelta < 0 {
			plan, err := drainTempHpAndHit(ctx, v.q, charID, p.HpCurrent, int(-*hpDelta))
			if err != nil {
				return p, err
			}
			p.HpCurrent = int64(plan.HpCurrent)
		} else if hpDelta != nil {
			p.HpCurrent += *hpDelta
		}
		if mpDelta != nil {
			p.MpCurrent += *mpDelta
		}
		return p, nil
	})
}

// ApplyAbsolute crava PV/PM absolutos no personagem (o "vitals-patch" do
// rastreador). Um valor absoluto é uma afirmação sobre o total, e não uma
// pancada, então ele NÃO drena os poços temporários — essa regra é do dano.
func (v sheetVitals) ApplyAbsolute(
	ctx context.Context, charID int64, hpCurrent, mpCurrent *int64,
) (*int64, *int64, error) {
	return v.applyRule(ctx, charID, func(p sheet.Pools) (sheet.Pools, error) {
		if hpCurrent != nil {
			p.HpCurrent = *hpCurrent
		}
		if mpCurrent != nil {
			p.MpCurrent = *mpCurrent
		}
		return p, nil
	})
}

// applyRule é o que os dois gestos têm em comum: achar a linha, passar pelo
// funil e devolver os DOIS vitais — inclusive o que o gesto não tocou, senão o
// rastreador voltaria a mostrar um número que a ficha não tem.
func (v sheetVitals) applyRule(
	ctx context.Context, charID int64, rule sheet.PoolRule,
) (*int64, *int64, error) {
	row, err := v.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	pools, err := sheet.ApplyToPools(ctx, v.q, v.catalogs(), row, rule)
	if err != nil {
		return nil, nil, err
	}
	return &pools.HpCurrent, &pools.MpCurrent, nil
}

// PoolsOf cumpre a metade de LEITURA da porta: o poço derivado de cada
// personagem da fila.
//
// Ela traduz o `sheet.Pools` no `live.VitalPool` porque o regime não pode
// conhecer o tipo da ficha — é o mesmo motivo de a porta existir.
func (v sheetVitals) PoolsOf(
	ctx context.Context, charIDs []int64,
) (map[int64]live.VitalPool, error) {
	pools, err := sheet.PoolsForCharacters(ctx, v.q, v.catalogs(), charIDs)
	if err != nil {
		return nil, err
	}
	fromQueue := make(map[int64]live.VitalPool, len(pools))
	for id, p := range pools {
		fromQueue[id] = live.VitalPool{
			HpMax: p.HpMax, HpCurrent: p.HpCurrent, MpMax: p.MpMax, MpCurrent: p.MpCurrent,
		}
	}
	return fromQueue, nil
}

// SUSTENTADA: o que a manutenção do turno precisa da ficha (T20 p227).
//
// O adaptador é o MESMO da `SheetVitals` porque a fonte é a mesma tabela e a
// mesma conexão; as portas é que são duas, porque mudam por razões diferentes.

// SustainedOf lista os efeitos que cobram mana por turno, do mais antigo para o
// mais novo — a ordem em que serão pagos quando o mana não cobrir todos.
func (v sheetVitals) SustainedOf(ctx context.Context, charID int64) ([]live.SustainedEffect, error) {
	rows, err := v.q.ListActiveEffectsByCharacter(ctx, charID)
	if err != nil {
		return nil, err
	}
	var sustained []live.SustainedEffect
	for _, l := range rows {
		duration, err := engine.ParseDuration(l.Scope)
		if err != nil || duration.Kind != engine.DurationSustained {
			continue
		}
		sustained = append(sustained, live.SustainedEffect{
			CatalogID: l.Catalogid, Label: spellLabel(l.Catalogid),
		})
	}
	return sustained, nil
}

// EndSustained derruba o efeito que não foi pago.
func (v sheetVitals) EndSustained(ctx context.Context, charID int64, catalogID string) error {
	return v.q.DeleteEffectsByCatalog(ctx, sqlcgen.DeleteEffectsByCatalogParams{
		Characterid: charID, Catalogid: catalogID,
	})
}

// ExpireTurnEffects derruba o que durava a vez que acabou.
func (v sheetVitals) ExpireTurnEffects(ctx context.Context, charID int64) error {
	return v.q.DeleteEffectsByScope(ctx, sqlcgen.DeleteEffectsByScopeParams{
		Characterid: charID, Scope: engine.TurnScope(),
	})
}

// ConditionsOf lê as condições ligadas na ficha.
func (v sheetVitals) ConditionsOf(ctx context.Context, charID int64) ([]string, error) {
	row, err := v.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, err
	}
	return sheet.UnmarshalStrings(row.Activeconditions), nil
}

// spellLabel é o nome que a MESA lê. Sem verbete, o id serve: um extrato que
// diz "velocidade" ainda responde qual efeito caiu, e um extrato vazio não.
//
// Ele mora AQUI e não no motor porque o `engine.Catalogs` não carrega magias —
// e é por isso que a aba Efeitos da ficha mostra "velocidade (cena)" no lugar
// do nome até hoje. Consertar aquilo é mexer no que o motor carrega, e é outra
// fatia.
func spellLabel(catalogID string) string {
	if spell, known := catalog.LookupSpell(catalogID); known && spell.Name != "" {
		return spell.Name
	}
	return catalogID
}
