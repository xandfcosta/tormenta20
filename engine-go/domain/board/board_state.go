package board

import (
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O TABULEIRO COMO DADO e as PEÇAS que estão nele: o que uma peça é, e as
// mutações que a põem, copiam, mexem e tiram.

// MaxTokens — teto de peças num tabuleiro. Espelha o `live.InitiativeMaxEntries`
// pelo mesmo motivo: sem teto, o estado cresce sem limite e TODO broadcast o
// carrega. Vinte tokens é uma mesa cheia; 200 é um acidente.
const MaxTokens = 200

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
	// re-sincronizado do motor a cada proposta, como o `RefreshCharacterVitals`
	// faz com o `hpMax`. Zero = nunca medido; vale o padrão do livro.
	SpeedSquares int `json:"speedSquares,omitempty"`
	// CameFrom é onde a peça estava ANTES do último movimento confirmado, e é
	// o que dá ao mestre o "voltar para onde estava".
	//
	// Guardado na PEÇA e não na memória da tela: o gesto que ele conserta —
	// "arrastei o dragão para o lugar errado na frente de seis pessoas" — é o
	// que se quer desfazer de QUALQUER tela, e memória de tela morre no F5.
	//
	// UMA posição e não uma pilha: o arrependimento é sobre o gesto que acabou
	// de acontecer, e um histórico convidaria a andar para trás na cena com um
	// botão que não diz até onde vai.
	CameFrom *engine.Square `json:"deOndeVeio,omitempty"`
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
	// ID identifica ESTE tabuleiro dentro da sessão, que pode ter vários abertos
	// ao mesmo tempo.
	//
	// Cunhado pelo servidor no `Open`, com o mesmo `newID` das peças, e não pelo
	// banco: o tabuleiro nasce em MEMÓRIA e a gravação vem depois, então um id
	// vindo do disco obrigaria a cena a esperar o disco para saber quem ela é.
	// Na hidratação ele vem da COLUNA e não deste campo — ver `hydrateLocked`.
	ID string `json:"id"`
	// Seq é a ORDEM DE ABERTURA dentro da sessão, e ela é a ordem das abas na
	// tela. Mora numa COLUNA e não no JSON — `json:"-"` — porque é propriedade
	// da aba e não da cena: o mesmo lugar arquivado reabre em qualquer posição,
	// e um número gravado dentro da cena reabriria a taverna no lugar de outra
	// aba.
	//
	// Contador e não carimbo de tempo: com o instante em milissegundos, duas
	// cenas abertas no mesmo milissegundo empatam e o desempate cai no id, que
	// é um UUID.
	Seq int64 `json:"-"`
	// Version sobe a cada mutação aceita. É o que vai permitir recusar um
	// movimento proposto sobre um tabuleiro que já mudou, e o que deixa o
	// cliente descartar um broadcast atrasado depois de reconectar.
	Version int64 `json:"version"`
	// Curtained é a CORTINA: o tabuleiro existe para o mestre e a mesa vê uma
	// cortina no lugar dele.
	//
	// Mora no estado e não numa lista de sessões em memória porque tem de
	// sobreviver a recarregar a página — o mestre monta a emboscada, fecha o
	// laptop, e a cortina continua fechada.
	//
	// O padrão é FALSO, e é o inverso do `BoardForRole`, que erra para o lado
	// que esconde: um tabuleiro nascido sob cortina sem o mestre pedir some da
	// mesa sem ninguém entender, e o erro caro é achar que se está montando
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
	// hoje não são consumidos por nada — alimentam o OLHO. O ataque se resolve
	// entre LINHAS DA FILA (`engine.ResolveAttack`, ALE-364) e não sabe em que
	// casa as peças estão. Um mapa afirmaria que são intercambiáveis, e a
	// Tabela 5-3 diz que não; e chave de string ainda convida ao erro mudo, com
	// `"elevated"` contra `"elevado"` virando lista vazia sem estourar.
	//
	// A assimetria é a parte que importa e a que um mapa esconderia: ela é
	// exatamente o que quem for ligar o terreno ao ataque precisa ver.
	//
	// A repetição está contida no `listForKind`, que é o único lugar que sabe
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
	// Markers são os LUGARES apontados no mapa. Não são peças: não
	// ocupam quadrado e não entram na conta de nada.
	Markers []BoardMarker `json:"markers,omitempty"`
	// Pending é o movimento proposto e ainda não confirmado — no máximo um.
	Pending *PendingMove `json:"pending,omitempty"`
}

// NewBoard abre um tabuleiro vazio num plano sem bordas.
func NewBoard(id, place, terrain string) *BoardState {
	return &BoardState{ID: id, Version: 1, Place: place, Terrain: terrain, Tokens: []BoardToken{}}
}

