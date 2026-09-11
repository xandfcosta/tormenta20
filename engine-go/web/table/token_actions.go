package table

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
	"t20engine/board"
	"t20engine/engine"
)

// O MENU DE CONTEXTO NA PEÇA (ALE-206), em Datastar.
//
// A issue foi escrita para a SPA e o desenho dela vale igual aqui: clique direito
// numa peça abre os verbos dela. O que MUDA é a pergunta que ela deixava em
// aberto — *"decidir se a barra continua existindo ao lado do menu, ou se o menu
// a substitui, faz parte da issue"*. Nesta Mesa não há barra: o menu é a única
// casa, e por isso ele carrega o conjunto INTEIRO de verbos, não só os cinco que
// a issue lista.
//
// # O buraco que isto fecha, e como ele passou despercebido
//
// A peça em Datastar não tinha gesto NENHUM além de arrastar, e o `BoardStore` já
// sabia esconder, duplicar, editar e remover desde a ALE-178 — a mesma forma da
// cortina: a capacidade no ar e invisível.
//
// Ele escapou da lista das dez superfícies porque aquele levantamento cruzou
// RÓTULOS, e os rótulos da peça são todos interpolados (`Esconder ${token.label}`):
// eles não casam com texto nenhum de nenhum dos dois lados. É a limitação
// conhecida daquele método, e vale anotá-la — a próxima varredura que confiar só
// em texto vai perder exatamente a mesma família.
//
// A pior das seis é ESCONDER, e a razão é que ela deixa outra superfície mentindo:
// "ver como jogador" (ALE-193) existe para conferir a emboscada, e sem um gesto
// de esconder ela responde sempre "nenhuma peça escondida nesta cena".
//
// # O clique direito já tem dono, e é a FERRAMENTA que arbitra
//
// A issue avisa: na SPA o clique direito apaga terreno com a borracha rápida, e
// "provavelmente o menu só existe fora das ferramentas de pintura". Nesta Mesa a
// regra sai de graça e por CONSTRUÇÃO: com ferramenta ligada a peça já é inerte
// ao ponteiro (o `.board-with-tool` da superfície 8), então o clique
// direito sobre ela nem chega à peça. Nenhuma condição a mais para lembrar.

func (s Scene) TokenActionRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro/pecas/{tokenId}"
	r.Post(base+"/visibilidade", s.gmBoardCommand(toggleVisibility))
	// TRÊS rotas de duplicar e não uma com parâmetro, porque são três VERBOS na
	// tela e o endereço é o que o menu escreve. O que muda entre elas é só o
	// modo — ver o `duplicatesWith`.
	r.Post(base+"/duplicar/"+modoSoAPeca, s.gmBoardCommand(duplicatesWith(modoSoAPeca)))
	r.Post(base+"/duplicar/"+modoJunto, s.gmBoardCommand(duplicatesWith(modoJunto)))
	r.Post(base+"/duplicar/"+modoSozinha, s.gmBoardCommand(duplicatesWith(modoSozinha)))
	r.Post(base+"/duplicar/"+modoBloco, s.gmBoardCommand(duplicatesWith(modoBloco)))
	// COLAR não é de uma peça e por isso não pende do `base`: a peça de origem
	// pode estar em OUTRA aba, e quem a nomeia é a área de transferência de quem
	// clicou, não o caminho. O que vem no caminho é o QUADRADO, que é a única
	// coisa que o cliente sabe e o servidor não — ele não conhece o zoom nem
	// onde cada pessoa está olhando.
	r.Post("/mesa/{campaignId}/{sessionId}/tabuleiro/colar/{x}/{y}", s.gmBoardCommand(pastesToken))
	r.Post(base+"/voltar", s.gmBoardCommand(wasWhereForTokenBack))
	r.Post(base+"/editar", s.gmBoardCommand(editsToken))
	r.Post(base+"/remover", s.gmBoardCommand(removesToken))
}

