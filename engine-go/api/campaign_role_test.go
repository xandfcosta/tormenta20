package api

import "testing"

// A mesa de outra pessoa diz DE QUEM ela é, não a postura de quem lê (ALE-120).
func TestTheRoleSaysWhoseSomeoneElsesTableIs(t *testing.T) {
	bruna := "Bruna"
	if got := papelNaCampanha("gm", &bruna); got != "Mesa de Bruna" {
		t.Errorf("papel = %q — a mesa alheia não pode dizer \"Mestrando\"", got)
	}
	if got := papelNaCampanha("gm", nil); got != "Mestrando" {
		t.Errorf("papel = %q", got)
	}
	if got := papelNaCampanha("player", nil); got != "Jogando" {
		t.Errorf("papel = %q", got)
	}
	vazio := ""
	if got := papelNaCampanha("player", &vazio); got != "Jogando" {
		t.Errorf("dono vazio virou %q em vez da postura", got)
	}
}
