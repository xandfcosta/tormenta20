package table

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/board"
)

// AS ABAS DE TABULEIRO: o mestre abre a cripta sem guardar a taverna, e cada
// pessoa na mesa escolhe qual das duas está olhando.
//
// # Por que a aba ativa é ESTADO DO SERVIDOR, e não um sinal do navegador
//
// Um sinal do navegador é o que se faria num framework em que o componente
// segura o próprio estado. Em Datastar quem desenha o mapa é o SERVIDOR, e uma
// aba local custaria duas coisas que não se pagam:
//
//   - o stream teria de mandar TODOS os tabuleiros abertos em TODO quadro,
//     para o cliente poder trocar sem ir ao servidor (o mapa sozinho já é
//     ~41,7 KB por pintura);
//   - e o jogador receberia no HTML a cena que ele não está olhando, que é
//     exatamente o que o `BoardForRole` existe para não fazer.
//
// A LENTE (`lenses`) é estado do servidor por `(sessão, pessoa)` pelo mesmo
// motivo. Esta é a irmã dela, e paga os mesmos preços: duas abas do navegador da
// mesma pessoa compartilham a escolha, e a escolha morre com o processo (todo
// mundo volta para a aba padrão, que é a mais antiga).

// chosenTabs guarda qual tabuleiro cada pessoa está olhando, e o PUXÃO do
// mestre por cima disso.
//
// Tipo próprio e não um `sync.Map` solto no `Server` pela mesma razão da lente:
// a chave é composta e a regra de leitura tem um caso — "a aba que eu escolhi
// foi fechada" — que precisa morar junto do dado.
//
// # O puxão é um CONTADOR da sessão, e não uma escrita na escolha de cada um
//
// "Mostrar esta à mesa" tem de alcançar TODO MUNDO, e o mapa de escolhas só
// conhece quem já escolheu: quem entrou e ficou na aba padrão não tem entrada
// nenhuma, e um laço sobre o mapa passaria por cima justamente de quem nunca
// mexeu em nada.
//
// Então o puxão mora na SESSÃO, com um número que só sobe, e cada pessoa guarda
// qual puxão ela já viu. Quem tem `StrengthSeen` menor está sendo puxado agora —
// inclusive quem nunca apareceu no mapa, cujo zero é menor que qualquer puxão.
//
// E ele **não sobrescreve a escolha de ninguém**, o que dá o "voltar para onde
// eu estava" de graça: a escolha anterior continua lá, intacta, e é ela que a
// tira do jogador oferece como saída. Sobrescrever seria apagar a informação de
// que a tira precisa.
type chosenTabs struct {
	mu     sync.RWMutex
	chosen map[tabKey]tabChoice
	// pull é o "parem tudo e olhem isto" de cada sessão.
	pull map[int64]pullTable
}

type tabKey struct {
	SessionID int64
	UserID    int64
}

// tabChoice é o que uma pessoa escolheu, mais o puxão que ela já consumiu.
type tabChoice struct {
	Board string
	// StrengthSeen é o número do último puxão que esta pessoa já viu. É o que faz
	// o puxão ser UM EMPURRÃO e não uma trava (decisão do dono): assim que ela
	// escolhe qualquer aba, ela consome o puxão e volta a decidir sozinha.
	StrengthSeen int64
}

type pullTable struct {
	Board string
	Seq   int64
}

func newTabs() *chosenTabs {
	return &chosenTabs{
		chosen: map[tabKey]tabChoice{},
		pull:   map[int64]pullTable{},
	}
}

