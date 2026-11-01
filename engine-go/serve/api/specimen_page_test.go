package api

import (
	"strings"
	"testing"
)

// A folha viva mostra UMA amostra de cada peça, que segue o seletor de modo, dividida
// em seções com um índice. Duas amostras lado a lado faziam o seletor parecer sem
// efeito.
func TestTheSpecimenDrawsOneSampleDividedInSections(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	body := pedeNoMestre(t, s, gm, "GET", "/sistema", "").Body.String()

	if n := strings.Count(body, "Rolar ataque"); n != 1 {
		t.Errorf("o botão de exemplo saiu %d vezes, quero uma amostra só", n)
	}
	if strings.Contains(body, "data-preview") {
		t.Error("a folha ainda força um modo num pedaço da página")
	}
	if !strings.Contains(body, `action="/modo"`) {
		t.Error("a folha perdeu o seletor de modo")
	}
	for _, id := range []string{"acoes", "campos", "navegacao", "listas", "numeros", "cartoes", "retorno"} {
		if !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("falta a seção %q", id)
		}
		if strings.Count(body, `href="#`+id+`"`) != 2 {
			t.Errorf("a seção %q não está nos dois índices, o do desktop e o do celular", id)
		}
	}
	if !strings.Contains(body, `class="ds-entry-card"`) {
		t.Error("o cartão de verbete não está na folha")
	}
}
