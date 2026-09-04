package sheet

import "testing"

// Tabela 4-1. Uma tabela de seis linhas que decide o custo de TODA magia do
// jogo — cara demais para ficar só implícita nos oráculos.
//
// Ela veio do `api/character_cast_rules_test.go` com a tabela que ela prende
// (ALE-278). Os outros cinco casos daquele arquivo montam um `Server` e ficaram
// lá: é a sexta vez nesta épica que a fronteira separa um arquivo de teste que
// misturava duas camadas.
func TestSpellBasePmCostTable(t *testing.T) {
	want := map[int]int{0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15}
	for circle, pm := range want {
		if got := SpellBasePmCost[circle]; got != pm {
			t.Errorf("círculo %d custa %d PM, want %d (Tabela 4-1, p170)", circle, got, pm)
		}
	}
	if len(SpellBasePmCost) != len(want) {
		t.Errorf("a tabela tem %d círculos, want %d", len(SpellBasePmCost), len(want))
	}
}

func TestTheReachableCircleRisesWithTheLevelAndHasAFloorInTheSpellItself(t *testing.T) {
	casos := []struct {
		nome    string
		classes []ClassDTO
		magia   int
		quer    int
	}{
		{"arcanista de 1º alcança o 1º", []ClassDTO{{ClassName: "Arcanista", Level: 1}}, 1, 1},
		{"arcanista de 4º ainda está no 1º", []ClassDTO{{ClassName: "Arcanista", Level: 4}}, 1, 1},
		{"arcanista de 5º abre o 2º", []ClassDTO{{ClassName: "Arcanista", Level: 5}}, 1, 2},
		{"arcanista de 17º chega ao 5º", []ClassDTO{{ClassName: "Arcanista", Level: 17}}, 1, 5},
		{"bardo de 20º para no 4º", []ClassDTO{{ClassName: "Bardo", Level: 20}}, 1, 4},
		{"guerreiro não conjura, e o piso é a magia", []ClassDTO{{ClassName: "Guerreiro", Level: 20}}, 2, 2},
		{"multiclasse fica com o MAIOR", []ClassDTO{
			{ClassName: "Guerreiro", Level: 10}, {ClassName: "Arcanista", Level: 9},
		}, 1, 3},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if got := HighestCastableCircle(caso.classes, caso.magia); got != caso.quer {
				t.Errorf("alcançou o %dº, quer o %dº", got, caso.quer)
			}
		})
	}
}
