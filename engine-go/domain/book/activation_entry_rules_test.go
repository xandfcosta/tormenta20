package book

import "testing"

// O VERBETE DONO DE UMA ATIVAÇÃO fica na procedência dela (ver `entryOf`): pelo id, ou pelo nome sem atravessar.
// A ativação do caçador serve também ao druida e ao dahllan, e a classe e a página dela continuam as do caçador; a
// Engenhosidade do kliren divide o nome com a do inventor e não ganha a classe dele.
func TestTheActivationEntryStaysInTheActivationsOwnScope(t *testing.T) {
	for _, c := range []struct {
		name, activation, class string
		page                    int
	}{
		{"a Coragem, ligada pelo nome", "class.paladino.julgamento-coragem", "Paladino", 83},
		{"a Empatia Selvagem do caçador", "class.cacador.empatia-selvagem", "Caçador", 51},
		{"a Engenhosidade do kliren", "race.kliren.engenhosidade", "", 28},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := ActivationOf(c.activation, "")
			if spec == nil {
				t.Fatalf("a ativação %s não está no registro", c.activation)
			}
			entry := entryOf(*spec)
			if entry.class != c.class || entry.page != c.page {
				t.Errorf("o verbete de %s é %+v; want classe %q e p%d", c.activation, entry, c.class, c.page)
			}
		})
	}
}
