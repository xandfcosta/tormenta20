package api

import (
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O DIÁLOGO ESCOLHER DA FICHA (ALE-473), pelo mockup aprovado: a
// lista dos passos com quanto falta em cada um, e um passo de cada vez — o
// valor curto como opção, a lista longa com interruptor, e nos poderes a
// busca e o filtro fixos acima da parte que rola. Todo gesto leva a marca da
// ficha, e nada do grimório entra no diálogo.
func TestTheParallelChoicesDialogOpensOneStepAtATime(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Paladino", 3, 0, 0)
	page := parallelSheet(t, f, id, "abilities")

	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="power-choices".*?</dialog>`).FindString(page)
	if dialog == "" {
		t.Fatal("a aba Poderes não desenhou o diálogo de escolhas")
	}
	if strings.Contains(dialog, "De onde vem a escolha") {
		t.Error("o diálogo ainda tem a fileira Raça · Origem · Classe, que repetia a lista dos passos")
	}

	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	list := findNode(doc, "class", "ds-steps-list")
	if list == nil {
		t.Fatal("o diálogo não tem a lista dos passos")
	}
	if !strings.Contains(textOf(list), "Faltam") {
		t.Error("a lista dos passos não diz quantas escolhas faltam")
	}

	// Cada passo: o toque na lista que o abre, e o que ele oferece por dentro
	// da parte que rola.
	steps := []struct{ what, step, inside string }{
		{"a devoção como opção", "devotion", `<button[^>]*class="ds-option"[^>]*data-on:click="[^"]*/poderes/devocao/`},
		{"o benefício da origem como interruptor", "origin", `<input[^>]*role="switch"[^>]*data-on:change="[^"]*/poderes/origem/`},
		{"o poder da classe como interruptor", "class-Paladino-powers", `<input[^>]*role="switch"[^>]*data-on:change="[^"]*/poderes/escolhe/`},
	}
	for _, s := range steps {
		if !regexp.MustCompile(`\$choice_step = (&#39;|')` + s.step + `(&#39;|')`).MatchString(renderNode(t, list)) {
			t.Errorf("a lista não abre o passo %s", s.step)
		}
		section := findNode(doc, "id", "choice-step-"+s.step)
		if section == nil {
			t.Errorf("falta o passo %s", s.step)
			continue
		}
		scroll := findNode(section, "class", "ds-dialog-scroll")
		if scroll == nil || !regexp.MustCompile(s.inside).MatchString(renderNode(t, scroll)) {
			t.Errorf("o passo %s não oferece %s na parte que rola", s.step, s.what)
		}
	}

	// NOS PODERES, a busca e o filtro ficam fora da parte que rola, e o filtro
	// é uma fileira de opções, não o segmentado que quebrava a 390px.
	powers := findNode(doc, "id", "choice-step-class-Paladino-powers")
	if powers == nil {
		return
	}
	bar := findNode(powers, "class", "ds-filter-bar")
	if bar == nil || findNode(findNode(powers, "class", "ds-dialog-scroll"), "class", "ds-filter-bar") != nil {
		t.Error("a busca dos poderes não está fixa acima da parte que rola")
	}
	if bar != nil && (findNode(bar, "aria-label", "Quais poderes") == nil || findNode(bar, "class", "ds-segmented") != nil) {
		t.Error("o filtro dos poderes não é a fileira de opções")
	}
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func renderNode(t *testing.T, n *html.Node) string {
	t.Helper()
	var b strings.Builder
	if err := html.Render(&b, n); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
