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
	var numero int
	if json.Unmarshal(spec.PmCost, &numero) == nil {
		return numero
	}
	return -1
}

// ── a DECISÃO de usar ────────────────────────────────────────────────────────

// UseDecision responde se o poder pode ser usado AGORA, e por que não.
//
// A ORDEM das recusas importa: a razão mostrada é a PRIMEIRA que barra, então
// "requer Fúria" aparece antes de "PM insuficiente" num poder que precisa das
// duas coisas — e é a que a pessoa pode resolver primeiro.
func UseDecision(spec Activation, contexto UseContext) (bool, string) {
	if CostIsVariable(spec) {
		return false, "custo variável"
	}
	if spec.RequiresFlag != "" && !contexto.Flags[spec.RequiresFlag] {
		return false, "requer " + spec.RequiresFlag
	}
	switch ChargedScope(spec) {
	case "scene":
		if contexto.UsadoNaCena >= 1 {
			return false, "limite por cena atingido"
		}
	case "day":
		if contexto.UsadoNoDia >= 1 {
			return false, "limite por dia atingido"
		}
	}
	if ActivationPm(spec) > contexto.PmAtual {
		return false, "PM insuficiente"
	}
	return true, ""
}

// UseContext é o que a decisão precisa saber da ficha AGORA.
type UseContext struct {
	PmAtual     int
	UsadoNaCena int
	UsadoNoDia  int
	Flags       map[string]bool
}

// ── a POSTURA de degraus ─────────────────────────────────────────────────────

// LevelSteps são os degraus EXTRAS que o nível na classe concede.
//
// O nível é o da CLASSE e não o do personagem (p40): um bárbaro 5/ladino 5 tem
// a Fúria de um bárbaro de nível 5, e não a de um personagem de nível 10.
func LevelSteps(escala ActivationScale, nivelNaClasse int) int {
	if escala.StepEveryLevels <= 0 || nivelNaClasse < escala.FirstStepLevel {
		return 0
	}
	return 1 + (nivelNaClasse-escala.FirstStepLevel)/escala.StepEveryLevels
}

// StanceCost é o que entrar custa com os degraus escolhidos.
func StanceCost(spec Activation, degraus int) int {
	if spec.Scaling == nil {
		return ActivationPm(spec)
	}
	return spec.Scaling.BasePm + degraus*spec.Scaling.StepPm
}

// StanceDecision responde se dá para entrar na postura com esses degraus.
func StanceDecision(spec Activation, degraus, maximo, pmAtual int) (bool, string) {
	if degraus < 0 || degraus > maximo {
		return false, "o nível permite até " + strconv.Itoa(maximo) + " degraus"
	}
	if custo := StanceCost(spec, degraus); custo > pmAtual {
		return false, "PM insuficiente"
	}
	return true, ""
}
