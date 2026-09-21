package api

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O CROMO DO MAPA NÃO PODE ROUBAR O CLIQUE DO QUE NÃO É BOTÃO, e a forma de não
// roubar tem uma armadilha que este guarda existe para prender.
//
// Os painéis que flutuam sobre a cena são CAIXAS: sem tratamento, eles recebem o
// clique na faixa inteira que ocupam, inclusive nos vãos entre os ícones — e ali
// pintar terreno ou pegar uma peça é um gesto que some sem nada na tela para
// explicar.
//
// O remédio é `pointer-events: none` no contêiner com `auto` em cada controle, e
// ele só é seguro se TODO controle reativar: um que fique de fora vira botão
// morto, que é pior que o defeito e não deixa erro em lugar nenhum. Um botão sem
// clique é indistinguível de um botão que não faz nada.
//
// Ele cruza as duas pontas, como o `TestEveryClassPositionedByColAndRowHasABox`:
// a FOLHA COMPILADA diz quem não recebe ponteiro e quem o devolve, o HTML
// SERVIDO diz quem precisa dele. Quem acrescentar um botão a um desses painéis
// amanhã cai aqui sem ler nada disto.
func TestNoChromeOverTheMapStealsTheClickOffItsControls(t *testing.T) {
	f := newSceneFixture(t)
	// A cena mais CHEIA que o mestre vê, senão o guarda mede o que sobrou: com
	// acervo (o baú só nasce com lugar guardado), com segunda aba (a barra de
	// abas), com peça (a camada de mover) e com movimento proposto.
	f.seedOpenBoard(t, "stone")
	f.savePlace(t, "Taverna do Javali")
	f.seedOpenBoard(t, "stone")
	f.openSecond(t, "Cripta do Rei Caolho")
	tokenID := f.onBoardAt(t, 4, 2)
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":7,"Y":3}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	noPointer, returns := pointerRulesOfTheMap(t)
	if len(noPointer) < 3 {
		t.Fatalf("a folha só tira o ponteiro de %d classes do tabuleiro: o canal não está aberto e o silêncio abaixo não é evidência", len(noPointer))
	}
	// O PAINEL DE VERBOS é o caso que escreveu a issue, e ele é nomeado porque a
	// varredura abaixo não o cobraria: um painel que recebe o clique inteiro não
	// tem controle morto nenhum — ele está errado do outro lado.
	if !noPointer["board-scene-verbs"] {
		t.Error("o painel de verbos recebe o clique na faixa INTEIRA que ocupa: medido a 390px, 328 dos 864 pontos dele (38%) caem no fundo entre os ícones, e ali pintar terreno, largar marcador ou pegar uma peça é um gesto que some")
	}

	root, err := html.Parse(strings.NewReader(screen))
	if err != nil {
		t.Fatalf("ler a cena servida: %v", err)
	}
	panels, controls := 0, 0
	var visit func(no *html.Node, panel string, reached bool)
	visit = func(no *html.Node, panel string, reached bool) {
		if no.Type == html.ElementNode {
			if panel == "" {
				for _, class := range strings.Fields(attr(no, "class")) {
					if !noPointer[class] {
						continue
					}
					panel, reached = class, reachedByRelief(no, true, returns[class])
					panels++
					break
				}
			} else if !reached {
				reached = reachedByRelief(no, false, returns[panel])
			}
			if panel != "" && needsThePointer(no) {
				controls++
				if !reached {
					t.Errorf("<%s %q> dentro de .%s precisa do ponteiro e a folha não o devolve: ele nasce morto",
						no.Data, attr(no, "aria-label"), panel)
				}
			}
		}
		for child := no.FirstChild; child != nil; child = child.NextSibling {
			visit(child, panel, reached)
		}
	}
	visit(root, "", false)

	// O DENOMINADOR, e ele é o que separa "nada reprovou" de "não mediu": a cena
	// montada acima tem o painel dos verbos inteiro do mestre — enquadramento,
	// cortina, lente, pôr no mapa, abrir outro, acervo e encerrar.
	if panels < 3 || controls < 7 {
		t.Fatalf("o guarda visitou %d painéis e %d controles: a montagem não produziu o que ele vem medir", panels, controls)
	}
}

// pointerRelief é uma regra de `pointer-events: auto` escrita para dentro de um
// nó que não recebe o ponteiro. São as TRÊS formas que a folha usa hoje, e uma
// quarta faz o guarda FALHAR em vez de passar por cima: um seletor que ele não
// sabe ler é um controle que ele não sabe conferir, e a primeira versão deste
// guarda ignorou em silêncio o `.board-notices>*` — 14 botões vivos saíram
// como mortos, com cara de defeito achado.
type pointerRelief struct {
	directChild bool            // `.painel>*`
	tags        map[string]bool // `.painel :is(a, button, …)` e `button.painel`
	selfNode    bool            // `button.painel`: a tag é do painel, não de dentro
}

