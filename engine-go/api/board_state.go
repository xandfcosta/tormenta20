package api

import (
	"fmt"
)

// boardMaxTokens — teto de peças num tabuleiro. Espelha o `initiativeMaxEntries`
// pelo mesmo motivo: sem teto, o estado cresce sem limite e TODO broadcast o
// carrega. Vinte tokens é uma mesa cheia; 200 é um acidente.
const boardMaxTokens = 200

// boardMaxSide — o maior lado aceito da grade, em quadrados. 60 quadrados são
// 90m (T20 p236: 1 quadrado = 1,5m), que é o alcance longo do livro (p224): um
// mapa maior que isso não cabe em nenhuma magia nem em nenhuma vista.
const boardMaxSide = 60

// BoardToken é uma peça no tabuleiro. X/Y são o canto superior-esquerdo em
// QUADRADOS, nunca em pixels: pixel amarraria o estado a um tamanho de tela, e
// o celular e o desktop passariam a discordar sobre onde o ogro está.
type BoardToken struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	// Footprint é o LADO da peça em quadrados (T20 p107, Tab. 1-21): Minúsculo,
	// Pequeno e Médio ocupam 1; Grande 2; Enorme 3; Colossal 6. Não existe 4 nem 5.
	Footprint int `json:"footprint"`
	// Kind separa quem luta de quem é cenário: "character" | "npc" | "object".
	Kind string `json:"kind"`
	// EntryID amarra a peça à linha da iniciativa — é por ele que o servidor
	// saberá de quem é a vez. Ausente em objeto: uma porta não tem turno.
	EntryID     *string `json:"entryId,omitempty"`
	CharacterID *int64  `json:"characterId,omitempty"`
	// Hidden: o mestre escondeu a peça. Diferente do `hpHidden` da iniciativa,
	// onde a linha SOBREVIVE sem os números — aqui a peça some inteira da cópia
	// do jogador, porque a existência dela é a emboscada.
	Hidden bool `json:"hidden,omitempty"`
}

// BoardState é o tabuleiro vivo de uma sessão. Ausente (sem linha em
// session_boards) = sessão sem tabuleiro, que é estado diferente de tabuleiro
// vazio: o segundo desenharia uma grade de 0×0 na tela.
type BoardState struct {
	// Version sobe a cada mutação aceita. É o que vai permitir recusar um
	// movimento proposto sobre um tabuleiro que já mudou, e o que deixa o
	// cliente descartar um broadcast atrasado depois de reconectar.
	Version int64 `json:"version"`
	// Place é o nome do lugar ("Taverna do Javali") — o mestre está montando uma
	// cena, não uma planilha.
	Place   string       `json:"place"`
	Cols    int          `json:"cols"`
	Rows    int          `json:"rows"`
	Terrain string       `json:"terrain"`
	Tokens  []BoardToken `json:"tokens"`
}

// newBoard abre um tabuleiro. Recusa grade fora da faixa jogável com o valor
// ofendido na mensagem — quem chamou precisa saber o que mandou.
func newBoard(place string, cols, rows int, terrain string) (*BoardState, error) {
	if cols < 1 || cols > boardMaxSide || rows < 1 || rows > boardMaxSide {
		return nil, fmt.Errorf("grade %dx%d fora da faixa: cada lado vai de 1 a %d quadrados", cols, rows, boardMaxSide)
	}
	return &BoardState{Version: 1, Place: place, Cols: cols, Rows: rows, Terrain: terrain, Tokens: []BoardToken{}}, nil
}

// addToken põe uma peça no tabuleiro, recusando o que sairia da grade.
func addToken(b *BoardState, t BoardToken, newID func() string) error {
	if len(b.Tokens) >= boardMaxTokens {
		return fmt.Errorf("o tabuleiro já tem %d peças (teto %d)", len(b.Tokens), boardMaxTokens)
	}
	if t.Footprint <= 0 {
		t.Footprint = 1
	}
	if err := assertInsideBoard(b, t); err != nil {
		return err
	}
	t.ID = newID()
	b.Tokens = append(b.Tokens, t)
	b.Version++
	return nil
}

// removeToken tira a peça do tabuleiro. Some em silêncio se ela já não está lá:
// dois cliques no mesmo botão não são erro do usuário.
func removeToken(b *BoardState, tokenID string) {
	for i, t := range b.Tokens {
		if t.ID != tokenID {
			continue
		}
		b.Tokens = append(b.Tokens[:i], b.Tokens[i+1:]...)
		b.Version++
		return
	}
}

// tokenPatch é a alteração parcial de uma peça: só os campos não-nulos entram,
// para "não mexer" ficar distinto de "zerar".
type tokenPatch struct {
	Label     *string `json:"label"`
	Hidden    *bool   `json:"hidden"`
	Footprint *int    `json:"footprint"`
	X         *int    `json:"x"`
	Y         *int    `json:"y"`
}

