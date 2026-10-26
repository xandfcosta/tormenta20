package engine

import "testing"

// A ESCOLHA DE HABILIDADE DE RAÇA (ALE-493): mora no blob da raça, pelo id da
// habilidade, como o `powerChoices` mora pelo id do poder.

func expertiseBonus(t *testing.T, r *Ruleset, ch Character, skill string) int {
	t.Helper()
	return StatFor(ComputeItemEffects(r.ActiveItemsFor(ch)), ModifierTarget{K: "expertise", Name: skill}).Total
}

func charismaFromTormenta(r *Ruleset, ch Character) int {
	for _, item := range r.ActiveItemsFor(ch) {
		if item.Source == "Poderes da Tormenta" {
			return int(item.Modifiers[0].Amount)
		}
	}
	return 0
}

// p24 Deformidade: "+2 em duas perícias a sua escolha" — e "você pode trocar
// um desses bônus por um poder da Tormenta a sua escolha (ele também não conta
// para perda de Carisma)".
func TestTheDeformitySwapsABonusForATormentaPowerThatCostsNoCharisma(t *testing.T) {
	r := BookRuleset(primeBook(t))
	lefou := Character{
		Races:                []CharacterRace{{Race: "lefou"}},
		RaceAttributeChoices: `{"floatingPicks":["strength","dexterity","constitution"],"abilityPicks":{"lefou-deformidade":["Furtividade","olhos-vermelhos"]}}`,
		ClassPowers:          "[]",
	}
	if got := expertiseBonus(t, r, lefou, "Furtividade"); got != 2 {
		t.Errorf("Furtividade escolhida na Deformidade = +%d, esperava +2", got)
	}
	if got := expertiseBonus(t, r, lefou, "Percepção"); got != 0 {
		t.Errorf("Percepção não foi escolhida e ganhou +%d", got)
	}
	if got := charismaFromTormenta(r, lefou); got != 0 {
		t.Errorf("o poder trocado na Deformidade tirou %d de Carisma, e a p24 diz que não conta", got)
	}
	if got := r.RaceHeldPowers(lefou); len(got) != 1 || got[0] != "olhos-vermelhos" {
		t.Errorf("o poder trocado não ficou com a ficha: %v", got)
	}

	// O CONTROLE: o mesmo poder ESCOLHIDO como poder conta, −1 (p136).
	lefou.ClassPowers = `["olhos-vermelhos"]`
	if got := charismaFromTormenta(r, lefou); got != -1 {
		t.Fatalf("controle: um poder da Tormenta escolhido tirou %d de Carisma, esperava −1", got)
	}
}

// p28 Vanguardista: "+2 em Ofício (um qualquer, a sua escolha)". A
// proficiência em armas de fogo é padrão da ficha, como a da classe
// (`book.GrantedProficiencies`).
func TestTheVanguardistAddsTwoToTheChosenCraft(t *testing.T) {
	r := BookRuleset(primeBook(t))
	kliren := Character{
		Races:                []CharacterRace{{Race: "kliren"}},
		RaceAttributeChoices: `{"abilityPicks":{"kliren-vanguardista":["Ofício (alquimista)"]}}`,
		ClassPowers:          "[]",
		Proficiencies:        "[]",
	}
	if got := expertiseBonus(t, r, kliren, "Ofício (alquimista)"); got != 2 {
		t.Errorf("Ofício escolhido no Vanguardista = +%d, esperava +2", got)
	}
}
