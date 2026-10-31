package api

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// As ESCOLHAS da forja no Tempestade (ALE-535): raça, classe e origem são linhas
// que abrem uma folha com busca, e a folha mora FORA do formulário.

// pickRow devolve o HTML da linha de escolha de um campo, do botão até o fim
// dele.
func pickRow(t *testing.T, body, field string) string {
	t.Helper()
	i := strings.Index(body, `id="forge-pick-`+field+`"`)
	if i < 0 {
		t.Fatalf("a folha não tem a linha de escolha de %q", field)
	}
	rest := body[i:]
	end := strings.Index(rest, "</button>")
	if end < 0 {
		t.Fatalf("a linha de %q não fecha", field)
	}
	return rest[:end]
}

// A linha diz o que foi escolhido: é ela que a pessoa lê para saber o que já
// respondeu, com as folhas fechadas.
func TestEachPickRowSaysWhatWasChosen(t *testing.T) {
	f := newSceneFixture(t)
	body := postaAForja(t, f, f.player, "/personagens/nova/esboco", url.Values{
		"race": {"elfo"}, "class": {"Guerreiro"}, "origin": {"acolito"},
	}).Body.String()

	for field, chosen := range map[string]string{"race": "Elfo", "class": "Guerreiro", "origin": "Acólito"} {
		if row := pickRow(t, body, field); !strings.Contains(row, chosen) {
			t.Errorf("a linha de %s não diz %q:\n%s", field, chosen, row)
		}
	}
	empty := f.requests(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()
	if row := pickRow(t, empty, "race"); !strings.Contains(row, "Escolher") {
		t.Errorf("a linha vazia não convida a escolher:\n%s", row)
	}
}

// NADA DE FORMULÁRIO ANINHADO. A folha do kit traz o próprio `<form
// method="dialog">` para o fechar; dentro do formulário da forja, o HTML o
// ignora e o botão de fechar passa a ENVIAR a forja. Por isso as folhas ficam
// fora, e cada rádio se liga ao formulário pelo atributo `form`.
func TestThePickSheetsLiveOutsideTheForgeForm(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()

	start := strings.Index(body, `id="forge-form"`)
	if start < 0 {
		t.Fatal("a folha não tem o formulário `forge-form`")
	}
	form := body[start:]
	form = form[:strings.Index(form, "</form>")]
	if strings.Contains(form, `method="dialog"`) || strings.Contains(form, "<dialog") {
		t.Error("há uma folha dentro do formulário da forja: o fechar dela enviaria a forja")
	}

	radio := regexp.MustCompile(`<input[^>]*type="radio"[^>]*name="(race|class|origin)"[^>]*>`)
	found := radio.FindAllString(body, -1)
	if len(found) < 3 {
		t.Fatalf("achei %d rádios de escolha; a folha não carregou as listas", len(found))
	}
	for _, tag := range found {
		if !strings.Contains(tag, `form="forge-form"`) {
			t.Errorf("rádio sem `form=\"forge-form\"` — ele não iria com o formulário: %s", tag)
		}
	}
}
