package book

import "testing"

// UM ID, UMA ATIVAÇÃO: o `ActivationOf` guarda a última lida, e a de cima vira
// dado que ninguém lê. O Ponto Fraco ficou com duas — a passiva antiga e o
// acréscimo da Marca da Presa (ALE-509) — sem nada acusar.
func TestNoTwoActivationsShareAnId(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range Activations() {
		if seen[a.ID] {
			t.Errorf("a ativação %s aparece duas vezes no registro: junte as duas numa só", a.ID)
		}
		seen[a.ID] = true
	}
	if len(seen) < 400 {
		t.Fatalf("mediu %d ids; o catálogo tem mais de 400", len(seen))
	}
}
