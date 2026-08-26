package tabuleiro

import "t20engine/aovivo"

import (
	"fmt"
	"strings"

	"t20engine/engine"
)

// boardMaxTokens — teto de peças num tabuleiro. Espelha o `aovivo.InitiativeMaxEntries`
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
	// re-sincronizado do motor a cada proposta, como o `RefreshCharacterMaxes`
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
	// Curtained é a CORTINA (ALE-202): o tabuleiro existe para o mestre e a
	// mesa vê uma cortina no lugar dele.
	//
	// Mora no estado e não numa lista de sessões em memória porque tem de
	// sobreviver a recarregar a página — o mestre monta a emboscada, fecha o
	// laptop, e a cortina continua fechada. A persistência do tabuleiro já leva
	// o estado inteiro desde a ALE-124, então isto vem de graça.
	//
	// O padrão é FALSO por uma razão que o comentário do `BoardForRole` já diz:
	// errar para o lado que mostra seria vazar por omissão. Aqui é o inverso —
	// um tabuleiro que nasce sob cortina sem o mestre pedir some da mesa sem
	// ninguém entender, e o erro caro desta issue é achar que se está montando
	// escondido e não estar. Quem fecha a cortina é um gesto explícito.
	Curtained bool `json:"curtained"`
	// Place é o nome do lugar ("Taverna do Javali") — o mestre está montando uma
	// cena, não uma planilha.
	Place   string       `json:"place"`
	Terrain string       `json:"terrain"`
	Tokens  []BoardToken `json:"tokens"`
	// As quatro espécies de TERRENO — o que o quadrado FAZ com quem está nele
	// (T20 p238, Tabela 5-3). Listas ESPARSAS, e não um mapa do tabuleiro: o
	// plano é infinito, então não existe "todas as casas" para preencher —
	// existem as poucas que o mestre pintou.
	//
	// QUATRO LISTAS IRMÃS e não um `map[string][]Square`, e o motivo é de
	// DOMÍNIO: as quatro não são variantes de uma coisa só. O difícil muda o
	// CUSTO DO MOVIMENTO e é consumido por regra (`PathCost`, e o alcance que
	// acende as casas); os outros três mudam Defesa, chance de falha e ataque, e
	// hoje não são consumidos por nada — alimentam o OLHO, porque o app não
	// resolve ataque contra Defesa em lugar nenhum. Um mapa afirmaria que são
	// intercambiáveis, e a Tabela 5-3 diz que não; e chave de string ainda
	// convida ao erro mudo, com `"elevated"` contra `"elevado"` virando lista
	// vazia sem estourar.
	//
	// A assimetria é a parte que importa e a que um mapa esconderia: ela é
	// exatamente o que quem for implementar a resolução de ataque precisa ver.
	//
	// NÃO é argumento contra o mapa que ele quebraria os Lugares já gravados.
	// Foi o meu primeiro, e ele não sustenta: o `Archive` faz `json.Marshal` do
	// estado inteiro, então um `UnmarshalJSON` que leia o `difficult` legado
	// resolveria em dez linhas. "Quebra os gravados" é razão para ESCREVER a
	// migração, não para evitar a forma — quem revisar isto não deve herdar o
	// argumento errado (achado da sessão da main, que foi ler a persistência).
	//
	// A repetição está contida no `listaDaEspecie`, que é o único lugar que sabe
	// qual lista guarda qual espécie.
	Difficult []engine.Square `json:"difficult,omitempty"`
	// Cover: +5 na Defesa de quem está nela (p238). Trincheira, árvore estreita.
	Cover []engine.Square `json:"cover,omitempty"`
	// Concealment: 20% de chance de falha no ataque contra quem está nela
	// (p238). Folhagens, moitas.
	Concealment []engine.Square `json:"concealment,omitempty"`
	// Elevated: +2 no ataque de quem ataca DE LÁ (p238). É a única espécie que
	// beneficia quem está nela em vez de proteger.
	Elevated []engine.Square `json:"elevated,omitempty"`
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