// Choose grava a aba que esta pessoa está olhando, e CONSOME o puxão em curso.
//
// Consumir aqui é o que solta a pessoa: ela foi trazida, olhou, e escolheu outra
// coisa — a partir daí a decisão é dela de novo, e a tira do puxão some. Vale
// também quando ela escolhe a própria aba para onde foi trazida: ficar é uma
// escolha, e a tira que continuasse acesa depois dela seria um modo sem gesto.
func (a *chosenTabs) Choose(sessionID, userID int64, boardID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	key := tabKey{SessionID: sessionID, UserID: userID}
	seen := a.pull[sessionID].Seq
	if boardID == "" && seen == 0 {
		// APAGA em vez de gravar vazio, como o `Toggle` da lente: o mapa vive
		// enquanto o processo viver, e uma sessão que acumulasse uma entrada
		// morta por pessoa nunca devolveria a memória. Com puxão em curso a
		// entrada TEM de existir, mesmo apontando para a padrão — ela é o
		// registro de que esta pessoa já o consumiu.
		delete(a.chosen, key)
		return
	}
	a.chosen[key] = tabChoice{Board: boardID, StrengthSeen: seen}
}

// Pull traz a mesa para uma aba, e devolve o número do puxão.
//
// Quem puxa já CONSUMIU o próprio puxão: ele está olhando aquela aba — foi por
// isso que a mostrou —, e a tira "o mestre trouxe você para cá" na tela do
// próprio mestre seria a cena contando a ele o que ele acabou de fazer.
func (a *chosenTabs) Pull(sessionID, userID int64, boardID string) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	seq := a.pull[sessionID].Seq + 1
	a.pull[sessionID] = pullTable{Board: boardID, Seq: seq}
	a.chosen[tabKey{SessionID: sessionID, UserID: userID}] = tabChoice{
		Board: boardID, StrengthSeen: seq,
	}
	return seq
}

// Resolve diz qual aba vale para esta pessoa AGORA, se ela está sendo puxada, e
// de onde ela veio.
//
// A ordem importa e é a regra inteira: o puxão ainda não consumido VENCE a
// escolha, e a escolha vence o padrão.
func (a *chosenTabs) Resolve(sessionID, userID int64) (boardID string, pulled bool, pulledFrom string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	mine := a.chosen[tabKey{SessionID: sessionID, UserID: userID}]
	if pull := a.pull[sessionID]; pull.Seq > mine.StrengthSeen {
		return pull.Board, true, mine.Board
	}
	return mine.Board, false, ""
}

// PullProgress é o número do puxão que esta pessoa ainda NÃO consumiu (0 = nenhum).
//
// Existe para o stream, que precisa empurrar a SUPERFÍCIE uma vez por puxão e
// não a cada quadro: empurrar sempre seria uma trava — a pessoa mandada para o
// tabuleiro não conseguiria voltar para a Mesa, porque o quadro seguinte a
// traria de volta no quadro seguinte, e ela concluiria que o botão está quebrado.
func (a *chosenTabs) PullProgress(sessionID, userID int64) int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	mine := a.chosen[tabKey{SessionID: sessionID, UserID: userID}]
	if pull := a.pull[sessionID]; pull.Seq > mine.StrengthSeen {
		return pull.Seq
	}
	return 0
}

// Erase esquece as escolhas e o puxão de uma sessão inteira.
//
// Chamado quando a última cena morre, pelo mesmo motivo da lente: uma escolha
// apontando para um tabuleiro que não existe mais é lixo que sobrevive à sessão.
// HowMany conta as escolhas guardadas desta sessão, mais o puxão dela se houver
// — ver o `Watching`, que soma as duas metades da memória efêmera.
func (a *chosenTabs) HowMany(sessionID int64) int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	kept := 0
	for key := range a.chosen {
		if key.SessionID == sessionID {
			kept++
		}
	}
	if _, pulled := a.pull[sessionID]; pulled {
		kept++
	}
	return kept
}

func (a *chosenTabs) Erase(sessionID int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for key := range a.chosen {
		if key.SessionID == sessionID {
			delete(a.chosen, key)
		}
	}
	delete(a.pull, sessionID)
}

