package api

import (
	"context"

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
	ctx context.Context, charID int64, regra sheet.PoolRule,
) (*int64, *int64, error) {
	row, err := v.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	pocos, err := sheet.ApplyToPools(ctx, v.q, v.catalogs(), row, regra)
	if err != nil {
		return nil, nil, err
	}
	return &pocos.HpCurrent, &pocos.MpCurrent, nil
}

// PoolsOf cumpre a metade de LEITURA da porta: o poço derivado de cada
// personagem da fila.
//
// Ela traduz o `sheet.Pools` no `live.VitalPool` porque o regime não pode
// conhecer o tipo da ficha — é o mesmo motivo de a porta existir.
func (v sheetVitals) PoolsOf(
	ctx context.Context, charIDs []int64,
) (map[int64]live.VitalPool, error) {
	pocos, err := sheet.PoolsForCharacters(ctx, v.q, v.catalogs(), charIDs)
	if err != nil {
		return nil, err
	}
	daFila := make(map[int64]live.VitalPool, len(pocos))
	for id, p := range pocos {
		daFila[id] = live.VitalPool{
			HpMax: p.HpMax, HpCurrent: p.HpCurrent, MpMax: p.MpMax, MpCurrent: p.MpCurrent,
		}
	}
	return daFila, nil
}
