package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Os guardas da ENTRADA DO PALCO (ALE-235, entregue na ALE-239) — nas DUAS
// cenas de seleção desde a ALE-297.
//
// A animação em si é do navegador e só o e2e a vê — linha do tempo, `animationstart`
// e duração não existem fora dele. O que se prende AQUI é o que o servidor
// escreve: as classes que substituem o mount e o gesto que diz o sentido. É a
// divisão de sempre — a regra na camada mais barata que a segura.
//
// # POR QUE ELES VARREM AS DUAS CENAS
//
// Enquanto o elenco era a única com palco, pedir `/personagens` era cobertura.
// No dia em que a campanha virou palco, o MESMO guarda passou a medir metade do
// terreno — e um gesto sem sentido nasceria em campanhas sem ninguém acusar. É a
// forma exata do "um guarda só mede o que ele VISITA", e a saída barata aqui é
// enumeração: são duas cenas, elas estão nesta tabela, e a terceira que nascer
// precisa entrar. Enumerar é remendo; o que restauraria a amostragem seria a
// cena nova não poder existir fora desta lista, e isso não é verdade hoje.

// cenaDePalco é uma cena de seleção montada e pronta para ser pedida.
type cenaDePalco struct {
	nome string
	rota string
	// itens é quantos itens de verdade o trilho tem, SEM a vaga do fim. Ele é o
	// denominador do controle: cada item do trilho escreve o cursor no clique e
	// no foco, e a vaga faz o mesmo.
	itens int
	quem  int64
	f     pilotoFixture
}

// asCenasDePalco monta as duas com TRÊS itens cada, e três não é número redondo
// escolhido por gosto: com um item só não há vizinho nenhum (as duas pontas
// viram espaçador), e é justamente o clique no retrato vizinho que o guarda
// precisa visitar. Com três, o do meio tem os dois.
func asCenasDePalco(t *testing.T) []cenaDePalco {
	t.Helper()

	elenco := novoPiloto(t)
	// O `novoPiloto` já semeia um herói para o jogador; faltam dois.
	seedCharacterAtLevel(t, elenco.s, elenco.jogador, "Anã Clériga", 4, 20, 20, 6, 6)
	seedCharacterAtLevel(t, elenco.s, elenco.jogador, "Elfo Ladino", 2, 14, 14, 0, 0)

	// E já semeia uma campanha para o mestre; faltam duas. A cena de campanhas é
	// pedida pelo MESTRE, e não pelo jogador: é ele que tem mesa.
	campanhas := novoPiloto(t)
	seedCampaign(t, campanhas.s, campanhas.mestre)
	seedCampaign(t, campanhas.s, campanhas.mestre)

	return []cenaDePalco{
		{nome: "o elenco", rota: "/personagens", itens: 3, quem: elenco.jogador, f: elenco},
		{nome: "as campanhas", rota: "/campanhas", itens: 3, quem: campanhas.mestre, f: campanhas},
	}
}

func (c cenaDePalco) tela(t *testing.T) string {
	t.Helper()
	return c.f.pede(t, c.quem, http.MethodGet, c.rota, "").Body.String()
}

// TODO GESTO QUE MOVE O CURSOR DIZ O SENTIDO — a varredura da convenção.
//
// São CINCO gestos por cena no mínimo (o marcador do trilho no clique e no foco,
// os dois retratos vizinhos e a vaga do fim), e cada um precisa escrever o
// sentido e o índice junto do cursor. Escrito à mão cinco vezes, o sexto é o que
// esquece: o palco entraria pelo lado errado, sem erro em lugar nenhum, e só
// quem conhece a animação notaria.
//
// O guarda falha com o TRECHO ofensor e com o NOME da cena, que é a diferença
// entre "conserte isto" e "procure".
func TestEveryGestureThatMovesTheCursorSaysTheDirection(t *testing.T) {
	for _, cena := range asCenasDePalco(t) {
		tela := cena.tela(t)

		// O CONTROLE vem primeiro: sem ele, "não achei escritor solto" é
		// indistinguível de "não achei escritor nenhum" — e as duas passam
		// verde.
		//
		// O piso sai do DESENHO e não do código medido: cada um dos `itens` do
		// trilho escreve o cursor no clique e no foco, e a vaga do fim também.
		// Os vizinhos acrescentam mais, e por isso é piso e não igualdade.
		piso := 2 * (cena.itens + 1)
		escritores := regexp.MustCompile(`\$cursor = \d+`).FindAllString(tela, -1)
		if len(escritores) < piso {
			t.Fatalf("%s: só %d gestos movem o cursor, e o desenho pede ao menos %d — o canal não está aberto, e a ausência abaixo não seria evidência",
				cena.nome, len(escritores), piso)
		}

		// Todo `$cursor =` tem de vir precedido da guarda que escreve o sentido.
		// A expressão inteira é `if ($indice != N) { … } $cursor = ID`, então
		// basta olhar o que vem ANTES na mesma expressão.
		for _, atributo := range regexp.MustCompile(`data-on:(?:click|focusin)="([^"]*\$cursor = \d+[^"]*)"`).FindAllStringSubmatch(tela, -1) {
			gesto := atributo[1]
			if !strings.Contains(gesto, "$sentido") || !strings.Contains(gesto, "$indice") {
				t.Errorf("%s: um gesto move o cursor sem dizer o sentido: %q — o palco entraria pelo lado errado, em silêncio", cena.nome, gesto)
			}
		}
	}
}