// AddToken põe uma peça no tabuleiro, recusando o que sairia da grade.
func AddToken(b *BoardState, t BoardToken, newID func() string) error {
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

//
// A MESMA convenção vive na tela, em `token-appearance.ts`, onde ela decide a
// cor e o selo da peça (ALE-179) — e as duas precisam concordar, senão a cópia
// nasce com um nome que o desenho colore como outra espécie. Os dois testes
// carregam a mesma tabela de exemplos de propósito, com a mesma armadilha: o
// número no MEIO do nome ("Recruta Nv1 Simples") não é instância.

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
	usados := make([]string, 0, len(b.Tokens))
	for _, token := range b.Tokens {
		usados = append(usados, token.Label)
	}
	return aovivo.NextInstanceLabelAmong(usados, label)
}

// DuplicateToken põe outra igual no tabuleiro — "mais um zumbi" é a operação
// mais repetida ao montar encontro (ALE-192).
//
// A cópia leva o corpo (rótulo renumerado, tamanho, tipo e o ocultamento: o
// segundo zumbi da emboscada também está escondido) e NÃO leva o vínculo:
// `entryId` e `characterId` ficam para trás porque a cópia é uma peça nova, não
// a mesma linha da iniciativa nem o mesmo personagem.
func DuplicateToken(b *BoardState, tokenID string, newID func() string) error {
	original := FindToken(b, tokenID)
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
	return AddToken(b, copia, newID)
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

// RemoveToken tira a peça do tabuleiro. Some em silêncio se ela já não está lá:
// dois cliques no mesmo botão não são erro do usuário.
func RemoveToken(b *BoardState, tokenID string) {
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

// UpdateToken aplica o patch. Não há borda para respeitar — só o guarda contra
// coordenada absurda, que é sobre lixo de cliente e não sobre o mapa.
func UpdateToken(b *BoardState, tokenID string, patch tokenPatch) error {
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

// boardMaxMarkers — teto de marcadores, pelo mesmo motivo do teto de peças: o
// estado inteiro viaja em todo broadcast.
const boardMaxMarkers = 100

// AddMarker põe um lugar marcado no mapa.
func AddMarker(b *BoardState, m BoardMarker, newID func() string) error {
	if len(b.Markers) >= boardMaxMarkers {
		return fmt.Errorf("o tabuleiro já tem %d marcadores (teto %d)", len(b.Markers), boardMaxMarkers)
	}
	if abs(m.X) > boardCoordLimit || abs(m.Y) > boardCoordLimit {
		return fmt.Errorf("marcador em (%d,%d) está além do limite de sanidade de %d quadrados", m.X, m.Y, boardCoordLimit)
	}
	if !CorDeMarcadorConhecida(m.Color) {
		m.Color = CorPadraoDeMarcador()
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

// UpdateMarker altera texto, cor ou o ocultamento — a posição não muda porque
// marcador que anda é peça, e peça já existe.
func UpdateMarker(b *BoardState, markerID string, patch markerPatch) error {
	for i := range b.Markers {
		if b.Markers[i].ID != markerID {
			continue
		}
		if patch.Text != nil {
			b.Markers[i].Text = trimMarkerText(*patch.Text)
		}
		if patch.Color != nil && CorDeMarcadorConhecida(*patch.Color) {
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

// RemoveMarker tira o lugar marcado. Some em silêncio se já não está lá.
func RemoveMarker(b *BoardState, markerID string) {
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

// EntrySelection nomeia as linhas da iniciativa que o mestre escolheu trazer
// (ALE-204).
//
// Nil é TODAS, e isso não é conveniência: é o que `board-Populate` sem
// `entryIds` sempre significou, e uma aba aberta antes desta mudança continua
// mandando exatamente isso. Lista VAZIA é diferente de ausente — "não escolhi"
// não é "escolhi ninguém".
type EntrySelection map[string]bool

func (s EntrySelection) wants(entryID string) bool { return s == nil || s[entryID] }

// populateBoard traz para o tabuleiro cada linha ESCOLHIDA da iniciativa que
// ainda não tem peça, com os PERSONAGENS de um lado e o resto do outro.
// Idempotente de propósito, como o `populateParty` do rastreador: clicar duas
// vezes não duplica ninguém.
//
// Antes todos caíam numa fileira única no meio do mapa (ALE-166), e esse é o
// estado em que o mestre encontra o tabuleiro no segundo em que o combate
// começa, com a mesa esperando: ele tinha de arrastar nove peças, uma a uma,
// antes de o tabuleiro servir para alguma coisa. Nascendo em dois lados, ele
// AJUSTA em vez de DISTRIBUIR — e a informação de lado já existia na linha.
//
// A ESCOLHA entrou depois (ALE-204): trazer a fila inteira punha no mapa o
// assassino que o mestre montou para aparecer no terceiro turno, e desfazer
// era peça por peça. Quem não foi escolhido não nasce — nem escondido: peça
// que não existe não vaza por bug de redação.
func populateBoard(b *BoardState, st *aovivo.SessionRuntimeState, newID func() string, chosen EntrySelection) int {
	placed := 0
	for _, entry := range st.Initiative {
		if !chosen.wants(entry.ID) || hasTokenForEntry(b, entry.ID) {
			continue
		}
		token := BoardToken{
			Label: entry.Label, Kind: entry.Type, Footprint: 1,
			EntryID: strPtr(entry.ID), CharacterID: entry.CharacterID,
		}
		spot := clusterSpot(b, entry.Type == "character")
		token.X, token.Y = spot.x, spot.y
		if err := AddToken(b, token, newID); err != nil {
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

// BoardForRole é o tabuleiro como UM papel pode vê-lo. Papel desconhecido cai em
// jogador: errar para o lado que MOSTRA seria vazar por omissão, a mesma regra
// do `aovivo.StateForRole`.
func BoardForRole(papel string, b *BoardState) *BoardState {
	if b == nil || papel == "gm" {
		return b
	}
	// A CORTINA vem ANTES da redação de peça (ALE-202): com ela fechada, a mesa
	// não recebe o mapa nenhum — nem as peças visíveis, nem o terreno, nem o
	// nome do lugar. Redigir peça por peça deixaria passar tudo o que não está
	// marcado como escondido, que é justamente a cena que o mestre está
	// montando.
	//
	// O que a mesa recebe é um tabuleiro VAZIO com a cortina ligada, e não
	// `nil`: `nil` significa "não há tabuleiro" e a cortina é outra coisa —
	// o jogador precisa saber que vem cena, sem ver qual (decisão do dono).
	if b.Curtained {
		// `Tokens` vai VAZIO e não nulo: fatia nil vira `null` no JSON, e o
		// cliente indexa `tokens.length` no cabeçalho — a cortina derrubaria a
		// tela da mesa em vez de escondê-la. O contrato do fio é "lista vazia é
		// uma lista", e quem o garante é aqui, não cada leitor.
		return &BoardState{Version: b.Version, Curtained: true, Tokens: []BoardToken{}}
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
	// Diagonals e Difficult são a CONTA que produziu o custo, não um resumo
	// dela: quantos passos dobraram por serem diagonais e quantos por entrarem
	// em terreno difícil (T20 p238). Viajam para a tela poder NOMEAR a regra
	// em vez de refazer a aritmética em JavaScript (ALE-190).
	Diagonals int `json:"diagonals"`
	Difficult int `json:"difficult"`
	// Budget é o orçamento contra o qual o caminho foi medido, em quadrados, ou
	// -1 quando não há (mestre, ou cena fora de combate).
	Budget int `json:"budget"`
	// ByUserID é quem propôs. O mestre confirma por qualquer um; o jogador só
	// confirma o que ele mesmo propôs.
	ByUserID int64 `json:"byUserId"`
	// Stops são as casas onde a pessoa CLICOU, na ordem, com a primeira sendo o
	// lugar de onde a peça saiu. O `Path` é o que elas produzem
	// (`CaminhoPorParadas`), e não o contrário.
	//
	// Existe porque o caminho NÃO deixa descobri-las: um trecho legítimo já tem
	// uma dobra (a diagonal vem primeiro), e ela é indistinguível da dobra de uma
	// parada. Sem esta lista, "desfazer a última perna" só poderia ser adivinhado
	// — e o que se adivinha errado aqui é o movimento que a mesa está vendo.
	//
	// NULO é um valor legítimo e quer dizer "não se sabe onde ela parou": é o que
	// o `ProposeMove` deixa quando o caminho chega pronto de fora. Quem propõe por
	// paradas usa o `ProposeMoveComParadas`, e só aí o desfazer de UMA existe.
	Stops []engine.Square `json:"stops,omitempty"`
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

// FindToken devolve o ponteiro para a peça viva (para mutação) ou nil.
func FindToken(b *BoardState, tokenID string) *BoardToken {
	for i := range b.Tokens {
		if b.Tokens[i].ID == tokenID {
			return &b.Tokens[i]
		}
	}
	return nil
}

// Mover descreve quem está tentando mexer numa peça. O papel vem do socket e a
// posse do banco; a decisão de deixar ou não é a função abaixo.
type Mover struct {
	UserID int64
	Role   string
	// OwnsCharacter: a peça é de um personagem DESTE usuário. Resolvido no
	// gateway, contra o banco — o cliente não é fonte de posse.
	OwnsCharacter bool
}

// assertMovable responde "esta pessoa pode Mover esta peça agora?" e, quando
// pode, com QUANTO de orçamento (T20 p106; -1 = sem orçamento).
//
// Três regras, e as duas exceções são deliberadas:
//   - o MESTRE move qualquer peça, a qualquer hora, sem orçamento — é a saída
//     para voo, empurrão, teleporte e "pode ir";
//   - FORA DE COMBATE (`turnIndex` < 0) não existe vez nem deslocamento de
//     turno: cada um anda com a própria peça, e o contador só informa;
//   - em combate, o jogador move a própria peça só na vez dela.
func assertMovable(b *BoardState, st *aovivo.SessionRuntimeState, tokenID string, by Mover) (*BoardToken, int, error) {
	token := FindToken(b, tokenID)
	if token == nil {
		return nil, 0, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if by.Role == "gm" {
		return token, -1, nil
	}
	if !by.OwnsCharacter {
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
func isTokenOnTurn(token *BoardToken, st *aovivo.SessionRuntimeState) bool {
	if token.EntryID == nil || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return false
	}
	return st.Initiative[st.TurnIndex].ID == *token.EntryID
}

// ProposeMove mede o caminho e guarda o provisório. Recusa o que estoura o
// deslocamento: a decisão do dono é BLOQUEAR no limite, e as saídas são o
// mestre (sem orçamento) e a cena fora de combate.
func ProposeMove(b *BoardState, st *aovivo.SessionRuntimeState, tokenID string, path []engine.Square, by Mover) error {
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
	b.Pending = &PendingMove{
		TokenID: tokenID, Path: path, Cost: cost.Squares, Budget: budget,
		Diagonals: cost.Diagonals, Difficult: cost.Difficult, ByUserID: by.UserID,
	}
	b.Version++
	return nil
}

// ProposeMoveComParadas propõe pelas casas em que a pessoa CLICOU, e guarda a
// lista junto (ALE-269, item 10).
//
// A primeira parada é onde a peça está; cada uma seguinte estende o caminho,
// contornando o que quem move quiser. É a forma que o piloto usa, e é ela que
// torna "desfazer a última perna" uma operação exata em vez de um palpite: o
// caminho se reconstrói pelas paradas que sobraram, e reconstruir é o que o
// `CaminhoPorParadas` já faz de graça.
//
// A validação inteira continua sendo a do `ProposeMove` — o orçamento, a vez, a
// posse, a contiguidade. Esta função não afrouxa nada; ela só LEMBRA de onde o
// caminho veio.
func ProposeMoveComParadas(b *BoardState, st *aovivo.SessionRuntimeState, tokenID string, paradas []engine.Square, by Mover) error {
	if err := ProposeMove(b, st, tokenID, engine.CaminhoPorParadas(paradas), by); err != nil {
		return err
	}
	b.Pending.Stops = paradas
	return nil
}

// PaintTerrain marca ou apaga UMA casa como terreno difícil (T20 p238).
//
// Recebe o valor DESEJADO e não alterna, e a razão mudou junto com a tela: o
// pincel pinta ARRASTANDO, e o arraste passa pela mesma casa mais de uma vez —
// alternar faria a casa piscar entre brejo e chão limpo debaixo do dedo. Com o
// valor explícito a mensagem é idempotente, que é o que um arraste precisa.
// Quem apaga é a borracha, que manda `false`.
func PaintTerrain(b *BoardState, square engine.Square, especie EspecieDeTerreno, ligado bool) {
	lista := listaDaEspecie(b, especie)
	if lista == nil {
		return // espécie que não existe não pinta nada, e não derruba a mesa
	}
	for i, existente := range *lista {
		if existente == square {
			if ligado {
				return // já é brejo: nada mudou, e a versão não sobe à toa
			}
			*lista = append((*lista)[:i], (*lista)[i+1:]...)
			b.Version++
			return
		}
	}
	if !ligado {
		return
	}
	*lista = append(*lista, square)
	b.Version++
}

// listaDaEspecie é o ÚNICO lugar que sabe qual lista guarda qual espécie.
//
// Devolve ponteiro para o campo porque o pincel escreve nele. É o que segura a
// repetição das quatro listas irmãs num ponto só: acrescentar uma quinta espécie
// é uma linha aqui e uma no `EspeciesDeTerreno`, e o resto do código não muda.
//
// nil para espécie desconhecida, e o pincel trata: o id vem do cliente, e uma
// espécie inventada não pode derrubar a mesa nem pintar a lista errada.
func listaDaEspecie(b *BoardState, especie EspecieDeTerreno) *[]engine.Square {
	switch especie {
	case TerrenoDificil:
		return &b.Difficult
	case TerrenoCobertura:
		return &b.Cover
	case TerrenoCamuflagem:
		return &b.Concealment
	case TerrenoElevado:
		return &b.Elevated
	}
	return nil
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

// CommitMove pousa a peça no fim do caminho proposto.
//
// `version` é a versão que o proponente tinha na mão: se o tabuleiro mudou
// desde a proposta, o commit é RECUSADO em vez de aplicado sobre outra cena.
// Isso mata os três casos que o last-write-wins quebra — dois clientes na mesma
// peça, o mestre arrastando enquanto o jogador confirma, e o broadcast atrasado
// que chega depois da re-hidratação. Versão 0 = "não sei em que versão eu
// estava", aceita, porque recusar um cliente honesto e desatualizado seria
// pior que aplicar o que ele acabou de ver na tela.
func CommitMove(b *BoardState, st *aovivo.SessionRuntimeState, version int64, by Mover) error {
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

// CancelMove desfaz o provisório sem mexer na peça.
func CancelMove(b *BoardState, by Mover) error {
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
func pendingFor(b *BoardState, by Mover) (*PendingMove, error) {
	if b.Pending == nil {
		return nil, fmt.Errorf("não há movimento proposto para confirmar")
	}
	if by.Role != "gm" && b.Pending.ByUserID != by.UserID {
		return nil, fmt.Errorf("o movimento proposto não é seu")
	}
	return b.Pending, nil
}

// PodeMover responde "esta pessoa pode mover esta peça agora?" para a TELA.
//
// Envelope fino sobre o `assertMovable`, e a razão de existir é que a tela
// precisa da MESMA resposta que a escrita — perguntar de outro jeito é como
// nasce um botão que existe e o servidor recusa, ou uma casa clicável que leva
// a "não é a vez de Arwen" depois do clique.
//
// Não devolve o porquê: quem só desenha não tem o que fazer com a frase, e a
// frase certa é a que a RECUSA escreve, no instante em que ela acontece.
func PodeMover(b *BoardState, st *aovivo.SessionRuntimeState, tokenID string, by Mover) bool {
	pode, _ := PodeMoverCom(b, st, tokenID, by)
	return pode
}

// PodeMoverCom devolve também o ORÇAMENTO, que é o que a tela precisa para
// desenhar até onde dá para ir (-1 = sem teto).
func PodeMoverCom(b *BoardState, st *aovivo.SessionRuntimeState, tokenID string, by Mover) (bool, int) {
	if b == nil {
		return false, 0
	}
	_, orcamento, err := assertMovable(b, st, tokenID, by)
	return err == nil, orcamento
}

// QuadradosDe são as casas pintadas de uma espécie, para quem só LÊ.
//
// Existe para o mapeamento espécie→lista continuar com um dono só: sem ela,
// quem desenha refaz o `switch` do `listaDaEspecie` do lado de fora, e é a cópia
// de fora que fica para trás quando a quinta espécie chegar. Devolve a fatia e
// não o ponteiro justamente por ser leitura — o pincel é quem escreve.
func QuadradosDe(b *BoardState, especie EspecieDeTerreno) []engine.Square {
	if b == nil {
		return nil
	}
	if lista := listaDaEspecie(b, especie); lista != nil {
		return *lista
	}
	return nil
}
