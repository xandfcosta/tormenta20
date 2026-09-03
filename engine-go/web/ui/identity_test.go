package ui

import (
	"strings"
	"testing"
)

// Os valores de referência foram MEDIDOS rodando o JS da SPA, não deduzidos:
//
//	node -e "function h(n){let x=0;for(const c of n)x=(x*31+c.charCodeAt(0))>>>0;return x%360}
//	         console.log(h('Sombras de Valkaria'))"   // 181
//
// Sem isso o port ficaria "parecido", e parecido aqui significa a mesma
// campanha com duas capas diferentes nas duas telas.
func TestTheNameHueMatchesTheJs(t *testing.T) {
	casos := map[string]int{
		"Sombras de Valkaria": 181,
		"Anão":                153,
		"Thal, o Errante":     71,
		"A":                   65,
		"Tormenta 20":         194,
		"Mesa do Mestre":      351,
	}
	for nome, quer := range casos {
		if got := NameHue(nome); got != quer {
			t.Errorf("NameHue(%q) = %d, o JS dá %d", nome, got, quer)
		}
	}
}

func TestTheInitialsMatchTheJs(t *testing.T) {
	casos := map[string]string{
		"Sombras de Valkaria": "SD",
		"Anão":                "A",
		"Thal, o Errante":     "TO",
		"A":                   "A",
		"Tormenta 20":         "T2",
		"Mesa do Mestre":      "MD",
		"   ":                 "?",
		"":                    "?",
	}
	for nome, quer := range casos {
		if got := Monogram(nome); got != quer {
			t.Errorf("Monogram(%q) = %q, queria %q", nome, got, quer)
		}
	}
}

// O gradiente usa o MESMO matiz nas três paradas — se ele divergisse, a capa
// sairia com um degradê de duas cores diferentes.
func TestTheGradientUsesTheSameHueAtAllThreeStops(t *testing.T) {
	g := NameGradient("Sombras de Valkaria")
	if strings.Count(g, "181") != 3 {
		t.Errorf("o gradiente não repetiu o matiz nas três paradas: %s", g)
	}
}