// AS DUAS PARTES QUE SE MOVEM existem no HTML, e são as que o CSS anima.
//
// A classe do palco não anima nada sozinha: quem tem `animation` são os
// descendentes `.palco-retrato` e `.palco-placa`. Um porte que renomeasse uma
// delas deixaria a animação viva e sem alvo — e o sintoma seria "metade do palco
// entra", que ninguém liga a um seletor de CSS.
func TestTheStageHasTheTwoPartsThatAnimate(t *testing.T) {
	for _, cena := range asCenasDePalco(t) {
		tela := cena.tela(t)

		for _, parte := range []string{"palco-retrato", "palco-placa"} {
			if !strings.Contains(tela, parte) {
				t.Errorf("%s não tem %q: a animação de entrada ficaria sem alvo", cena.nome, parte)
			}
		}
		// E a classe que ENTRA é escrita por `data-class`, não pelo `class`
		// fixo: no `class` ela nasceria em todos os palcos ao mesmo tempo, e a
		// animação tocaria uma vez só, na carga.
		if !strings.Contains(tela, "palco-entra-adiante") || !strings.Contains(tela, "palco-entra-atras") {
			t.Errorf("%s não escreve as duas direções da entrada", cena.nome)
		}
		if strings.Contains(tela, `class="relative flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-2 palco-entra`) {
			t.Errorf("%s pôs a classe de entrada no `class` fixo: ela tocaria na carga e nunca mais", cena.nome)
		}
	}
}

// Os sinais que o gesto ESCREVE são declarados pela cena, e são cenas
// diferentes que declaram.
//
// Uma cena que esquecesse `sentido` e `indice` não daria erro nenhum: o gesto
// escreveria em sinais recém-inventados e o primeiro passo do cursor entraria
// pelo lado errado, porque `undefined >= undefined` é `false`. O unitário do
// `ui` prende a FRASE; este prende que a cena de fato a escreve.
func TestEverySelectionSceneDeclaresTheSignalsTheGestureWrites(t *testing.T) {
	for _, cena := range asCenasDePalco(t) {
		tela := cena.tela(t)
		for _, sinal := range []string{"cursor:", "sentido:", "indice:"} {
			if !strings.Contains(tela, sinal) {
				t.Errorf("%s não declara %q, e o gesto escreve nele", cena.nome, sinal)
			}
		}
	}
}

// O LIVRO DE COURO NÃO VOLTA (ALE-297).
//
// A campanha em foco era um tomo aberto, e ele saiu por decisão do dono. As
// classes dele foram apagadas da folha junto, então um `class="grimorio-book"`
// escrito de novo não desenharia livro nenhum — desenharia uma caixa sem estilo,
// em silêncio, que é a família do "tinta sem caixa não desenha" pelo avesso.
//
// O `ui.TomeSheet` NÃO é isto e continua de pé: ele é a folha das telas de
// FORMULÁRIO (abrir campanha, entrar, forjar, a ficha), e é a identidade delas.
func TestNoSelectionSceneDrawsTheLeatherBook(t *testing.T) {
	for _, cena := range asCenasDePalco(t) {
		tela := cena.tela(t)
		for _, morta := range []string{"grimorio-book", "grimorio-leaf"} {
			if strings.Contains(tela, morta) {
				t.Errorf("%s escreve %q, e essa classe não existe mais na folha: a caixa sairia sem estilo nenhum", cena.nome, morta)
			}
		}
	}
}
