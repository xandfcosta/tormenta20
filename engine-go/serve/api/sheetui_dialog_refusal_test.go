package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// A RECUSA APARECE DENTRO DO DIÁLOGO. Um modal deixa o resto da página atrás dele, e a
// recusa desenhada só na página parece um gesto que não fez nada.

const twoPowersRefusal = "troca só uma das perícias"

// refusedOnTab provoca a mesma recusa com a aba pedida aberta: cada aba desenha os diálogos
// dela.
func refusedOnTab(t *testing.T, f sceneFixture, id int64, tab string) *html.Node {
	t.Helper()
	body := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/poderes/raca/humano-versatil/sortudo?tab=%s", id, tab), "").Body.String()
	if !strings.Contains(sceneRefusal(body), twoPowersRefusal) {
		t.Fatalf("a aba %s não trouxe a recusa: %q", tab, sceneRefusal(body))
	}
	start := strings.Index(body, "<")
	doc, err := html.Parse(strings.NewReader(body[start:]))
	if err != nil {
		t.Fatalf("ler a resposta: %v", err)
	}
	return doc
}

func dialogsIn(n *html.Node) []*html.Node {
	found := []*html.Node{}
	if n.Type == html.ElementNode && n.Data == "dialog" {
		found = append(found, n)
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		found = append(found, dialogsIn(child)...)
	}
	return found
}

func attrOf(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func alertsIn(n *html.Node) []*html.Node {
	found := []*html.Node{}
	if n.Type == html.ElementNode && attrOf(n, "role") == "alert" {
		found = append(found, n)
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		found = append(found, alertsIn(child)...)
	}
	return found
}

func humanWithAGeneralPower(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := humanBarbarian(t)
	if refusal := raceAbilityCommand(t, f, id, versatile, "vontade-de-ferro"); refusal != "" {
		t.Fatalf("o primeiro poder: %q", refusal)
	}
	return f, id
}

// A varredura é pelos diálogos que a página desenha: diálogo novo entra sozinho, e reprova
// com o id dele se usar o `ds.Dialog` direto.
func TestEverySheetDialogShowsTheRefusal(t *testing.T) {
	f, id := humanWithAGeneralPower(t)
	seen := 0
	for _, tab := range []string{"expertises", "combat", "bag", "proficiencies", "conditionals", "abilities", "spells"} {
		for _, dialog := range dialogsIn(refusedOnTab(t, f, id, tab)) {
			seen++
			if !strings.Contains(textOf(dialog), twoPowersRefusal) || len(alertsIn(dialog)) == 0 {
				t.Errorf("aba %s: o diálogo %q não mostra a recusa: use o `sheetDialog`", tab, attrOf(dialog, "id"))
			}
		}
	}
	if seen == 0 {
		t.Fatal("a varredura não achou diálogo nenhum: ela não mede nada")
	}
}

// No Escolher, cada passo tem o próprio alerta, abaixo do cabeçalho, porque o corpo dele rola
// e o passo aberto é o único à vista.
func TestTheRefusalIsInEveryStepOfTheChoices(t *testing.T) {
	f, id := humanWithAGeneralPower(t)
	doc := refusedOnTab(t, f, id, "abilities")
	dialog := findNode(doc, "id", "power-choices")
	if dialog == nil {
		t.Fatal("a página não desenhou o Escolher")
	}
	step := findNode(dialog, "id", "choice-step-race-0")
	if step == nil {
		t.Fatal("o Escolher não tem o passo da raça")
	}
	if alerts := alertsIn(step); len(alerts) != 1 || !strings.Contains(textOf(alerts[0]), twoPowersRefusal) {
		t.Errorf("o passo aberto não mostra a recusa: %d alertas", len(alerts))
	}
}

// Sem recusa, nenhum diálogo desenha o alerta vazio.
func TestNoRefusalDrawsNoAlertInTheDialogs(t *testing.T) {
	f, id := humanBarbarian(t)
	body := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ler a página: %v", err)
	}
	for _, dialog := range dialogsIn(doc) {
		for _, alert := range alertsIn(dialog) {
			if strings.TrimSpace(textOf(alert)) == "" {
				t.Errorf("o diálogo %q desenha um alerta vazio", attrOf(dialog, "id"))
			}
		}
	}
}

// Cada diálogo tem um alerta só: dois repetem a frase, e o leitor de tela a lê duas vezes.
func TestEachSheetDialogHasOneRefusal(t *testing.T) {
	f, id := humanWithAGeneralPower(t)
	for _, tab := range []string{"expertises", "combat", "bag", "proficiencies", "conditionals", "abilities", "spells"} {
		for _, dialog := range dialogsIn(refusedOnTab(t, f, id, tab)) {
			if attrOf(dialog, "id") == "power-choices" {
				continue // um por passo, e só o passo aberto aparece
			}
			if alerts := alertsIn(dialog); len(alerts) > 1 {
				t.Errorf("aba %s: o diálogo %q tem %d alertas", tab, attrOf(dialog, "id"), len(alerts))
			}
		}
	}
}
