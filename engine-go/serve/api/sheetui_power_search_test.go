package api

import (
	"regexp"
	"testing"
)

// A BUSCA DE PODERES FILTRA A ABA PODERES, E SÓ ELA.
//
// O painel dos poderes com termo de busca montava só o resultado, e a lista de ações
// ficava vazia; mas a postura e a declaração em vigor tiram dessa lista o texto e a
// página do poder, e as Ações a leem inteira. Quem buscava um poder e ia para Efeitos
// via a Fúria em vigor sem texto.

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
