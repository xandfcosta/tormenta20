package api

import (
	"regexp"
	"strings"
	"testing"
)

// A BUSCA DE PODERES FILTRA A ABA PODERES, E SÓ ELA.
//
// O painel dos poderes com termo de busca montava só o resultado, e a lista de ações
// ficava vazia; mas a postura e a declaração em vigor tiram dessa lista o texto e a
// página do poder, e as Ações a leem inteira. Quem buscava um poder e ia para Efeitos
// via a Fúria em vigor sem texto. E o Escolher buscava pelo MESMO sinal: digitar ali
// deixava a aba Poderes filtrada atrás.

func stanceDialog(t *testing.T, page string) string {
	t.Helper()
	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="effect-stance-furia".*?</dialog>`).FindString(page)
	if dialog == "" {
		t.Fatal("a ficha não desenhou a descrição da Fúria em vigor")
	}
	return dialog
}

func TestAPowerSearchDoesNotEmptyTheStanceInPlay(t *testing.T) {
	f, id, _ := effectsInPlay(t)
	plain := stanceDialog(t, sheetTabWith(t, f, id, "conditionals", `{}`))
	// O CONTROLE: sem busca, a descrição traz a página do poder.
	if !regexp.MustCompile(`p\d+`).MatchString(plain) {
		t.Fatalf("a Fúria em vigor não tem página nem sem busca — o caso mediria o vazio:\n%s", plain)
	}
	searching := stanceDialog(t, sheetTabWith(t, f, id, "conditionals", `{"power_search":"zzz"}`))
	if searching != plain {
		t.Errorf("com uma busca de poderes na aba de trás, a Fúria em vigor mudou:\nsem busca:\n%s\ncom busca:\n%s", plain, searching)
	}
}

// classPowerSwitches conta os poderes oferecidos no passo da classe do Escolher.
func classPowerSwitches(t *testing.T, page string) int {
	t.Helper()
	_, step, ok := strings.Cut(page, `id="choice-step-class-Paladino-powers"`)
	if !ok {
		t.Fatal("o Escolher não tem o passo dos poderes do Paladino")
	}
	step, _, _ = strings.Cut(step, `id="choice-step-`)
	return strings.Count(step, "/poderes/escolhe/")
}

func TestTheChoicesSearchHasItsOwnSignal(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Paladino", 3, 0, 0)
	all := classPowerSwitches(t, sheetTabWith(t, f, id, "abilities", `{"sheet_dialog":"power-choices"}`))
	if all == 0 {
		t.Fatal("o passo do Paladino não oferece poder nenhum — o caso mediria o vazio")
	}

	// A busca do Escolher filtra o passo — "aura" casa uma parte dos poderes do Paladino...
	page := sheetTabWith(t, f, id, "abilities", `{"sheet_dialog":"power-choices","choice_search":"aura"}`)
	if got := classPowerSwitches(t, page); got == 0 || got >= all {
		t.Errorf("a busca do Escolher por \"aura\" deixou %d de %d poderes", got, all)
	}
	// ...e não a aba Poderes atrás dele.
	if strings.Contains(page, `Nenhum poder para "aura"`) {
		t.Error("buscar no Escolher filtrou a aba Poderes")
	}

	// E a busca da aba Poderes não filtra o Escolher.
	if got := classPowerSwitches(t, sheetTabWith(t, f, id, "abilities", `{"sheet_dialog":"power-choices","power_search":"aura"}`)); got != all {
		t.Errorf("a busca da aba Poderes filtrou o Escolher: %d de %d poderes", got, all)
	}
}
