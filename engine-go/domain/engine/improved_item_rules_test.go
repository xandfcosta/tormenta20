package engine

import "testing"

// A MELHORIA APRIMORADO: "só pode ser aplicada a uma ferramenta ou vestuário que modifique uma perícia
// (reduza uma penalidade ou forneça um bônus) e fornece um bônus de +1 nessa perícia (ou aumenta o bônus
// fornecido em +1). Por exemplo, uma maleta de medicamentos aprimorada fornece +1 em Cura e uma luva de
// pelica aprimorada fornece +2 em Ladinagem" (p164).
func TestTheImprovedItemAddsOneToItsSkill(t *testing.T) {
	world := BookRuleset(primeBook(t))
	skillTotal := func(item, skill string, improved bool, on map[string]bool) int {
		hero := switchHero(`[]`, 0, item)
		hero.Expertises = []CharacterExpertise{{Name: skill, Attribute: "wisdom"}}
		if item == "luva-pelica" {
			hero.Items[0].Equipped = ptrTo("vested")
		}
		if improved {
			hero.Items[0].Improvements = `["melhoria-aprimorado"]`
		}
		for _, row := range world.ComputeSheet(hero, on).Expertises {
			if row.Name == skill {
				return row.Total
			}
		}
		t.Fatalf("a ficha não tem a perícia %s", skill)
		return 0
	}
	for _, c := range []struct{ name, item, skill string }{
		{"a maleta, que só tira a penalidade", "maleta-medicamentos", "Cura"},
		{"a luva, que já dá bônus", "luva-pelica", "Ladinagem"},
		{"o instrumento, ligado a Atuação (p158)", "instrumento-musical", "Atuação"},
		{"os instrumentos de ofício, no Ofício da ficha", "instrumentos-de-oficio", "Ofício (ferreiro)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := skillTotal(c.item, c.skill, true, nil) - skillTotal(c.item, c.skill, false, nil); got != 1 {
				t.Errorf("a Aprimorado somou %+d em %s", got, c.skill)
			}
		})
	}
}

// Onde o item só vale numa situação, o +1 vale nela: o interruptor é o mesmo do bônus do item, e a gazua
// aprimorada não soma em todo teste de Ladinagem.
func TestTheImprovedItemFollowsTheSituationOfTheItem(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct{ item, skill string }{
		{"gazua", "Ladinagem"},
		{"estojo-de-disfarces", "Enganação"},
		{"equipamento-de-viagem", "Sobrevivência"},
		{"sela", "Cavalgar"},
		{"traje-corte", "Diplomacia"},
		{"traje-corte", "Jogatina"},
	} {
		hero := switchHero(`[]`, 0, c.item)
		hero.Expertises = []CharacterExpertise{{Name: c.skill, Attribute: "wisdom"}}
		plain := world.ComputeSheet(hero, nil)
		hero.Items[0].Improvements = `["melhoria-aprimorado"]`
		improved := world.ComputeSheet(hero, nil)
		if plain.Expertises[0].Total != improved.Expertises[0].Total {
			t.Errorf("%s: o +1 em %s somou fora da situação do item", c.item, c.skill)
		}
		found := false
		for _, offered := range ComputeItemEffects(world.ActiveItemsFor(hero)).Conditional {
			if offered.Target.K == "expertise" && offered.Target.Name == c.skill && offered.Amount == 1 {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: o +1 em %s não é oferecido na situação do item", c.item, c.skill)
		}
	}
}