// toggleVisibility é o gesto da EMBOSCADA.
//
// ALTERNA e não recebe o estado desejado, ao contrário do pincel de terreno: é UM
// estado com dois lados e um botão com `aria-pressed`. Mandar o valor da tela
// faria dois cliques rápidos com a resposta atrasada apagarem um ao outro — e
// aqui o resultado desse empate é a emboscada aparecendo para a mesa.
func toggleVisibility(st Scene, c commandCtx) (*board.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().UpdateToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID,
		board.ParseTokenPatch(map[string]any{"hidden": !peca.Hidden}))
}

// OS TRÊS DUPLICARES, e a diferença entre eles é o que a cópia faz com a LINHA
// DA FILA (ALE-206).
//
// A cópia nasce AO LADO da original e com o número seguinte no nome, e o servidor
// é quem numera — duas telas escolhendo por conta própria é como nasce o segundo
// "Zumbi 3" no mesmo mapa.
//
// O eixo é a LINHA e não a ficha, e a issue nasceu dizendo o contrário. A barra
// de PV da peça é indexada por `entryId` (`saude[*t.EntryID]`, no `board_view`),
// então é a linha que decide se um dano aparece nas duas peças ou só numa — e o
// zumbi do exemplo sequer tem ficha, porque NPC entra na fila com `characterId`
// nulo por construção. Duplicar "apontando para a mesma ficha" seria um no-op
// exatamente no caso que motivou a issue.

// duplicatesWith é o construtor dos TRÊS duplicares: a peça é a mesma, e o que
// muda é o laço.
//
// Um construtor e não três funções porque a diferença entre eles cabe inteira no
// `bondForMode` — três corpos seriam três lugares para o "sangrando junto" do
// duplicar e o do colar discordarem sobre o que a palavra significa.
func duplicatesWith(modo string) func(Scene, commandCtx) (*board.BoardState, error) {
	return func(st Scene, c commandCtx) (*board.BoardState, error) {
		peca, err := st.tokenOfCommand(c)
		if err != nil {
			return nil, err
		}
		laco, err := st.bondForMode(c, modo, peca)
		if err != nil {
			return nil, err
		}
		return st.deps.Boards().DuplicateToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID, laco)
	}
}

// clipboardSignals é a ÁREA DE TRANSFERÊNCIA de quem clicou, e ela viaja do
// cliente porque é dele: a área é de quem copiou, não da mesa.
//
// Nomes em `snake_case` pela mesma razão do `tokenSignals`: o analisador de HTML
// minuscula chave de atributo, então caixa alta ali liga um sinal novo e o
// servidor fica lendo o antigo, para sempre vazio. O `_` atravessa intacto.
type clipboardSignals struct {
	Peca      string `json:"area_token"`
	Tabuleiro string `json:"area_board"`
	Modo      string `json:"area_mode"`
}

// pastesToken põe outra igual onde a pessoa está OLHANDO (ALE-206).
//
// # Por que ele existe, tendo duplicar
//
// O duplicar põe a cópia colada na original. O colar faz três coisas que ele não
// faz: repete sem perguntar de novo, pousa onde se está olhando depois de
// arrastar o mapa, e ATRAVESSA AS ABAS — copiar o zumbi na Cripta e colá-lo na
// Taverna. É essa terceira que não tinha caminho nenhum antes.
//
// # O modo foi decidido no COPIAR, e não aqui
//
// Decisão do dono (2026-09-05): a issue dizia "ao colar, perguntar", e perguntar
// a cada `CTRL+V` mataria o valor do teclado, que é repetir. A pergunta continua
// sendo feita uma vez, no menu, no momento de copiar — e o que a área guarda é a
// resposta.
//
// # Colar ENTRE abas com "sangrando junto" é permitido, e é decisão
//
// O mesmo combatente passa a ter peça em dois mapas, com uma barra de PV só.
// Poderia ser recusado, e não é: a regra do modo é "um combatente, um PV", e ela
// não fala de mapa. Quem cola na outra aba escolheu isso na tela que diz o modo.
// O que ele custa está escrito para o dia em que incomodar: a marca da VEZ
// acende nas duas abas, e quem estiver olhando a Taverna vê o ogro aceso com o
// combate acontecendo na Cripta.
func pastesToken(st Scene, c commandCtx) (*board.BoardState, error) {
	var area clipboardSignals
	if err := datastar.ReadSignals(c.R, &area); err != nil {
		return nil, fmt.Errorf("não entendi o que há na área: %v", err)
	}
	if area.Peca == "" {
		return nil, fmt.Errorf("não há peça na área — copie uma primeiro, pelo menu dela")
	}
	// A ORIGEM é o tabuleiro de onde a peça foi copiada, e não o que está na
	// tela: são diferentes justamente quando o colar mais serve.
	origem := st.deps.Boards().Get(c.R.Context(), c.SessionID, area.Tabuleiro)
	modelo := board.FindToken(origem, area.Peca)
	if modelo == nil {
		return nil, fmt.Errorf("a peça que estava na área não está mais no tabuleiro de origem")
	}
	laco, err := st.bondForMode(c, area.Modo, modelo)
	if err != nil {
		return nil, err
	}
	x, y, err := squareOfCommand(c)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().PasteToken(c.R.Context(), c.SessionID, c.TabuleiroID, *modelo, laco, x, y)
}

