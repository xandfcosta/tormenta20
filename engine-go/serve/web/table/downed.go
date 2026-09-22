package table

import (
	"context"
	"slices"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ui"
)

// QUEM CAIU, na barra: a palavra do estado ao lado do PV de uma FICHA a 0 ou
// menos — morrendo, estável, morto (T20 p236, ALE-366).
//
// O Sangrando que separa morrendo de estável mora na FICHA, e é lido a cada
// desenho pelo mesmo motivo que a reserva de PV temporário é: um terceiro dado
// espelhado na linha da fila envelheceria no primeiro sítio de escrita que
// esquecesse dele.

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

// bleedingOf diz quais dessas fichas estão com a condição Sangrando. Leitura que
// falha devolve ninguém: a barra sai sem a palavra, e não com uma errada.
func (s Scene) bleedingOf(ctx context.Context, ids []int64) map[int64]bool {
	bleeding := map[int64]bool{}
	if len(ids) == 0 {
		return bleeding
	}
	rows, err := s.deps.Queries().ListCharactersByIDs(ctx, ids)
	if err != nil {
		return bleeding
	}
	for _, r := range rows {
		bleeding[r.ID] = slices.Contains(sheet.UnmarshalStrings(r.Activeconditions), engine.ConditionBleeding)
	}
	return bleeding
}

// markDowned escreve a palavra numa barra de PV de ficha. Barra escondida pelo
// mestre não ganha palavra: ela contaria à mesa o que o mestre escondeu.
func markDowned(b *tableBar, characterID int64, bleeding map[int64]bool) {
	if b == nil || b.Hidden || b.Max <= 0 {
		return
	}
	b.Down = ui.DownedWord(b.Current, b.Max, bleeding[characterID])
}

// markQueueDowned faz o mesmo nas linhas da fila que têm ficha atrás. A fila da
// vista nasce na ordem da fila do estado, e é esse o casamento.
func markQueueDowned(v *View, st *live.SessionRuntimeState, bleeding map[int64]bool) {
	for i := range v.Queue {
		if i < len(st.Initiative) && st.Initiative[i].CharacterID != nil {
			markDowned(v.Queue[i].PV, *st.Initiative[i].CharacterID, bleeding)
		}
	}
}