// chosenTabOf resolve qual tabuleiro esta pessoa está olhando AGORA, conferindo
// contra os que existem — ver `pullTab`, que faz a conferência.
//
// Devolve o id VAZIO quando a escolha ainda vale por ser a padrão, e é
// deliberado: vazio é a palavra do store para "a primeira aberta", e reescrevê-la
// aqui como um id concreto faria a resposta envelhecer no instante em que a
// primeira aba trocasse.
func (s Scene) chosenTabOf(ctx context.Context, sessionID, userID int64) (string, error) {
	aba, _, _, err := s.pullTab(ctx, sessionID, userID)
	return aba, err
}

// pullTab é a resolução INTEIRA: a aba que vale, se ela veio de um puxão
// do mestre, e de onde a pessoa foi trazida.
//
// As rotas usam o `chosenTabOf`, que joga fora as duas últimas — para um comando só
// importa ONDE ele age. Quem precisa do resto é a CENA, que tem de dizer à
// pessoa que ela foi trazida e como voltar: um puxão silencioso trocaria o mapa
// debaixo dela no meio de um turno, e ela procuraria o defeito na própria tela.
//
// O puxão para uma cena JÁ ENCERRADA cai na escolha de quem olha, e não numa
// tela morta: o mestre mostra a cripta, encerra a cripta, e quem foi trazido
// volta para onde estava em vez de ficar olhando um tabuleiro que não existe.
func (s Scene) pullTab(ctx context.Context, sessionID, userID int64) (aba string, pulled bool, pulledFrom string, err error) {
	open, err := s.deps.Boards().OpenBoards(ctx, sessionID)
	if err != nil {
		return "", false, "", err
	}
	isOpen := func(id string) bool {
		if id == "" {
			return false
		}
		for _, b := range open {
			if b.ID == id {
				return true
			}
		}
		return false
	}
	target, pulled, pulledFrom := s.chosenTabs.Resolve(sessionID, userID)
	if pulled && !isOpen(target) {
		target, pulled, pulledFrom = pulledFrom, false, ""
	}
	if !isOpen(target) {
		// A aba que a pessoa escolheu pode ter sido fechada pelo mestre enquanto
		// ela olhava. Sem cair no padrão, a tela dela ficaria dizendo "esta sessão
		// não tem tabuleiro" com duas cenas abertas na mesa ao lado — e o gesto
		// que causou isso foi de outra pessoa, então ela não teria como ligar uma
		// coisa à outra.
		target = ""
	}
	return target, pulled, pulledFrom, nil
}

func (s Scene) TabRoutes(r chi.Router) {
	// A troca é de TODO MUNDO, e é a metade da issue que o jogador ganha: ele
	// não fica preso ao que o mestre está olhando — quem está na cripta abre a
	// aba da cripta porque quer.
	r.Post(sessionPattern+"/tabuleiro/aba/{tabuleiroId}",
		s.tableCommand(swapBoard))
	// MOSTRAR À MESA é só do mestre, e a trava é do servidor: um jogador que
	// puxasse a mesa para a aba dele tiraria dos outros cinco o que esta issue
	// acabou de lhes dar.
	r.Post(sessionPattern+"/tabuleiro/aba/{tabuleiroId}/mostrar",
		s.gmBoardCommand(showTableIsTab))
}

