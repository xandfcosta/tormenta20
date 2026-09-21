package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"t20engine/domain/catalog"
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

// p140. O kit é um só; o que a classe muda são a arma marcial, a brunea e o escudo.
func TestTheStartingKitFollowsTheProficiencies(t *testing.T) {
	cases := []struct {
		name          string
		class         string
		proficiencies []string
		martial       bool
		armors        []string
		shield        string
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kit := StartingKitFor(tc.class, tc.proficiencies)
			if kit.MartialWeapon != tc.martial {
				t.Errorf("arma marcial: %v, esperado %v", kit.MartialWeapon, tc.martial)
			}
			if !reflect.DeepEqual(kit.Armors, tc.armors) {
				t.Errorf("armaduras: %v, esperado %v", kit.Armors, tc.armors)
			}
			if kit.Shield != tc.shield {
				t.Errorf("escudo: %q, esperado %q", kit.Shield, tc.shield)
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

// p140: "Se você tiver proficiência com armaduras pesadas, em vez disso PODE
// começar com uma brunea" — quem usa pesadas escolhe entre QUATRO. Trocar a
// lista pela brunea deixaria o guerreiro que quer gibão de peles sem como pedir.
func TestTheBruneaIsAChoiceAndNotASwap(t *testing.T) {
	kit := StartingKitFor("Guerreiro", guerreiro)
	if len(kit.Armors) != 4 {
		t.Fatalf("%d armaduras à escolha, esperado 4: %v", len(kit.Armors), kit.Armors)
	}
	if kit.Armors[len(kit.Armors)-1] != "brunea" {
		t.Errorf("a brunea entra ao lado das leves, não no lugar delas: %v", kit.Armors)
	}
}

// Guarda de varredura: o kit concede
// por ID, e um ID que não casa não estoura — ele concede NADA, em silêncio, e o
// herói nasce sem mochila sem ninguém ver.
func TestEveryStartingKitItemExistsInTheCatalog(t *testing.T) {
	raw, ok := catalog.Resource("items")
	if !ok {
		t.Fatal("catálogo de itens ausente")
	}
	var items []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("itens: %v", err)
	}
	inCatalog := make(map[string]bool, len(items))
	for _, item := range items {
		inCatalog[item.ID] = true
	}

	// O denominador: uma lista de ausentes vazia e um catálogo que não carregou
	// se parecem no terminal.
	granted := append([]string{}, startingKitBaseItems...)
	granted = append(granted, startingLightArmors...)
	granted = append(granted, startingHeavyArmor, startingShield)
	if len(granted) != 8 {
		t.Fatalf("%d itens conferidos, esperado 8", len(granted))
	}
	for _, id := range granted {
		if !inCatalog[id] {
			t.Errorf("o kit concede %q, que não existe em items.json", id)
		}
	}
}
