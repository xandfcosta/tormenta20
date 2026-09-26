package engine

import (
	"path/filepath"
	"testing"
)

// TestTheGodPowerReachesTheSheet (ALE-397).
//
// # O defeito
//
// A devoção NÃO era fonte de modificador. O `GodPower` entrava no motor por uma
// porta só — o `vitalGrantMods`, que existe para PV e PM e descarta todo alvo
// que não seja `maxPv`/`maxPm` —, e a lista de sistemas do coletor tinha SEIS
// fontes: item, raça, origem, classe, poder geral e Tormenta. Devoção não
// estava lá, e nunca esteve.
//
// O efeito na mesa: o texto de Escamas Dracônicas aparecia na ficha e a Defesa
// não mudava. De 72 poderes concedidos, um mexia em alguma conta — a Bênção do
// Mana, justamente porque é `maxPm`, a única família que aquela porta deixa
// passar.
//
// # Os casos
//
// Dois deuses, para o caso não medir uma entrada só: Escamas Dracônicas
// (p133) toca DOIS alvos de naturezas diferentes — Defesa e uma perícia —, e
// Astúcia da Serpente (p132) toca três perícias. Um poder que acertasse a
// perícia e errasse a Defesa passaria com um caso só.
//
// # O controle
//
// O MESMO personagem sem devoção nenhuma. Sem ele, um +2 que já viesse de
// outro lugar — raça, classe, origem — passaria por mérito do poder concedido,
// e o teste ficaria verde sobre nada.
func TestTheGodPowerReachesTheSheet(t *testing.T) {
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))

	pericias := []CharacterExpertise{
		{Name: "Fortitude", Attribute: "constitution"},
		{Name: "Enganação", Attribute: "charisma"},
		{Name: "Furtividade", Attribute: "dexterity"},
		{Name: "Intuição", Attribute: "wisdom"},
	}
	semDevocao := ComputeItemEffects(world.ActiveItemsFor(
		Character{Level: 1, Expertises: pericias}))

	comPoder := func(poder string) ItemEffects {
		return ComputeItemEffects(world.ActiveItemsFor(
			Character{Level: 1, GodPower: poder, Expertises: pericias}))
	}
	subiu := func(efeitos ItemEffects, alvo ModifierTarget) int {
		return StatFor(efeitos, alvo).Total - StatFor(semDevocao, alvo).Total
	}

	escamas := comPoder("Escamas Dracônicas")
	for _, caso := range []struct {
		alvo  ModifierTarget
		nome  string
		quero int
	}{
		{ModifierTarget{K: "defense"}, "Defesa", 2},
		{ModifierTarget{K: "expertise", Name: "Fortitude"}, "Fortitude", 2},
	} {
		if got := subiu(escamas, caso.alvo); got != caso.quero {
			t.Errorf("Escamas Dracônicas subiu %s em %d, e a p133 dá +%d "+
				"(\"Você recebe +2 na Defesa e em Fortitude\")", caso.nome, got, caso.quero)
		}
	}

	astucia := comPoder("Astúcia da Serpente")
	for _, pericia := range []string{"Enganação", "Furtividade", "Intuição"} {
		if got := subiu(astucia, ModifierTarget{K: "expertise", Name: pericia}); got != 2 {
			t.Errorf("Astúcia da Serpente subiu %s em %d, e a p132 dá +2 "+
				"(\"Você recebe +2 em Enganação, Furtividade e Intuição\")", pericia, got)
		}
	}

	// O CONTROLE, ao contrário: um poder concedido que o livro NÃO dá a este
	// personagem não pode aparecer. Se `ActiveItemsFor` passasse a somar todos
	// os poderes do catálogo, os dois casos acima ficariam verdes pelo motivo
	// errado.
	if got := subiu(escamas, ModifierTarget{K: "expertise", Name: "Enganação"}); got != 0 {
		t.Errorf("Escamas Dracônicas subiu Enganação em %d, e ela não dá perícia "+
			"nenhuma além de Fortitude — o motor está somando poder que o "+
			"personagem não tem", got)
	}
}
