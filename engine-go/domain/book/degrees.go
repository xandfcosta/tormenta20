package book

import (
	"encoding/json"
	"sync"

	"t20engine/domain/engine"
)

// O DEGRAU QUE UM PODER VENDE, fora da postura.
//
// O livro escreve a mesma oferta em quatro poderes de três classes: um bônus de
// base por um custo em PM, e *"a cada N níveis, pode gastar +1 PM para aumentar
// o bônus em M"*. A Fúria (p41) e a Inspiração (p44) são POSTURAS e a oferta
// delas mora na ativação, com a máquina de entrar e sair que a ALE-423
// construiu. O Baluarte e o Duelo (p53) não são — a p54 reserva o nome "Postura
// de Combate" a outro conjunto do cavaleiro —, e o que carrega o degrau deles é
// o INTERRUPTOR do situacional (decisão do dono, ALE-454).
//
// Por isso a oferta deles mora no VERBETE do poder e não numa ativação: criar
// uma ativação só para hospedar a conta daria ao Baluarte um botão de "Usar" na
// aba Poderes, e o app não cobra o PM dele — ele informa o preço e a mesa paga,
// como em todo situacional.

// Degrees é a oferta do livro: o preço do primeiro PM, o preço e o tamanho do
// degrau, e de quantos em quantos níveis o TETO cresce.
type Degrees struct {
	BasePm int `json:"basePm"`
	StepPm int `json:"stepPm"`
	// StepAmount é quanto CADA degrau soma ao bônus — +2 no Baluarte e +1 no
	// Duelo. Ele existe porque a máquina de postura assumia +1 e dava o número
	// errado para o Baluarte sem dizer nada.
	StepAmount      int    `json:"stepAmount"`
	FirstStepLevel  int    `json:"firstStepLevel"`
	StepEveryLevels int    `json:"stepEveryLevels"`
	StepLabel       string `json:"stepLabel"`
}

// DegreeOffer é a oferta já ligada à FLAG do situacional que a liga, com o
// bônus de base que o verbete concede.
type DegreeOffer struct {
	Flag      string
	PowerID   string
	PowerName string
	ClassName string
	Degrees   Degrees
	// Base são os modificadores do verbete, com a condição da flag ainda neles.
	Base []engine.Modifier
}

var (
	degreesOnce sync.Once
	degreesByID map[string]DegreeOffer
)

// DegreesByFlag liga cada flag de situacional à oferta de degrau do poder que a
// declara, ou devolve o mapa vazio quando nenhum poder vende degrau.
//
// A flag sai do PRIMEIRO modificador condicionado do verbete, e não de um campo
// próprio: a flag do degrau e a que os modificadores usam têm de ser a MESMA, e
// um segundo campo seria a segunda verdade que diverge. É a mesma derivação que
// o `ClassPowerFlags` faz.
//
// @example DegreesByFlag()["baluarte"].Degrees.StepAmount // 2
func DegreesByFlag() map[string]DegreeOffer {
	degreesOnce.Do(func() {
		degreesByID = map[string]DegreeOffer{}
		for _, power := range classPowersWithDegrees() {
			if power.Degrees == nil {
				continue
			}
			flag := firstConditionFlag(power.Modifiers)
			if flag == "" {
				continue
			}
			degreesByID[flag] = DegreeOffer{
				Flag: flag, PowerID: power.ID, PowerName: power.Name,
				ClassName: power.ClassName, Degrees: *power.Degrees,
				Base: power.Modifiers,
			}
		}
	})
	return degreesByID
}

// MaxDegrees é quantos degraus o nível NA CLASSE autoriza comprar.
//
// Mesma conta do `LevelSteps` das posturas, e de propósito: é a mesma frase do
// livro nos quatro poderes que vendem degrau. Nível abaixo do primeiro degrau
// dá zero, que é o cavaleiro de 1º — ele tem Baluarte e não tem o que comprar.
func (d Degrees) MaxDegrees(classLevel int) int {
	return LevelSteps(ActivationScale{
		FirstStepLevel: d.FirstStepLevel, StepEveryLevels: d.StepEveryLevels,
	}, classLevel)
}

// Cost é o que ligar o situacional com estes degraus custa de PM.
func (d Degrees) Cost(steps int) int {
	return d.BasePm + steps*d.StepPm
}

// Amount é o bônus TOTAL com os degraus pagos, a partir do bônus de base.
func (d Degrees) Amount(base, steps int) int {
	return base + steps*d.StepAmount
}

// DegreeModifiers é o que os degraus PAGOS valem: o bônus de base com os
// degraus somados, uma vez por alvo que o poder move.
//
// O DEGRAU SUBSTITUI O BÔNUS, e não soma com ele — *"pode gastar +1 PM para
// aumentar o bônus em +2"* (p53) é UM bônus que fica maior. O modificador sai
// com o valor TOTAL e com o MESMO `bonusType` da base, e é o não-empilhamento
// da p105 que faz o maior vencer. É a mesma escolha do `StanceDegreeModifiers`,
// e pela mesma razão: um `+degraus` solto daria o mesmo número hoje e erraria
// no dia em que outro bônus do mesmo tipo disputasse o alvo.
//
// SEM A CONDIÇÃO da flag: quem chama grava um efeito ativo que nasce ao ligar o
// interruptor e morre ao desligá-lo, então a EXISTÊNCIA da linha já é a
// condição. Carregá-la de novo faria o mesmo bônus ser diferido e dobrado de
// volta pelo mesmo interruptor, e a tela o mostraria como uma segunda linha.
func DegreeModifiers(offer DegreeOffer, steps int) []engine.Modifier {
	if steps <= 0 {
		return nil
	}
	out := []engine.Modifier{}
	for _, m := range offer.Base {
		m.Amount = offer.Degrees.Amount(m.Amount, steps)
		m.Condition = nil
		out = append(out, m)
	}
	return out
}

// powerWithDegrees é a leitura do verbete que esta pergunta precisa — o par
// id/nome, os modificadores e o bloco de degraus.
type powerWithDegrees struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	ClassName string            `json:"className"`
	Modifiers []engine.Modifier `json:"modifiers"`
	Degrees   *Degrees          `json:"degrees"`
}

func classPowersWithDegrees() []powerWithDegrees {
	raw := classPowersResolved()
	if raw == nil {
		return nil
	}
	var powers []powerWithDegrees
	_ = json.Unmarshal(raw, &powers)
	return powers
}

// firstConditionFlag é a flag do primeiro modificador que tem uma.
func firstConditionFlag(mods []engine.Modifier) string {
	for _, m := range mods {
		if m.Condition != nil && m.Condition.Flag != "" {
			return m.Condition.Flag
		}
	}
	return ""
}