// updateToken aplica o patch, validando a posição RESULTANTE — mudar só o
// footprint pode jogar uma peça de 3×3 para fora da grade sem que ninguém tenha
// mexido em X ou Y.
func updateToken(b *BoardState, tokenID string, patch tokenPatch) error {
	for i := range b.Tokens {
		t := &b.Tokens[i]
		if t.ID != tokenID {
			continue
		}
		next := *t
		applyTokenPatch(&next, patch)
		if err := assertInsideBoard(b, next); err != nil {
			return err
		}
		*t = next
		b.Version++
		return nil
	}
	return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
}

func applyTokenPatch(t *BoardToken, patch tokenPatch) {
	if patch.Label != nil {
		t.Label = *patch.Label
	}
	if patch.Hidden != nil {
		t.Hidden = *patch.Hidden
	}
	if patch.Footprint != nil && *patch.Footprint > 0 {
		t.Footprint = *patch.Footprint
	}
	if patch.X != nil {
		t.X = *patch.X
	}
	if patch.Y != nil {
		t.Y = *patch.Y
	}
}

// assertInsideBoard cobra a grade inteira da peça, e não só o canto: uma peça
// Grande (2×2, p107) ancorada na última coluna teria metade do corpo fora.
func assertInsideBoard(b *BoardState, t BoardToken) error {
	side := t.Footprint
	if side <= 0 {
		side = 1
	}
	if t.X < 0 || t.Y < 0 || t.X+side > b.Cols || t.Y+side > b.Rows {
		return fmt.Errorf("peça %dx%d em (%d,%d) sai da grade de %dx%d", side, side, t.X, t.Y, b.Cols, b.Rows)
	}
	return nil
}

// populateBoard traz para o tabuleiro cada linha da iniciativa que ainda não
// tem peça, em fileira a partir do canto. Idempotente de propósito, como o
// `populateParty` do rastreador: clicar duas vezes não duplica ninguém.
func populateBoard(b *BoardState, st *SessionRuntimeState, newID func() string) int {
	placed := 0
	for _, entry := range st.Initiative {
		if hasTokenForEntry(b, entry.ID) {
			continue
		}
		token := BoardToken{
			Label: entry.Label, Kind: entry.Type, Footprint: 1,
			EntryID: strPtr(entry.ID), CharacterID: entry.CharacterID,
		}
		spot, ok := nextFreeSpot(b)
		if !ok {
			break // grade cheia: para em vez de empilhar todo mundo no canto
		}
		token.X, token.Y = spot.x, spot.y
		if err := addToken(b, token, newID); err != nil {
			break
		}
		placed++
	}
	return placed
}

func hasTokenForEntry(b *BoardState, entryID string) bool {
	for _, t := range b.Tokens {
		if t.EntryID != nil && *t.EntryID == entryID {
			return true
		}
	}
	return false
}

type boardSpot struct{ x, y int }

// nextFreeSpot devolve o primeiro quadrado vazio em ordem de leitura. Não
// precisa de offset: cada peça entra antes da busca seguinte, então a varredura
// já enxerga quem acabou de chegar. Varredura simples porque o teto é de 200
// peças — um índice espacial aqui seria estrutura para um problema que não existe.
func nextFreeSpot(b *BoardState) (boardSpot, bool) {
	for y := 0; y < b.Rows; y++ {
		for x := 0; x < b.Cols; x++ {
			if !occupied(b, x, y) {
				return boardSpot{x, y}, true
			}
		}
	}
	return boardSpot{}, false
}

func occupied(b *BoardState, x, y int) bool {
	for _, t := range b.Tokens {
		side := t.Footprint
		if side <= 0 {
			side = 1
		}
		if x >= t.X && x < t.X+side && y >= t.Y && y < t.Y+side {
			return true
		}
	}
	return false
}

// boardForRole é o tabuleiro como UM papel pode vê-lo. Papel desconhecido cai em
// jogador: errar para o lado que MOSTRA seria vazar por omissão, a mesma regra
// do `stateForRole`.
func boardForRole(role string, b *BoardState) *BoardState {
	if b == nil || role == "gm" {
		return b
	}
	return redactBoardForPlayers(b)
}

// redactBoardForPlayers apaga da cópia do jogador as peças que o mestre
// escondeu. Some a peça INTEIRA — e essa é a assimetria deliberada em relação ao
// `hpHidden` da iniciativa, onde a linha fica sem os números: aqui a existência
// da peça é a informação, e uma peça "presente porém anônima" entregaria a
// emboscada do mesmo jeito (ALE-124).
func redactBoardForPlayers(b *BoardState) *BoardState {
	out := *b
	out.Tokens = make([]BoardToken, 0, len(b.Tokens))
	for _, t := range b.Tokens {
		if t.Hidden {
			continue
		}
		out.Tokens = append(out.Tokens, t)
	}
	return &out
}

func strPtr(s string) *string { return &s }
