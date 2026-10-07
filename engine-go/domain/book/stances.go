package book

import (
	"strconv"
	"sync"

	"t20engine/domain/engine"
)

// AS POSTURAS DO CATÁLOGO, ligadas à flag que cada uma acende.

// Stance é uma postura do livro vista de fora: a flag que ela acende, o nome, o
// custo e a página.
type Stance struct {
	Flag string
	Name string
	PM   int
	Page int
	// ID é o do poder que a concede, para quem precisa perguntar a posse.
	ID string
	// Group é o conjunto de mutuamente exclusivas — ver `Activation.StanceGroup`.
	Group string
	// Action é o que ASSUMI-LA custa do turno: livre na Fúria, padrão na
	// Inspiração, movimento nas Posturas de Combate (p54).
	Action string
}

var (
	stancesOnce   sync.Once
	stancesByFlag map[string]Stance
)

// StancesFromCatalog liga as ativações de `kind: "stance"` à flag que elas
// acendem.
//
// Ela morava na cena da ficha e subiu na ALE-351, pela mesma razão do
// `activation_rules.go`: quem decide entrar numa postura é o caso de uso, e ele
// não alcança o `serve/web`.
//
// A FLAG DECLARADA VENCE, e a derivada é o recuo.
//
// A derivação lê o `condition.flag` dos modificadores do poder de mesmo id, e
// ela existe porque a flag do catálogo e a que os modificadores usam têm de ser
// a MESMA. Ela continua atendendo a Fúria e a Inspiração, que não declaram
// nada — e o que a obrigou a ganhar companhia foram as Posturas de Combate
// (p54): três das seis não têm modificador nenhum, então não há de onde
// derivar. Ver o campo `Flag` da `Activation`.
func StancesFromCatalog() map[string]Stance {
	stancesOnce.Do(func() {
		stancesByFlag = map[string]Stance{}
		flags := ClassPowerFlags()
		for _, a := range Activations() {
			if a.Kind != "stance" {
				continue
			}
			flag := a.Flag
			if flag == "" {
				flag = flags[a.ID]
			}
			if flag == "" {
				flag = flags[a.ID+stanceStep(flags, a.ID)]
			}
			if flag == "" {
				continue
			}
			stancesByFlag[flag] = Stance{
				Flag: flag, Name: a.Name, PM: ActivationPm(a), Page: a.BookPage,
				Group: a.StanceGroup, Action: a.Action, ID: a.ID,
			}
		}
	})
	return stancesByFlag
}

// StanceSiblings são as OUTRAS posturas do mesmo grupo — as que esta derruba ao
// ser assumida (p54).
//
// Grupo vazio não tem irmã nenhuma: a Fúria não derruba a Inspiração.
func StanceSiblings(group, flag string) []Stance {
	if group == "" {
		return nil
	}
	outside := []Stance{}
	for _, s := range StancesFromCatalog() {
		if s.Group == group && s.Flag != flag {
			outside = append(outside, s)
		}
	}
	return outside
}

// stanceStep acha o sufixo do DEGRAU quando a postura não declara a flag no
// poder de id exato.
//
// O catálogo trata as duas posturas de formas DIFERENTES:
// `class.barbaro.furia` carrega os modificadores no poder de id exato, enquanto
// `class.bardo.inspiracao` os põe nos degraus numerados (`inspiracao-1`, `-2`,
// …) e deixa o id base sem modificador nenhum. Ligar só pelo id exato acha UMA
// das duas, e passa calado: a outra simplesmente não aparece na lista.
//
// O sufixo aceito é `-<dígitos>` e MAIS NADA. Um prefixo solto casaria
// `class.barbaro.furia-da-savana`, que é outro poder — e no dia em que ele
// ligasse uma flag, a postura errada herdaria a dele.
func stanceStep(flags map[string]string, base string) string {
	for i := 1; i <= 9; i++ {
		suffix := "-" + strconv.Itoa(i)
		if flags[base+suffix] != "" {
			return suffix
		}
	}
	return ""
}

// FlagGrants são as ativações de gatilho daquela flag que CONCEDEM algo.
//
// Ela NÃO filtra pelo que o personagem possui, e isso é uma folga deliberada:
// quem chega aqui já entrou na postura, e a postura é de uma classe. Filtrar
// duas vezes daria uma segunda leitura da posse.
func FlagGrants(flag string) []Activation {
	outside := []Activation{}
	for _, spec := range Activations() {
		if spec.RequiresFlag == flag && spec.Grant != nil {
			outside = append(outside, spec)
		}
	}
	return outside
}

// StanceTargets são os alvos de modificador que esta postura move, lidos do
// PODER que a concede.
//
// Eles saem do catálogo e não de uma lista escrita à mão: a Fúria mexe em
// ataque e dano, a Inspiração em perícia, a Muralha em Defesa e Reflexos, e
// três das Posturas de Combate não mexem em nada. Uma lista fixa acertava as
// duas primeiras posturas que existiram e erraria calada em todas as outras.
func StanceTargets(flag string) []engine.ModifierTarget {
	out := []engine.ModifierTarget{}
	for _, p := range ClassPowersWithModifiers() {
		for _, m := range p.Modifiers {
			if m.Condition != nil && m.Condition.C == "flagOn" && m.Condition.Flag == flag {
				out = append(out, m.Target)
			}
		}
	}
	return out
}
