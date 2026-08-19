package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

// BoardMarker é um LUGAR marcado no mapa que não é uma peça (ALE-195): a
// armadilha, a porta que range, o ponto de encontro.
//
// Nem tudo que importa no mapa é criatura ou móvel. Até agora o mestre só tinha
// a saída de criar uma PEÇA `object` para dizer "a armadilha é aqui", e peça
// ocupa quadrado, entra na conta de quem está na área e aparece na lista de
// quem o gabarito pega. O marcador não ocupa nada: ele aponta.
//
// Texto de DUAS letras ("1A", "B3") e cor de um conjunto fechado — é o que o
// Roll20 dá de graça no pin dele, e é justamente o que a decisão de síntese
// desta casa já dispensa de asset.
type BoardMarker struct {
	ID    string `json:"id"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Text  string `json:"text"`
	Color string `json:"color"`
	// Hidden: nasce escondido e o mestre revela, pela MESMA redação por papel da
	// peça — uma segunda política sobre o que a mesa vê seria a forma mais fácil
	// de vazar a armadilha.
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
	// Difficult são as casas que custam o dobro (T20 p238). Lista ESPARSA, e não
	// um mapa do tabuleiro: o plano é infinito, então não existe "todas as
	// casas" para preencher — existem as poucas que o mestre pintou.
	Difficult []engine.Square `json:"difficult,omitempty"`
	// Markers são os LUGARES apontados no mapa (ALE-195). Não são peças: não
	// ocupam quadrado e não entram na conta de nada.
	Markers []BoardMarker `json:"markers,omitempty"`
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

// instanceSuffix é o número que a mesa usa para contar iguais: "Zumbi 3".
//
// A MESMA convenção vive na tela, em `token-appearance.ts`, onde ela decide a
// cor e o selo da peça (ALE-179) — e as duas precisam concordar, senão a cópia
// nasce com um nome que o desenho colore como outra espécie. Os dois testes
// carregam a mesma tabela de exemplos de propósito, com a mesma armadilha: o
// número no MEIO do nome ("Recruta Nv1 Simples") não é instância.
var instanceSuffix = regexp.MustCompile(`^(.*\S)\s+(\d{1,3})$`)

// speciesOf separa a espécie do número da instância.
func speciesOf(label string) (string, int) {
	match := instanceSuffix.FindStringSubmatch(strings.TrimSpace(label))
	if match == nil {
		return strings.TrimSpace(label), 0
	}
	numero, err := strconv.Atoi(match[2])
	if err != nil {
		return strings.TrimSpace(label), 0
	}
	return match[1], numero
}

// nextInstanceLabel devolve o rótulo da cópia: a mesma espécie com o MENOR
// número livre.
//
// Menor livre e não "maior mais um": depois de tirar o Zumbi 2 do tabuleiro, a
// próxima cópia volta a ser o Zumbi 2 e a numeração continua colada — uma mesa
// com "Zumbi 1, 3 e 7" faz a pessoa procurar os que não existem. A peça SEM
// número conta como a instância 1, senão duplicar o "Ogro" produziria um
// "Ogro 1" que ninguém distingue do original.
//
// O original NUNCA é renomeado: ele pode estar amarrado a uma linha da
// iniciativa, e mudar o nome dele por baixo faria a lista e o mapa discordarem
// sobre quem é quem.
func nextInstanceLabel(b *BoardState, label string) string {
	especie, _ := speciesOf(label)
	usados := map[int]bool{}
	for _, token := range b.Tokens {
		outra, numero := speciesOf(token.Label)
		if outra != especie {
			continue
		}
		if numero == 0 {
			numero = 1 // a peça sem número ocupa o 1
		}
		usados[numero] = true
	}
	for numero := 1; ; numero++ {
		if !usados[numero] {
			return fmt.Sprintf("%s %d", especie, numero)
		}
	}
}

// duplicateToken põe outra igual no tabuleiro — "mais um zumbi" é a operação
// mais repetida ao montar encontro (ALE-192).
//
// A cópia leva o corpo (rótulo renumerado, tamanho, tipo e o ocultamento: o
// segundo zumbi da emboscada também está escondido) e NÃO leva o vínculo:
// `entryId` e `characterId` ficam para trás porque a cópia é uma peça nova, não
// a mesma linha da iniciativa nem o mesmo personagem.
func duplicateToken(b *BoardState, tokenID string, newID func() string) error {
	original := findToken(b, tokenID)
	if original == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	copia := *original
	copia.EntryID = nil
	copia.CharacterID = nil
	copia.SpeedSquares = original.SpeedSquares
	copia.Label = nextInstanceLabel(b, original.Label)
	spot := freeSpotNear(b, boardSpot{x: original.X, y: original.Y})
	copia.X, copia.Y = spot.x, spot.y
	return addToken(b, copia, newID)
}

// freeSpotNear acha o primeiro quadrado livre em volta de um ponto, em anéis
// que crescem.
//
// AO LADO do original, e não na fileira de entrada: quem duplica o zumbi que
// está no canto do mapa espera o irmão dele ali do lado, não a dez quadrados de
// distância no lugar combinado onde as peças avulsas nascem (ALE-166).
func freeSpotNear(b *BoardState, from boardSpot) boardSpot {
	for anel := 1; anel <= boardCoordLimit; anel++ {
		for dy := -anel; dy <= anel; dy++ {
			for dx := -anel; dx <= anel; dx++ {
				if abs(dx) != anel && abs(dy) != anel {
					continue // o miolo já foi visto nos anéis de dentro
				}
				spot := boardSpot{x: from.x + dx, y: from.y + dy}
				if !occupied(b, spot.x, spot.y) {
					return spot
				}
			}
		}
	}
	return from
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

// markerColors é o conjunto FECHADO de cores do marcador. Fechado porque a cor
// vira classe na tela: aceitar qualquer string deixaria o cliente escrever CSS
// no estado da mesa.
var markerColors = map[string]bool{"ouro": true, "carmim": true, "azul": true, "verde": true}

// boardMaxMarkers — teto de marcadores, pelo mesmo motivo do teto de peças: o
// estado inteiro viaja em todo broadcast.
const boardMaxMarkers = 100

// addMarker põe um lugar marcado no mapa.
func addMarker(b *BoardState, m BoardMarker, newID func() string) error {
	if len(b.Markers) >= boardMaxMarkers {
		return fmt.Errorf("o tabuleiro já tem %d marcadores (teto %d)", len(b.Markers), boardMaxMarkers)
	}
	if abs(m.X) > boardCoordLimit || abs(m.Y) > boardCoordLimit {
		return fmt.Errorf("marcador em (%d,%d) está além do limite de sanidade de %d quadrados", m.X, m.Y, boardCoordLimit)
	}
	if !markerColors[m.Color] {
		m.Color = "ouro"
	}
	m.Text = trimMarkerText(m.Text)
	m.ID = newID()
	b.Markers = append(b.Markers, m)
	b.Version++
	return nil
}

// trimMarkerText corta o rótulo em DUAS letras — em runas e não em bytes, senão
// "Ê2" viraria meio caractere e a tela desenharia lixo.
func trimMarkerText(text string) string {
	runas := []rune(strings.TrimSpace(text))
	if len(runas) > 2 {
		runas = runas[:2]
	}
	return string(runas)
}

// updateMarker altera texto, cor ou o ocultamento — a posição não muda porque
// marcador que anda é peça, e peça já existe.
func updateMarker(b *BoardState, markerID string, patch markerPatch) error {
	for i := range b.Markers {
		if b.Markers[i].ID != markerID {
			continue
		}
		if patch.Text != nil {
			b.Markers[i].Text = trimMarkerText(*patch.Text)
		}
		if patch.Color != nil && markerColors[*patch.Color] {
			b.Markers[i].Color = *patch.Color
		}
		if patch.Hidden != nil {
			b.Markers[i].Hidden = *patch.Hidden
		}
		b.Version++
		return nil
	}
	return fmt.Errorf("marcador %q não está no tabuleiro", markerID)
}

// markerPatch é a alteração parcial: ausente é "não mexa", não "zere".
type markerPatch struct {
	Text   *string `json:"text"`
	Color  *string `json:"color"`
	Hidden *bool   `json:"hidden"`
}

// removeMarker tira o lugar marcado. Some em silêncio se já não está lá.
func removeMarker(b *BoardState, markerID string) {
	for i, m := range b.Markers {
		if m.ID != markerID {
			continue
		}
		b.Markers = append(b.Markers[:i], b.Markers[i+1:]...)
		b.Version++
		return
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
// tem peça, com os PERSONAGENS de um lado e o resto do outro. Idempotente de
// propósito, como o `populateParty` do rastreador: clicar duas vezes não
// duplica ninguém.
//
// Antes todos caíam numa fileira única no meio do mapa (ALE-166), e esse é o
// estado em que o mestre encontra o tabuleiro no segundo em que o combate
// começa, com a mesa esperando: ele tinha de arrastar nove peças, uma a uma,
// antes de o tabuleiro servir para alguma coisa. Nascendo em dois lados, ele
// AJUSTA em vez de DISTRIBUIR — e a informação de lado já existia na linha.
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
		spot := clusterSpot(b, entry.Type == "character")
		token.X, token.Y = spot.x, spot.y
		if err := addToken(b, token, newID); err != nil {
			break
		}
		placed++
	}
	return placed
}

// Os dois lados onde uma cena de combate começa, em quadrados.
//
// A distância entre as bordas é de 6 quadrados — 9m, o alcance curto do livro
// (T20 p224). É perto o bastante para a briga começar sem ninguém andar meia
// tela, e longe o bastante para o primeiro turno ainda ter escolha: aproximar,
// atirar ou conjurar.
const (
	partySideX  = -5 // personagens: colunas -5, -4, -3
	enemySideX  = 3  // o resto: colunas 3, 4, 5
	clusterCols = 3
)

// clusterSpot devolve o primeiro quadrado livre do lado pedido, preenchendo em
// blocos de três colunas que crescem para baixo.
//
// Continua havendo um lugar COMBINADO onde a peça nova aparece, que é o que um
// plano infinito exige — só que agora são dois, um por lado. E continua
// respeitando quem já está no tabuleiro: o mestre pode ter posicionado alguém
// ali antes de trazer o resto.
func clusterSpot(b *BoardState, isParty bool) boardSpot {
	baseX := enemySideX
	if isParty {
		baseX = partySideX
	}
	for i := 0; ; i++ {
		spot := boardSpot{x: baseX + i%clusterCols, y: i / clusterCols}
		if !occupied(b, spot.x, spot.y) {
			return spot
		}
	}
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
	// O marcador escondido some INTEIRO, como a peça: o mestre marca a armadilha
	// antes da mesa chegar nela, e um marcador "presente porém anônimo" diria à
	// mesa exatamente onde não pisar (ALE-195).
	out.Markers = make([]BoardMarker, 0, len(b.Markers))
	for _, marker := range b.Markers {
		if marker.Hidden {
			continue
		}
		out.Markers = append(out.Markers, marker)
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
	cost := engine.PathCost(path, moveTerrainOf(b), budget)
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

// paintTerrain marca ou apaga UMA casa como terreno difícil (T20 p238).
//
// Recebe o valor DESEJADO e não alterna, e a razão mudou junto com a tela: o
// pincel pinta ARRASTANDO, e o arraste passa pela mesma casa mais de uma vez —
// alternar faria a casa piscar entre brejo e chão limpo debaixo do dedo. Com o
// valor explícito a mensagem é idempotente, que é o que um arraste precisa.
// Quem apaga é a borracha, que manda `false`.
func paintTerrain(b *BoardState, square engine.Square, difficult bool) {
	for i, existente := range b.Difficult {
		if existente == square {
			if difficult {
				return // já é brejo: nada mudou, e a versão não sobe à toa
			}
			b.Difficult = append(b.Difficult[:i], b.Difficult[i+1:]...)
			b.Version++
			return
		}
	}
	if !difficult {
		return
	}
	b.Difficult = append(b.Difficult, square)
	b.Version++
}

// moveTerrainOf traduz a lista esparsa para o que o motor cobra. A conversão
// mora aqui e não no motor porque o motor não conhece tabuleiro: ele responde
// sobre um caminho e um chão, e quem tem chão é o estado.
func moveTerrainOf(b *BoardState) engine.MoveTerrain {
	if len(b.Difficult) == 0 {
		return engine.MoveTerrain{}
	}
	difficult := make(map[engine.Square]bool, len(b.Difficult))
	for _, square := range b.Difficult {
		difficult[square] = true
	}
	return engine.MoveTerrain{Difficult: difficult}
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
