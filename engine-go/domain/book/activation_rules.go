package book

import (
	"encoding/json"
	"strconv"

	"t20engine/domain/engine"
)

// AS REGRAS DE ATIVAR UM PODER: se ele pode ser usado agora, qual limite o
// prende, e quanto custa entrar numa postura de degraus.
//
// Elas moravam na CENA da ficha, e o `activations.go` ao lado dizia por escrito
// que esta era "a metade que a cena decide". Deixou de ser: quem decide usar um
// poder é o caso de uso (`character.Plays`), e a cena continua lendo as mesmas
// funções para desenhar o botão desligado (ALE-351). Duas cópias — uma para
// decidir, outra para desenhar — dariam um botão aceso sobre um gesto recusado.
//
// # O que é COBRADO e o que é só crachá
//
// "1/cena" e "1/dia" são cobrados — eles têm contador no banco. Um "3/dia" e um
// "1/rodada" saem como crachá e nada mais, e isso é decisão registrada, não
// esquecimento: a mesa conta rodadas, a ficha não.
//
// O REGISTRO em si mora no `activations.go`, ao lado: ler o catálogo é uma
// coisa, decidir a partir dele é outra, e por isso são dois arquivos.

// ── o LIMITE de usos ─────────────────────────────────────────────────────────

// ChargedScope é "scene", "day" ou "" — e "" quer dizer que o limite existe
// no livro e a ficha NÃO o cobra.
//
// A DECISÃO de cobrar mora no `engine.UsageLimit`, junto com a janela: aqui só
// se traduz para a string que a coluna de contadores usa. Enquanto as duas
// coisas eram um `switch` só, "a ficha não conta rodadas" era uma ausência no
// meio de um `case` — e ausência não se lê (ALE-365).
func ChargedScope(spec Activation) string {
	limit, err := engine.ParseUsageLimit(unquoted(spec.Uses))
	if err != nil || !limit.ChargedBySheet() {
		return ""
	}
	return string(limit.Window)
}

// unquoted tira as aspas do JSON cru do `uses`. Ele é cru porque o catálogo
// escreve naturezas diferentes ali — ver o `activations.go`.
func unquoted(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return ""
	}
	return text
}

// CostIsVariable diz que o custo é NEGOCIADO com a mesa, e não um número.
func CostIsVariable(spec Activation) bool {
	return ActivationPm(spec) < 0
}

// ActivationPm é o custo em PM, ou -1 quando ele é variável.
//
// Menos um e não zero: zero é um custo LEGÍTIMO (a maioria das passivas), e
// confundir os dois é o que faria um poder de graça ser tratado como negociado
// com a mesa.
func ActivationPm(spec Activation) int {
	var number int
	if json.Unmarshal(spec.PmCost, &number) == nil {
		return number
	}
	return -1
}

// ── a DECISÃO de usar ────────────────────────────────────────────────────────

// UseDecision responde se o poder pode ser usado AGORA, e por que não.
//
// A ORDEM das recusas importa: a razão mostrada é a PRIMEIRA que barra, então
// "requer Fúria" aparece antes de "PM insuficiente" num poder que precisa das
// duas coisas — e é a que a pessoa pode resolver primeiro.
func UseDecision(spec Activation, context UseContext) (bool, string) {
	if CostIsVariable(spec) {
		return false, "custo variável"
	}
	if spec.RequiresFlag != "" && !context.Flags[spec.RequiresFlag] {
		return false, "requer " + spec.RequiresFlag
	}
	switch ChargedScope(spec) {
	case "scene":
		if context.UsedThisScene >= 1 {
			return false, "limite por cena atingido"
		}
	case "day":
		if context.UsedToday >= 1 {
			return false, "limite por dia atingido"
		}
	}
	if ActivationPm(spec) > context.CurrentPM {
		return false, "PM insuficiente"
	}
	return true, ""
}

// UseContext é o que a decisão precisa saber da ficha AGORA.
type UseContext struct {
	CurrentPM     int
	UsedThisScene int
	UsedToday     int
	Flags         map[string]bool
}

