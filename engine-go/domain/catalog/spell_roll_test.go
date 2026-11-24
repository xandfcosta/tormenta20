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
	Duration   string          `json:"duration"`
	BaseEffect string          `json:"baseEffect"`
	Rolls      []spellRollSeen `json:"rolls"`
	TurnRoll   *struct {
		At       string `json:"at"`
		Versions []struct {
			Name  string          `json:"name"`
			Rolls []spellRollSeen `json:"rolls"`
		} `json:"versions"`
	} `json:"turnRoll"`
	Augments []struct {
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
	"enxame-de-pestes":                turnRollOnly,
	"enxame-rubro-de-ichabod":         turnRollOnly,
	"circulo-da-restauracao":          "a cura é no fim do turno de quem está no círculo",
	"manto-do-cruzado":                turnRollOnly,
	"criar-elementos":                 "o dano depende do elemento escolhido",
	"controlar-fogo":                  "o dano depende do efeito escolhido",
	"controlar-agua":                  "o dano depende do efeito escolhido",
	"controlar-terra":                 "o dano depende do efeito escolhido",
	"muralha-elemental":               "o dano depende do elemento escolhido",
	"telecinesia":                     "o dano depende do que é arremessado",
	"terremoto":                       "o dano depende do lugar",
	"palavra-primordial":              "o dano depende da palavra escolhida e da resistência",
}

// turnRollOnly é o motivo da magia cujo dado é só o da vez de quem conjurou (ver GLOSSARY, **dado da vez**): ela
// declara o `TurnRoll`, e a varredura confere.
const turnRollOnly = "o dado é o da vez de quem conjurou"

// augmentDiceNotTheCastRoll são os aprimoramentos com dado na prosa, de magias que rolam na conjuração, cujo
// dado não é o da conjuração.
var augmentDiceNotTheCastRoll = map[string]string{
	"raio-solar[3]":              "o dado é da duração da cegueira",
	"enxame-rubro-de-ichabod[5]": "o dado diz se surge um enxame novo, e não é dano",
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
		if (reason == turnRollOnly) != (m.TurnRoll != nil && len(m.Rolls) == 0) {
			t.Errorf("%s (%s): o motivo %q e o dado da vez não concordam: a magia que só rola na vez declara `TurnRoll` e tem este motivo", m.Name, id, reason)
		}
		// O aprimoramento mexe no dado da conjuração E no da vez (`SpellOnTheSheet.TurnRoll`).
		rolls := len(m.Rolls) > 0 || m.TurnRoll != nil
		for i, a := range m.Augments {
			key := fmt.Sprintf("%s[%d]", id, i)
			declares := len(a.Adds) > 0 || len(a.Rolls) > 0 || a.ChangesTheRoll
			reason, excused := augmentDiceNotTheCastRoll[key]
			switch {
			case !rolls && declares:
				t.Errorf("%s (%s) aprimoramento %d mexe no dado de uma magia que não rola na conjuração nem na vez", m.Name, id, i)
			case rolls && diceInText.MatchString(a.Description) && !declares && !excused:
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

// TestEveryTurnRollSaysWhenAndWhatItRolls confere a forma do dado da vez (ver GLOSSARY, **dado da vez**): o momento
// é o início ou o fim do turno, toda versão tem parcela, e o nome da versão existe só quando há mais de uma —
// é ele que vai no botão de quem rola.
func TestEveryTurnRollSaysWhenAndWhatItRolls(t *testing.T) {
	spells := decodeResource[map[string]spellForRollSweep](t, "spells")
	turning := 0
	for id, m := range spells {
		if m.TurnRoll == nil {
			continue
		}
		turning++
		if m.TurnRoll.At != "inicio" && m.TurnRoll.At != "fim" {
			t.Errorf("%s (%s): o dado da vez é no %q, e o momento é \"inicio\" ou \"fim\"", m.Name, id, m.TurnRoll.At)
		}
		// A conjuração abre a linha em vigor da sustentada ou da de cena, e é dela que o dado da vez sai: outra
		// duração não teria de onde rolar.
		if m.Duration != "sustentada" && m.Duration != "cena" {
			t.Errorf("%s (%s) rola na vez e dura %q: a conjuração só abre linha em vigor para \"sustentada\" e \"cena\"", m.Name, id, m.Duration)
		}
		if len(m.TurnRoll.Versions) == 0 {
			t.Errorf("%s (%s): o dado da vez não tem versão nenhuma", m.Name, id)
		}
		named := map[string]bool{}
		for _, v := range m.TurnRoll.Versions {
			if (v.Name != "") != (len(m.TurnRoll.Versions) > 1) {
				t.Errorf("%s (%s): a versão %q tem nome com %d versão(ões); o nome existe só quando há mais de uma", m.Name, id, v.Name, len(m.TurnRoll.Versions))
			}
			if named[v.Name] {
				t.Errorf("%s (%s): duas versões se chamam %q", m.Name, id, v.Name)
			}
			named[v.Name] = true
			if len(v.Rolls) == 0 {
				t.Errorf("%s (%s): a versão %q não rola nada", m.Name, id, v.Name)
			}
			for _, r := range v.Rolls {
				if !diceShape.MatchString(r.Dice) {
					t.Errorf("%s (%s): o dado da vez %q não tem a forma NdM ou NdM+K", m.Name, id, r.Dice)
				}
			}
		}
	}
	if turning == 0 {
		t.Fatal("nenhuma magia declara o dado da vez: a varredura não mediu nada")
	}
}
