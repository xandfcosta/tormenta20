package catalog

import "testing"

// TODA ARMADURA LEVANTA A FLAG DO PESO DELA (p69, p125, ALE-443).
//
// A flag é o que a CONDIÇÃO de um poder consulta: o Encouraçado exige
// `armadura-pesada` (p125) e o Couraceiro exige `armadura-leve` (p69) para
// trocar a Destreza pela Inteligência na Defesa.
//
// Uma armadura transcrita sem ela não dá erro em lugar nenhum: ela veste, soma
// a Defesa dela, e os poderes que dependiam do peso simplesmente não acendem —
// em silêncio, e com a ficha inteira plausível. É por isso que a convenção vira
// guarda em vez de parágrafo.
//
// O denominador sai da CATEGORIA, e não de uma lista escrita aqui: armadura
// nova nasce medida.
func TestEveryArmorRaisesTheFlagOfItsWeightClass(t *testing.T) {
	type armorEntry struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Armor *struct {
			Heavy bool `json:"heavy"`
		} `json:"armor"`
		Modifiers []struct {
			Target struct {
				K    string `json:"k"`
				Name string `json:"name"`
			} `json:"target"`
		} `json:"modifiers"`
		// ComposedOf é o item específico (p340), que não declara peso nem
		// bandeira: ele É uma armadura base com sobreposições, e as duas coisas
		// se herdam. Aqui a pergunta se transfere para a BASE em vez de o item
		// sair da conta — sair seria um buraco calado na varredura.
		ComposedOf *struct {
			Base string `json:"base"`
		} `json:"composedOf"`
	}

	bases := map[string]armorEntry{}
	for _, category := range []string{"armor-light", "armor-heavy"} {
		for _, armor := range itemsOfCategory[armorEntry](t, category) {
			bases[armor.ID] = armor
		}
	}

	for _, caso := range []struct {
		category string
		flag     string
		heavy    bool
	}{
		{category: "armor-light", flag: "armadura-leve", heavy: false},
		{category: "armor-heavy", flag: "armadura-pesada", heavy: true},
	} {
		medidas := 0
		for _, armor := range itemsOfCategory[armorEntry](t, caso.category) {
			medidas++
			if armor.ComposedOf != nil {
				base, achada := bases[armor.ComposedOf.Base]
				if !achada {
					t.Errorf("%s é composta sobre %q, que não é armadura de categoria "+
						"nenhuma", armor.Name, armor.ComposedOf.Base)
					continue
				}
				armor = base
			}
			// O PESO declarado no verbete tem de concordar com a categoria,
			// senão as duas metades deste guarda mediriam listas diferentes.
			if armor.Armor == nil || armor.Armor.Heavy != caso.heavy {
				t.Errorf("%s está em %q e o `armor.heavy` dela não diz %v",
					armor.Name, caso.category, caso.heavy)
			}
			levanta := false
			for _, m := range armor.Modifiers {
				if m.Target.K == "flag" && m.Target.Name == caso.flag {
					levanta = true
				}
			}
			if !levanta {
				t.Errorf("a armadura %q (%s) não levanta a flag %q.\n"+
					"Sem ela, todo poder que exige esse peso deixa de valer para "+
					"quem a veste — e nada reclama: a Defesa dela continua certa.",
					armor.Name, armor.ID, caso.flag)
			}
		}
		t.Logf("%s: %d armaduras medidas contra a flag %q", caso.category, medidas, caso.flag)
	}
}
