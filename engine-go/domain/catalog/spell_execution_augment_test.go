package catalog

import (
	"regexp"
	"strings"
	"testing"
)

// spellForExecutionSweep é o que a varredura do aprimoramento que muda a execução precisa.
type spellForExecutionSweep struct {
	Name     string `json:"name"`
	Augments []struct {
		Description string `json:"description"`
		Execution   string `json:"execution"`
	} `json:"augments"`
}

// changesExecution casa o início da frase, e não a palavra solta: "Como parte da execução: ataque corpo a
// corpo" não muda a execução de nada.
var changesExecution = regexp.MustCompile(`(?i)^(muda (a )?)?execução (para )?`)

// executionWritten é a execução que o texto dá, na grafia do turno (`engine.ActionCost`): o que não é ação
// de um turno — 1 minuto, 1 hora, 1 dia — é a longa, que numa cena de ação não cabe, como o ritual (p131).
func executionWritten(rest string) string {
	rest = strings.ToLower(rest)
	for _, c := range []struct{ prefix, cost string }{
		{"reação", "reacao"}, {"ação de movimento", "movimento"}, {"movimento", "movimento"},
		{"padrão", "padrao"}, {"completa", "completa"}, {"ação livre", "livre"}, {"livre", "livre"},
	} {
		if strings.HasPrefix(rest, c.prefix) {
			return c.cost
		}
	}
	return "longa"
}

// TestEveryAugmentThatChangesTheExecutionSaysWhichOne amarra a marca `execution` do aprimoramento ao texto.
//
// A conjuração cobra do turno a execução que o aprimoramento escolhido dá (p233): o Armamento da Natureza
// com "Muda a execução para ação de movimento" gasta o movimento, e não a padrão. O texto é prosa, e a
// marca é a única leitura dele — o conjurador a usa para a execução e para o que a Magia Acelerada apaga.
func TestEveryAugmentThatChangesTheExecutionSaysWhichOne(t *testing.T) {
	spells := decodeResource[map[string]spellForExecutionSweep](t, "spells")
	if len(spells) == 0 {
		t.Fatal("nenhuma magia no catálogo: não há o que varrer, e verde aqui não valeria nada")
	}
	changing := 0
	for id, m := range spells {
		for i, a := range m.Augments {
			want := ""
			if at := changesExecution.FindStringIndex(a.Description); at != nil {
				want = executionWritten(a.Description[at[1]:])
				changing++
			}
			if a.Execution != want {
				t.Errorf("%s (%s) aprimoramento %d: o texto dá a execução %q e o campo `execution` diz %q: %q",
					m.Name, id, i, want, a.Execution, a.Description)
			}
		}
	}
	if changing == 0 {
		t.Fatal("a varredura não achou aprimoramento que muda a execução: a regex parou de casar o texto")
	}
}
