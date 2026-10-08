package book

import (
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
//
// Havia um TERCEIRO recuo, que procurava a flag num poder de id SUFIXADO
// (`inspiracao-1`, `-2`, …): o catálogo punha os modificadores da Inspiração
// nos degraus numerados e deixava o id base sem nenhum. Os degraus numerados
// eram a divergência que a ALE-423 consertou — eles concediam por NÍVEL o bônus
// que o livro VENDE —, e com eles fora as duas posturas carregam o bônus no
// poder de id exato. O recuo deixou de ter o que achar.
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

// StanceBase são os modificadores que esta postura liga no degrau ZERO, lidos
// do PODER que a concede.
//
// Eles saem do catálogo e não de uma lista escrita à mão: a Fúria mexe em
// ataque e dano, a Inspiração em perícia, a Muralha em Defesa e Reflexos, e
// três das Posturas de Combate não mexem em nada. Uma lista fixa acertava as
// duas primeiras posturas que existiram e erraria calada em todas as outras.
func StanceBase(flag string) []engine.Modifier {
	out := []engine.Modifier{}
	for _, p := range ClassPowersWithModifiers() {
		for _, m := range p.Modifiers {
			if m.Condition != nil && m.Condition.C == "flagOn" && m.Condition.Flag == flag {
				out = append(out, m)
			}
		}
	}
	return out
}

// StanceTargets são os alvos que a postura move — o que a tela precisa para
// dizer QUAIS números um chip mexe, sem o resto do modificador.
func StanceTargets(flag string) []engine.ModifierTarget {
	base := StanceBase(flag)
	out := make([]engine.ModifierTarget, 0, len(base))
	for _, m := range base {
		out = append(out, m.Target)
	}
	return out
}

// StanceDegreeModifiers é o que os DEGRAUS PAGOS valem: o bônus de base com o
// número de degraus somado, uma vez por alvo que a postura move.
//
// # O DEGRAU SUBSTITUI O BÔNUS, e não soma com ele
//
// O livro diz *"pode gastar +1 PM para aumentar os bônus em +1"* (p41): é UM
// bônus que fica maior, e não um segundo bônus ao lado do primeiro. Então o
// degrau sai com o valor TOTAL e com o MESMO `bonusType` da base, e é a regra
// de não-empilhamento (p105) que faz o maior vencer — o +3 entra e o +2 da base
// perde a disputa.
//
// Um modificador de `+degraus` solto daria o mesmo número hoje e erraria no dia
// em que outro bônus do MESMO tipo disputasse o alvo: a Inspiração de +3 compete
// como +3, e não como +1 mais dois avulsos. O `TestEveryStanceDegreeRidesANonStackingBonus`
// é quem mantém essa substituição representável.
//
// SEM A CONDIÇÃO da flag: quem chama grava um efeito ativo que nasce ao entrar
// na postura e morre ao encerrá-la, então a EXISTÊNCIA da linha já é a condição.
// Carregá-la de novo faria o mesmo bônus ser diferido e dobrado de volta pelo
// mesmo interruptor, e a tela mostraria o degrau como uma segunda linha.
func StanceDegreeModifiers(flag string, steps int) []engine.Modifier {
	if steps <= 0 {
		return nil
	}
	out := []engine.Modifier{}
	for _, m := range StanceBase(flag) {
		m.Amount += steps
		m.Condition = nil
		out = append(out, m)
	}
	return out
}