// pointerRulesOfTheMap lê a folha COMPILADA e devolve, do tabuleiro, quem não
// recebe ponteiro e o que o devolve dentro de cada um.
//
// Só entra no primeiro conjunto o seletor de UMA CLASSE SOZINHA: um
// `.board-with-tool .board-token` tira o ponteiro sob uma condição
// que não está no nó, e ler isso como "a peça está morta" reprovaria a cena
// inteira toda vez que uma ferramenta estivesse desligada.
func pointerRulesOfTheMap(t *testing.T) (map[string]bool, map[string][]pointerRelief) {
	t.Helper()
	noPointer := map[string]bool{}
	returns := map[string][]pointerRelief{}
	for _, rule := range strings.Split(compiledStylesheet(t), "}") {
		opens := strings.Index(rule, "{")
		if opens < 0 {
			continue
		}
		body, selectors := rule[opens:], splitSelectors(rule[:opens])
		for _, selector := range selectors {
			selector = strings.TrimSpace(selector)
			switch {
			case strings.Contains(body, "pointer-events:none"):
				name := strings.TrimPrefix(selector, ".")
				if strings.HasPrefix(name, "board-") && !strings.ContainsAny(name, " >+~.:[") {
					noPointer[name] = true
				}
			case strings.Contains(body, "pointer-events:auto"):
				if panel, relief, ok := parseRelief(t, selector); ok {
					returns[panel] = append(returns[panel], relief)
				}
			}
		}
	}
	return noPointer, returns
}

// splitSelectors reparte a lista de seletores de uma regra pelas vírgulas que
// separam SELETORES, e não pelas que estão dentro de um `:is(a, button, …)`.
//
// Um `strings.Split` cru quebrava `.painel :is(a,button)` em `.painel :is(a` e
// `button`, e as duas metades caíam no ramo que ignora em silêncio — os dez
// botões do painel saíram como mortos, com o alívio escrito na folha.
func splitSelectors(list string) []string {
	selectors, start, depth := []string{}, 0, 0
	for i, r := range list {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				selectors = append(selectors, list[start:i])
				start = i + 1
			}
		}
	}
	return append(selectors, list[start:])
}

func parseRelief(t *testing.T, selector string) (string, pointerRelief, bool) {
	t.Helper()
	// `button.board-marker`: a folha diz que ESTA tag do painel recebe o
	// ponteiro. O marcador é `none` para a mesa e `auto` para o mestre, e a
	// diferença entre os dois é a tag que o servidor escreve.
	if tag, name, found := strings.Cut(selector, "."); found && tag != "" && !strings.ContainsAny(name, " >+~.:[") {
		if strings.HasPrefix(name, "board-") {
			return name, pointerRelief{tags: map[string]bool{tag: true}, selfNode: true}, true
		}
		return "", pointerRelief{}, false
	}
	if name, found := strings.CutSuffix(strings.TrimPrefix(selector, "."), ">*"); found {
		if !strings.HasPrefix(name, "board-") {
			return "", pointerRelief{}, false
		}
		return name, pointerRelief{directChild: true}, true
	}
	label, rest, hit := strings.Cut(strings.TrimPrefix(selector, "."), " ")
	if !hit || !strings.HasPrefix(label, "board-") {
		// Seletor que não fala de dentro de um painel do tabuleiro — o
		// `.board-marker-actions` é um nó solto e não um alívio.
		return "", pointerRelief{}, false
	}
	list, ok := strings.CutPrefix(rest, ":is(")
	if !ok {
		t.Fatalf("a folha devolve o ponteiro por %q, uma forma que este guarda não sabe ler: ensine-o antes de confiar no verde", selector)
	}
	tags := map[string]bool{}
	for _, tag := range strings.Split(strings.TrimSuffix(list, ")"), ",") {
		tags[strings.TrimSpace(tag)] = true
	}
	return label, pointerRelief{tags: tags}, true
}

// reachedByRelief diz se ESTE nó recebe o ponteiro de volta. A subárvore dele vem
// junto por herança, e é por isso que quem chama carrega o `reached` adiante.
func reachedByRelief(no *html.Node, isThePanel bool, reliefs []pointerRelief) bool {
	for _, relief := range reliefs {
		if relief.selfNode != isThePanel {
			continue
		}
		if relief.directChild || relief.tags[no.Data] {
			return true
		}
	}
	return false
}

// needsThePointer: o nó é alvo de dedo, seja pela tag ou por um gesto de
// ponteiro escrito nele.
//
// O `__window` e o `__document` ficam de FORA, e a distinção não é sutileza: o
// `clipboardStrip` carrega um `data-on:keydown__window` e o ouvinte dele mora na
// janela — aquele `<div>` não precisa receber clique nenhum para o CTRL+V
// funcionar, e cobrá-lo mataria justamente os vãos que a issue vem devolver.
func needsThePointer(no *html.Node) bool {
	switch no.Data {
	case "a", "button", "input", "label", "select", "summary", "textarea":
		return true
	}
	for _, a := range no.Attr {
		gesture, ok := strings.CutPrefix(a.Key, "data-on:")
		if !ok || strings.Contains(gesture, "__window") || strings.Contains(gesture, "__document") {
			continue
		}
		switch strings.Split(gesture, "__")[0] {
		case "click", "contextmenu", "dblclick", "pointerdown", "pointerup", "pointermove", "wheel":
			return true
		}
	}
	return false
}

func attr(no *html.Node, name string) string {
	for _, a := range no.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}
