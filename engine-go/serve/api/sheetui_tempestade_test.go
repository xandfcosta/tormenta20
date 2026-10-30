package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// A FICHA NO TEMPESTADE (ALE-473): ela foi construída aba por aba atrás da
// marca `?tempestade=1`, e com a última aba migrada a marca saiu. O que o
// servidor DECIDE é o que se prende aqui: qual sistema a página veste, o que a
// barra de baixo oferece a cada personagem, e a resposta de cada gesto.

func parallelSheet(t *testing.T, f sceneFixture, id int64, tab string) string {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=%s", id, tab), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ficha, aba %q: status %d", tab, rec.Code)
	}
	return rec.Body.String()
}

var sceneShell = regexp.MustCompile(`<section[^>]*data-slot="scene-shell"[^>]*>`)

// A PÁGINA E O GESTO vestem o Tempestade, sem marca nenhuma no endereço.
func TestTheSheetWearsTempestade(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)

	if shell := sceneShell.FindString(parallelSheet(t, f, id, "expertises")); !strings.Contains(shell, "scene-ds") || strings.Contains(shell, "scene-grimorio") {
		t.Errorf("a ficha não saiu só no Tempestade: %s", shell)
	}
	answer := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pv/-1?tab=expertises", id), "").Body.String()
	if !strings.Contains(answer, `data-system="tempestade"`) {
		t.Error("o gesto da ficha voltou com a cena do grimório")
	}
}

// A BARRA DE BAIXO tem seis lugares, e o quinto depende de quem conjura: a
// Mochila entra no lugar das Magias, que vão para o Mais (decisão do dono).
func TestTheBottomBarGivesTheFifthPlaceToSpellsOnlyToWhoCasts(t *testing.T) {
	f := newSceneFixture(t)
	warrior := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)
	wizard := seedCharacterAtLevel(t, f.s, f.player, "Ana", "Arcanista", 3, 0, 0)

	bar := regexp.MustCompile(`(?s)<nav class="ds-bottom-nav"[^>]*>(.*?)</nav>`)
	labels := func(id int64) string {
		m := bar.FindStringSubmatch(parallelSheet(t, f, id, "expertises"))
		if m == nil {
			t.Fatalf("personagem %d: a ficha não desenhou a barra de baixo", id)
		}
		return m[1]
	}

	if got := labels(wizard); !strings.Contains(got, "Magias") || strings.Contains(got, "Mochila") {
		t.Errorf("a arcanista não tem Magias na barra, ou tem a Mochila: %s", got)
	}
	if got := labels(warrior); strings.Contains(got, "Magias") || !strings.Contains(got, "Mochila") {
		t.Errorf("o guerreiro tem Magias na barra, ou não tem a Mochila: %s", got)
	}
}

// O GESTO NUNCA DEVOLVE O GRIMÓRIO dentro da Tempestade: é a mistura que a
// transição por escopo proíbe, e todo gesto redesenha a cena inteira.
func TestAGestureOfTheParallelSheetNeverDrawsTheGrimoire(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)

	// O recorte é a CENA, e a cena sozinha é o que um gesto devolve: a página
	// traz também os diálogos da casca (o livro, o buscador), que são de toda
	// tela e migram com ela, não com a ficha.
	scene := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pv/-1?tab=proficiencies", id), "").Body.String()
	if !strings.Contains(scene, "sheet-scene") {
		t.Fatal("o gesto não devolveu a cena da ficha")
	}
	if grimoire := regexp.MustCompile(`[\w:-]*grimorio[\w-]*`).FindAllString(scene, 5); len(grimoire) > 0 {
		t.Errorf("a ficha desenhou classe do grimório: %v", grimoire)
	}
}

// O ⋯ DIZ QUEM É (decisão do dono): no celular o cabeçalho só cabe o nome, e
// o retrato, a classe, a raça e a divindade moram no diálogo do ⋯.
func TestTheSheetMenuSaysWhoTheCharacterIs(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)
	seedRaca(t, f.s, id, "osteon")
	devoteOf(t, f, id, "Khalmyr", "Dom da Verdade")

	menu := regexp.MustCompile(`(?s)<dialog[^>]*id="sheet-menu".*?</dialog>`).FindString(parallelSheet(t, f, id, "expertises"))
	if menu == "" {
		t.Fatal("a ficha não desenhou o diálogo do ⋯")
	}
	for _, want := range []string{"ds-avatar", "Guerreiro 3", "Osteon", "Khalmyr", "Soldado"} {
		if !strings.Contains(menu, want) {
			t.Errorf("o ⋯ não diz %q", want)
		}
	}
}

// O EFEITO ATIVO DA COLUNA ABRE A DESCRIÇÃO DELE (decisão do dono): o nome do
// crachá abre um diálogo com o que a condição faz e a página, e o X continua
// tirando.
func TestAnActiveEffectInTheRailOpensItsDescription(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)
	if rec := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/efeitos/condicao/caido?tab=expertises", id), ""); rec.Code != http.StatusOK {
		t.Fatalf("pôr Caído respondeu %d", rec.Code)
	}
	page := parallelSheet(t, f, id, "expertises")

	opener := regexp.MustCompile(`<button[^>]*commandfor="(effect-[^"]*)"[^>]*>Caído</button>`).FindStringSubmatch(page)
	if opener == nil {
		t.Fatal("o crachá do Caído não abre nada")
	}
	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="` + opener[1] + `".*?</dialog>`).FindString(page)
	if !strings.Contains(dialog, "Condição") || !strings.Contains(dialog, `class="ds-card-text"`) {
		t.Errorf("o diálogo do Caído não descreve a condição:\n%s", dialog)
	}
	if !strings.Contains(page, `aria-label="Tirar Caído"`) {
		t.Error("o crachá perdeu o X que tira o Caído")
	}
}

// O PEDIDO DO DATASTAR RECEBE SÓ A CENA (ALE-520): a troca macia de aba e a
// busca pedem a ficha pelo `@get`, e a página inteira — a casca, o buscador, o
// livro — fazia o navegador comparar tudo de novo, ~450ms num celular médio.
// A navegação de verdade continua recebendo a página.
func TestADatastarGetOfTheParallelSheetGetsOnlyTheScene(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)
	path := fmt.Sprintf("/personagens/%d?tab=combat", id)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.player))
	req.Header.Set("Datastar-Request", "true")
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/event-stream") || strings.Contains(body, "<html") {
		t.Errorf("o @get da ficha recebeu a página inteira (%s)", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, `id="sheet-scene"`) || !strings.Contains(body, `aria-label="Combate"`) {
		t.Error("o @get da ficha não trouxe a cena na aba pedida")
	}

	if page := parallelSheet(t, f, id, "combat"); !strings.Contains(page, "<html") {
		t.Error("a navegação de verdade deixou de receber a página")
	}
}
