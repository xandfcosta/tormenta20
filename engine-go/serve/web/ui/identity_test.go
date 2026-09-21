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
	cases := map[string]int{
		"Sombras de Valkaria": 181,
		"Anão":                153,
		"Thal, o Errante":     71,
		"A":                   65,
		"Tormenta 20":         194,
		"Mesa do Mestre":      351,
	}
	for name, want := range cases {
		if got := NameHue(name); got != want {
			t.Errorf("NameHue(%q) = %d, o JS dá %d", name, got, want)
		}
	}
}

func TestTheInitialsMatchTheJs(t *testing.T) {
	cases := map[string]string{
		"Sombras de Valkaria": "SD",
		"Anão":                "A",
		"Thal, o Errante":     "TO",
		"A":                   "A",
		"Tormenta 20":         "T2",
		"Mesa do Mestre":      "MD",
		"   ":                 "?",
		"":                    "?",
	}
	for name, want := range cases {
		if got := Monogram(name); got != want {
			t.Errorf("Monogram(%q) = %q, queria %q", name, got, want)
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
