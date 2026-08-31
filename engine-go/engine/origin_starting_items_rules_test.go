package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"t20engine/catalog"
)

// TestAsConcessoesDeOrigemSaoClassificadas — as frases são as do livro (p85–95),
// copiadas de `origens.json` na mão.
func TestAsConcessoesDeOrigemSaoClassificadas(t *testing.T) {
	casos := []struct {
		frase    string
		kind     OriginItemKind
		conferir func(*testing.T, OriginItemGrant)
	}{
		{"Símbolo sagrado", OriginItemFixed, func(t *testing.T, g OriginItemGrant) {
			if g.Name != "Símbolo sagrado" {
				t.Errorf("nome: %q", g.Name)
			}
		}},
		{"Arma marcial", OriginItemWeapon, func(t *testing.T, g OriginItemGrant) {
			if !reflect.DeepEqual(g.Categories, []string{"weapon-martial"}) {
				t.Errorf("categorias: %v", g.Categories)
			}
		}},
		{
			// O "ou" MINÚSCULO é texto corrido: são duas categorias de arma, e
			// não duas alternativas escritas.
			"Arma marcial ou exótica", OriginItemWeapon, func(t *testing.T, g OriginItemGrant) {
				if !reflect.DeepEqual(g.Categories, []string{"weapon-martial", "weapon-exotic"}) {
					t.Errorf("categorias: %v", g.Categories)
				}
			},
		},
		{"Estojo de disfarces OU gazua", OriginItemOneOf, func(t *testing.T, g OriginItemGrant) {
			if !reflect.DeepEqual(g.Options, []string{"Estojo de disfarces", "gazua"}) {
				t.Errorf("alternativas: %v", g.Options)
			}
		}},
		{"Um item estrangeiro (até T$ 100)", OriginItemAny, func(t *testing.T, g OriginItemGrant) {
			if g.MaxPrice != 100 {
				t.Errorf("teto: %d", g.MaxPrice)
			}
		}},
		{"Itens variados (até T$ 500, aprovados pelo mestre)", OriginItemAny, func(t *testing.T, g OriginItemGrant) {
			if g.MaxPrice != 500 {
				t.Errorf("teto: %d", g.MaxPrice)
			}
		}},
		{"T$ 2d6 (último salário)", OriginItemMoney, func(t *testing.T, g OriginItemGrant) {
			if g.Dice != "2d6" {
				t.Errorf("dado: %q", g.Dice)
			}
		}},
		{
			"Cão de caça, cavalo, pônei ou trobo (escolha)", OriginItemOneOf,
			func(t *testing.T, g OriginItemGrant) {
				esperado := []string{"Cão de caça", "Cavalo", "Pônei", "Trobo"}
				if !reflect.DeepEqual(g.Options, esperado) {
					t.Errorf("alternativas: %v, esperado %v", g.Options, esperado)
				}
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.frase, func(t *testing.T) {
			concessao := ParseOriginItem(caso.frase)
			if concessao.Kind != caso.kind {
				t.Fatalf("tipo %q, esperado %q", concessao.Kind, caso.kind)
			}
			if concessao.Label != caso.frase {
				t.Errorf("o rótulo perdeu a frase do livro: %q", concessao.Label)
			}
			caso.conferir(t, concessao)
		})
	}
}

// TestNenhumaConcessaoDeOrigemComEscolhaNasceFixa é guarda de varredura sobre o
// catálogo inteiro.
//
// O tipo `fixed` é o que vira ITEM na Mochila. Uma frase de escolha classificada
// como fixa não estoura — ela vira uma linha chamada "Arma marcial" ocupando
// carga, e é o tipo de defeito que só aparece quando um jogador olha a mochila
// dele. Como o classificador é por FORMA, uma origem nova com uma forma nova
// falha aqui em vez de nascer errada.
func TestNenhumaConcessaoDeOrigemComEscolhaNasceFixa(t *testing.T) {
	bruto, ok := catalog.Resource("origens")
	if !ok {
		t.Fatal("catálogo de origens ausente")
	}
	// O catálogo de origens é um OBJETO indexado por id, e não uma lista como o
	// de itens e o de perícias.
	var origens map[string]struct {
		Name          string   `json:"name"`
		ItensIniciais []string `json:"itensIniciais"`
	}
	if err := json.Unmarshal(bruto, &origens); err != nil {
		t.Fatalf("origens: %v", err)
	}

	marcas := []string{" OU ", "até T$", "(escolha)"}
	medidos, escolhas := 0, 0
	for _, origem := range origens {
		for _, frase := range origem.ItensIniciais {
			medidos++
			concessao := ParseOriginItem(frase)
			if concessao.Kind != OriginItemFixed {
				escolhas++
				continue
			}
			for _, marca := range marcas {
				if strings.Contains(frase, marca) {
					t.Errorf("%s: %q nasceu fixa e carrega %q", origem.Name, frase, marca)
				}
			}
			if originMoneyDice.MatchString(frase) {
				t.Errorf("%s: %q nasceu fixa e é dinheiro", origem.Name, frase)
			}
		}
	}
	// O DENOMINADOR: uma lista de reprovados vazia e um catálogo que não casou
	// com o `json` se parecem no terminal. Hoje são 71 frases, 14 delas
	// escolha ou dinheiro; o piso é frouxo de propósito, porque o catálogo
	// cresce e o que este guarda protege é a FORMA, não a contagem.
	if medidos < 60 || escolhas < 10 {
		t.Fatalf("mediu %d frases (%d de escolha) — o catálogo não chegou inteiro", medidos, escolhas)
	}
}
