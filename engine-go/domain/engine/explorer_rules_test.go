package engine

import "testing"

// p50 Explorador: "Quando estiver no tipo de terreno escolhido, você soma sua
// Sabedoria (mínimo +1) na Defesa e nos testes de Acrobacia, Atletismo,
// Furtividade, Percepção e Sobrevivência. […] escolha outro tipo de terreno
// para receber o bônus ou aumente o bônus em um tipo de terreno já escolhido
// em +2." Cada terreno é UM interruptor (ALE-503).

func explorerSheet(wisdom int, picks string) Character {
	return Character{
		Wisdom:       wisdom,
		Classes:      []CharacterClass{{ClassName: "Caçador", Level: 7}},
		ClassPowers:  "[]",
		PowerChoices: `{"class.cacador.explorador":` + picks + `}`,
	}
}

func TestTheExplorerAddsWisdomInTheChosenTerrain(t *testing.T) {
	r := BookRuleset(primeBook(t))
	ch := explorerSheet(3, `["floresta","floresta","urbano"]`)

	// O que se mede é a DIFERENÇA que o interruptor faz: outras habilidades do
	// caçador já somam em algumas destas perícias.
	off := r.sheetEffects(ch, nil)
	gain := func(on string, target ModifierTarget) int {
		return StatFor(r.sheetEffects(ch, map[string]bool{FlagGroupID(on): true}), target).Total - StatFor(off, target).Total
	}
	for _, target := range []ModifierTarget{{K: "defense"}, {K: "expertise", Name: "Furtividade"}, {K: "expertise", Name: "Sobrevivência"}} {
		if got := gain("explorador:floresta", target); got != 5 {
			t.Errorf("na floresta escolhida duas vezes, %v ganha +%d, esperava +5 (Sab 3 + 2)", target, got)
		}
	}
	if got := gain("explorador:urbano", ModifierTarget{K: "expertise", Name: "Percepção"}); got != 3 {
		t.Errorf("no urbano escolhido uma vez, Percepção ganha +%d, esperava +3", got)
	}
	if got := gain("explorador:urbano", ModifierTarget{K: "expertise", Name: "Luta"}); got != 0 {
		t.Errorf("Luta não é perícia do Explorador e ganhou +%d", got)
	}

	// "mínimo +1": Sabedoria −1 ainda soma 1.
	ch = explorerSheet(-1, `["floresta"]`)
	off = r.sheetEffects(ch, nil)
	if got := gain("explorador:floresta", ModifierTarget{K: "defense"}); got != 1 {
		t.Errorf("Sabedoria −1 na floresta dá %+d na Defesa, esperava +1 (mínimo)", got)
	}
}
