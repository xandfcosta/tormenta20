package table

import (
	"t20engine/domain/live"
)

// O QUE A FILA EMPRESTA AO TABULEIRO.
//
// Três consultas que leem o estado da SESSÃO para o mapa desenhar: quem tem
// bloco de criatura, quanto de PV sobrou em cada linha, e de quem é a vez. Elas
// saíram do `board_view.go` por terem outra razão para mudar — ele muda quando
// o desenho do mapa muda, e isto quando a FILA muda —, e foi o teto de 500
// linhas que cobrou a separação.

// blocosDaFila diz quais combatentes têm bloco de criatura do mestre.
//
// Mapa por `entryId` como a saúde, e pela mesma razão: não é do tabuleiro, é da
// FILA, e o tabuleiro só mostra. Ele decide se o menu da peça OFERECE o
// "com bloco próprio" — oferecer o que o servidor vai recusar é desenhar um erro,
// que é o que o `sessionConfig` já escreve com todas as letras.
func blocosDaFila(st *live.SessionRuntimeState) map[string]bool {
	withBlock := map[string]bool{}
	if st == nil {
		return withBlock
	}
	for i := range st.Initiative {
		if st.Initiative[i].CreatureID != nil {
			withBlock[st.Initiative[i].ID] = true
		}
	}
	return withBlock
}

// saudeDaFila é quanto de PV resta a cada combatente, em porcentagem.
//
// Lê o estado JÁ REDIGIDO: o combatente cujo PV o mestre ocultou chega sem
// `HpMax`, não entra no mapa, e a peça dele sai sem barra. É assim que a redação
// por papel alcança o tabuleiro sem uma segunda decisão sobre quem vê o quê.
func saudeDaFila(st *live.SessionRuntimeState) map[string]int {
	health := map[string]int{}
	if st == nil {
		return health
	}
	for i := range st.Initiative {
		e := &st.Initiative[i]
		if e.HpMax == nil || *e.HpMax <= 0 {
			continue
		}
		health[e.ID] = tableBarOf(live.DerefOr(e.HpCurrent, 0), *e.HpMax, false).Pct
	}
	return health
}

// turnCombatant é o `entryId` de quem está na vez, ou vazio fora de combate.
// A peça acende com o MESMO dourado da linha, porque é o mesmo fato.
func turnCombatant(st *live.SessionRuntimeState) string {
	if st == nil || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return ""
	}
	return st.Initiative[st.TurnIndex].ID
}
