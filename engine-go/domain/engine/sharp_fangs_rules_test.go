package engine

import "testing"

// p63 "Presas Afiadas. A margem de ameaça de suas armas naturais aumenta em +2."
func TestTheSharpFangsWidenOnlyTheNaturalWeapons(t *testing.T) {
	world := BookRuleset(primeBook(t))

	druid := wielding(attributeSumHero("Druida", 5, `["class.druida.presas-afiadas"]`), "adaga")
	druid.Races = []CharacterRace{{Race: "minotauro"}}
	cards := world.ComputeWeaponCards(druid, nil)

	horns := -1
	for i, card := range cards {
		if card.Natural != nil && card.Name == "Chifres" {
			horns = i
		}
	}
	if horns < 0 {
		t.Fatalf("o druida minotauro devia ter a carta dos Chifres: %d cartas", len(cards))
	}
	if cards[horns].CritRange != 18 {
		t.Errorf("Chifres com Presas Afiadas: margem %d, want 18 (20 e +2)", cards[horns].CritRange)
	}
	if cards[0].Name != "adaga" || cards[0].CritRange != 19 {
		t.Errorf("a adaga do druida: %s %d, want 19 — a Presas Afiadas é só das armas naturais", cards[0].Name, cards[0].CritRange)
	}
	if unarmed := unarmedCard(t, world, druid); unarmed.CritRange != 20 {
		t.Errorf("o desarmado do druida: %d, want 20 — ataque desarmado não é arma natural", unarmed.CritRange)
	}
}
