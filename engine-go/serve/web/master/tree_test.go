package master

import (
	"bytes"
	"context"
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/bookui"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/ui"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A ÁRVORE QUE O NAVEGADOR MONTA, e não a string que o servidor escreveu.
//
// A marcação é STRING montada no servidor, e não há compilador olhando: o
// `templ` confere se as tags fecham, não o modelo de conteúdo delas. O parser do
// navegador conserta o aninhamento inválido em silêncio, e ninguém escreve nada
// em lugar nenhum — foi assim que 24 parágrafos vazios atravessaram compilador,
// typecheck e guarda de contraste.
//
// A barreira se constrói com o `x/net/html`, que implementa o MESMO algoritmo de
// correção do navegador: se a árvore que ele monta não é a que o template
// descreve, o navegador também vai discordar.
//
// Ele é mais largo que o guarda de grep do `@ui.SectionLabel`: aquele conhece um
// componente, este mede o RESULTADO de qualquer cena.

// sceneAddresses são as cenas que este guarda visita.
//
// A lista é ENUMERAÇÃO, e enumeração é remendo. Ela existe porque renderizar uma
// cena exige montar a view dela, e não há como descobrir isso por reflexão.
// **Cena nova que não entrar aqui nasce sem medição.**
func sceneAddresses(t *testing.T) map[string]string {
	t.Helper()
	outside := map[string]string{}
	builds := func(name string, render func() (string, error)) {
		html, err := render()
		if err != nil {
			t.Fatalf("renderizar %s: %v", name, err)
		}
		outside[name] = html
	}

	ctx := context.Background()
	builds("bestiario", func() (string, error) {
		return ui.RenderFragment(ctx, bestiaryScene(LoadBestiaryFrom(routes.MasterBestiary, bookui.BookAddress{}, "", nil, book.CRMin, book.CRMax, "")))
	})
	builds("catalogos", func() (string, error) {
		return ui.RenderFragment(ctx, collectionScene(loadCollection(collectionCriteria{Term: "", Aba: "condicoes"}, bookui.BookAddress{})))
	})
	builds("catalogos-busca", func() (string, error) {
		return ui.RenderFragment(ctx, collectionScene(loadCollection(collectionCriteria{Term: "fogo", Aba: ""}, bookui.BookAddress{})))
	})
	builds("encontros", func() (string, error) {
		v := loadEncounters(3, 4, []encounterRow{{ID: "ogro", Qtd: 2}}, "ogro")
		return ui.RenderFragment(ctx, encountersScene(v))
	})
	builds("improviso", func() (string, error) {
		v := loadImprov(improvView{
			Rooms: 14,
			Ruin:  []roll{{Roll: 4, Text: "Vazia"}, {Roll: 2, Text: "Vazia"}},
		})
		return ui.RenderFragment(ctx, improvScene(v))
	})
	return outside
}

// O sinal é o PARÁGRAFO VAZIO: quando o parser encontra conteúdo de fluxo
// dentro de um `<p>`, ele fecha o parágrafo antes do intruso e o hoista — e
// sobra uma casca sem texto que o template nunca pediu. Um `<p>` deliberadamente
// vazio não existe em nenhuma cena da casa; se um dia existir, ele terá de
// ganhar uma exceção nomeada aqui, e essa conversa é melhor que o silêncio.
//
// Provado VERMELHO pondo o `<h4>` de volta dentro do `@ui.SectionLabel`: acusa o
// bestiário com 24 parágrafos vazios.
func TestTheBrowserDoesNotHaveToFixTheMarkup(t *testing.T) {
	scenes := sceneAddresses(t)
	if len(scenes) == 0 {
		t.Fatal("nenhuma cena foi montada: o guarda não visitaria nada e o verde não valeria")
	}
	for name, marking := range scenes {
		t.Run(name, func(t *testing.T) {
			root, err := html.Parse(strings.NewReader(marking))
			if err != nil {
				t.Fatalf("o parser recusou a marcação: %v", err)
			}
			empty := parágrafosVazios(root)
			if len(empty) > 0 {
				t.Errorf("%d parágrafo(s) vazio(s) na árvore de %s — o navegador CONSERTOU "+
					"aninhamento inválido, expulsando conteúdo de fluxo de dentro de um `<p>` "+
					"e deixando a casca para trás. A classe do parágrafo não alcança mais o "+
					"conteúdo, e nenhum guarda de contraste vê isso porque casca vazia não "+
					"tem texto para medir.", len(empty), name)
			}
		})
	}
}

// TestHeadingsAreNotChildrenOfAParagraph é o outro lado da mesma moeda, e ele
// pega o caso em que o parser hoista SEM deixar casca — quando o `<p>` tinha
// texto antes do cabeçalho, ele fica com o texto e o cabeçalho sai.
func TestHeadingsAreNotChildrenOfAParagraph(t *testing.T) {
	for name, marking := range sceneAddresses(t) {
		t.Run(name, func(t *testing.T) {
			root, err := html.Parse(strings.NewReader(marking))
			if err != nil {
				t.Fatalf("parser: %v", err)
			}
			// Reserializar e comparar a CONTAGEM de cabeçalhos por pai é caro;
			// o que basta é afirmar que a árvore não tem cabeçalho órfão logo
			// depois de um parágrafo vazio, que é a assinatura do hoist.
			var problems []string
			percorre(root, func(n *html.Node) {
				if n.Type != html.ElementNode || n.DataAtom != atom.P {
					return
				}
				if textoDe(n) != "" {
					return
				}
				if s := n.NextSibling; s != nil && ehCabecalho(s) {
					problems = append(problems, s.Data)
				}
			})
			if len(problems) > 0 {
				t.Errorf("em %s, %v vieram logo depois de um parágrafo vazio — assinatura de "+
					"cabeçalho expulso de dentro do `<p>`", name, problems)
			}
		})
	}
}

func parágrafosVazios(root *html.Node) []*html.Node {
	var outside []*html.Node
	percorre(root, func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.P && textoDe(n) == "" && n.FirstChild == nil {
			outside = append(outside, n)
		}
	})
	return outside
}

func percorre(n *html.Node, f func(*html.Node)) {
	f(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		percorre(c, f)
	}
}

func textoDe(n *html.Node) string {
	var b bytes.Buffer
	percorre(n, func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
	})
	return strings.TrimSpace(b.String())
}

func ehCabecalho(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return true
	}
	return false
}
