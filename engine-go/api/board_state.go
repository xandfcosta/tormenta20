package api

import (
	"fmt"

	"t20engine/engine"
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
	// SpeedSquares é o orçamento de movimento da peça em QUADRADOS (T20 p106:
	// 9m = 6 quadrados). Mora na peça porque a tela precisa dele ANTES de
	// propor — para contar o gasto e acender o que dá para alcançar —, e é
	// re-sincronizado do motor a cada proposta, como o `refreshCharacterMaxes`
	// faz com o `hpMax`. Zero = nunca medido; vale o padrão do livro.
	SpeedSquares int `json:"speedSquares,omitempty"`
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
	// Pending é o movimento proposto e ainda não confirmado — no máximo um.
	Pending *PendingMove `json:"pending,omitempty"`
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
		// O provisório daquela peça morre com ela: um movimento proposto para
		// quem não está mais no tabuleiro nunca poderia ser confirmado, e
		// ficaria pendurado no estado de todo mundo.
		if b.Pending != nil && b.Pending.TokenID == tokenID {
			b.Pending = nil
		}
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
	hidden := map[string]bool{}
	for _, t := range b.Tokens {
		if t.Hidden {
			hidden[t.ID] = true
			continue
		}
		out.Tokens = append(out.Tokens, t)
	}
	// O provisório de uma peça escondida entregaria a emboscada por outro
	// caminho: um caminho desenhado saindo do nada é a peça sem o círculo.
	if out.Pending != nil && hidden[out.Pending.TokenID] {
		out.Pending = nil
	}
	return &out
}

func strPtr(s string) *string { return &s }

// PendingMove é um movimento PROPOSTO e ainda não confirmado (ALE-124).
//
// Ele é ESTADO, e não um evento de arraste no fio. Todo broadcast desta casa
// carrega o estado inteiro; um fantasma a 60fps seriam kilobytes por quadro
// vezes N clientes, e outras tantas gravações do blob por segundo. Como estado,
// são duas mensagens — soltar e confirmar —, o provisório sobrevive à
// reconexão, e ele é redigível por papel como qualquer outra parte do tabuleiro.
//
// No máximo UM por tabuleiro: dois movimentos provisórios simultâneos são duas
// verdades sobre a mesma cena, e a mesa não teria como saber qual confirmar.
type PendingMove struct {
	TokenID string `json:"tokenId"`
	// Path é o caminho INTEIRO, do quadrado onde a peça está até o destino. O
	// custo depende do percurso (diagonal custa o dobro, T20 p238), então
	// guardar só o destino perderia a conta que a mesa acabou de ver.
	Path []engine.Square `json:"path"`
	Cost int             `json:"cost"`
	// Budget é o orçamento contra o qual o caminho foi medido, em quadrados, ou
	// -1 quando não há (mestre, ou cena fora de combate).
	Budget int `json:"budget"`
	// ByUserID é quem propôs. O mestre confirma por qualquer um; o jogador só
	// confirma o que ele mesmo propôs.
	ByUserID int64 `json:"byUserId"`
}

// boardDefaultSpeedSquares é o orçamento de quem não declarou deslocamento: 9m,
// o padrão do livro (T20 p106), que são 6 quadrados. Vale para o capanga que o
// mestre digitou à mão — inventar zero o deixaria pregado no chão, e inventar
// infinito tiraria a regra da mesa.
const boardDefaultSpeedSquares = 6

// speedOf devolve o orçamento da peça em quadrados, com o padrão do livro para
// quem nunca teve o deslocamento medido.
func speedOf(t BoardToken) int {
	if t.SpeedSquares > 0 {
		return t.SpeedSquares
	}
	return boardDefaultSpeedSquares
}

// findToken devolve o ponteiro para a peça viva (para mutação) ou nil.
func findToken(b *BoardState, tokenID string) *BoardToken {
	for i := range b.Tokens {
		if b.Tokens[i].ID == tokenID {
			return &b.Tokens[i]
		}
	}
	return nil
}

// mover descreve quem está tentando mexer numa peça. O papel vem do socket e a
// posse do banco; a decisão de deixar ou não é a função abaixo.
type mover struct {
	userID int64
	role   string
	// ownsCharacter: a peça é de um personagem DESTE usuário. Resolvido no
	// gateway, contra o banco — o cliente não é fonte de posse.
	ownsCharacter bool
}

