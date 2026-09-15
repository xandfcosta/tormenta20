package campaigns

import "testing"

// A mesa de outra pessoa diz DE QUEM ela é, não a postura de quem lê (ALE-120).
//
// O `dono` deixou de ser `*string` na ALE-278: a cena declara o `ListRow` e ali
// ausente é a string VAZIA, então o ponteiro não carregava informação nenhuma —
// só a obrigação de desreferenciar. Os quatro casos continuam os mesmos.
func TestTheRoleSaysWhoseSomeoneElsesTableIs(t *testing.T) {
	if got := roleLabel("gm", "Bruna"); got != "Mesa de Bruna" {
		t.Errorf("papel = %q — a mesa alheia não pode dizer \"Mestrando\"", got)
	}
	if got := roleLabel("gm", ""); got != "Mestrando" {
		t.Errorf("papel = %q", got)
	}
	if got := roleLabel("player", ""); got != "Jogando" {
		t.Errorf("papel = %q", got)
	}
	// O dono VAZIO é o mesmo caso do ausente, e não um terceiro: quem lê a
	// própria mesa vê a postura.
	if got := roleLabel("player", ""); got != "Jogando" {
		t.Errorf("dono vazio virou %q em vez da postura", got)
	}
}
