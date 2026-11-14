package api

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// SINAL QUE NÃO SE LÊ VAI PARA O LOG. Um sinal com o tipo errado faz o `ReadSignals` recusar o corpo
// INTEIRO, e a ficha segue sem nenhum sinal daquele pedido — a busca, o filtro, o diálogo aberto. O
// `$race_attributes = null` do Escolher viveu assim, sem testemunha, até um diálogo se esvaziar na tela.
func TestASheetGestureWithUnreadableSignalsLogsTheCause(t *testing.T) {
	f := newSceneFixture(t)
	var logged bytes.Buffer
	f.s.log = slog.New(slog.NewJSONHandler(&logged, nil))

	path := fmt.Sprintf("/personagens/%d/vitais/pv/1", f.charID)
	f.requests(t, f.player, http.MethodPost, path+"?tab=combat", `{"race_attributes":""}`)

	out := logged.String()
	if !strings.Contains(out, `"path":"`+path+`"`) {
		t.Errorf("o log não diz qual gesto perdeu os sinais:\n%s", out)
	}
	if !strings.Contains(out, "race_attributes") {
		t.Errorf("o log não diz qual sinal não se leu (race_attributes):\n%s", out)
	}
}

// O CORPO VAZIO é "sem sinais", como o `GET` sem o parâmetro `datastar` é para o SDK, e não um erro: o
// log é para o sinal que veio e não se leu.
func TestASheetGestureWithAnEmptyBodyLogsNothing(t *testing.T) {
	f := newSceneFixture(t)
	var logged bytes.Buffer
	f.s.log = slog.New(slog.NewJSONHandler(&logged, nil))

	f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/vitais/pv/1?tab=combat", f.charID), "")

	if logged.Len() > 0 {
		t.Errorf("o gesto sem corpo deixou aviso no log:\n%s", logged.String())
	}
}
