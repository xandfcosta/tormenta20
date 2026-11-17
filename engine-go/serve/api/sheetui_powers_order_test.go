package api

import (
	"strings"
	"testing"
)

// AS AÇÕES SE ORDENAM PELO CUSTO DO VERBETE, e não pelo texto que a tela escreveu. A
// ordem desmontava o "POSTURA · 2+ PM" à procura de um número, e o "2+" não é um: a
// Fúria de um bárbaro de 6º, que custa 2 PM, subia como se custasse 0, acima do Brado
// Assustador de 1 PM.
func TestTheActionsAreSortedByTheEntryCostAndNotByTheWrittenOne(t *testing.T) {
	f, id := barbaro(t, 6)
	seedPowerChoices(t, f.s, id, `["class.barbaro.brado-assustador"]`, `{}`)
	page := sheetTabWith(t, f, id, "abilities", `{}`)
	_, actions, ok := strings.Cut(page, ">Ações<")
	if !ok {
		t.Fatal("a aba Poderes não tem a lista de Ações")
	}
	actions, _, _ = strings.Cut(actions, "Passivas")
	// O CONTROLE: os dois estão na lista, e a Fúria escreve o preço com degraus.
	if !strings.Contains(actions, "POSTURA · 2+ PM") || !strings.Contains(actions, "Brado Assustador") {
		t.Fatalf("a lista não tem a Fúria com degraus e o Brado — o caso mediria outra coisa:\n%s", actions)
	}
	if strings.Index(actions, "Brado Assustador") > strings.Index(actions, "Fúria") {
		t.Error("a Fúria (2 PM) vem antes do Brado Assustador (1 PM): o \"2+\" foi lido como zero")
	}
}
