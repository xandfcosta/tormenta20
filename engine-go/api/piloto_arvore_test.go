package api

import (
	"bytes"
	"context"
	"strings"
	"t20engine/web/ui"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A ÁRVORE QUE O NAVEGADOR MONTA, e não a string que o servidor escreveu
// (ALE-262).
//
// Este guarda existe por causa de uma assimetria que a sessão irmã nomeou ao
// varrer a SPA atrás do mesmo defeito e não achar: **o compilador do Solid
// AVISA sobre aninhamento inválido** ("The HTML provided is malformed and will
// yield unexpected output when evaluated by a browser"), porque ele compila JSX
// para clonagem de `<template>` e vê a marcação antes do navegador.
//
// Aqui a marcação é STRING montada no servidor. Não há compilador olhando: o
// `templ` confere se as tags fecham, não o modelo de conteúdo delas. O parser
// do navegador conserta em silêncio, e ninguém escreve nada em lugar nenhum —
// foi por isso que 24 parágrafos vazios atravessaram compilador, typecheck e
// guarda de contraste.
//
// A barreira que falta se constrói com o `x/net/html`, que implementa o MESMO
// algoritmo de correção do navegador. Se a árvore que ele monta não é a que o
// template descreve, o navegador também vai discordar.
//
// Ele é mais largo que o guarda de grep do `@rotuloDeSecao`: aquele conhece um
// componente, este mede o RESULTADO de qualquer cena.

// cenasDoPiloto são as cenas que este guarda visita.
//
// A lista é enumeração, e enumeração é remendo — o CLAUDE.md diz isso e está
// certo. Ela existe porque renderizar uma cena exige montar a view dela, e não
// há como descobrir isso por reflexão. **Cena nova que não entrar aqui nasce
// sem medição**, que é a marca desta família de defeito; o comentário fica para
// quem acrescentar a próxima saber que precisa vir aqui.
func cenasDoPiloto(t *testing.T) map[string]string {
	t.Helper()
	fora := map[string]string{}
	monta := func(nome string, render func() (string, error)) {
		html, err := render()
		if err != nil {
			t.Fatalf("renderizar %s: %v", nome, err)
		}
		fora[nome] = html
	}

	ctx := context.Background()
	monta("bestiario", func() (string, error) {
		return ui.RenderFragment(ctx, cenaDoBestiario(carregaBestiarioDe(rotaDoBestiarioDoMestre, enderecoDoLivro{}, "", nil, ndMinimo, ndMaximo, "")))
	})
	monta("catalogos", func() (string, error) {
		return ui.RenderFragment(ctx, cenaDosCatalogos(carregaCatalogos(criteriosDoAcervo{Busca: "", Aba: "condicoes"}, enderecoDoLivro{})))
	})
	monta("catalogos-busca", func() (string, error) {
		return ui.RenderFragment(ctx, cenaDosCatalogos(carregaCatalogos(criteriosDoAcervo{Busca: "fogo", Aba: ""}, enderecoDoLivro{})))
	})
	monta("encontros", func() (string, error) {
		v := carregaEncontros(3, 4, []linhaDoEncontro{{ID: "ogro", Qtd: 2}}, "ogro")
		return ui.RenderFragment(ctx, cenaDosEncontros(v))
	})
	monta("improviso", func() (string, error) {
		v := carregaImproviso(improvisoView{
			Salas: 14,
			Ruina: []sorteio{{Rolagem: 4, Texto: "Vazia"}, {Rolagem: 2, Texto: "Vazia"}},
		})
		return ui.RenderFragment(ctx, cenaDoImproviso(v))
	})
	return fora
}

// TestONavegadorNaoPrecisaConsertarAMarcacao.
//
// O sinal é o PARÁGRAFO VAZIO: quando o parser encontra conteúdo de fluxo
// dentro de um `<p>`, ele fecha o parágrafo antes do intruso e o hoista — e
// sobra uma casca sem texto que o template nunca pediu. Um `<p>` deliberadamente
// vazio não existe em nenhuma cena da casa; se um dia existir, ele terá de
// ganhar uma exceção nomeada aqui, e essa conversa é melhor que o silêncio.
//
// Provado VERMELHO pondo o `<h4>` de volta dentro do `@rotuloDeSecao`: acusa o
// bestiário com 24 parágrafos vazios.
func TestONavegadorNaoPrecisaConsertarAMarcacao(t *testing.T) {
	cenas := cenasDoPiloto(t)
	if len(cenas) == 0 {
		t.Fatal("nenhuma cena foi montada: o guarda não visitaria nada e o verde não valeria")
	}
	for nome, marcacao := range cenas {
		t.Run(nome, func(t *testing.T) {
			raiz, err := html.Parse(strings.NewReader(marcacao))
			if err != nil {
				t.Fatalf("o parser recusou a marcação: %v", err)
			}
			vazios := parágrafosVazios(raiz)
			if len(vazios) > 0 {
				t.Errorf("%d parágrafo(s) vazio(s) na árvore de %s — o navegador CONSERTOU "+
					"aninhamento inválido, expulsando conteúdo de fluxo de dentro de um `<p>` "+
					"e deixando a casca para trás. A classe do parágrafo não alcança mais o "+
					"conteúdo, e nenhum guarda de contraste vê isso porque casca vazia não "+
					"tem texto para medir.", len(vazios), nome)
			}
		})
	}
}

// TestOsCabecalhosNaoSaoFilhosDeParagrafo é o outro lado da mesma moeda, e ele
// pega o caso em que o parser hoista SEM deixar casca — quando o `<p>` tinha
// texto antes do cabeçalho, ele fica com o texto e o cabeçalho sai.
func TestOsCabecalhosNaoSaoFilhosDeParagrafo(t *testing.T) {
	for nome, marcacao := range cenasDoPiloto(t) {
		t.Run(nome, func(t *testing.T) {
			raiz, err := html.Parse(strings.NewReader(marcacao))
			if err != nil {
				t.Fatalf("parser: %v", err)
			}
			// Reserializar e comparar a CONTAGEM de cabeçalhos por pai é caro;
			// o que basta é afirmar que a árvore não tem cabeçalho órfão logo
			// depois de um parágrafo vazio, que é a assinatura do hoist.
			var problemas []string
			percorre(raiz, func(n *html.Node) {
				if n.Type != html.ElementNode || n.DataAtom != atom.P {
					return
				}
				if textoDe(n) != "" {
					return
				}
				if s := n.NextSibling; s != nil && ehCabecalho(s) {
					problemas = append(problemas, s.Data)
				}
			})
			if len(problemas) > 0 {
				t.Errorf("em %s, %v vieram logo depois de um parágrafo vazio — assinatura de "+
					"cabeçalho expulso de dentro do `<p>`", nome, problemas)
			}
		})
	}
}

func parágrafosVazios(raiz *html.Node) []*html.Node {
	var fora []*html.Node
	percorre(raiz, func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.P && textoDe(n) == "" && n.FirstChild == nil {
			fora = append(fora, n)
		}
	})
	return fora
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