// AddToken põe uma peça no tabuleiro, recusando o que sairia da grade.
func AddToken(b *BoardState, t BoardToken, newID func() string) error {
	if len(b.Tokens) >= MaxTokens {
		return fmt.Errorf("o tabuleiro já tem %d peças (teto %d)", len(b.Tokens), MaxTokens)
	}
	if t.Footprint <= 0 {
		t.Footprint = 1
	}
	if err := AssertSaneCoords(t); err != nil {
		return err
	}
	t.ID = newID()
	b.Tokens = append(b.Tokens, t)
	b.Version++
	return nil
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
	used := make([]string, 0, len(b.Tokens))
	for _, token := range b.Tokens {
		used = append(used, token.Label)
	}
	return live.NextInstanceLabelAmong(used, label)
}

// DuplicateToken põe outra igual no tabuleiro — "mais um zumbi" é a operação
// mais repetida ao montar encontro.
//
// A cópia leva o corpo: rótulo renumerado, tamanho, tipo e o ocultamento (o
// segundo zumbi da emboscada também está escondido).
//
// O LAÇO é a decisão inteira, e ele é a LINHA DA FILA: a barra de PV de uma
// peça é indexada por `entryId` — o `board_view` lê `saude[*t.EntryID]` —,
// então é a LINHA, e não a ficha, que decide se um dano aparece nas duas peças
// ou só numa. São três usos, e o chamador escolhe passando ou não uma linha:
//
//   - `laco == nil` → PEÃO MUDO: sem fila e sem PV, que é o certo para cenário
//     e para a peça que entra na fila depois;
//   - `loop` = a linha DA ORIGINAL → as duas peças sangram JUNTO, com uma barra
//     só. Serve para o mesmo inimigo desenhado em dois pontos;
//   - `loop` = uma linha NOVA → a cópia sangra SOZINHA, com PV próprio. Quem
//     cria a linha é o chamador, porque ela mora no `app/session` e não aqui.
//
// A FICHA vem do laço e nunca da original: a linha nova de um NPC não tem
// ficha, e herdar o `characterId` da original ali daria uma peça dizendo ser de
// um personagem que a fila dela não conhece — posse e deslocamento, os dois que
// o `characterId` decide, sairiam da ficha errada.
func DuplicateToken(b *BoardState, tokenID string, loop *live.InitiativeEntry, newID func() string) error {
	original := FindToken(b, tokenID)
	if original == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	dup := *original
	dup.EntryID, dup.CharacterID = nil, nil
	if loop != nil {
		dup.EntryID = strPtr(loop.ID)
		dup.CharacterID = loop.CharacterID
	}
	dup.SpeedSquares = original.SpeedSquares
	dup.Label = nextInstanceLabel(b, original.Label)
	spot := freeSpotNear(b, boardSpot{x: original.X, y: original.Y})
	dup.X, dup.Y = spot.x, spot.y
	return AddToken(b, dup, newID)
}

// PasteToken põe no tabuleiro uma cópia de uma peça que veio de OUTRO LUGAR —
// de outra aba, ou da mesma depois de o mestre ter arrastado a vista.
//
// A diferença para o `DuplicateToken` é o DESTINO e nada mais: lá a cópia nasce
// colada na original, aqui ela nasce onde a pessoa está olhando. As duas regras
// que valem nos dois casos ficam escritas uma vez só — o rótulo é renumerado
// contra as peças DESTE tabuleiro, e a casa é a primeira livre a partir do alvo,
// porque pousar uma peça em cima de outra esconde a de baixo sem dizer nada.
//
// O `template` é a peça de ORIGEM e ela pode não estar neste tabuleiro: por isso
// ela chega por valor e não por id. O `loop` decide o que a cópia é, exatamente
// como no `DuplicateToken` — ver a explicação lá, que é onde a decisão mora.
func PasteToken(b *BoardState, template BoardToken, loop *live.InitiativeEntry, x, y int, newID func() string) error {
	dup := template
	dup.EntryID, dup.CharacterID = nil, nil
	if loop != nil {
		dup.EntryID = strPtr(loop.ID)
		dup.CharacterID = loop.CharacterID
	}
	// O DE-ONDE-VEIO não viaja: ele é a memória do último pouso DESTA peça, e
	// uma cópia que nasce agora não tem para onde voltar. Herdá-lo daria um
	// "voltar" que manda a cópia para um lugar onde ela nunca esteve — e, colando
	// entre abas, para um quadrado de outro mapa.
	dup.CameFrom = nil
	dup.Label = nextInstanceLabel(b, template.Label)
	// O ALVO PRIMEIRO, e só depois a vizinhança. O `freeSpotNear` começa no anel
	// 1 e nunca olha o próprio quadrado — ele foi escrito para o duplicar, onde
	// pousar EM CIMA da original é justamente o que não se quer. No colar o alvo
	// é o alvo: quem apertou CTRL+V está olhando para aquele quadrado, e sem
	// esta linha a cópia pousaria na primeira casa do anel de fora.
	dup.X, dup.Y = x, y
	if occupied(b, x, y) {
		spot := freeSpotNear(b, boardSpot{x: x, y: y})
		dup.X, dup.Y = spot.x, spot.y
	}
	return AddToken(b, dup, newID)
}

