package engine

import "testing"

// OS SITUACIONAIS DA ARMA (ALE-649).

// PÃO E CIRCO: "pode escolher causar dano não letal sem sofrer a penalidade de –5" (p91). A carta de arma letal
// troca o tipo sem a linha da Tabela 5-3, como o Estilo Desarmado; o desarmado, que já é não letal, não muda.
func TestBreadAndCircusesHitsNonLethalWithoutPenalty(t *testing.T) {
	world := BookRuleset(primeBook(t))
	gladiator := func(chosen string) []WeaponCard {
		ch := wielding(attributeSumHero("Guerreiro", 5, `[]`), "espada-longa")
		ch.Origin, ch.OriginChoices = "gladiador", chosen
		return world.ComputeWeaponCards(ch, nil)
	}
	with, without := gladiator(`["origin-gladiador-unique"]`), gladiator(`[]`)
	if !cardOf(t, with, "espada-longa").SwitchesDamageFreely {
		t.Error("a espada do gladiador com Pão e Circo não troca para não letal sem a penalidade")
	}
	if cardOf(t, without, "espada-longa").SwitchesDamageFreely {
		t.Error("a espada sem Pão e Circo troca o tipo de dano de graça")
	}
	for _, card := range with {
		if card.Name == UnarmedName && card.SwitchesDamageFreely {
			t.Error("o desarmado, já não letal, troca para letal sem a penalidade: o Pão e Circo é só o não letal")
		}
	}
}

// GRUDAR O CANO: "quando faz um ataque à distância com uma arma de fogo contra um oponente adjacente, você não
// sofre a penalidade de –5 no teste de ataque e aumenta seu dano em um passo" (p48). O adjacente é do mestre e do
// mapa: o jogador liga o interruptor, e a arma de fogo, só ela, ganha as duas coisas.
func TestGluingTheBarrelWorksOnlyWithTheFirearmAndTheSwitch(t *testing.T) {
	world := BookRuleset(primeBook(t))
	glued := map[string]bool{FlagGroupID("grudar-o-cano"): true}
	gunner := func(weapon string) Character {
		ch := wielding(attributeSumHero("Bucaneiro", 5, `["class.bucaneiro.pistoleiro","class.bucaneiro.grudar-o-cano"]`), weapon)
		ch.Proficiencies = `["armas-simples","armas-marciais","armas-de-fogo"]`
		return ch
	}
	off := cardOf(t, world.ComputeWeaponCards(gunner("pistola"), nil), "pistola")
	on := cardOf(t, world.ComputeWeaponCards(gunner("pistola"), glued), "pistola")
	if on.Damage != DamageStep(off.Damage, 1) || !on.ShootsIntoMelee {
		t.Errorf("a pistola com o cano grudado causa %s (antes %s) e atira em corpo a corpo: %v; want um passo e sem o –5",
			on.Damage, off.Damage, on.ShootsIntoMelee)
	}
	if off.ShootsIntoMelee {
		t.Error("a pistola sem o interruptor atira em corpo a corpo sem o –5")
	}
	bow := cardOf(t, world.ComputeWeaponCards(gunner("arco-curto"), glued), "arco-curto")
	if bow.ShootsIntoMelee || bow.Damage != cardOf(t, world.ComputeWeaponCards(gunner("arco-curto"), nil), "arco-curto").Damage {
		t.Error("o arco ganhou o Grudar o Cano, que é da arma de fogo")
	}
}
