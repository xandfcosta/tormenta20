package main

import "testing"

// O NOME CASA SEM O QUALIFICADOR entre parênteses, dos dois lados: o catálogo guarda "Corda (10 m)" porque os 10 m
// importam na mesa, e a tabela diz "Corda"; a tabela diz "Ração de viagem (por dia)", e o catálogo "Ração de Viagem".
func TestTheNameMatchesWithoutTheQualifier(t *testing.T) {
	for _, pair := range [][2]string{
		{"Corda (10 m)", "Corda"},
		{"Coleção de livros (Guerra)", "Coleção de livros"},
		{"Harmonizada (arma)", "Harmonizada"},
		{"Ração de Viagem", "Ração de viagem (por dia)"},
	} {
		if a, b := nameKey(pair[0]), nameKey(pair[1]); a != b || a == "" {
			t.Errorf("%q e %q não casam: %q contra %q", pair[0], pair[1], a, b)
		}
	}
	if nameKey("Corda") == nameKey("Cordas de alaúde") {
		t.Error("o controle falhou: nomes diferentes casaram")
	}
	if nameKey("(por dia)") != "" {
		t.Errorf("uma célula só de qualificador virou a chave %q, e casaria com qualquer coisa vazia", nameKey("(por dia)"))
	}
}
