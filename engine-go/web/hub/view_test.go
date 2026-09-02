package hub

import "testing"

// A INICIAL do retrato corta por RUNA e não por byte.
//
// "Áurea" em UTF-8 começa com dois bytes, então `nome[:1]` devolveria meio
// caractere — e a tela mostraria um losango de substituição no lugar da letra.
// O caso veio do `api` junto com o hub (ALE-278): ele é unitário e puro, e mora
// onde a regra mora.

// Por RUNA e não por byte: "Áurea" começa com dois bytes, e cortar o primeiro
// desenha o losango de erro no lugar da letra.
func TestTheInitialCutsByRuneAndNotByByte(t *testing.T) {
	casos := map[string]string{
		"Mestre":            "M",
		"Áurea":             "Á",
		"  ébano  ":         "É",
		"":                  "?",
		"   ":               "?",
		"jogador@t20.local": "J",
	}
	for entrada, quer := range casos {
		if got := initialOf(entrada); got != quer {
			t.Errorf("inicialDe(%q) = %q, queria %q", entrada, got, quer)
		}
	}
}
