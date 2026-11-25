package catalog

import (
	"fmt"
	"regexp"
	"testing"
)

// spellForRollSweep é o que a varredura do dado da magia precisa: o efeito, o dado declarado e os
// aprimoramentos.
type spellForRollSweep struct {
	Name       string            `json:"name"`
	Duration   string            `json:"duration"`
	BaseEffect string            `json:"baseEffect"`
	Rolls      []spellRollSeen   `json:"rolls"`
	Actions    *spellActionsSeen `json:"actions"`
	Lingering  *struct {
		At          string          `json:"at"`
		Rolls       []spellRollSeen `json:"rolls"`
		Turns       int             `json:"turns"`
		RepeatsSave bool            `json:"repeatsSave"`
		Burning     bool            `json:"burning"`
	} `json:"lingering"`
	TurnRoll *struct {
		At       string `json:"at"`
		Versions []struct {
			Name  string          `json:"name"`
			Rolls []spellRollSeen `json:"rolls"`
		} `json:"versions"`
	} `json:"turnRoll"`
	Augments []struct {
		Description    string            `json:"description"`
		Adds           []spellRollSeen   `json:"adds"`
		Rolls          []spellRollSeen   `json:"rolls"`
		ChangesTheRoll bool              `json:"changesTheRoll"`
		Actions        *spellActionsSeen `json:"actions"`
		Charges        int               `json:"charges"`
		LingerAdds     []spellRollSeen   `json:"lingerAdds"`
	} `json:"augments"`
}

