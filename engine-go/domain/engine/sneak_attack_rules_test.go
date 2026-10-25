package engine

import "testing"

// O ATAQUE FURTIVO É UM DADO QUE CRESCE (p73, ALE-486).
//
// "quando atinge uma criatura desprevenida com um ataque corpo a corpo ou em
// alcance curto, ou uma criatura que esteja flanqueando, você causa 1d6 pontos
// de dano extra. A cada dois níveis, esse dano extra aumenta em +1d6."
//
// Os níveis são de LADINO (p226, "Para classes, use seu nível naquela classe"):
// 1d6 no 1º, 2d6 no 3º … 10d6 no 19º. A situação é da mesa — desprevenido e
// flanqueado o mestre diz —, então o dado é OFERECIDO como interruptor e só
// entra na carta da arma quando o jogador o liga.
func TestTheSneakAttackDiceGrowEveryTwoRogueLevels(t *testing.T) {
	world := BookRuleset(primeBook(t))
	on := map[string]bool{FlagGroupID("ataque-furtivo"): true}

	for _, tt := range []struct {
		level int
		dice  string
	}{{1, "1d6"}, {2, "1d6"}, {3, "2d6"}, {5, "3d6"}, {10, "5d6"}, {19, "10d6"}, {20, "10d6"}} {
		rogue := wielding(attributeSumHero("Ladino", tt.level, `[]`), "adaga")
		off := wieldedCards(world.ComputeWeaponCards(rogue, nil))
		if len(off) != 1 {
			t.Fatalf("Ladino %d com adaga virou %d cartas", tt.level, len(off))
		}
		if len(off[0].ExtraDamage) != 0 {
			t.Errorf("Ladino %d: o Ataque Furtivo entrou sem ser ligado: %+v", tt.level, off[0].ExtraDamage)
		}
		cards := wieldedCards(world.ComputeWeaponCards(rogue, on))
		if len(cards[0].ExtraDamage) != 1 || cards[0].ExtraDamage[0].Dice != tt.dice {
			t.Errorf("Ladino %d, ligado: a adaga traz %+v, want um dado extra de %s", tt.level, cards[0].ExtraDamage, tt.dice)
		}
	}

	// Multiclasse: o dado é do nível de LADINO.
	multi := wielding(attributeSumHero("Ladino", 10, `[]`), "adaga")
	multi.Classes = []CharacterClass{{ClassName: "Guerreiro", Level: 7}, {ClassName: "Ladino", Level: 3}}
	if got := wieldedCards(world.ComputeWeaponCards(multi, on))[0].ExtraDamage; len(got) != 1 || got[0].Dice != "2d6" {
		t.Errorf("Guerreiro 7 / Ladino 3, ligado: %+v, want 2d6 — é o nível NA CLASSE", got)
	}
}
