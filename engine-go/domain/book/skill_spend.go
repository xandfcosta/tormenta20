package book

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/catalog"
)

// O PM QUE SE PAGA NUM TESTE DE PERÍCIA (ALE-489).
//
// Os verbetes que vendem bônus no teste: Audácia (+Carisma, p47),
// Engenhosidade (+Inteligência, p68 e o kliren), Orgulho (+2 por PM, até o
// Carisma, p79), A Pessoa Certa para o Trabalho (+10, só na lista de ladino,
// p74), Eclético (treinado neste teste, p45), Cultura Exótica (ignora a
// exigência de treino, p90) e Especialista (dobra o treino nas perícias
// escolhidas, p73). As trocas e os dois d20 vieram na ALE-500. O verbete escreve a forma no bloco `skillSpend` da
// ativação, e esta é a conta: o que cada um oferece para UMA perícia, e o que o
// jogador pagou.
//
// Vários no mesmo teste SOMAM (decisão do dono, p226), e o PM sai ao rolar —
// o teste de perícia não tem confirmação do mestre.

// SkillSpend é a forma de um gasto no teste.
type SkillSpend struct {
	Kind string `json:"kind"`
	// Attribute é o atributo somado (`add-attribute`) ou o que limita os PM
	// (`per-pm`).
	Attribute string `json:"attribute,omitempty"`
	// Amount é o +10 do `flat` e o +2 por PM do `per-pm`.
	Amount int `json:"amount,omitempty"`
	// SkillsOf limita às perícias da LISTA dessa classe (p74).
	SkillsOf string `json:"skillsOf,omitempty"`
	// Group junta o mesmo poder recebido de duas fontes: "caso receba esta
	// habilidade novamente, seu custo é reduzido em –1 PM" (kliren, p28).
	Group string `json:"group,omitempty"`
	// Skills e Attributes dizem ONDE a troca e os dois d20 valem: nestas
	// perícias, ou nas perícias deste atributo do LIVRO (Palpite Fundamentado:
	// "originalmente baseada em Inteligência ou Sabedoria", p90).
	Skills     []string `json:"skills,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
	// SwapTo é a perícia que o teste passa a usar (`swap`).
	SwapTo string `json:"swapTo,omitempty"`
	// RequiresFlag é a flag que um GESTO levantou e pagou — o Detetive (p91)
	// paga 1 PM uma vez e troca de graça até o fim da cena.
	RequiresFlag string `json:"requiresFlag,omitempty"`
	// Context é o quando do livro que o mestre julga ("para interrogar").
	Context string `json:"context,omitempty"`
}

// As formas de gasto.
const (
	SpendAddAttribute   = "add-attribute"
	SpendPerPM          = "per-pm"
	SpendFlat           = "flat"
	SpendAsTrained      = "as-trained"
	SpendIgnoreTraining = "ignore-training"
	// SpendSwap usa o total de OUTRA perícia no teste (ALE-500).
	SpendSwap = "swap"
	// SpendTwoDice rola dois d20 e usa o melhor (ALE-500).
	SpendTwoDice = "two-dice"
	// SpendDoubleTraining soma o treino de novo, nas perícias que o poder
	// escolheu (Especialista, p73, ALE-501).
	SpendDoubleTraining = "double-training"
)

// SkillSpendTarget é o que a oferta precisa saber da perícia e da ficha.
type SkillSpendTarget struct {
	Skill       string
	Trained     bool
	TrainedOnly bool
	// TrainingBonus é o +2/+4/+6 do nível, que o Eclético empresta (p45).
	TrainingBonus int
	Attributes    map[string]int
	// Attribute é o atributo da perícia no LIVRO; Totals e TrainedIn são o
	// total e o treino de cada perícia da ficha, para a troca; Flags são as
	// flags acesas (a cena do Detetive).
	Attribute string
	Totals    map[string]int
	TrainedIn map[string]bool
	Flags     map[string]bool
	// PowerPicks é o `powerChoices` da ficha: as perícias que o Especialista
	// escolheu, pelo id do poder.
	PowerPicks map[string][]string
}

// SkillSpendOffer é um gasto que a ficha pode fazer NESTE teste.
type SkillSpendOffer struct {
	ID string
	// Key é o nome do gasto no SINAL da tela (`SpendKey`), e é por ele que o
	// pedido volta.
	Key  string
	Name string
	Kind string
	// PM é o custo fixo, já com o desconto do poder recebido duas vezes. No
	// `per-pm`, é o custo de UM PM, e MaxPM o teto.
	PM    int
	MaxPM int
	// Bonus é o que soma no teste — por PM, no `per-pm`.
	Bonus int
	// Unlocks diz que o gasto deixa rolar a perícia só para treinados.
	Unlocks bool
	// SwapTo é a perícia que a troca usa; Context, o quando do livro.
	SwapTo  string
	Context string
}

// SkillSpendOffers são os gastos que as ativações POSSUÍDAS oferecem para esta
// perícia, na ordem do catálogo.
//
// O que não se aplica não é oferecido: o Eclético numa perícia já treinada, a
// Cultura Exótica numa perícia que não pede treino, A Pessoa Certa fora da
// lista de ladino, o Orgulho de quem tem Carisma zero.
func SkillSpendOffers(owned []Activation, target SkillSpendTarget) []SkillSpendOffer {
	out := []SkillSpendOffer{}
	byGroup := map[string]int{}
	for _, a := range owned {
		spend := a.SkillSpend
		if spend == nil {
			continue
		}
		if spend.Group != "" {
			if i, seen := byGroup[spend.Group]; seen {
				out[i].PM = max(0, out[i].PM-1)
				continue
			}
		}
		offer := SkillSpendOffer{ID: a.ID, Key: SpendKey(a.ID), Name: a.Name, Kind: spend.Kind, PM: ActivationPm(a)}
		switch spend.Kind {
		case SpendAddAttribute:
			offer.Bonus = target.Attributes[spend.Attribute]
		case SpendPerPM:
			offer.PM, offer.MaxPM, offer.Bonus = 1, target.Attributes[spend.Attribute], spend.Amount
			if offer.MaxPM < 1 {
				continue
			}
		case SpendFlat:
			if spend.SkillsOf != "" && !slices.Contains(ClassSkillList(spend.SkillsOf), target.Skill) {
				continue
			}
			offer.Bonus = spend.Amount
		case SpendAsTrained:
			if target.Trained {
				continue
			}
			offer.Bonus, offer.Unlocks = target.TrainingBonus, true
		case SpendIgnoreTraining:
			if target.Trained || !target.TrainedOnly {
				continue
			}
			offer.Unlocks = true
		case SpendSwap:
			if !appliesTo(*spend, target) || !swapUsable(spend.SwapTo, target) {
				continue
			}
			if spend.RequiresFlag != "" {
				// O PM foi do gesto que acendeu a flag.
				offer.PM = 0
			}
			offer.SwapTo, offer.Context, offer.Unlocks = spend.SwapTo, spend.Context, true
			offer.Bonus = target.Totals[spend.SwapTo] - target.Totals[target.Skill]
		case SpendTwoDice:
			if !appliesTo(*spend, target) {
				continue
			}
			offer.Context = spend.Context
		case SpendDoubleTraining:
			if !target.Trained || !slices.Contains(target.PowerPicks[a.ID], target.Skill) {
				continue
			}
			offer.Bonus = target.TrainingBonus
		default:
			continue
		}
		if spend.Group != "" {
			byGroup[spend.Group] = len(out)
		}
		out = append(out, offer)
	}
	return out
}

// appliesTo diz se a troca ou os dois d20 valem nesta perícia, e se a flag que
// eles pedem está acesa.
func appliesTo(spend SkillSpend, target SkillSpendTarget) bool {
	if spend.RequiresFlag != "" && !target.Flags[spend.RequiresFlag] {
		return false
	}
	if target.Skill == spend.SwapTo {
		return false
	}
	return slices.Contains(spend.Skills, target.Skill) || slices.Contains(spend.Attributes, target.Attribute)
}

// swapUsable diz se a perícia de destino existe na ficha e pode ser rolada: a
// só para treinados, sem treino, não serve de troca (p115).
func swapUsable(skill string, target SkillSpendTarget) bool {
	if _, found := target.Totals[skill]; !found {
		return false
	}
	for _, e := range Expertises() {
		if e.Name == skill && e.TrainedOnly {
			return target.TrainedIn[skill]
		}
	}
	return true
}

// PaysTraining diz se o gasto mexe no TREINO da perícia original — contar como
// treinado, ignorar o treino, dobrá-lo (Especialista, ALE-501). É o que a troca
// de perícia recusa.
func PaysTraining(kind string) bool {
	return kind == SpendAsTrained || kind == SpendIgnoreTraining || kind == SpendDoubleTraining
}

// SkillSpendChoice é um gasto pedido: a chave e, no `per-pm`, quantos PM.
type SkillSpendChoice struct {
	Key string
	PM  int
}

// SpendKey é o nome do gasto no sinal da tela: o fim do id, em `snake_case`
// (o `-` não atravessa o Datastar — ver o `CLAUDE.md`, "Sinal"). As duas
// Engenhosidades dão a mesma chave, e é a oferta única do grupo.
//
//	SpendKey("class.bucaneiro.audacia") // "audacia"
func SpendKey(id string) string {
	return strings.ReplaceAll(id[strings.LastIndex(id, ".")+1:], "-", "_")
}

// PaidSkillSpends é o que os gastos pedidos valem no teste.
type PaidSkillSpends struct {
	Bonus    int
	PM       int
	Unlocked bool
	Parts    []SpentPart
	// SwapTo é a perícia que o teste usou no lugar; TwoDice, que se rolam dois
	// d20 e vale o melhor.
	SwapTo  string
	TwoDice bool
	// trainingPaid marca o treino pago, que a troca recusa.
	trainingPaid bool
}

// SpentPart é um gasto escrito para a mesa — "Audácia +3 · 2 PM".
type SpentPart struct {
	Name  string
	Kind  string
	Bonus int
	PM    int
}

// PaySkillSpends confere os pedidos contra as ofertas e soma.
//
//	paid, err := PaySkillSpends(offers, []SkillSpendChoice{{Key: "orgulho", PM: 2}})
func PaySkillSpends(offers []SkillSpendOffer, chosen []SkillSpendChoice) (PaidSkillSpends, error) {
	paid := PaidSkillSpends{}
	seen := map[string]bool{}
	for _, c := range chosen {
		if seen[c.Key] {
			return PaidSkillSpends{}, fmt.Errorf("o gasto %q veio duas vezes no mesmo teste", c.Key)
		}
		seen[c.Key] = true
		i := slices.IndexFunc(offers, func(o SkillSpendOffer) bool { return o.Key == c.Key })
		if i < 0 {
			return PaidSkillSpends{}, fmt.Errorf("o gasto %q não vale neste teste — ou a ficha não o tem, ou a perícia não o aceita", c.Key)
		}
		offer := offers[i]
		part := SpentPart{Name: offer.Name, Kind: offer.Kind, Bonus: offer.Bonus, PM: offer.PM}
		if offer.Kind == SpendPerPM {
			if c.PM < 1 || c.PM > offer.MaxPM {
				return PaidSkillSpends{}, fmt.Errorf("%s: %d PM pedidos, e o limite é de 1 a %d", offer.Name, c.PM, offer.MaxPM)
			}
			part.Bonus, part.PM = offer.Bonus*c.PM, offer.PM*c.PM
		}
		switch offer.Kind {
		case SpendSwap:
			if paid.SwapTo != "" {
				return PaidSkillSpends{}, fmt.Errorf("duas trocas de perícia no mesmo teste (%s e %s): o teste usa uma perícia só",
					paid.SwapTo, offer.SwapTo)
			}
			paid.SwapTo = offer.SwapTo
		case SpendTwoDice:
			paid.TwoDice = true
		}
		paid.Bonus += part.Bonus
		paid.PM += part.PM
		paid.Unlocked = paid.Unlocked || offer.Unlocks
		paid.Parts = append(paid.Parts, part)
		paid.trainingPaid = paid.trainingPaid || PaysTraining(offer.Kind)
	}
	// A TROCA NÃO SE MISTURA COM TREINO PAGO: o Eclético diz "treinado NESSA
	// perícia" (p45), e a troca muda qual perícia é.
	if paid.SwapTo != "" && paid.trainingPaid {
		return PaidSkillSpends{}, fmt.Errorf("a troca por %s não combina com treino pago: o treino é da perícia original",
			paid.SwapTo)
	}
	return paid, nil
}

// ClassSkillList é a LISTA de perícias da classe: as fixas, a piscina de
// escolha e a escolha entre duas.
//
//	ClassSkillList("Ladino") // [Ladinagem, Ofício, Acrobacia, …]
func ClassSkillList(class string) []string {
	var all map[string]ClassExpertises
	if raw, ok := catalog.Resource("class-expertises"); ok {
		_ = json.Unmarshal(raw, &all)
	}
	rule, found := all[class]
	if !found {
		return nil
	}
	out := append(append([]string{}, rule.Fixed...), rule.ChoosePool...)
	if rule.EitherOr != nil {
		out = append(out, rule.EitherOr.Options...)
	}
	return out
}