// freeSpotNear acha o primeiro quadrado livre em volta de um ponto, em anéis
// que crescem.
//
// AO LADO do original, e não na fileira de entrada: quem duplica o zumbi que
// está no canto do mapa espera o irmão dele ali do lado, não a dez quadrados de
// distância no lugar combinado onde as peças avulsas nascem.
func freeSpotNear(b *BoardState, from boardSpot) boardSpot {
	for ring := 1; ring <= boardCoordLimit; ring++ {
		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if abs(dx) != ring && abs(dy) != ring {
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

// TokenPatch é a alteração parcial de uma peça: só os campos não-nulos entram,
// para "não mexer" ficar distinto de "zerar".
type TokenPatch struct {
	Label     *string `json:"label"`
	Hidden    *bool   `json:"hidden"`
	Footprint *int    `json:"footprint"`
	X         *int    `json:"x"`
	Y         *int    `json:"y"`
}

// UpdateToken aplica o patch. Não há borda para respeitar — só o guarda contra
// coordenada absurda, que é sobre lixo de cliente e não sobre o mapa.
func UpdateToken(b *BoardState, tokenID string, patch TokenPatch) error {
	for i := range b.Tokens {
		t := &b.Tokens[i]
		if t.ID != tokenID {
			continue
		}
		next := *t
		applyTokenPatch(&next, patch)
		if err := AssertSaneCoords(next); err != nil {
			return err
		}
		*t = next
		b.Version++
		return nil
	}
	return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
}

func applyTokenPatch(t *BoardToken, patch TokenPatch) {
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

// AssertSaneCoords recusa coordenada que só pode ter vindo de cliente quebrado.
// Não é borda do mapa — o mapa não tem borda; é o guarda que impede um número
// absurdo de estourar a serialização e a tela de todo mundo na mesa.
func AssertSaneCoords(t BoardToken) error {
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

type boardSpot struct{ x, y int }

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

// FindToken devolve o ponteiro para a peça viva (para mutação) ou nil.
func FindToken(b *BoardState, tokenID string) *BoardToken {
	for i := range b.Tokens {
		if b.Tokens[i].ID == tokenID {
			return &b.Tokens[i]
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }

// UnbindOrphanTokens desamarra as peças cuja linha da fila não existe mais, e
// devolve quantas foram. A peça FICA no mapa — só o vínculo sai.
//
// # Por que desamarrar, e não remover a peça
//
// Porque a peça sobreviver é o desenho, e não um efeito colateral. "Reiniciar o
// combate" promete na tela que *"a partida CONTINUA no ar"*: o mestre reiniciou
// o COMBATE, não a CENA, e o mapa que ele montou é trabalho dele (decisão do
// dono, ALE-377).
//
// O que não pode sobreviver é o PONTEIRO. Uma peça apontando para uma linha
// morta mente de um jeito silencioso: ela continua se anunciando como
// combatente, com o botão "Atacar" no menu, e o gesto responde 200 sem fazer
// nada nem recusar. O `EntryID` é `*string` com `omitempty` exatamente para
// distinguir "sem linha" de "linha vazia" — a peça avulsa (porta, baú, barril)
// já vive assim.
//
// # Ela é a reconciliação das TRÊS fontes
//
// Tirar um combatente da fila, reiniciar o combate e reabrir um lugar do acervo
// noutra sessão deixavam o mesmo estado por três caminhos. Passar o estado da
// fila que VALE AGORA responde aos três — inclusive ao terceiro, com `st` nulo:
// no acervo da campanha não existe fila nenhuma, e nenhum `EntryID` de sessão
// tem sentido lá.
func UnbindOrphanTokens(b *BoardState, st *live.SessionRuntimeState) int {
	if b == nil {
		return 0
	}
	alive := map[string]bool{}
	if st != nil {
		for i := range st.Initiative {
			alive[st.Initiative[i].ID] = true
		}
	}
	unbound := 0
	for i := range b.Tokens {
		if b.Tokens[i].EntryID == nil || alive[*b.Tokens[i].EntryID] {
			continue
		}
		// O `CharacterID` vai junto: ele é a outra metade do mesmo vínculo, e
		// uma peça que diz ter ficha sem ter linha desenha barra de PV de um
		// combatente que não está na mesa.
		b.Tokens[i].EntryID, b.Tokens[i].CharacterID = nil, nil
		unbound++
	}
	return unbound
}