// ── a POSTURA de degraus ─────────────────────────────────────────────────────

// LevelSteps são os degraus EXTRAS que o nível na classe concede.
//
// O nível é o da CLASSE e não o do personagem (p40): um bárbaro 5/ladino 5 tem
// a Fúria de um bárbaro de nível 5, e não a de um personagem de nível 10.
func LevelSteps(scale ActivationScale, classLevel int) int {
	if scale.StepEveryLevels <= 0 || classLevel < scale.FirstStepLevel {
		return 0
	}
	return 1 + (classLevel-scale.FirstStepLevel)/scale.StepEveryLevels
}

// StanceCost é o que entrar custa com os degraus escolhidos.
func StanceCost(spec Activation, steps int) int {
	if spec.Scaling == nil {
		return ActivationPm(spec)
	}
	return spec.Scaling.BasePm + steps*spec.Scaling.StepPm
}

// StanceDecision responde se dá para entrar na postura com esses degraus.
func StanceDecision(spec Activation, steps, max, currentPM int) (bool, string) {
	if steps < 0 || steps > max {
		return false, "o nível permite até " + strconv.Itoa(max) + " degraus"
	}
	if cost := StanceCost(spec, steps); cost > currentPM {
		return false, "PM insuficiente"
	}
	return true, ""
}

// O BÔNUS CUMULATIVO DE CENA (p42).
//
// A regra é de uma linha e mora aqui por isso: ela é do LIVRO — quanto cada
// gatilho dá e onde ele para —, e quem a chama é o caso de uso, que sabe o
// valor de agora e o nível. Nada aqui lê banco nem escreve nada.

// cumulativeTriggers é o vocabulário FECHADO dos gatilhos.
//
// Fechado pela mesma razão das cinco manobras: o casamento é por TEXTO, e um
// `on` digitado errado viraria um poder que nunca acumula — calado, como todo
// defeito de casamento por texto. Quem o cobra é o
// `TestEveryCumulativeBonusDeclaresATriggerTheEngineKnows`.
var cumulativeTriggers = map[string]bool{
	// "quando faz um acerto crítico ou reduz um inimigo a 0 PV" (p42).
	"criticalOrDrop": true,
}

// CumulativeTriggerIsKnown diz se o motor sabe disparar este gatilho.
func CumulativeTriggerIsKnown(on string) bool { return cumulativeTriggers[on] }

// CumulativeCap é o TETO do bônus — *"limitado pelo seu nível"* (p42).
//
// Devolve zero quando o teto é de uma espécie que o motor não conhece, e zero
// é o lado seguro: um teto desconhecido vira "não acumula", em vez de virar
// "acumula para sempre". É o oposto do `default` do `evalModifierScale`, que
// engole o `per` desconhecido e some com o bônus — aqui o silêncio também
// custa, mas custa um poder inerte e não um número inventado.
func CumulativeCap(spec CumulativeBonus, level int) int {
	if spec.CapPer == "level" {
		return level
	}
	return 0
}

// CumulativeNext é quanto o bônus vale DEPOIS de um gatilho, com o teto já
// aplicado.
//
// @example CumulativeNext(spec, 5, 6) // 6, e o próximo gatilho devolve 6 também
func CumulativeNext(spec CumulativeBonus, current, level int) int {
	return min(current+spec.Amount, CumulativeCap(spec, level))
}

// FlagCumulatives são as ativações que ACUMULAM enquanto esta flag está acesa.
//
// Irmã da `FlagGrants`, e separada dela porque as duas respondem perguntas
// diferentes: aquela diz o que a postura concede ao ENTRAR, esta diz o que ela
// faz crescer DEPOIS. Quem precisa das duas juntas é só o encerramento, que
// apaga os efeitos das duas famílias.
func FlagCumulatives(flag string) []Activation {
	outside := []Activation{}
	for _, spec := range Activations() {
		if spec.RequiresFlag == flag && spec.Cumulative != nil {
			outside = append(outside, spec)
		}
	}
	return outside
}
