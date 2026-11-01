package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do IMPROVISO (ALE-261).
//
// O dado em si é do `engine` e tem teste lá. O que se prende aqui é a tradução
// da linha do livro para a tela — que é onde um campo trocado passa por dado
// plausível.

// TestRollingReturnsAPatchAndStacks.
func TestRollingReturnsAPatchAndStacks(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/ruina",
		`{"ruin_rolls":[{"r":2,"t":"Vazia"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — a rolagem navegou em vez de remendar", ct)
	}
	// A anterior tem de sobreviver: o histórico é o que separa esta tabela de um
	// botão que só mostra o último.
	if !strings.Contains(rec.Body.String(), "Vazia") {
		t.Error("o remendo perdeu a rolagem anterior")
	}
}

// TestAnInventedTableIsRefused: a rota é montada a partir da própria lista,
// então nome errado só chega por URL digitada à mão — e devolver a cena intacta
// faria parecer que o botão não funciona.
func TestAnInventedTableIsRefused(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/tarot", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, quero 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "tarot") {
		t.Error("a recusa não diz qual tabela foi recusada")
	}
}

func TestClearingResetsOnlyThatTable(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	signals := `{"ruin_rolls":[{"r":4,"t":"Vazia"}],"chase_rolls":[{"r":9,"t":"Obstáculo"}],` +
		`"reward_rolls":[{"r":2,"t":"Favor"}],"idea_rolls":[{"r":7,"t":"Cripta"}]}`
	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/ruina/limpar", signals)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Vazia") {
		t.Error("a ruína não foi limpa")
	}
	for _, survivor := range []string{"Obstáculo", "Favor", "Cripta"} {
		if !strings.Contains(body, survivor) {
			t.Errorf("limpar a ruína levou junto %q — as tabelas são independentes", survivor)
		}
	}
}

// TestClearingAnInventedTableIsRefused, como a rolagem.
func TestClearingAnInventedTableIsRefused(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")
	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/tarot/limpar", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, quero 400", rec.Code)
	}
}

// O Improviso veste a moldura da Mesa do Mestre, com o nome no cabeçalho.
func TestTheImprovWearsTheMasterFrame(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")
	body := pedeNoMestre(t, s, eu, "GET", "/mestre/improviso", "").Body.String()

	if !strings.Contains(body, "scene-ds") {
		t.Fatal("o Improviso não veste o Tempestade")
	}
	if !strings.Contains(body, `<h1 class="ds-header-title">Improviso</h1>`) {
		t.Error("o cabeçalho não diz Improviso")
	}
	menu := sliceBetween(t, body, `class="ds-side-menu"`, "</nav>")
	if current := sliceBetween(t, menu, `href="/mestre/improviso"`, ">"); !strings.Contains(current, `aria-current="page"`) {
		t.Errorf("o menu não marca o Improviso: %s", current)
	}
}

// A página de cada tabela leva ao livro: o número do dado só vale se dá para conferir.
// As páginas estão escritas à mão porque foram conferidas contra o PDF (ver o cabeçalho
// de `improv.templ`), e a 6-7 é a armadilha: o corpo dela começa na p279, e não na p278.
func TestEveryImprovTableLinksItsPageInTheBook(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")
	body := pedeNoMestre(t, s, eu, "GET", "/mestre/improviso", "").Body.String()

	for _, page := range []string{"269", "274", "279", "263", "347"} {
		if !strings.Contains(body, "/livro/ler?p="+page) {
			t.Errorf("nenhuma tabela leva à p%s do livro", page)
		}
	}
	if !strings.Contains(body, "Tabela 6-5 · ") {
		t.Error("a citação da tabela sumiu de perto da página")
	}
}

// Sem livro servido a página continua escrita: o mestre com o livro de papel na mesa
// usa exatamente isso.
func TestWithoutTheBookTheImprovPageStaysWritten(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")
	body := pedeNoMestre(t, s, eu, "GET", "/mestre/improviso", "").Body.String()

	if strings.Contains(body, "/livro/ler?p=274") {
		t.Error("sem livro não há para onde o link levar")
	}
	if !strings.Contains(body, "<span>p274</span>") {
		t.Error("sem livro a página sumiu da tabela")
	}
}