// squareOfCommand lê o quadrado que o cliente calculou.
func squareOfCommand(c commandCtx) (int, int, error) {
	x, errX := strconv.Atoi(chi.URLParam(c.R, "x"))
	y, errY := strconv.Atoi(chi.URLParam(c.R, "y"))
	if errX != nil || errY != nil {
		return 0, 0, fmt.Errorf("o quadrado de destino veio ilegível: %q, %q",
			chi.URLParam(c.R, "x"), chi.URLParam(c.R, "y"))
	}
	return x, y, nil
}

// bondForMode traduz o modo guardado na área para o LAÇO da cópia.
//
// Ele é o mesmo mapa que os três verbos de duplicar usam, escrito uma vez: o
// colar e o duplicar têm de concordar sobre o que "sangrando junto" significa, e
// duas traduções seriam dois lugares para discordar.
func (s Scene) bondForMode(c commandCtx, modo string, modelo *board.BoardToken) (*aovivo.InitiativeEntry, error) {
	if modo == modoSoAPeca {
		return nil, nil
	}
	linha := s.queueLineOf(c.SessionID, modelo)
	if linha == nil {
		return nil, fmt.Errorf("%s não é um combatente da fila, e sem PV não há o que dividir nem o que copiar", modelo.Label)
	}
	if modo == modoJunto {
		return linha, nil
	}
	if modo != modoSozinha && modo != modoBloco {
		return nil, fmt.Errorf("modo de cópia desconhecido: %q", modo)
	}
	linhaModelo := *linha
	if modo == modoBloco {
		// O BLOCO é clonado ANTES da linha, e a ordem importa: a linha nova já
		// nasce apontando para a cópia. Criar a linha primeiro e remendá-la
		// depois deixaria uma janela em que ela aponta para o bloco da original —
		// e nessa janela um remendo da cena desenharia o chefe com a ficha errada.
		if linha.CreatureID == nil {
			return nil, fmt.Errorf("%s não tem bloco de criatura: não há o que copiar", linha.Label)
		}
		nomeDaCopia := s.nextNameForTheLine(c.SessionID, linha.Label)
		blocoNovo, err := s.deps.CloneCreatureBlock(c.R.Context(), *linha.CreatureID, c.CampaignID, nomeDaCopia)
		if err != nil {
			return nil, err
		}
		linhaModelo.CreatureID = &blocoNovo
	}
	nova, err := s.addsACopyOfTheLine(c.SessionID, linhaModelo)
	if err != nil {
		return nil, err
	}
	if fila := s.deps.Sessions().GetState(c.SessionID); fila != nil {
		s.deps.PublishSessionState(c.SessionID, fila)
	}
	return nova, nil
}

// Os três modos, e eles são a MESMA palavra na rota do duplicar, no sinal da
// área e aqui. Escritos uma vez porque um terceiro lugar com a string à mão é o
// lugar onde alguém digita "sozinho".
const (
	modoSoAPeca = "peca"
	modoJunto   = "junto"
	modoSozinha = "sozinha"
	// modoBloco é o "chefe que ganha nome": linha nova E bloco de criatura
	// próprio, para o mestre editar um sem mexer nos outros.
	modoBloco = "bloco"
)

