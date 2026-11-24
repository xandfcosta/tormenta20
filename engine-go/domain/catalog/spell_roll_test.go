package catalog

import (
	"fmt"
	"regexp"
	"testing"
)

// spellForRollSweep é o que a varredura do dado da magia precisa: o efeito, o dado declarado e os
// aprimoramentos.
type spellForRollSweep struct {
	Name       string          `json:"name"`
	BaseEffect string          `json:"baseEffect"`
	Rolls      []spellRollSeen `json:"rolls"`
	Augments   []struct {
		Description    string          `json:"description"`
		Adds           []spellRollSeen `json:"adds"`
		Rolls          []spellRollSeen `json:"rolls"`
		ChangesTheRoll bool            `json:"changesTheRoll"`
	} `json:"augments"`
}

type spellRollSeen struct {
	Dice string `json:"dice"`
	Kind string `json:"kind"`
}

var (
	diceInText = regexp.MustCompile(`\d+d\d+`)
	diceShape  = regexp.MustCompile(`^\d+d\d+(\+\d+)?$`)
)

// noCastingRoll são as magias com dado na prosa que NÃO rolam ao ser conjuradas, e por quê (ver GLOSSARY, **dado
// da magia**). A lista é de exceções com motivo: magia nova com dado reprova até alguém decidir de que lado ela
// fica.
var noCastingRoll = map[string]string{
	"amedrontar":                      "o dado é de duração",
	"sono":                            "o dado é de duração",
	"dissipar-magia":                  "o dado é de duração",
	"anular-a-luz":                    "o dado é de duração",
	"dispersar-as-trevas":             "o dado é de duração",
	"explosao-caleidoscopica":         "o dado é de duração",
	"banimento":                       "o dado é da execução e da duração",
	"augurio":                         "o dado é do mestre, em segredo",
	"comunhao-com-a-natureza":         "o dado é de auxílio, gasto nos testes",
	"contato-extraplanar":             "o dado é de auxílio, gasto nos testes",
	"teletransporte":                  "o dado é da distância do erro",
	"viagem-planar":                   "o dado é da distância da chegada",
	"arma-espiritual":                 "o dano é da reação, depois",
	"conjurar-mortos-vivos":           "o dano é do ataque dos esqueletos, depois",
	"runa-de-protecao":                "o dano é da ativação da runa, depois",
	"tentaculos-de-trevas":            "o dano é só em quem já está agarrado, depois da manobra",
	"viagem-arborea":                  "o dano é da árvore destruída, depois",
	"assassino-fantasmagorico":        "o dano é do espectro, depois",
	"controlar-a-gravidade":           "o dano é da queda, depois",
	"forma-eterea":                    "o dano é de materializar em espaço ocupado, depois",
	"mao-poderosa-de-talude":          "o dano é do esmagar, uma ação depois",
	"relampago-flamejante-de-reynard": "o dano é das rajadas, uma ação depois",
	"furia-do-panteao":                "o dano é da ação escolhida a cada turno",
	"enxame-de-pestes":                "o dano é no fim do turno",
	"enxame-rubro-de-ichabod":         "o dano é no fim do turno",
	"circulo-da-restauracao":          "a cura é no fim do turno de quem está no círculo",
	"manto-do-cruzado":                "a cura e o dano são no início do turno",
	"criar-elementos":                 "o dano depende do elemento escolhido",
	"controlar-fogo":                  "o dano depende do efeito escolhido",
	"controlar-agua":                  "o dano depende do efeito escolhido",
	"controlar-terra":                 "o dano depende do efeito escolhido",
	"muralha-elemental":               "o dano depende do elemento escolhido",
	"telecinesia":                     "o dano depende do que é arremessado",
	"terremoto":                       "o dano depende do lugar",
	"palavra-primordial":              "o dano depende da palavra escolhida e da resistência",
}

// augmentDiceNotTheCastRoll são os aprimoramentos com dado na prosa, de magias que rolam na conjuração, cujo
// dado não é o da conjuração.
var augmentDiceNotTheCastRoll = map[string]string{
	"raio-solar[3]": "o dado é da duração da cegueira",
}

// TestEverySpellSaysWhatItRollsWhenCast amarra o dado da magia à prosa do efeito: toda magia com dado no texto
// declara o que rola ao ser conjurada, ou está em `noCastingRoll` com o motivo. Sem a varredura, uma magia
// transcrita sem o dado se conjuraria sem rolar nada, e nada reprovaria.
func TestEverySpellSaysWhatItRollsWhenCast(t *testing.T) {
	spells := decodeResource[map[string]spellForRollSweep](t, "spells")
	if len(spells) == 0 {
		t.Fatal("nenhuma magia no catálogo: não há o que varrer, e verde aqui não valeria nada")
	}
	rolling := 0
	for id, m := range spells {
		written := diceInText.MatchString(m.BaseEffect)
		reason, excused := noCastingRoll[id]
		switch {
		case len(m.Rolls) > 0 && excused:
			t.Errorf("%s (%s) declara o dado e está em `noCastingRoll` (%q): tire-a da lista", m.Name, id, reason)
		case len(m.Rolls) > 0 && !written:
			t.Errorf("%s (%s) declara o dado, e o efeito não tem dado nenhum", m.Name, id)
		case len(m.Rolls) == 0 && written && !excused:
			t.Errorf("%s (%s) tem dado no efeito e não declara o que rola na conjuração: declare `Rolls`, ou ponha-a em `noCastingRoll` com o motivo", m.Name, id)
		case excused && !written:
			t.Errorf("%s (%s) está em `noCastingRoll` e o efeito não tem dado: tire-a da lista", m.Name, id)
		}
		if len(m.Rolls) > 0 {
			rolling++
		}
		for _, r := range m.Rolls {
			if !diceShape.MatchString(r.Dice) {
				t.Errorf("%s (%s): o dado %q não tem a forma NdM ou NdM+K", m.Name, id, r.Dice)
			}
		}
		for i, a := range m.Augments {
			key := fmt.Sprintf("%s[%d]", id, i)
			declares := len(a.Adds) > 0 || len(a.Rolls) > 0 || a.ChangesTheRoll
			reason, excused := augmentDiceNotTheCastRoll[key]
			switch {
			case len(m.Rolls) == 0 && declares:
				t.Errorf("%s (%s) aprimoramento %d mexe no dado de uma magia que não rola na conjuração", m.Name, id, i)
			case len(m.Rolls) > 0 && diceInText.MatchString(a.Description) && !declares && !excused:
				t.Errorf("%s (%s) aprimoramento %d tem dado e não diz o que faz com o da conjuração: declare `Adds`, `Rolls` ou `ChangesTheRoll`, ou ponha-o em `augmentDiceNotTheCastRoll` com o motivo: %q",
					m.Name, id, i, a.Description)
			case declares && excused:
				t.Errorf("%s está em `augmentDiceNotTheCastRoll` (%q) e mexe no dado: tire-o da lista", key, reason)
			}
			for _, r := range append(append([]spellRollSeen{}, a.Adds...), a.Rolls...) {
				if !diceShape.MatchString(r.Dice) {
					t.Errorf("%s (%s) aprimoramento %d: o dado %q não tem a forma NdM ou NdM+K", m.Name, id, i, r.Dice)
				}
			}
		}
	}
	for id := range noCastingRoll {
		if _, exists := spells[id]; !exists {
			t.Errorf("%s está em `noCastingRoll` e não é magia do catálogo: tire-a da lista", id)
		}
	}
	if rolling == 0 {
		t.Fatal("nenhuma magia declara o dado da conjuração: a varredura não mediu nada")
	}
}
