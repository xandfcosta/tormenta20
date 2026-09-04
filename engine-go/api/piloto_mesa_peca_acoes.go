package api

import (
	"fmt"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/engine"
	"t20engine/tabuleiro"
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
// ao ponteiro (o `.tabuleiro-com-ferramenta` da superfície 8), então o clique
// direito sobre ela nem chega à peça. Nenhuma condição a mais para lembrar.

func (s *Server) TokenActionRoutes(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro/pecas/{tokenId}"
	r.Post(base+"/visibilidade", s.gmBoardCommand(toggleVisibility))
	r.Post(base+"/duplicar", s.gmBoardCommand(duplicatesToken))
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
func toggleVisibility(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	return st.boards.UpdateToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID,
		tabuleiro.ParseTokenPatch(map[string]any{"hidden": !peca.Hidden}))
}

// duplicatesToken é "mais um zumbi" (ALE-192), a operação mais repetida ao montar
// encontro.
//
// A cópia nasce AO LADO da original e com o número seguinte no nome, e o servidor
// é quem numera — duas telas escolhendo por conta própria é como nasce o segundo
// "Zumbi 3" no mesmo mapa.
func duplicatesToken(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	return st.boards.DuplicateToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID)
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
func wasWhereForTokenBack(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	if peca.DeOndeVeio == nil {
		return nil, fmt.Errorf("%s não foi movida nesta cena: não há para onde voltar", peca.Label)
	}
	return st.boards.ReturnToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID)
}

// tokenSignals é o que o diálogo de editar manda.
//
// Nomes TODOS MINÚSCULOS porque viram chave de atributo, e o analisador de HTML
// minuscula chave — um `data-bind:pecaTamanho` chega como `pecatamanho` e liga um
// sinal NOVO, com o servidor lendo o antigo para sempre vazio.
type tokenSignals struct {
	Nome    string `json:"pecanome"`
	Tamanho int    `json:"pecatamanho"`
}

// editsToken muda o NOME e o TAMANHO.
//
// Os dois juntos porque são a mesma pergunta — "o que é esta peça?" —, e porque o
// tamanho é o que decide quantos quadrados ela ocupa (T20 p107, Tab. 1-21): uma
// peça Grande desenhada em 1×1 mente sobre quem o gabarito pega e sobre onde cabe
// passar.
func editsToken(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
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
	return st.boards.UpdateToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID,
		tabuleiro.ParseTokenPatch(map[string]any{"label": nome, "footprint": sinais.Tamanho}))
}

// removesToken tira a peça do tabuleiro, e SÓ do tabuleiro.
//
// A linha da iniciativa fica: são dois gestos porque respondem a duas perguntas —
// "ele saiu do mapa" e "ele saiu do combate" —, e juntá-los faria o mestre perder
// o combatente ao arrumar a cena. É a mesma separação que o elenco e a fila já
// têm (superfície 6b).
func removesToken(st *Server, c commandCtx) (*tabuleiro.BoardState, error) {
	peca, err := st.tokenOfCommand(c)
	if err != nil {
		return nil, err
	}
	return st.boards.RemoveToken(c.R.Context(), c.SessionID, c.TabuleiroID, peca.ID)
}

// tokenOfCommand lê a peça pelo id do CAMINHO, e recusa a que não existe.
//
// Sem a leitura, cada verbo teria de tratar "a peça sumiu" por conta própria — e
// ela some de verdade: outra aba do mestre pode ter removido a mesma peça meio
// segundo antes. A frase diz o id porque é ele que o botão carregava.
func (s *Server) tokenOfCommand(c commandCtx) (*tabuleiro.BoardToken, error) {
	b := s.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID)
	if b == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto nesta mesa")
	}
	tokenID := chi.URLParam(c.R, "tokenId")
	peca := tabuleiro.FindToken(b, tokenID)
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
// tempo. Com o id, a exclusão é por construção — a mesma escolha do `$ferramenta`
// e do `$marcadorescolhido`.
func chosenToken(id string) string {
	return fmt.Sprintf("$pecaescolhida === %q", id)
}

// openMenuToken é o clique DIREITO.
//
// `preventDefault` porque o menu do navegador cobriria o nosso; e o gesto NUNCA é
// o único caminho — a issue pede isso e a peça continua tendo o clique esquerdo
// para mover, o teclado para focar e o `Enter` para abrir o mesmo menu.
func openMenuToken(id string) string {
	return fmt.Sprintf("evt.preventDefault(); $pecaescolhida = %q", id)
}

// closeMenuToken é a saída, e ela existe em três lugares: o ✕ do menu, a tecla
// Esc e o gesto que abre OUTRA peça (que é o mesmo sinal recebendo outro id).
const closeMenuToken = "$pecaescolhida = ''"

// tokenCommand escreve o gesto de um verbo do menu.
func tokenCommand(v boardView, id, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/pecas/%s/%s')",
		v.CampaignID, v.SessionID, id, acao)
}

// openEditToken semeia o formulário com o que a peça É hoje, e só então abre.
//
// Semear no GESTO e não no HTML é a regra da casa para nó COMPARTILHADO: o
// diálogo de editar é UM só para todas as peças, e quem troca de peça é quem tem
// de limpar o que a anterior deixou. Sem isto, abrir o Ogro depois do Zumbi
// mostraria o nome do Zumbi sobre o Ogro — o defeito do link de redefinição de
// senha, de novo.
func openEditToken(p boardToken) string {
	return fmt.Sprintf("$pecaeditada = %q; $pecanome = %q; $pecatamanho = %d; %s; "+
		"document.getElementById('editar-peca').showModal()",
		p.ID, p.Rotulo, p.Pegada, closeMenuToken)
}

// saveEditToken manda o formulário para a peça que o gesto de abrir marcou.
//
// O id vem de `$pecaeditada` e não de `$pecaescolhida`, e os dois existem por
// isso: abrir o diálogo FECHA o menu — senão ele ficaria aceso atrás do modal —,
// e um sinal só faria o gesto de abrir apagar o alvo do gesto de salvar.
// FECHA ANTES de comandar, que é o que todo diálogo desta cena faz — o de abrir
// a cena, o de encerrar, o do acervo. A recusa cai no `erroDoComando` do rodapé
// do mestre, e ela só é legível com o modal fora do caminho: uma frase escrita
// atrás de um `<dialog>` aberto é uma frase que ninguém lê.
func saveEditToken(v boardView) string {
	return fmt.Sprintf(
		"document.getElementById('editar-peca').close(); "+
			"@post('/mesa/%d/%d/tabuleiro/pecas/' + $pecaeditada + '/editar')",
		v.CampaignID, v.SessionID)
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
	return coordenada(q.X, q.Y)
}