// queueLineOf é a linha da fila por trás de uma peça, ou nulo.
func (s Scene) queueLineOf(sessionID int64, peca *board.BoardToken) *aovivo.InitiativeEntry {
	if peca.EntryID == nil {
		return nil
	}
	estado := s.deps.Sessions().GetState(sessionID)
	if estado == nil {
		return nil
	}
	for i := range estado.Initiative {
		if estado.Initiative[i].ID == *peca.EntryID {
			return &estado.Initiative[i]
		}
	}
	return nil
}

// nextNameForTheLine é o nome que a cópia VAI receber, calculado ANTES de ela
// existir.
//
// Ele existe porque o bloco é clonado antes da linha, e o bloco leva nome: sem
// isto, o acervo do mestre ficaria com dois "Zumbi" e a linha com "Zumbi 2" —
// dois nomes para a mesma criatura, e o olho da fila abrindo um bloco que se
// chama outra coisa.
//
// A conta é a MESMA do `AddEntry` (o `numberedLabel`), e repeti-la aqui é o
// preço de precisar do nome cedo. Ela não briga: quem numera de verdade continua
// sendo o `AddEntry`, e este valor só decide como o BLOCO se chama.
func (s Scene) nextNameForTheLine(sessionID int64, rotulo string) string {
	estado := s.deps.Sessions().GetState(sessionID)
	if estado == nil {
		return rotulo
	}
	usados := make([]string, 0, len(estado.Initiative))
	for i := range estado.Initiative {
		usados = append(usados, estado.Initiative[i].Label)
	}
	return aovivo.NextInstanceLabelAmong(usados, rotulo)
}

// addsACopyOfTheLine põe na fila outra linha igual à dada, e devolve a que
// nasceu.
//
// O PV ATUAL vira o MÁXIMO da nova, e não o máximo da original: o segundo zumbi
// chega inteiro, não com os 12 de 130 que o primeiro levou de porrada. Quem
// quisesse o contrário estaria pedindo "sangra junto", que é o outro verbo.
//
// As CONDIÇÕES ficam para trás pela mesma razão: caído e sangrando são estado de
// combate do primeiro, e o que entra agora entra de pé.
//
// A linha nova é achada por DIFERENÇA e nunca pelo último da lista: o `AddEntry`
// ORDENA a fila por iniciativa depois de inserir, então a recém-chegada pode
// pousar em qualquer posição. Pegar `Initiative[len-1]` daria a de menor
// iniciativa da mesa, e daria certo por acaso sempre que o zumbi fosse lento.
func (s Scene) addsACopyOfTheLine(sessionID int64, modelo aovivo.InitiativeEntry) (*aovivo.InitiativeEntry, error) {
	antes := map[string]bool{}
	if estado := s.deps.Sessions().GetState(sessionID); estado != nil {
		for i := range estado.Initiative {
			antes[estado.Initiative[i].ID] = true
		}
	}
	nova := modelo
	nova.ID = ""
	nova.Conditions = nil
	if modelo.HpMax != nil {
		cheia := aovivo.DerefOr(modelo.HpMax, 0)
		nova.HpCurrent, nova.HpMax = &cheia, &cheia
	}
	depois, err := s.deps.Sessions().AddInitiativeEntry(sessionID, nova)
	if err != nil {
		return nil, err
	}
	for i := range depois.Initiative {
		if !antes[depois.Initiative[i].ID] {
			return &depois.Initiative[i], nil
		}
	}
	return nil, fmt.Errorf("a linha de %s não entrou na fila", modelo.Label)
}

// wasWhereForTokenBack desfaz o último pouso (ALE-206).
//
// "Arrastei o dragão para o lugar errado na frente de seis pessoas" é o gesto que
// ela conserta, e é por isso que a memória mora na PEÇA e não na tela: quem
// precisa desfazer pode ter recarregado a página, ou estar na outra aba.
//
// UMA vez e não uma pilha: voltar limpa o registro, então o botão some depois de
// usado. Um "voltar" que continuasse disponível andaria para trás na cena sem
// dizer até onde vai.
func wasWhereForTokenBack(st Scene, c commandCtx) (*board.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	if peca.DeOndeVeio == nil {
		return nil, fmt.Errorf("%s não foi movida nesta cena: não há para onde voltar", peca.Label)
	}
	return st.deps.Boards().ReturnToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID)
}

