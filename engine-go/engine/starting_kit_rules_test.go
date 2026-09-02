package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"t20engine/catalog"
)

// As proficiências como o catálogo as guarda (a linha "Proficiências." de cada
// bloco de classe, p36–83). Escritas aqui à mão de propósito: derivar o esperado
// do mesmo `classes.json` que a produção lê faria o teste concordar com um
// catálogo errado.
var (
	guerreiro = []string{"armas-marciais", "armaduras-pesadas", "escudos"}
	clerigo   = []string{"armaduras-pesadas", "escudos"}
	druida    = []string{"escudos"}
	bardo     = []string{"armas-marciais"}
	ladino    = []string{}
)

// TestTheStartingKitFollowsTheProficiencies — p140.
//
// O kit é um só; o que a classe muda são a arma marcial, a brunea e o escudo.
func TestTheStartingKitFollowsTheProficiencies(t *testing.T) {
	casos := []struct {
		nome          string
		classe        string
		proficiencias []string
		martial       bool
		armaduras     []string
		escudo        string
	}{
		{
			"guerreiro leva tudo o que o kit condiciona",
			"Guerreiro", guerreiro, true,
			[]string{"armadura-couro", "couro-batido", "gibao-peles", "brunea"},
			"escudo-leve",
		},
		{
			// A classe usa pesadas e escudos sem usar marciais: as três linhas
			// do kit são INDEPENDENTES, e este é o caso que prova.
			"clérigo tem brunea e escudo, e nenhuma arma marcial",
			"Clérigo", clerigo, false,
			[]string{"armadura-couro", "couro-batido", "gibao-peles", "brunea"},
			"escudo-leve",
		},
		{
			"druida usa escudos e não usa pesadas: escudo sim, brunea não",
			"Druida", druida, false,
			[]string{"armadura-couro", "couro-batido", "gibao-peles"},
			"escudo-leve",
		},
		{
			"bardo só ganha a arma marcial",
			"Bardo", bardo, true,
			[]string{"armadura-couro", "couro-batido", "gibao-peles"},
			"",
		},
		{
			// "Proficiências. Nenhuma." (p72) — e mesmo assim a armadura leve
			// vem, porque o kit a dá a todo mundo e não por proficiência.
			"ladino não é proficiente em nada e ainda escolhe armadura leve",
			"Ladino", ladino, false,
			[]string{"armadura-couro", "couro-batido", "gibao-peles"},
			"",
		},
		{
			// "Exceção: arcanistas começam sem armadura" (p140), e o livro
			// escreve a exceção pelo NOME da classe.
			"arcanista começa sem armadura nenhuma",
			"Arcanista", []string{}, false,
			nil,
			"",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			kit := StartingKitFor(caso.classe, caso.proficiencias)
			if kit.MartialWeapon != caso.martial {
				t.Errorf("arma marcial: %v, esperado %v", kit.MartialWeapon, caso.martial)
			}
			if !reflect.DeepEqual(kit.Armors, caso.armaduras) {
				t.Errorf("armaduras: %v, esperado %v", kit.Armors, caso.armaduras)
			}
			if kit.Shield != caso.escudo {
				t.Errorf("escudo: %q, esperado %q", kit.Shield, caso.escudo)
			}
			if !reflect.DeepEqual(kit.BaseItems, []string{"mochila", "saco-de-dormir", "traje-viajante"}) {
				t.Errorf("itens de base: %v", kit.BaseItems)
			}
			if kit.MoneyDice != "4d6" {
				t.Errorf("dado da bolsa: %q, esperado 4d6", kit.MoneyDice)
			}
		})
	}
}

// TestTheBruneaIsAChoiceAndNotASwap — p140, e é a divergência que o porte corrigiu.
//
// "Se você tiver proficiência com armaduras pesadas, em vez disso PODE começar
// com uma brunea": quem usa pesadas escolhe entre QUATRO. O
// `class-starting-kits.ts` devolvia `armor: 'brunea'` e apagava as três leves,
// de modo que um guerreiro que quisesse gibão de peles não tinha como pedir.
func TestTheBruneaIsAChoiceAndNotASwap(t *testing.T) {
	kit := StartingKitFor("Guerreiro", guerreiro)
	if len(kit.Armors) != 4 {
		t.Fatalf("%d armaduras à escolha, esperado 4: %v", len(kit.Armors), kit.Armors)
	}
	if kit.Armors[len(kit.Armors)-1] != "brunea" {
		t.Errorf("a brunea entra ao lado das leves, não no lugar delas: %v", kit.Armors)
	}
}

// TestEveryStartingKitItemExistsInTheCatalog é guarda de varredura: o kit concede
// por ID, e um ID que não casa não estoura — ele concede NADA, em silêncio, e o
// herói nasce sem mochila sem ninguém ver.
func TestEveryStartingKitItemExistsInTheCatalog(t *testing.T) {
	bruto, ok := catalog.Resource("items")
	if !ok {
		t.Fatal("catálogo de itens ausente")
	}
	var itens []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(bruto, &itens); err != nil {
		t.Fatalf("itens: %v", err)
	}
	noCatalogo := make(map[string]bool, len(itens))
	for _, item := range itens {
		noCatalogo[item.ID] = true
	}

	// O denominador: uma lista de ausentes vazia e um catálogo que não carregou
	// se parecem no terminal.
	concedidos := append([]string{}, startingKitBaseItems...)
	concedidos = append(concedidos, startingLightArmors...)
	concedidos = append(concedidos, startingHeavyArmor, startingShield)
	if len(concedidos) != 8 {
		t.Fatalf("%d itens conferidos, esperado 8", len(concedidos))
	}
	for _, id := range concedidos {
		if !noCatalogo[id] {
			t.Errorf("o kit concede %q, que não existe em items.json", id)
		}
	}
}
