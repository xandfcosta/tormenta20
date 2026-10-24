package engine

import (
	"testing"
)

// A ISENÇÃO DE DESLOCAMENTO TEM DUAS METADES, e nem todo mundo compra as duas
// (ALE-430).
//
//	"Fanático. Seu deslocamento não é reduzido por usar armaduras pesadas."
//	(p128)
//
//	"Devagar e Sempre. Seu deslocamento é 6m (em vez de 9m). Porém, seu
//	 deslocamento não é reduzido por uso de armadura ou excesso de carga."
//	(p20)
//
// O anão compra as duas; o Fanático compra só a ARMADURA. Enquanto houve uma
// flag só — `displacement-ignores-armor-and-load` —, dar a isenção ao Fanático
// era dar junto a da carga, que o livro não dá a ele.
//
// O deslocamento é contado em QUADRADOS de 1,5m: a armadura pesada tira 3m, que
// são dois quadrados.
func TestTheFanaticIgnoresHeavyArmorAndNotTheLoad(t *testing.T) {
	world := BookRuleset(primeBook(t))
	vested := "vested"
	completa := "armadura-completa"
	pedras := "pedra-funda"
	guerreiro := func(powers string, items ...CharacterItem) Character {
		return Character{
			Level: 12, Strength: 2, ClassPowers: powers,
			Races:   []CharacterRace{{Race: "humano"}},
			Classes: []CharacterClass{{ClassName: "Guerreiro", Level: 12}},
			Items:   items,
		}
	}
	armadura := CharacterItem{CatalogID: &completa, Quantity: 1, Slots: 1, Equipped: &vested}
	// CINQUENTA PEDRAS passam de qualquer limite de espaços: a sobrecarga é a
	// outra metade da isenção, e ela precisa estar no ar para o caso dizer algo.
	carga := CharacterItem{CatalogID: &pedras, Quantity: 50, Slots: 1}
	deslocamentoDe := func(ch Character) int {
		return world.ComputeSheet(ch, map[string]bool{}).Displacement.Total
	}

	semPoder := deslocamentoDe(guerreiro(`[]`, armadura))
	if semPoder != 4 {
		t.Fatalf("o guerreiro de armadura completa anda %d quadrados, e a armadura "+
			"pesada tira 3m de 9m: 4 quadrados. A bancada mediu outra coisa", semPoder)
	}
	if comPoder := deslocamentoDe(guerreiro(`["fanatico"]`, armadura)); comPoder != 6 {
		t.Errorf("o Fanático anda %d quadrados de armadura completa, e a p128 diz "+
			"que o deslocamento dele não é reduzido por ela: 6", comPoder)
	}

	// A CARGA CONTINUA PESANDO: é a metade que o Fanático não compra, e é ela que
	// separa a isenção dele da do anão.
	if sobrecarregado := deslocamentoDe(guerreiro(`["fanatico"]`, armadura, carga)); sobrecarregado != 4 {
		t.Errorf("o Fanático sobrecarregado anda %d quadrados, e a p128 só o isenta "+
			"da ARMADURA — a carga continua tirando 3m (p141): 4", sobrecarregado)
	}

	// O ANÃO NÃO REGRIDE, e ele é quem prova que a outra metade sobreviveu ao
	// corte: 6m de base, armadura pesada e sobrecarga, e nada o alcança (p20).
	anao := guerreiro(`[]`, armadura, carga)
	anao.Races = []CharacterRace{{Race: "anao"}}
	if got := deslocamentoDe(anao); got != 4 {
		t.Errorf("o anão de armadura completa e sobrecarregado anda %d quadrados, "+
			"e o Devagar e Sempre isenta as duas coisas: 4 (os 6m dele)", got)
	}
}