// tokenSignals é o que o diálogo de editar manda.
//
// Nomes em `snake_case` porque viram chave de atributo, e o analisador de HTML
// minuscula chave de atributo, então caixa alta ali chega minúscula e liga um
// sinal NOVO, com o servidor lendo o antigo para sempre vazio.
type tokenSignals struct {
	Nome    string `json:"token_name"`
	Tamanho int    `json:"token_size"`
}

// editsToken muda o NOME e o TAMANHO.
//
// Os dois juntos porque são a mesma pergunta — "o que é esta peça?" —, e porque o
// tamanho é o que decide quantos quadrados ela ocupa (T20 p107, Tab. 1-21): uma
// peça Grande desenhada em 1×1 mente sobre quem o gabarito pega e sobre onde cabe
// passar.
func editsToken(st Scene, c commandCtx) (*board.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	var sinais tokenSignals
	if err := datastar.ReadSignals(c.R, &sinais); err != nil {
		return nil, fmt.Errorf("não entendi o formulário da peça: %v", err)
	}
	nome := strings.TrimSpace(sinais.Nome)
	if nome == "" {
		return nil, fmt.Errorf("a peça precisa de um nome")
	}
	if !tokenSize(sinais.Tamanho) {
		return nil, fmt.Errorf("uma peça ocupa 1, 2, 3 ou 6 quadrados de lado (p107); veio %d", sinais.Tamanho)
	}
	return st.deps.Boards().UpdateToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID,
		board.ParseTokenPatch(map[string]any{"label": nome, "footprint": sinais.Tamanho}))
}

// removesToken tira a peça do tabuleiro, e SÓ do tabuleiro.
//
// A linha da iniciativa fica: são dois gestos porque respondem a duas perguntas —
// "ele saiu do mapa" e "ele saiu do combate" —, e juntá-los faria o mestre perder
// o combatente ao arrumar a cena. É a mesma separação que o elenco e a fila já
// têm (superfície 6b).
func removesToken(st Scene, c commandCtx) (*board.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().RemoveToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID)
}

