package engine

import "testing"

// A PROFICIÊNCIA QUE VEM DE PODER E DE HABILIDADE DE RAÇA (ALE-649). O motor a calcula pelo que a ficha TEM, e
// não pela coluna gravada: a categoria inteira (o Pistoleiro, p48), a categoria num propósito (o Armeiro, "armas
// marciais corpo a corpo", p68; a Balística, "de ataque à distância", p69) e a arma nomeada (a Tradição de
// Heredrimm, p20; o Mestre do Tridente, p30). O –5 de quem não sabe usar some só onde ela alcança (p142).

const unproficientPenalty = 5

// onlySimple é quem só sabe usar armas simples.
func onlySimple(ch Character) Character {
	ch.Proficiencies = `["armas-simples"]`
	return ch
}

// attackWith é o ataque da carta da arma, com quem a empunha sozinha.
func attackWith(t *testing.T, world *Ruleset, ch Character, weapon string) int {
	t.Helper()
	return cardOf(t, world.ComputeWeaponCards(wielding(ch, weapon), nil), weapon).Attack
}

func TestThePowerProficiencyLiftsThePenaltyWhereItReaches(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		why    string
		with   Character
		weapon string
		lifted bool
	}{
		{"o Pistoleiro com a pistola", attributeSumHero("Bucaneiro", 5, `["class.bucaneiro.pistoleiro"]`), "pistola", true},
		{"o Armeiro com a espada longa", attributeSumHero("Inventor", 5, `["class.inventor.armeiro"]`), "espada-longa", true},
		{"o Armeiro com o arco longo", attributeSumHero("Inventor", 5, `["class.inventor.armeiro"]`), "arco-longo", false},
		{"a Balística com o arco longo", attributeSumHero("Inventor", 5, `["class.inventor.balistica"]`), "arco-longo", true},
		{"a Balística com a pistola", attributeSumHero("Inventor", 5, `["class.inventor.balistica"]`), "pistola", true},
		{"a Balística com a espada longa", attributeSumHero("Inventor", 5, `["class.inventor.balistica"]`), "espada-longa", false},
		{"o anão com o machado de batalha", ofRace(attributeSumHero("Inventor", 5, `[]`), "anao"), "machado-batalha", true},
		{"o anão com o machado anão, que é exótico", ofRace(attributeSumHero("Inventor", 5, `[]`), "anao"), "machado-anao", true},
		{"o anão com a espada longa", ofRace(attributeSumHero("Inventor", 5, `[]`), "anao"), "espada-longa", false},
		{"o tritão com o tridente", ofRace(attributeSumHero("Inventor", 5, `[]`), "sereia-tritao"), "tridente", true},
	} {
		// A MESMA FICHA com todas as categorias gravadas é o ataque sem penalidade nenhuma: a diferença é só
		// o –5, e o +2 do Heredrimm não entra na conta.
		every := c.with
		every.Proficiencies = `["armas-simples","armas-marciais","armas-exoticas","armas-de-fogo"]`
		got := attackWith(t, world, every, c.weapon) - attackWith(t, world, onlySimple(c.with), c.weapon)
		want := unproficientPenalty
		if c.lifted {
			want = 0
		}
		if got != want {
			t.Errorf("%s: só com armas simples gravadas, o ataque fica %d abaixo; want %d", c.why, got, want)
		}
	}
}
