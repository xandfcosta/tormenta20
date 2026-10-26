package engine

import "testing"

// "Inteligência, exceto bônus temporários" (Especialista, p73): o efeito
// aplicado com duração sai da conta, e o item vestido fica — decisão do dono
// na ALE-501. O controle é o total da ficha, que soma os dois.
func TestThePermanentAttributeLeavesTheAppliedEffectOut(t *testing.T) {
	catalogs := primeBook(t)
	earrings, vested := "brincos-da-sagacidade", "vested"
	ch := Character{
		Intelligence: 2,
		Items:        []CharacterItem{{CatalogID: &earrings, Name: "Brincos da sagacidade", Quantity: 1, Equipped: &vested}},
		ActiveEffects: []ActiveEffectRow{{CatalogID: "astucia", Scope: "scene",
			Modifiers: `[{"target":{"k":"attribute","name":"intelligence"},"amount":2,"bonusType":"untyped"}]`}},
		ActiveConditions: "[]",
	}
	ruleset := BookRuleset(catalogs)

	if got := ruleset.PermanentAttribute(ch, "intelligence"); got != 3 {
		t.Errorf("Inteligência sem temporários = %d, esperava 3 (2 da ficha + 1 dos brincos, sem o +2 da cena)", got)
	}
	whole := attributeBreakdown(ch, "intelligence", ComputeItemEffects(ruleset.ActiveItemsFor(ch)))
	if whole.Total != 5 {
		t.Fatalf("controle: a Inteligência da ficha = %d, esperava 5 — o efeito ou o item não chegou à conta", whole.Total)
	}
}
