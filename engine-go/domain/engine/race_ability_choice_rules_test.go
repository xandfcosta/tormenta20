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

// p29 Memória Póstuma: "recebe um poder geral a sua escolha". O poder vale
// como se escolhido — e um da Tormenta CONTA na perda de Carisma: a isenção é
// só da troca da Deformidade (ALE-502).
func TestTheMemoryGivesAGeneralPowerThatCountsLikeAnyOther(t *testing.T) {
	r := BookRuleset(primeBook(t))
	osteon := Character{
		Races:                []CharacterRace{{Race: "osteon"}},
		RaceAttributeChoices: `{"abilityPicks":{"osteon-memoria-postuma":["vontade-de-ferro"]}}`,
		ClassPowers:          "[]",
	}
	if got := expertiseBonus(t, r, osteon, "Vontade"); got != 2 {
		t.Errorf("Vontade de Ferro pela Memória Póstuma deu +%d em Vontade, esperava +2", got)
	}
	osteon.RaceAttributeChoices = `{"abilityPicks":{"osteon-memoria-postuma":["antenas"]}}`
	if got := charismaFromTormenta(r, osteon); got != -1 {
		t.Errorf("Antenas pela Memória Póstuma tirou %d de Carisma, esperava −1 (p136)", got)
	}
}

// p29 Memória Póstuma: "você pode ser um osteon de outra raça humanoide que não
// humano. Neste caso, você ganha uma habilidade dessa raça a sua escolha. Se a
// raça era de tamanho diferente de Médio, você também possui sua categoria de
// tamanho." Uma HABILIDADE — não os atributos da raça (ALE-504).
func TestTheMemoryBorrowsAnAbilityAndTheSizeOfAnotherRace(t *testing.T) {
	r := BookRuleset(primeBook(t))
	osteon := Character{
		Races:                []CharacterRace{{Race: "osteon"}},
		RaceAttributeChoices: `{"abilityPicks":{"osteon-memoria-postuma":["goblin-rato-das-ruas"]}}`,
		ClassPowers:          "[]",
	}
	if got := expertiseBonus(t, r, osteon, "Fortitude"); got != 2 {
		t.Errorf("Rato das Ruas emprestado deu +%d em Fortitude, esperava +2", got)
	}
	if got := expertiseBonus(t, r, osteon, "Furtividade"); got != 2 {
		t.Errorf("o osteon goblin é Pequeno e ganhou %+d em Furtividade, esperava +2 (Tab. 1-21)", got)
	}
	effects := ComputeItemEffects(r.ActiveItemsFor(osteon))
	if got := StatFor(effects, ModifierTarget{K: "attribute", Name: "dexterity"}).Total; got != 0 {
		t.Errorf("o osteon goblin ganhou %+d de Destreza: a Memória empresta uma habilidade, e não os atributos", got)
	}

	// A habilidade emprestada que pede escolha a guarda pela chave dela.
	osteon.RaceAttributeChoices = `{"abilityPicks":{"osteon-memoria-postuma":["qareen-resistencia-elemental"],` +
		`"qareen-resistencia-elemental":["qareen-res-fogo"]}}`
	effects = ComputeItemEffects(r.ActiveItemsFor(osteon))
	if got := StatFor(effects, ModifierTarget{K: "damageReduction", DamageType: "fogo"}).Total; got != 10 {
		t.Errorf("a Resistência Elemental emprestada com fogo deu RD %d a fogo, esperava 10", got)
	}
	if got := expertiseBonus(t, r, osteon, "Furtividade"); got != 0 {
		t.Errorf("o osteon qareen é Médio e ganhou %+d em Furtividade", got)
	}
}

// p31 suraggel: "Luz Sagrada (Aggelus). Você recebe +2 em Diplomacia e
// Intuição" e "Sombras Profanas (Sulfure). Você recebe +2 em Enganação e
// Furtividade". A habilidade vale pela ASCENDÊNCIA escolhida na distribuição
// de atributo — um lugar só para a escolha (ALE-505).
func TestTheSuraggelAbilityFollowsTheAncestry(t *testing.T) {
	r := BookRuleset(primeBook(t))
	suraggel := func(ancestry string) Character {
		return Character{
			Races:                []CharacterRace{{Race: "suraggel"}},
			RaceAttributeChoices: `{"floatingPicks":[],"ascendencia":"` + ancestry + `"}`,
			ClassPowers:          "[]",
		}
	}
	aggelus, sulfure := suraggel("aggelus"), suraggel("sulfure")
	for _, c := range []struct {
		name  string
		ch    Character
		skill string
		want  int
	}{
		{"aggelus", aggelus, "Diplomacia", 2}, {"aggelus", aggelus, "Intuição", 2}, {"aggelus", aggelus, "Enganação", 0},
		{"sulfure", sulfure, "Enganação", 2}, {"sulfure", sulfure, "Furtividade", 2}, {"sulfure", sulfure, "Diplomacia", 0},
	} {
		if got := expertiseBonus(t, r, c.ch, c.skill); got != c.want {
			t.Errorf("suraggel %s: %s = %+d, esperava %+d", c.name, c.skill, got, c.want)
		}
	}
}
