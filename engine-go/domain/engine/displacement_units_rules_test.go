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

// O LENTO CORTA O DESLOCAMENTO PELA METADE, E O IMÓVEL O ZERA (ALE-390).
//
//	"LENTO. Todas as formas de deslocamento do personagem são reduzidas à metade
//	(arredonde para baixo para o primeiro incremento de 1,5m)" — p395
//	"IMÓVEL. Todas as formas de deslocamento do personagem são reduzidas a 0m."
//	— p394
//
// Nenhuma das duas é uma PARCELA, e é por isso que elas não existiam: o motor
// só somava. Em quadrados a metade é divisão inteira, que É o arredondamento
// que a condição pede.
func TestSlowHalvesTheDisplacementAndImmobileZeroesIt(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	world := BookRuleset(primeFromDump(t, dir))

	andar := func(condicoes string) int {
		ch := Character{Races: []CharacterRace{{Race: "Humano"}}, ActiveConditions: condicoes}
		return world.ComputeSheet(ch, nil).Displacement.Total
	}

	// O humano anda 9m — 6 quadrados.
	if base := andar("[]"); base != 6 {
		t.Fatalf("o controle já estava errado: o humano anda %d quadrados e o livro dá 9m (6)", base)
	}
	if got := andar(`["lento"]`); got != 3 {
		t.Errorf("lento deu %d quadrados e a metade de 6 é 3 (4,5m, p395)", got)
	}
	if got := andar(`["imovel"]`); got != 0 {
		t.Errorf("imóvel deu %d quadrados e o livro manda 0m (p394)", got)
	}

	// OS DOIS JUNTOS valem o MAIS SEVERO, e não compõem: "aplique apenas o mais
	// severo" (p394). Compondo daria 1 quadrado, que o livro não imprime.
	if got := andar(`["lento","imovel"]`); got != 0 {
		t.Errorf("lento + imóvel deu %d quadrados e o mais severo é 0 (p394)", got)
	}
}

// O CEGO E O EXAUSTO FICAM LENTOS, e o livro diz isso com todas as letras.
//
//	"CEGO. O personagem fica desprevenido e LENTO […]" — p394
//	"EXAUSTO. O personagem fica debilitado, LENTO e vulnerável." — p395
//
// Os comentários da tabela de condições já citavam as duas frases; o que faltava
// era poder escrever a lentidão. Um personagem cego andava a velocidade cheia.
func TestBlindAndExhaustedMoveAtHalfSpeed(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	world := BookRuleset(primeFromDump(t, dir))

	andar := func(condicao string) int {
		ch := Character{
			Races:            []CharacterRace{{Race: "Humano"}},
			ActiveConditions: `["` + condicao + `"]`,
		}
		return world.ComputeSheet(ch, nil).Displacement.Total
	}
	for _, caso := range []struct {
		condicao string
		pagina   string
	}{
		{"cego", "p394"},
		{"exausto", "p395"},
	} {
		if got := andar(caso.condicao); got != 3 {
			t.Errorf("%s: %d quadrados, esperava 3 — o livro diz que ele fica LENTO (%s), "+
				"e a metade de 6 é 3", caso.condicao, got, caso.pagina)
		}
	}
}

// A ISENÇÃO DO ANÃO NÃO ALCANÇA O LENTO, e isso é o texto do verbete.
//
// "Devagar e Sempre" isenta de armadura e CARGA (p20) — nada mais. O cabeçalho
// do `displacementBreakdown` já dizia: "o que ela NÃO isenta é qualquer outra
// redução: uma magia de lentidão continua valendo".
func TestTheDwarfExemptionDoesNotCoverBeingSlowed(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	world := BookRuleset(primeFromDump(t, dir))

	ch := Character{Races: []CharacterRace{{Race: "Anão"}}}
	if base := world.ComputeSheet(ch, nil).Displacement.Total; base != 4 {
		t.Fatalf("o controle já estava errado: o anão anda %d quadrados e o livro dá 6m (4)", base)
	}
	ch.ActiveConditions = `["lento"]`
	if got := world.ComputeSheet(ch, nil).Displacement.Total; got != 2 {
		t.Errorf("anão lento deu %d quadrados e esperava 2 — a isenção da p20 é de armadura e "+
			"carga, e não de lentidão", got)
	}
}
