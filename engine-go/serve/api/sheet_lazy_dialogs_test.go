package api

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// O DIÁLOGO PESADO SÓ SE DESENHA ABERTO (ALE-642).
//
// O Aprender magia, o Adicionar item e o Escolher eram desenhados fechados a cada vez que a aba
// aparecia e a cada gesto nela: 86 a 154 KB por diálogo, mais da metade da cena. Quem abre escreve o
// `$sheet_dialog` e pede a cena; o servidor desenha o corpo só quando o sinal diz que ele está aberto.

// sheetTabWith é a aba pedida com os sinais `signals`, como o `@get` da cena os manda.
func sheetTabWith(t *testing.T, f sceneFixture, id int64, tab, signals string) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=%s&datastar=%s", id, tab, url.QueryEscape(signals)), "").Body.String()
}

func TestTheHeavyDialogsAreDrawnOnlyWhenOpen(t *testing.T) {
	f := newSceneFixture(t)
	for _, c := range []struct {
		dialog, tab, body string
	}{
		{"spell-learn", "spells", "/magias/aprende/"},
		{"item-add", "bag", "/itens/adiciona/"},
		{"power-choices", "abilities", `class="ds-steps"`},
	} {
		closed := sheetTabWith(t, f, f.charID, c.tab, `{}`)
		if strings.Contains(closed, c.body) {
			t.Errorf("o %s fechado já traz o corpo (%s) na aba %s", c.dialog, c.body, c.tab)
		}
		if !strings.Contains(html.UnescapeString(closed), `$sheet_dialog = '`+c.dialog+`'`) {
			t.Errorf("quem abre o %s não escreve o $sheet_dialog", c.dialog)
		}
		if !strings.Contains(closed, `id="`+c.dialog+`-loading"`) {
			t.Errorf("o %s fechado não diz que está carregando, para o instante entre abrir e chegar", c.dialog)
		}
		if open := sheetTabWith(t, f, f.charID, c.tab, `{"sheet_dialog":"`+c.dialog+`"}`); !strings.Contains(open, c.body) {
			t.Errorf("com o $sheet_dialog = %q, a aba %s não traz o corpo dele (%s)", c.dialog, c.tab, c.body)
		}
	}
}

// dialogOfTab é o diálogo pesado de cada aba: o que o teste que lê o conteúdo dele precisa ABRIR, como
// o jogador abre, antes de ler.
var dialogOfTab = map[string]string{"abilities": "power-choices", "spells": "spell-learn", "bag": "item-add"}

// openDialogSignals são os sinais de quem está com o diálogo pesado de `tab` aberto, ou `{}`.
func openDialogSignals(tab string) string {
	if dialog, ok := dialogOfTab[tab]; ok {
		return `{"sheet_dialog":"` + dialog + `"}`
	}
	return `{}`
}

// sheetTabOpen é a aba com o diálogo pesado dela aberto.
func sheetTabOpen(t *testing.T, f sceneFixture, id int64, tab string) string {
	t.Helper()
	return sheetTabWith(t, f, id, tab, openDialogSignals(tab))
}
