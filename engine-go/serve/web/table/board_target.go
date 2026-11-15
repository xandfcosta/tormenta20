package table

import (
	"fmt"

	"t20engine/domain/board"
	"t20engine/domain/live"
)

// AGIR NO TABULEIRO (ALE-660, mockup aprovado pelo dono). Tocar na peça de outra linha da fila escolhe o alvo da
// vez do jogador — o mesmo `$turn_target` do "Contra" das Ações — e abre o painel, que é a própria superfície
// Ações num modo por cima do mapa (`ds-board-panel`). O cabeçalho do painel mora AQUI, no tabuleiro, e não na
// superfície: o remendo do tabuleiro chega a cada passo de peça, e a distância andaria atrasada no outro lugar.
//
// Só do jogador (decisão do dono): o mestre tem o menu da peça, que é outro gesto.

// picksTargets marca as peças que o jogador pode escolher como alvo, e a distância até cada uma, medida da peça
// dele que está na vez — ou da primeira dele no mapa, fora da vez.
func picksTargets(v *BoardView, b *board.BoardState, st *live.SessionRuntimeState, mine map[int64]bool) {
	if v.GM || b == nil || st == nil {
		return
	}
	from := viewerToken(b, st, mine)
	for i := range v.Tokens {
		token := &v.Tokens[i]
		if token.EntryID == "" || entryIsMine(st, token.EntryID, mine) {
			continue
		}
		token.PicksTarget = true
		to := board.FindToken(b, token.ID)
		if from != nil && to != nil {
			token.Reach = reachSentence(board.ApartBetween(*from, *to))
		}
	}
}

// viewerToken é a peça de onde o jogador mede: a dele que está na vez, ou a primeira dele no mapa.
func viewerToken(b *board.BoardState, st *live.SessionRuntimeState, mine map[int64]bool) *board.BoardToken {
	var first *board.BoardToken
	for i := range b.Tokens {
		token := &b.Tokens[i]
		if token.EntryID == nil || !entryIsMine(st, *token.EntryID, mine) {
			continue
		}
		if board.IsTokenOnTurn(token, st) {
			return token
		}
		if first == nil {
			first = token
		}
	}
	return first
}

// entryIsMine diz que a linha da fila é de um personagem de quem olha.
func entryIsMine(st *live.SessionRuntimeState, entryID string, mine map[int64]bool) bool {
	if i := live.FindEntryIndex(st, entryID); i >= 0 {
		if id := st.Initiative[i].CharacterID; id != nil {
			return mine[*id]
		}
	}
	return false
}

// reachSentence é a distância como a mesa fala: "a 1,5m · corpo a corpo" quando o alcance natural chega, e a
// faixa do livro (p224) quando não — "a 4,5m · alcance curto". O sistema informa; quem diz se o golpe alcança é o
// mestre.
func reachSentence(apart board.Apart) string {
	where := rulerBand(apart.Measurement)
	if apart.Reaches {
		where = "corpo a corpo"
	}
	return fmt.Sprintf("a %sm · %s", meters(apart.Metres), where)
}

// boardPanelSheetShare é a fração da altura da janela que a folha do painel ocupa no celular — o `52vh` do
// `.ds-board-panel` no `ds.css`. Os dois andam juntos: com a folha maior que a conta, a peça sobe para baixo dela.
const boardPanelSheetShare = 0.52

// picksTheTarget é o toque na peça: escolhe o alvo da vez e abre o painel. No celular, onde o painel é uma folha
// que sobe de baixo, ele traz a peça para a faixa ACIMA da folha, e só se ela ficaria coberta — o "rolar o mínimo"
// do `viewportFollowsFocus`: o alvo tem de continuar à vista enquanto se escolhe o que fazer com ele.
func picksTheTarget(p boardToken) string {
	center := fmt.Sprintf("(%d + %d / 2) * $square", p.Y, max(p.Footprint, 1))
	return fmt.Sprintf("$turn_target = '%s'; $board_panel = true; "+
		"if (window.matchMedia('(max-width: 63.99rem)').matches) { "+
		"const cena = document.getElementById(%q); const visivel = cena.clientHeight - window.innerHeight * %.2f; "+
		"if (%s - $viewport_y > visivel - $square) { $viewport_y = %s - visivel / 2 } }",
		p.EntryID, sceneId, boardPanelSheetShare, center, center)
}

// chosenTargetStyling acrescenta à peça a marca de alvo escolhido.
func chosenTargetStyling(styling string, p boardToken) string {
	if !p.PicksTarget {
		return styling
	}
	return styling[:len(styling)-1] + fmt.Sprintf(", 'board-token-chosen': $turn_target === '%s'}", p.EntryID)
}

// actionsSurfaceShows é quando a superfície Ações aparece: na aba dela, e no tabuleiro com o painel aberto.
func actionsSurfaceShows() string {
	return surface(superficieDasAcoes) + " || (" + surface(superficieDoTabuleiro) + " && $board_panel)"
}

// actionsSurfaceMode é o modo de painel: no tabuleiro, a superfície flutua por cima do mapa e mostra só as linhas
// que pedem alvo (o CSS de `.ds-board-panel`).
func actionsSurfaceMode() string {
	return "{'ds-board-panel': " + surface(superficieDoTabuleiro) + "}"
}
