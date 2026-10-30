package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A FICHA PARALELA (ALE-473): a ficha vestida no Tempestade nasce no mesmo
// endereço com a marca `?tempestade=1`, e a de hoje fica intacta até a última
// aba migrar. O que o servidor DECIDE é o que se prende aqui: qual sistema a
// página veste, que a marca sobrevive a todo gesto, o que a barra de baixo
// oferece a cada personagem, e que aba não migrada nunca mostra o grimório.

func parallelSheet(t *testing.T, f sceneFixture, id int64, tab string) string {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=%s&tempestade=1", id, tab), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ficha paralela, aba %q: status %d", tab, rec.Code)
	}
	return rec.Body.String()
}

var sceneShell = regexp.MustCompile(`<section[^>]*data-slot="scene-shell"[^>]*>`)

func TestTheParallelSheetWearsTempestadeAndTheOldOneStays(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)

	if shell := sceneShell.FindString(parallelSheet(t, f, id, "expertises")); !strings.Contains(shell, "scene-ds") || strings.Contains(shell, "scene-grimorio") {
		t.Errorf("com a marca, a ficha não saiu só no Tempestade: %s", shell)
	}
	old := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	if shell := sceneShell.FindString(old); strings.Contains(shell, "scene-ds") || !strings.Contains(shell, "scene-grimorio") {
		t.Errorf("sem a marca, a ficha de hoje deixou o grimório: %s", shell)
	}
}

// A MARCA VIAJA EM TODO GESTO, pela razão da aba: todo comando redesenha a
// cena inteira, e um que a perdesse devolveria a ficha do grimório dentro da
// página Tempestade.
func TestEveryGestureOfTheParallelSheetKeepsTheMark(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)

	// O templ escapa a aspa do atributo (`&#39;`), e o endereço tem `&amp;`
	// dentro: o fim do endereço é a aspa, escapada ou não.
	gestures := regexp.MustCompile(`@(?:post|get)\(([^)]*)\)`)
	// O link para a ficha de HOJE é a exceção, e declarada: ele existe para
	// sair da paralela, e é o atributo que o diz.
	links := regexp.MustCompile(`<a(?:[^>](?:[^>]*data-sheet-of-today)?)*?href="(/personagens/[^"]*)"[^>]*>`)
	page := parallelSheet(t, f, id, "expertises")
	seen := 0
	for _, m := range append(gestures.FindAllStringSubmatch(page, -1), links.FindAllStringSubmatch(page, -1)...) {
		// Só os endereços DA FICHA: a casca da página traz o buscador, que é
		// de toda cena e não sabe de ficha nenhuma.
		if strings.Contains(m[0], "data-sheet-of-today") || !strings.Contains(m[1], "/personagens/") {
			continue
		}
		seen++
		if !strings.Contains(m[1], "tempestade=1") {
			t.Errorf("o endereço %q sai da ficha paralela", m[1])
		}
	}
	if seen < 8 {
		t.Fatalf("só %d endereços medidos; a casca tem pelo menos as seis da barra e os dois vitais", seen)
	}

	// E a RESPOSTA do gesto continua Tempestade.
	answer := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pv/-1?tab=expertises&tempestade=1", id), "").Body.String()
	if !strings.Contains(answer, `data-system="tempestade"`) {
		t.Error("o gesto da ficha paralela voltou com a cena do grimório")
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
			t.Fatalf("personagem %d: a ficha paralela não desenhou a barra de baixo", id)
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

// ABA QUE AINDA NÃO MIGROU não desenha o grimório dentro da Tempestade: ela
// diz que está na ficha de hoje e leva até lá, na mesma aba.
func TestATabNotYetMigratedLeadsToTheSheetOfToday(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)

	page := parallelSheet(t, f, id, "proficiencies")
	if !strings.Contains(page, fmt.Sprintf(`href="/personagens/%d?tab=proficiencies"`, id)) {
		t.Error("a aba não migrada não leva às Proficiências da ficha de hoje")
	}
	// O recorte é a CENA, e a cena sozinha é o que um gesto devolve: a página
	// traz também os diálogos da casca (o livro, o buscador), que são de toda
	// tela e migram com ela, não com a ficha.
	scene := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pv/-1?tab=proficiencies&tempestade=1", id), "").Body.String()
	if !strings.Contains(scene, "sheet-scene") {
		t.Fatal("o gesto não devolveu a cena da ficha")
	}
	if grimoire := regexp.MustCompile(`[\w:-]*grimorio[\w-]*`).FindAllString(scene, 5); len(grimoire) > 0 {
		t.Errorf("a ficha paralela desenhou classe do grimório: %v", grimoire)
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
		t.Fatal("a ficha paralela não desenhou o diálogo do ⋯")
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
		fmt.Sprintf("/personagens/%d/efeitos/condicao/caido?tab=expertises&tempestade=1", id), ""); rec.Code != http.StatusOK {
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