// assertMovable responde "esta pessoa pode mover esta peça agora?" e, quando
// pode, com QUANTO de orçamento (T20 p106; -1 = sem orçamento).
//
// Três regras, e as duas exceções são deliberadas:
//   - o MESTRE move qualquer peça, a qualquer hora, sem orçamento — é a saída
//     para voo, empurrão, teleporte e "pode ir";
//   - FORA DE COMBATE (`turnIndex` < 0) não existe vez nem deslocamento de
//     turno: cada um anda com a própria peça, e o contador só informa;
//   - em combate, o jogador move a própria peça só na vez dela.
func assertMovable(b *BoardState, st *SessionRuntimeState, tokenID string, by mover) (*BoardToken, int, error) {
	token := findToken(b, tokenID)
	if token == nil {
		return nil, 0, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if by.role == "gm" {
		return token, -1, nil
	}
	if !by.ownsCharacter {
		return nil, 0, fmt.Errorf("a peça %q não é sua", token.Label)
	}
	if st == nil || st.TurnIndex < 0 {
		return token, -1, nil
	}
	if !isTokenOnTurn(token, st) {
		return nil, 0, fmt.Errorf("não é a vez de %s", token.Label)
	}
	return token, speedOf(*token), nil
}

// isTokenOnTurn amarra a peça à LINHA da iniciativa: a vez não é copiada para o
// tabuleiro, ela é perguntada ao rastreador — duas cópias da vez divergiriam no
// primeiro turno passado com o tabuleiro fechado.
func isTokenOnTurn(token *BoardToken, st *SessionRuntimeState) bool {
	if token.EntryID == nil || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return false
	}
	return st.Initiative[st.TurnIndex].ID == *token.EntryID
}

// proposeMove mede o caminho e guarda o provisório. Recusa o que estoura o
// deslocamento: a decisão do dono é BLOQUEAR no limite, e as saídas são o
// mestre (sem orçamento) e a cena fora de combate.
func proposeMove(b *BoardState, st *SessionRuntimeState, tokenID string, path []engine.Square, by mover) error {
	token, budget, err := assertMovable(b, st, tokenID, by)
	if err != nil {
		return err
	}
	if len(path) < 2 {
		return fmt.Errorf("o caminho tem %d quadrados: precisa de origem e destino", len(path))
	}
	if path[0].X != token.X || path[0].Y != token.Y {
		return fmt.Errorf("o caminho começa em (%d,%d) e a peça está em (%d,%d)", path[0].X, path[0].Y, token.X, token.Y)
	}
	// Terreno difícil ainda não existe no estado (é a fatia do mapa): a régua
	// já sabe cobrá-lo, e o mapa vazio custa o que o chão limpo custa.
	cost := engine.PathCost(path, engine.MoveTerrain{}, budget)
	if !cost.Legal {
		return fmt.Errorf("%s", cost.Reason)
	}
	if err := assertSaneCoords(BoardToken{X: path[len(path)-1].X, Y: path[len(path)-1].Y}); err != nil {
		return err
	}
	b.Pending = &PendingMove{TokenID: tokenID, Path: path, Cost: cost.Squares, Budget: budget, ByUserID: by.userID}
	b.Version++
	return nil
}

// commitMove pousa a peça no fim do caminho proposto.
//
// `version` é a versão que o proponente tinha na mão: se o tabuleiro mudou
// desde a proposta, o commit é RECUSADO em vez de aplicado sobre outra cena.
// Isso mata os três casos que o last-write-wins quebra — dois clientes na mesma
// peça, o mestre arrastando enquanto o jogador confirma, e o broadcast atrasado
// que chega depois da re-hidratação. Versão 0 = "não sei em que versão eu
// estava", aceita, porque recusar um cliente honesto e desatualizado seria
// pior que aplicar o que ele acabou de ver na tela.
func commitMove(b *BoardState, st *SessionRuntimeState, version int64, by mover) error {
	pending, err := pendingFor(b, by)
	if err != nil {
		return err
	}
	if version > 0 && version != b.Version {
		return fmt.Errorf("o tabuleiro mudou (versão %d, você viu a %d): refaça o movimento", b.Version, version)
	}
	// A vez é conferida DE NOVO na confirmação: entre propor e confirmar o
	// mestre pode ter passado o turno, e o que vale é a mesa no instante em que
	// a peça pousa. O mestre confirma por qualquer um, então ele passa por aqui
	// sem restrição.
	token, _, err := assertMovable(b, st, pending.TokenID, by)
	if err != nil {
		return err
	}
	destination := pending.Path[len(pending.Path)-1]
	token.X, token.Y = destination.X, destination.Y
	b.Pending = nil
	b.Version++
	return nil
}

// cancelMove desfaz o provisório sem mexer na peça.
func cancelMove(b *BoardState, by mover) error {
	if _, err := pendingFor(b, by); err != nil {
		return err
	}
	b.Pending = nil
	b.Version++
	return nil
}

// pendingFor devolve o provisório se esta pessoa pode decidir sobre ele. O
// mestre decide por qualquer um — é ele quem toca a mesa quando o jogador
// travou ou caiu da rede —, e o jogador só sobre o que ele mesmo propôs.
func pendingFor(b *BoardState, by mover) (*PendingMove, error) {
	if b.Pending == nil {
		return nil, fmt.Errorf("não há movimento proposto para confirmar")
	}
	if by.role != "gm" && b.Pending.ByUserID != by.userID {
		return nil, fmt.Errorf("o movimento proposto não é seu")
	}
	return b.Pending, nil
}
