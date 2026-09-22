package table

import (
	"context"
	"slices"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ui"
)

// O QUE A MESA LÊ DA FICHA a cada desenho: as condições do personagem
// (ALE-368) e, delas, a palavra de quem caiu — morrendo, estável, morto (T20
// p236, ALE-366).
//
// Lido a cada desenho pelo mesmo motivo que a reserva de PV temporário é: um
// dado espelhado na linha da fila envelheceria no primeiro sítio de escrita que
// esquecesse dele — e a lista de condições espelhada foi exatamente isso.

// characterIDsOnTable são as fichas que a tela desenha: na fila e no grupo,
// sem repetir.
func characterIDsOnTable(st *live.SessionRuntimeState, group []Member) []int64 {
	ids := []int64{}
	for i := range st.Initiative {
		if id := st.Initiative[i].CharacterID; id != nil && !slices.Contains(ids, *id) {
			ids = append(ids, *id)
		}
	}
	for i := range group {
		if !slices.Contains(ids, group[i].CharacterID) {
			ids = append(ids, group[i].CharacterID)
		}
	}
	return ids
}

// sheetConditionsOf lê as condições de cada ficha na mesa — o que a linha da
// fila MOSTRA para personagem (ALE-368) e de onde sai o Sangrando que separa
// morrendo de estável. Leitura que falha devolve nenhuma: a linha sai sem
// crachá e sem palavra, e não com um errado.
func (s Scene) sheetConditionsOf(ctx context.Context, ids []int64) map[int64][]string {
	conditions := map[int64][]string{}
	if len(ids) == 0 {
		return conditions
	}
	rows, err := s.deps.Queries().ListCharactersByIDs(ctx, ids)
	if err != nil {
		return conditions
	}
	for _, r := range rows {
		conditions[r.ID] = sheet.UnmarshalStrings(r.Activeconditions)
	}
	return conditions
}

// markDowned escreve a palavra numa barra de PV de ficha. Barra escondida pelo
// mestre não ganha palavra: ela contaria à mesa o que o mestre escondeu.
func markDowned(b *tableBar, conditions []string) {
	if b == nil || b.Hidden || b.Max <= 0 {
		return
	}
	b.Down = ui.DownedWord(b.Current, b.Max, slices.Contains(conditions, engine.ConditionBleeding))
}

// withSheetState põe nas linhas da fila que têm ficha atrás o que é DA FICHA:
// as condições (o crachá) e a palavra de quem caiu. A fila da vista nasce na
// ordem da fila do estado, e é esse o casamento.
func withSheetState(v *View, st *live.SessionRuntimeState, conditions map[int64][]string) {
	for i := range v.Queue {
		if i >= len(st.Initiative) || st.Initiative[i].CharacterID == nil {
			continue
		}
		id := *st.Initiative[i].CharacterID
		v.Queue[i].Conditions = conditions[id]
		markDowned(v.Queue[i].PV, conditions[id])
	}
}
