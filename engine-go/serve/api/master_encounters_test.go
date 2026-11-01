package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do CONSTRUTOR DE ENCONTROS (ALE-259).
//
// A conta em si é do `engine` e tem os testes dela lá, contra o livro. O que se
// prende aqui é a ÁLGEBRA DO RASCUNHO e a tradução do gesto — que é onde um
// erro apaga o encontro do mestre sem avisar.

// TestTheAddGestureReturnsAPatch, e não uma página: recarregar no meio
// da montagem perderia o rascunho, que só vive nos sinais.
func TestTheAddGestureReturnsAPatch(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/encontros/adicionar/ogro", `{"encounter_rows":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — o gesto navegou em vez de remendar", ct)
	}
	if !strings.Contains(rec.Body.String(), "Ogro") {
		t.Error("o remendo não trouxe a criatura acrescentada")
	}
}

// O construtor veste a moldura da Mesa do Mestre, com o nome no cabeçalho, a lista de
// criaturas à vista sem precisar de busca e o diálogo de acrescentar do celular.
func TestTheEncounterBuilderWearsTheMasterFrame(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")
	body := pedeNoMestre(t, s, eu, "GET", "/mestre/encontros?nivel=3&grupo=4&c=ogro:2", "").Body.String()

	if !strings.Contains(body, "scene-ds") {
		t.Fatal("o construtor não veste o Tempestade")
	}
	if !strings.Contains(body, `<h1 class="ds-header-title">Encontros</h1>`) {
		t.Error("o cabeçalho não diz Encontros")
	}
	menu := sliceBetween(t, body, `class="ds-side-menu"`, "</nav>")
	if current := sliceBetween(t, menu, `href="/mestre/encontros"`, ">"); !strings.Contains(current, `aria-current="page"`) {
		t.Errorf("o menu não marca Encontros: %s", current)
	}
	if !strings.Contains(body, `aria-label="Acrescentar Bandido ao encontro"`) {
		t.Error("sem termo digitado, a lista não traz as criaturas")
	}
	if !strings.Contains(body, `commandfor="encounter-add"`) || !strings.Contains(body, `id="encounter-add"`) {
		t.Error("o celular não tem o diálogo de acrescentar")
	}
	if !strings.Contains(body, ">2×</span>") {
		t.Error("a lista não diz que já há 2 ogros no encontro")
	}
	// O link fica na cena, que é o que o remendo atualiza: o botão mora no cabeçalho,
	// que o remendo não alcança, e carregaria o endereço de quando a página abriu.
	scene := sliceBetween(t, body, `id="encounters"`, ">")
	if !strings.Contains(scene, "nivel=3") || !strings.Contains(scene, "c=ogro%3A2") {
		t.Errorf("a cena não carrega o endereço do encontro: %s", scene)
	}
}

// A dificuldade sai num selo, com o tom de cada faixa. Dois ogros valem ND 6 pelo
// livro (4 + 2·log2(2), p282), e a faixa é a diferença para o nível do grupo (p281).
func TestTheDifficultySealHasTheToneOfItsBand(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	for level, want := range map[string]string{
		"3": `<span class="ds-seal" data-tone="danger">Mortal</span>`,
		"5": `<span class="ds-seal" data-tone="warn">Difícil</span>`,
		"6": `<span class="ds-seal" data-tone="neutral">Médio</span>`,
		"8": `<span class="ds-seal" data-tone="good">Fácil</span>`,
		"9": `<span class="ds-seal" data-tone="good">Trivial</span>`,
	} {
		body := pedeNoMestre(t, s, eu, "GET", "/mestre/encontros?grupo=4&c=ogro:2&nivel="+level, "").Body.String()
		if !strings.Contains(body, want) {
			t.Errorf("nível %s: a página não traz %s", level, want)
		}
	}
}