type spellActionsSeen struct {
	OncePerTurn bool `json:"oncePerTurn"`
	Options     []struct {
		Name     string          `json:"name"`
		Action   string          `json:"action"`
		Rolls    []spellRollSeen `json:"rolls"`
		CastRoll bool            `json:"castRoll"`
	} `json:"options"`
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
	"mao-poderosa-de-talude":          actionRollOnly,
	"relampago-flamejante-de-reynard": actionRollOnly,
	"furia-do-panteao":                actionRollOnly,
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

// actionRollOnly é o motivo da magia cujo dado é só o das ações dela em vigor (ver GLOSSARY, **ação da magia**):
// ela declara `Actions`, e a varredura confere.
const actionRollOnly = "o dado é o das ações da magia em vigor"

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
		if (reason == actionRollOnly) != (m.Actions != nil && len(m.Rolls) == 0) {
			t.Errorf("%s (%s): o motivo %q e as ações da magia não concordam: a magia que só rola nas ações declara `Actions` e tem este motivo", m.Name, id, reason)
		}
		// O aprimoramento mexe no dado da conjuração, no da vez (`SpellOnTheSheet.TurnRoll`) e no das ações.
		rolls := len(m.Rolls) > 0 || m.TurnRoll != nil || m.Actions != nil
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

// TestEveryLingeringRollSaysWhenAndWhatItRolls confere a forma da marca na fila (ver GLOSSARY, **marca na fila**): o
// momento é o início ou o fim do turno de quem foi atingido, há uma parcela, a duração não é negativa, as chamas são
// do início do turno (a condição Em Chamas, p394), e o aumento que soma ao dado do alvo é de magia que deixa marca.
func TestEveryLingeringRollSaysWhenAndWhatItRolls(t *testing.T) {
	spells := decodeResource[map[string]spellForRollSweep](t, "spells")
	lingering := 0
	for id, m := range spells {
		for i, a := range m.Augments {
			if len(a.LingerAdds) > 0 && m.Lingering == nil {
				t.Errorf("%s (%s), aprimoramento %d, soma ao dado do alvo e a magia não deixa marca", m.Name, id, i)
			}
		}
		if m.Lingering == nil {
			continue
		}
		lingering++
		l := m.Lingering
		if l.At != "inicio" && l.At != "fim" {
			t.Errorf("%s (%s): a marca rola no %q, e o momento é \"inicio\" ou \"fim\"", m.Name, id, l.At)
		}
		if l.Burning && l.At != "inicio" {
			t.Errorf("%s (%s): as chamas rolam no início do turno (p394), e a marca diz %q", m.Name, id, l.At)
		}
		if l.Turns < 0 {
			t.Errorf("%s (%s): a marca dura %d turnos", m.Name, id, l.Turns)
		}
		// UMA PARCELA SÓ: a linha do mestre tem um campo da mão por dado da vez, e uma segunda parcela não teria
		// onde receber o total da mesa.
		if len(l.Rolls) != 1 {
			t.Errorf("%s (%s): a marca rola %d parcelas, e a linha do mestre recebe o total da mão de uma", m.Name, id, len(l.Rolls))
		}
		for _, r := range l.Rolls {
			if !diceShape.MatchString(r.Dice) {
				t.Errorf("%s (%s): o dado da marca %q não tem a forma NdM ou NdM+K", m.Name, id, r.Dice)
			}
		}
	}
	if lingering == 0 {
		t.Fatal("nenhuma magia deixa marca: a varredura não mediu nada")
	}
}

// TestEverySpellActionSaysWhatItCostsAndRolls confere a forma das ações da magia (ver GLOSSARY, **ação da magia**):
// toda opção custa uma ação do turno e rola um dado — o dela, ou o da conjuração —, os nomes não se repetem, e o
// aprimoramento que dá ações diz quantas vezes. A magia com ações próprias é sustentada: é a linha em vigor dela
// que as oferece.
func TestEverySpellActionSaysWhatItCostsAndRolls(t *testing.T) {
	spells := decodeResource[map[string]spellForRollSweep](t, "spells")
	acting := 0
	check := func(where string, actions *spellActionsSeen, castRolls int) {
		acting++
		if len(actions.Options) == 0 {
			t.Errorf("%s: as ações não têm opção nenhuma", where)
		}
		named := map[string]bool{}
		for _, o := range actions.Options {
			switch o.Action {
			case "padrao", "movimento", "livre", "completa":
			default:
				t.Errorf("%s, %q: a ação %q não é uma do turno (padrao, movimento, livre, completa)", where, o.Name, o.Action)
			}
			if o.Name == "" || named[o.Name] {
				t.Errorf("%s: a opção %q não tem nome ou o repete", where, o.Name)
			}
			named[o.Name] = true
			if o.CastRoll == (len(o.Rolls) > 0) {
				t.Errorf("%s, %q: a opção rola o dado dela OU o da conjuração, e não os dois nem nenhum", where, o.Name)
			}
			if o.CastRoll && castRolls == 0 {
				t.Errorf("%s, %q: a opção rola o dado da conjuração, e a magia não rola nada ao ser conjurada", where, o.Name)
			}
			for _, r := range o.Rolls {
				if !diceShape.MatchString(r.Dice) {
					t.Errorf("%s, %q: o dado %q não tem a forma NdM ou NdM+K", where, o.Name, r.Dice)
				}
			}
		}
	}
	for id, m := range spells {
		if m.Actions != nil {
			if m.Duration != "sustentada" {
				t.Errorf("%s (%s) tem ações e dura %q: a linha em vigor que as oferece é a da sustentada", m.Name, id, m.Duration)
			}
			check(fmt.Sprintf("%s (%s)", m.Name, id), m.Actions, len(m.Rolls))
		}
		for i, a := range m.Augments {
			if (a.Actions != nil) != (a.Charges > 0) {
				t.Errorf("%s (%s) aprimoramento %d: as ações (%v) e as cargas (%d) andam juntas", m.Name, id, i, a.Actions != nil, a.Charges)
			}
			if a.Actions != nil {
				check(fmt.Sprintf("%s (%s) aprimoramento %d", m.Name, id, i), a.Actions, len(m.Rolls))
			}
		}
	}
	if acting == 0 {
		t.Fatal("nenhuma magia declara ações: a varredura não mediu nada")
	}
}
