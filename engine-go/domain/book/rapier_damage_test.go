package book

import "testing"

// A exceção da tabela: o catálogo dava 1d8, e a Tabela 3-3 dá 1d6 ao Florete (p144).
func TestTheRapierDealsTheDamageOfTheBookTable(t *testing.T) {
	rapier := ItemByID("florete")
	if rapier == nil || rapier.Weapon == nil {
		t.Fatal("o catálogo não tem o Florete como arma")
	}
	if rapier.Weapon.Damage != "1d6" {
		t.Errorf("o Florete dá %s, e a Tabela 3-3 dá 1d6 (p144)", rapier.Weapon.Damage)
	}
	// O Florete fugaz é feito dele (p337), e o dano vem da arma de base.
	fleeting := ItemByID("florete-fugaz")
	if fleeting == nil || fleeting.Weapon == nil {
		t.Fatal("o catálogo não tem o Florete fugaz como arma")
	}
	if fleeting.Weapon.Damage != "1d6" {
		t.Errorf("o Florete fugaz dá %s, e a arma de base dá 1d6", fleeting.Weapon.Damage)
	}
}
