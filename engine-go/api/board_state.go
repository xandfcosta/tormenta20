package api

import (
	"fmt"
)

// boardMaxTokens — teto de peças num tabuleiro. Espelha o `initiativeMaxEntries`
// pelo mesmo motivo: sem teto, o estado cresce sem limite e TODO broadcast o
// carrega. Vinte tokens é uma mesa cheia; 200 é um acidente.
const boardMaxTokens = 200

// boardCoordLimit — o tabuleiro é INFINITO e este número não é uma borda: é um
// guarda contra lixo. 5000 quadrados são 7,5km (T20 p236: 1 quadrado = 1,5m), e
// nenhuma cena de mesa chega perto disso; uma peça em 10^9 só pode ter vindo de
// um cliente quebrado, e aceitá-la estouraria a serialização e a tela.
const boardCoordLimit = 5000

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
// vazio: o segundo é uma cena aberta e ainda sem peça.
//
// NÃO tem largura nem altura: o plano é INFINITO nas quatro direções, e a peça
// pode estar em coordenada negativa. Quem tem tamanho é a JANELA, que mora no
// cliente — dois jogadores olhando pedaços diferentes da mesma cena é uma
// propriedade, não um bug.
type BoardState struct {
	// Version sobe a cada mutação aceita. É o que vai permitir recusar um
	// movimento proposto sobre um tabuleiro que já mudou, e o que deixa o
	// cliente descartar um broadcast atrasado depois de reconectar.
	Version int64 `json:"version"`
	// Place é o nome do lugar ("Taverna do Javali") — o mestre está montando uma
	// cena, não uma planilha.
	Place   string       `json:"place"`
	Terrain string       `json:"terrain"`
	Tokens  []BoardToken `json:"tokens"`
}

// newBoard abre um tabuleiro vazio num plano sem bordas.
func newBoard(place, terrain string) *BoardState {
	return &BoardState{Version: 1, Place: place, Terrain: terrain, Tokens: []BoardToken{}}
}

// addToken põe uma peça no tabuleiro, recusando o que sairia da grade.
func addToken(b *BoardState, t BoardToken, newID func() string) error {
	if len(b.Tokens) >= boardMaxTokens {
		return fmt.Errorf("o tabuleiro já tem %d peças (teto %d)", len(b.Tokens), boardMaxTokens)
	}
	if t.Footprint <= 0 {
		t.Footprint = 1
	}
	if err := assertSaneCoords(t); err != nil {
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

// updateToken aplica o patch. Não há borda para respeitar — só o guarda contra
// coordenada absurda, que é sobre lixo de cliente e não sobre o mapa.
func updateToken(b *BoardState, tokenID string, patch tokenPatch) error {
	for i := range b.Tokens {
		t := &b.Tokens[i]
		if t.ID != tokenID {
			continue
		}
		next := *t
		applyTokenPatch(&next, patch)
		if err := assertSaneCoords(next); err != nil {
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

// assertSaneCoords recusa coordenada que só pode ter vindo de cliente quebrado.
// Não é borda do mapa — o mapa não tem borda; é o guarda que impede um número
// absurdo de estourar a serialização e a tela de todo mundo na mesa.
func assertSaneCoords(t BoardToken) error {
	if abs(t.X) > boardCoordLimit || abs(t.Y) > boardCoordLimit {
		return fmt.Errorf("peça em (%d,%d) está além do limite de sanidade de %d quadrados", t.X, t.Y, boardCoordLimit)
	}
	return nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
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
		spot := nextFreeSpot(b)
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

// nextFreeSpot devolve o primeiro quadrado vazio de uma fileira que começa na
// origem e cresce para a direita, quebrando a cada dez quadrados. Num plano sem
// bordas não existe "varrer a grade": existe um lugar combinado onde a peça
// nova aparece, e a origem é ele — o mestre acha o grupo em "Centralizar".
//
// Não precisa de offset: cada peça entra antes da busca seguinte, então a
// varredura já enxerga quem acabou de chegar.
func nextFreeSpot(b *BoardState) boardSpot {
	for i := 0; ; i++ {
		spot := boardSpot{x: i % boardRowWidth, y: i / boardRowWidth}
		if !occupied(b, spot.x, spot.y) {
			return spot
		}
	}
}

// boardRowWidth é o comprimento da fileira em que as peças novas nascem. Dez
// quadrados são 15m: cabe numa tela e é a largura de uma sala.
const boardRowWidth = 10

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
