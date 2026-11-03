package sheetui

import (
	"testing"

	"t20engine/domain/engine"
	"t20engine/serve/web/ds"
)

// A CONTA DA ARMA DIZ POR QUE A DESTREZA ENTROU, e o dano é outra conta, com o
// atributo que de fato soma nele. A adaga usa a Destreza no ATAQUE (p146) e o
// dano continua com a Força; a Acuidade com Arma leva a Destreza aos dois.
func TestTheWeaponSumSaysWhyDexterityAndCountsDamageApart(t *testing.T) {
	dagger := engine.WeaponCard{
		Name: "Adaga", CatalogID: "adaga", Skill: "Luta", Attack: 11, Damage: "1d4",
		DexBy: engine.DexByWeapon, StrDamage: 0,
		Expertise: engine.ExpertiseBreakdown{Attribute: "dexterity", AttrValue: 3, HalfLevel: 4, Training: 4, Trained: true, Total: 11},
	}
	attack, damage := weaponSumTerms(dagger)
	wantAttack := []ds.SumTerm{
		{Label: "Destreza", Note: "Adaga: Destreza no lugar de Força (p146)", Value: "+3"},
		{Label: "Metade do nível", Value: "+4"},
		{Label: "Treinada", Value: "+4"},
	}
	sameTerms(t, "ataque da adaga", attack, wantAttack)
	sameTerms(t, "dano da adaga", damage, []ds.SumTerm{{Label: "Força", Value: "+0"}})

	finesse := dagger
	finesse.Name, finesse.CatalogID, finesse.DexBy, finesse.DexDamage, finesse.StrDamage, finesse.DamageBonus = "Espada curta", "espada-curta", engine.DexByFinesse, true, 3, 3
	attack, damage = weaponSumTerms(finesse)
	if attack[0].Note != "Acuidade com Arma: Destreza no lugar de Força" {
		t.Errorf("a espada curta com Acuidade não diz por que usa a Destreza: %+v", attack[0])
	}
	sameTerms(t, "dano com Acuidade", damage, []ds.SumTerm{{Label: "Destreza", Note: "Acuidade com Arma", Value: "+3"}})
}

// A ARMA DE ARREMESSO ataca com Pontaria e soma a Força no dano (p142): a conta dá nome
// a ela, e a arma de disparo não tem atributo nenhum para mostrar.
func TestTheThrownWeaponSumNamesTheStrengthInTheDamage(t *testing.T) {
	spear := engine.WeaponCard{Name: "Lança (arremesso)", Skill: "Pontaria", Damage: "1d6", StrDamage: 4, DamageBonus: 4}
	_, damage := weaponSumTerms(spear)
	sameTerms(t, "dano da lança arremessada", damage, []ds.SumTerm{{Label: "Força", Value: "+4"}})
	_, rows := weaponRows(spear)
	if len(rows) != 1 || rows[0].Label != "FOR" || rows[0].Value != "+4" {
		t.Errorf("a decomposição do dano da lança arremessada: %+v, esperado FOR +4", rows)
	}

	bow := engine.WeaponCard{Name: "Arco curto", Skill: "Pontaria", Damage: "1d6"}
	_, damage = weaponSumTerms(bow)
	sameTerms(t, "dano do arco", damage, []ds.SumTerm{})
}

func sameTerms(t *testing.T, what string, got, want []ds.SumTerm) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %+v\nesperado: %+v", what, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s, parcela %d: %+v, esperado %+v", what, i, got[i], want[i])
		}
	}
}