// showTableIsTab é o "parem tudo e olhem isto".
//
// Devolve `nil` como a lente e a troca, e é a MESMA razão apesar de o alcance
// ser outro: puxar não muda a CENA. Nenhuma peça andou, nenhum terreno foi
// pintado — o que mudou foi para onde cada pessoa está olhando, e isso não é um
// quadro do tabuleiro. Quem leva o puxão às outras telas é o batimento do
// stream, que redesenha a cena de cada um a cada batimento e já pergunta ao
// `pullTab` qual aba vale.
func showTableIsTab(st Scene, c commandCtx) (*board.BoardState, error) {
	target := chi.URLParam(c.R, "tabuleiroId")
	// Puxar para uma aba que não existe deixaria a mesa inteira caindo no padrão
	// sem nada dizendo por quê. O id vem do caminho, então isto é a conferência
	// de sempre: o que o cliente manda não é a verdade.
	open, err := st.deps.Boards().OpenBoards(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	for _, open := range open {
		if open.ID == target {
			st.chosenTabs.Pull(c.SessionID, c.User, target)
			return nil, nil
		}
	}
	return nil, fmt.Errorf("o tabuleiro %q não está aberto nesta sessão", target)
}

// swapBoard põe outra aba na tela de quem clicou.
//
// O nome não é `pickTab` porque o EDITOR DE NPC já tem uma função com esse
// nome — as abas dele são outra coisa (Números / Ataques / Perícias). Duas abas
// no mesmo pacote é uma colisão de palavra que o GLOSSARY registra; o que
// resolve aqui é o verbo dizer o que se troca.
//
// Devolve `nil` como a lente, e pelo mesmo motivo: trocar de aba NÃO é mutação
// da cena. Publicar aqui acordaria a mesa inteira para uma escolha que é de uma
// pessoa só. Quem redesenha a tela de quem clicou é a resposta do comando.
//
// Não confere se o id existe, e isso não é descuido: quem confere é o `chosenTabOf`,
// a cada leitura, porque a aba pode morrer DEPOIS da escolha. Uma conferência
// aqui daria a mesma resposta e ainda deixaria a outra necessária.
func swapBoard(st Scene, c commandCtx) (*board.BoardState, error) {
	st.chosenTabs.Choose(c.SessionID, c.User, chi.URLParam(c.R, "tabuleiroId"))
	return nil, nil
}

// boardTab é uma ficha da barra de abas.
type boardTab struct {
	ID   string
	Name string
	// Active é a que esta pessoa está olhando. Ela não vira botão: é o `<h2>` que
	// nomeia a região.
	Active bool
	// Curtain diz que esta aba está sob cortina PARA QUEM OLHA. Para o mestre é
	// a marca de que ele está montando escondido; para o jogador é tudo o que
	// existe daquela aba — o nome não atravessa (ver `BoardForRole`).
	Curtain bool
	Command string
	// ShowsTable é o gesto do mestre "parem tudo e olhem isto", e ele só é
	// escrito na aba ATIVA (decisão do dono): ele já está olhando a cena que quer
	// mostrar — foi por isso que trocou para ela —, e um alvo de clique por ficha
	// encheria uma barra que é estreita por natureza.
	ShowsTable string
}

// pullScreen é a tira "o mestre trouxe você para cá", com a saída dela.
//
// Ela existe porque o puxão é a única coisa nesta cena que muda o que a pessoa
// está vendo SEM ela ter feito nada. A cortina e a lente são estados que o
// próprio dono da tela ligou; este não — e um mapa que troca sozinho, no meio de
// um turno, é lido como defeito.
type pullScreen struct {
	// Scene é para onde a mesa foi trazida, e Volta é de onde esta pessoa veio.
	//
	// `Volta` VAZIO é quem já estava nesta aba: para essa pessoa o puxão mudou a
	// superfície e não a cena, então não há para onde voltar — o `ReturnCommand`
	// aponta para a aba atual, e clicar nele é dizer "vi", que é o mesmo gesto
	// que solta qualquer um do puxão.
	Scene         string
	Back          string
	ReturnCommand string
}

// tableTabs monta a barra a partir dos tabuleiros abertos, JÁ REDIGIDOS pelo
// papel de quem olha.
//
// A redação acontece antes de o nome ser lido, e é por isso que a aba sob
// cortina chega aqui sem `Place`: o rótulo dela é escrito pelo `tabName`, e
// não pelo que o servidor guardou. Ler o nome do estado CRU e "esconder na tela"
// seria pôr "Cripta do Rei" no HTML de quem não pode saber que há uma cripta —
// o vazamento que não aparece na tela, só no ver-código-fonte.
func tableTabs(open []*board.BoardState, role, active string, campaignID, sessionID int64) []boardTab {
	// UMA aba não é uma barra: com um tabuleiro só não há o que trocar, e a
	// tira de fichas seria enfeite ocupando mapa. A tela cai no `<h2>` de sempre.
	if len(open) < 2 {
		return nil
	}
	bar := make([]boardTab, 0, len(open))
	for i, isOpen := range open {
		fromTable := board.BoardForRole(role, isOpen)
		sheet := boardTab{
			ID:      fromTable.ID,
			Name:    tabName(fromTable, i),
			Active:  activeTabIs(fromTable.ID, active, i),
			Curtain: fromTable.Curtained,
			Command: fmt.Sprintf("@post('%s/aba/%s')", tableBoardBase(campaignID, sessionID), fromTable.ID),
		}
		if sheet.Active && role == "gm" {
			sheet.ShowsTable = fmt.Sprintf("@post('%s/aba/%s/mostrar')",
				tableBoardBase(campaignID, sessionID), fromTable.ID)
		}
		bar = append(bar, sheet)
	}
	return bar
}

// removePull monta o aviso a partir da barra JÁ REDIGIDA, ou devolve nil.
//
// Ela lê a barra e não o estado cru de propósito: os nomes já passaram pelo
// papel de quem olha, então a tira de um jogador trazido para uma cena sob
// cortina diz "Cena 2" — o mesmo rótulo da ficha dele — em vez de contar o nome
// que a cortina esconde.
//
// Devolve nil quando não há PARA ONDE VOLTAR, e isso não é economia: um aviso
// que diz "você foi trazido" sem oferecer a saída é um modo que a pessoa não
// tem como desfazer, e a casa já decidiu que cada tira carrega a própria saída.
// Acontece de verdade — o mestre puxa e depois encerra a cena de onde a pessoa
// veio — e o caminho de volta passa a ser a barra, que está ali do lado.
func removePull(bar []boardTab, pulledFrom string) *pullScreen {
	if len(bar) == 0 {
		return nil
	}
	var current, back *boardTab
	for i := range bar {
		if bar[i].Active {
			current = &bar[i]
		}
		// `pulledFrom` vazio é a aba PADRÃO, que é a primeira da barra: quem nunca
		// escolheu estava nela, e é para lá que "voltar" o leva.
		if bar[i].ID == pulledFrom || (pulledFrom == "" && i == 0) {
			back = &bar[i]
		}
	}
	if current == nil {
		return nil
	}
	// QUEM JÁ ESTAVA NA CENA também recebe a tira, e isso não é ruído: o puxão
	// traz a pessoa para a superfície do TABULEIRO, e o caso mais comum da mesa é
	// exatamente este — o jogador abre na Mesa (a superfície padrão) e fica na
	// aba padrão. Sem a tira, a tela dele trocaria de superfície sozinha e sem
	// explicação, que é a leitura de defeito.
	//
	// O que muda é a SAÍDA: quem veio de outra aba tem para onde voltar; quem já
	// estava aqui só precisa de um jeito de dizer "vi" — e dizer "vi" é escolher
	// esta aba, que é o mesmo gesto que solta qualquer um do puxão.
	if back == nil || back.ID == current.ID {
		return &pullScreen{Scene: current.Name, ReturnCommand: current.Command}
	}
	return &pullScreen{Scene: current.Name, Back: back.Name, ReturnCommand: back.Command}
}

// activeTabIs resolve o id vazio, que é "a primeira aberta".
func activeTabIs(id, active string, position int) bool {
	if active == "" {
		return position == 0
	}
	return id == active
}

// tabName escreve o rótulo da ficha.
//
// A aba SOB CORTINA não tem nome para mostrar a quem está do outro lado dela, e
// mesmo assim precisa de uma palavra: a decisão do dono foi que ela APARECE para
// o jogador — sumir e voltar conforme o mestre corre a cortina trocaria a aba
// debaixo do dedo de quem estava olhando. Então ela se chama pela POSIÇÃO, que
// é o que se pode dizer sem contar nada: "Cena 2".
//
// Para o MESTRE o nome atravessa, porque a cortina não é sobre ele.
func tabName(fromTable *board.BoardState, position int) string {
	if fromTable.Place != "" {
		return fromTable.Place
	}
	return fmt.Sprintf("Cena %d", position+1)
}