// tokenOfCommand lê a peça pelo id do CAMINHO, e recusa a que não existe.
//
// Sem a leitura, cada verbo teria de tratar "a peça sumiu" por conta própria — e
// ela some de verdade: outra aba do mestre pode ter removido a mesma peça meio
// segundo antes. A frase diz o id porque é ele que o botão carregava.
func (s Scene) tokenOfCommand(c commandCtx) (*board.BoardToken, error) {
	b := s.deps.Boards().Get(c.R.Context(), c.SessionID, c.TabuleiroID)
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	tokenID := chi.URLParam(c.R, "tokenId")
	peca := board.FindToken(b, tokenID)
	if peca == nil {
		return nil, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	return peca, nil
}

// tokenSizes são os lados que o livro define (T20 p107, Tab. 1-21).
//
// NÃO existe 4 nem 5, e é por isso que isto é uma lista fechada e não um campo de
// número: Minúsculo, Pequeno e Médio ocupam 1; Grande 2; Enorme 3; Colossal 6.
// Um seletor com os números do livro impede a peça de lado 4 que nenhuma criatura
// tem.
var tokenSizes = []struct {
	Lado   int
	Rotulo string
}{
	{1, "Médio ou menor · 1×1"},
	{2, "Grande · 2×2"},
	{3, "Enorme · 3×3"},
	{6, "Colossal · 6×6"},
}

func tokenSize(lado int) bool {
	for _, t := range tokenSizes {
		if t.Lado == lado {
			return true
		}
	}
	return false
}

// ── as expressões da tela ────────────────────────────────────────────────────

// chosenToken é o teste que abre o menu de UMA peça.
//
// UM SINAL com o id dentro, e não um booleano por peça: com dez zumbis no mapa,
// dez sinais dariam dez lugares onde dois menus podem estar abertos ao mesmo
// tempo. Com o id, a exclusão é por construção — a mesma escolha do `$tool`
// e do `$marker_chosen`.
func chosenToken(id string) string {
	return fmt.Sprintf("$token_chosen === %q", id)
}

// openMenuToken é o clique DIREITO.
//
// `preventDefault` porque o menu do navegador cobriria o nosso; e o gesto NUNCA é
// o único caminho — a issue pede isso e a peça continua tendo o clique esquerdo
// para mover, o teclado para focar e o `Enter` para abrir o mesmo menu.
func openMenuToken(id string) string {
	return fmt.Sprintf("evt.preventDefault(); $token_chosen = %q", id)
}

// closeMenuToken é a saída, e ela existe em DOIS lugares: o ✕ do menu e o gesto
// que abre outra peça (o mesmo sinal recebendo outro id).
//
// Aqui morava "três lugares: o ✕ do menu, a tecla Esc e …", e o Esc nunca
// funcionou — medido na ALE-206: com o menu aberto, `Escape` o deixa `display:
// flex` e só o ✕ o fecha. Não é um defeito a consertar, é uma promessa a
// retirar: o `scene.js` mapeia Escape para "voltar" e chama `stopPropagation` no
// documento, então ele não chega. O `railKeyboard` e o `clickedPointRuler` já
// tinham medido exatamente isso, cada um no seu canto, e os dois escrevem que
// um ramo de Escape ali "seria uma promessa que a tela não cumpre". Este
// comentário era essa promessa, escrita.
//
// Ele apaga o SUBMENU junto, e o `openMenuToken` também: sem isso, abrir o menu
// de outra peça a mostraria com a segunda camada já aberta, porque o
// `$pecacopia` guarda um id e não um booleano. É a mesma armadilha do nó
// COMPARTILHADO que o `openEditToken` registra logo abaixo — quem troca de peça
// é quem tem de limpar o que a anterior deixou.
const closeMenuToken = "$token_chosen = ''"

// copyMenuId nomeia a segunda camada de UMA peça.
//
// Um id por peça porque o popover nativo casa gatilho e painel por id, e o menu
// existe no HTML de toda peça do mapa.
func copyMenuId(tokenID string) string {
	return "peca-copia-" + tokenID
}

// closesTheCopyMenu fecha a segunda camada por JS, e ela existe porque escolher
// um modo tem de fechar as DUAS camadas: o popover não se fecha sozinho quando o
// clique é num botão dentro dele.
func closesTheCopyMenu(tokenID string) string {
	return fmt.Sprintf("document.getElementById(%q)?.hidePopover(); ", copyMenuId(tokenID))
}

// putsInTheClipboard é o gesto de COPIAR: ele não chama o servidor.
//
// Copiar é decisão de quem olha, e ela mora no cliente inteira — a área é de
// QUEM COPIOU, não da mesa. Uma rota aqui gravaria por usuário e por sessão um
// estado que ninguém pediu, e que o mestre encontraria cheio no dia seguinte.
//
// O RÓTULO viaja junto para a faixa poder dizer o que está na área sem uma
// segunda ida ao servidor. Ele é só para ler: quem manda no que se cola é o par
// `areapeca` + `areatabuleiro`.
func putsInTheClipboard(v BoardView, p boardToken, modo, frase string) string {
	return closesTheCopyMenu(p.ID) + fmt.Sprintf(
		"$area_token = %q; $area_board = %q; $area_mode = %q; $area_label = %q; $area_phrase = %q; ",
		p.ID, v.TabuleiroID, modo, p.Rotulo, frase,
	) + closeMenuToken
}

// emptiesTheClipboard limpa a área, e o gesto é um BOTÃO e nunca o Esc.
//
// O Esc não chega: o `scene.js` o mapeia para "voltar" e o mata no documento —
// medido na ALE-206, e o `railKeyboard` e o `clickedPointRuler` já registram o
// mesmo. Uma faixa que dissesse "Esc limpa" prometeria o que a tela não cumpre.
const emptiesTheClipboard = "$area_token = ''; $area_board = ''; $area_mode = ''; " +
	"$area_label = ''; $area_phrase = ''"

// pasteInTheMiddleOfTheView é o `CTRL + V`, e o quadrado é o CENTRO do que se vê.
//
// A conta é a inversa do `centerViewport`, e é do cliente pelo mesmo motivo de
// sempre: o servidor não sabe o zoom nem para onde cada pessoa arrastou o mapa.
//
// `preventDefault` porque o `CTRL + V` é do navegador antes de ser nosso, e sem
// ele o colar da página dispara junto. O `typingTargetWithout` é o outro lado da
// mesma promessa: dentro de um campo de texto a tecla continua sendo do texto,
// que é o que a issue pede com todas as letras.
func pasteInTheMiddleOfTheView(v BoardView) string {
	meioX := fmt.Sprintf("Math.floor(($viewport_x + document.getElementById(%q).clientWidth / 2) / $square)", sceneId)
	meioY := fmt.Sprintf("Math.floor(($viewport_y + document.getElementById(%q).clientHeight / 2) / $square)", sceneId)
	return typingTargetWithout +
		fmt.Sprintf("(evt.key === 'v' || evt.key === 'V') && (evt.ctrlKey || evt.metaKey) && $area_token !== '' "+
			"? (evt.preventDefault(), @post('%s/colar/' + (%s) + '/' + (%s))) : null",
			v.Base, meioX, meioY)
}

// copyCommand é o gesto de um dos três modos: manda e fecha as duas camadas.
//
// Fechar faz parte do gesto porque a resposta REDESENHA o tabuleiro inteiro, e um
// submenu que sobrevive ao redesenho fica pendurado sobre uma peça que já ganhou
// irmã — pedindo um segundo clique para dizer que acabou.
func copyCommand(v BoardView, id, modo string) string {
	return closesTheCopyMenu(id) + tokenCommand(v, id, "duplicar/"+modo) + "; " + closeMenuToken
}

// tokenCommand escreve o gesto de um verbo do menu.
func tokenCommand(v BoardView, id, acao string) string {
	return fmt.Sprintf("@post('%s/pecas/%s/%s')", v.Base, id, acao)
}

// openEditToken semeia o formulário com o que a peça É hoje, e só então abre.
//
// Semear no GESTO e não no HTML é a regra da casa para nó COMPARTILHADO: o
// diálogo de editar é UM só para todas as peças, e quem troca de peça é quem tem
// de limpar o que a anterior deixou. Sem isto, abrir o Ogro depois do Zumbi
// mostraria o nome do Zumbi sobre o Ogro — o defeito do link de redefinição de
// senha, de novo.
func openEditToken(p boardToken) string {
	return fmt.Sprintf("$token_edited = %q; $token_name = %q; $token_size = %d; %s; "+
		"document.getElementById('edit-token').showModal()",
		p.ID, p.Rotulo, p.Pegada, closeMenuToken)
}

// saveEditToken manda o formulário para a peça que o gesto de abrir marcou.
//
// O id vem de `$token_edited` e não de `$token_chosen`, e os dois existem por
// isso: abrir o diálogo FECHA o menu — senão ele ficaria aceso atrás do modal —,
// e um sinal só faria o gesto de abrir apagar o alvo do gesto de salvar.
// FECHA ANTES de comandar, que é o que todo diálogo desta cena faz — o de abrir
// a cena, o de encerrar, o do acervo. A recusa cai no `command_error` do rodapé
// do mestre, e ela só é legível com o modal fora do caminho: uma frase escrita
// atrás de um `<dialog>` aberto é uma frase que ninguém lê.
func saveEditToken(v BoardView) string {
	return fmt.Sprintf(
		"document.getElementById('edit-token').close(); "+
			"@post('%s/pecas/' + $token_edited + '/editar')",
		v.Base)
}

// visibilityName diz o VERBO que o clique executa, e não o estado atual.
//
// "Esconder" numa peça visível e "Mostrar" numa escondida: nome acessível de
// botão é o que ele FAZ. O estado quem carrega é o `aria-pressed`.
func visibilityName(p boardToken) string {
	if p.Oculta {
		return "Mostrar " + p.Rotulo + " à mesa"
	}
	return "Esconder " + p.Rotulo + " da mesa"
}

// tokenSquare é onde ela estava, para a frase do "voltar" dizer o destino.
func tokenSquare(q *engine.Square) string {
	if q == nil {
		return ""
	}
	return Coordinate(q.X, q.Y)
}
