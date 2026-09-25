package engine

import (
	"path/filepath"
	"testing"
)

// O DESLOCAMENTO É CONTADO EM QUADRADOS DE 1,5m (ALE-390).
//
// O livro mede deslocamento em metros e joga num grid de 1,5m — "para
// simplificar, você pode se referir a distâncias em 'quadrados' (de 1,5m)"
// (p236) —, e TODO valor que ele imprime é múltiplo de 1,5.
//
// Guardar em metros obrigava a arredondar na fronteira do JSON, e meio metro não
// sobrevivia a isso.
func TestTheReinforcedBootsGiveOneSquareAndNotTwoMetres(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	item := book.Item("botas-reforcadas")
	if item == nil || len(item.Modifiers) != 1 {
		t.Fatalf("o verbete das botas mudou de forma; este caso partia de 1 modificador: %+v", item)
	}

	// A p159 dá +1,5m, que é UM quadrado. Antes desta fatia o motor guardava 2 —
	// meio metro de bônus que o livro não dá, inventado pelo `math.Round`.
	if got := item.Modifiers[0].Amount; got != 1 {
		t.Errorf("as botas reforçadas valem %d quadrados e o livro dá +1,5m, que é 1 (p159).\n"+
			"2 quer dizer que o valor foi arredondado como se metros fossem a unidade do motor.", got)
	}
}

// A ARMADURA TIRA DOIS QUADRADOS, e é a mesma conversão vista do outro lado.
//
// O caso existe para o de cima não passar sozinho com uma conversão que só
// acerta o 1,5: −3m tem de virar −2, e não −3 nem −2,5 arredondado.
func TestArmourCostsTwoSquares(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)

	for _, id := range []string{"armadura-completa", "cota-malha", "brunea"} {
		item := book.Item(id)
		if item == nil {
			t.Fatalf("o verbete %q sumiu do catálogo", id)
		}
		achou := false
		for _, m := range item.Modifiers {
			if m.Target.K != "displacement" {
				continue
			}
			achou = true
			if m.Amount != -2 {
				t.Errorf("%s tira %d quadrados e o livro tira 3m, que são 2", id, m.Amount)
			}
		}
		if !achou {
			t.Errorf("%s deixou de ter modificador de deslocamento — o caso mediria o vazio", id)
		}
	}
}

// O DESLOCAMENTO DA RAÇA também é convertido, senão a base e os modificadores
// estariam em unidades diferentes — que é a pior forma de errar uma conta.
func TestTheRaceBaseIsInSquaresToo(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	world := BookRuleset(primeFromDump(t, dir))

	// O humano anda 9m (6 quadrados); o anão, 6m (4) — "é 6m EM VEZ DE 9m" (p20).
	casos := map[string]int{"Humano": 6, "Anão": 4}
	for raca, quadrados := range casos {
		ch := Character{Races: []CharacterRace{{Race: raca}}}
		if got := world.raceDisplacement(ch); got != quadrados {
			t.Errorf("%s anda %d quadrados e esperava %d", raca, got, quadrados)
		}
	}
	// Sem raça, o padrão do livro: 9m = 6 quadrados.
	if got := world.raceDisplacement(Character{}); got != 6 {
		t.Errorf("o padrão do livro deu %d quadrados e 9m são 6", got)
	}
}
