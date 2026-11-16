package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

// TODA ARMA PREFERIDA DO PANTEÃO É UMA ARMA DO CATÁLOGO (ALE-652): o nome do verbete do deus acha exatamente
// um item com arma. Um nome que não achasse calaria a Arma Sagrada para os devotos daquele deus.
func TestEveryFavoredWeaponOfThePantheonIsOneCatalogWeapon(t *testing.T) {
	gods, err := resourceJSON("gods")
	if err != nil {
		t.Fatal(err)
	}
	items, err := resourceJSON("items")
	if err != nil {
		t.Fatal(err)
	}
	byGod, err := favoredWeapons(gods, items)
	if err != nil {
		t.Fatalf("o panteão tem arma preferida fora do catálogo: %v", err)
	}
	for god, want := range map[string]string{"khalmyr": "espada-longa", "lin-wu": "katana", "valkaria": "mangual", "aharadak": "corrente-espinhos"} {
		if byGod[god] != want {
			t.Errorf("a arma preferida de %s saiu %q, want %q", god, byGod[god], want)
		}
	}
	// Nimb é "Nenhuma e todas!" (p102), e Lena e Marah não têm: os três ficam com o mestre, fora do mapa.
	for _, god := range []string{"nimb", "lena", "marah"} {
		if weapon, found := byGod[god]; found {
			t.Errorf("%s ganhou a arma preferida %q, e o livro não dá uma só", god, weapon)
		}
	}
}

func TestAFavoredWeaponOutsideTheCatalogNamesTheGod(t *testing.T) {
	gods := json.RawMessage(`[{"id":"khalmyr","armaPreferida":"Espada bastarda de vidro"}]`)
	items := json.RawMessage(`[{"id":"espada-longa","name":"Espada longa","weapon":{}}]`)
	_, err := favoredWeapons(gods, items)
	if err == nil || !strings.Contains(err.Error(), "khalmyr") || !strings.Contains(err.Error(), "Espada bastarda de vidro") {
		t.Errorf("a arma que não existe deu %v; want o erro com o deus e o nome", err)
	}
}
