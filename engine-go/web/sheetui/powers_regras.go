package sheetui

import (
	"encoding/json"
	"strconv"
	"t20engine/book"
)

// AS REGRAS DE ATIVAR UM PODER (ALE-272, fatia 8).
//
// Elas vinham do `power-rules.ts`, que era o metade-regra do antigo hook de
// React: decidir se um poder pode ser usado agora, qual limite o prende, e
// quanto custa entrar numa postura de degraus.
//
// # O que é COBRADO e o que é só crachá
//
// "1/cena" e "1/dia" são cobrados — eles têm contador no banco desde a ALE-222.
// Um "3/dia" e um "1/rodada" saem como crachá e nada mais, e isso é decisão
// registrada, não esquecimento: a mesa conta rodadas, a ficha não. Decisão do
// dono, ALE-272 fatia 8: fica como está.

// O REGISTRO em si mora no `book` (`activations.go`) desde a ALE-278: ele é
// lido do catálogo, e ler catálogo é do livro. O que segue é o que a TELA decide
// a partir dele.

// ── o LIMITE de usos ─────────────────────────────────────────────────────────

// chargedScope é "scene", "day" ou "" — e "" quer dizer que o limite existe
// no livro e a ficha NÃO o cobra.
func chargedScope(spec book.Activation) string {
	switch string(spec.Uses) {
	case `"cena"`:
		return "scene"
	case `"dia"`:
		return "day"
	}
	return ""
}

// limitBadge é o que a tela escreve do limite: "1/cena", "3/dia", ou "".
func limitBadge(spec book.Activation) string {
	cru := string(spec.Uses)
	switch cru {
	case "", "null":
		return ""
	case `"cena"`:
		return "1/cena"
	case `"dia"`:
		return "1/dia"
	case `"rodada"`:
		return "1/rodada"
	}
	var numero int
	if json.Unmarshal(spec.Uses, &numero) == nil {
		return strconv.Itoa(numero) + "/dia"
	}
	return ""
}

// costVariableEh diz se o PM do poder não é um número — "PM variável" na tela,
// e o servidor recusa cobrar por ele: quem sabe o total é a mesa.
func costVariableEh(spec book.Activation) bool {
	return activationPm(spec) < 0
}

// activationPm é o custo em PM, ou -1 quando ele é variável.
//
// Menos um e não zero: zero é um custo LEGÍTIMO (a maioria das passivas), e
// confundir os dois é o que faria um poder de graça ser tratado como negociado
// com a mesa.
func activationPm(spec book.Activation) int {
	var numero int
	if json.Unmarshal(spec.PmCost, &numero) == nil {
		return numero
	}
	return -1
}

// ── a DECISÃO de usar ────────────────────────────────────────────────────────

// useDecision responde se o poder pode ser usado AGORA, e por que não.
//
// A ordem das recusas é a da tela antiga, e ela importa: a razão mostrada é a
// PRIMEIRA que barra, então "requer Fúria" aparece antes de "PM insuficiente"
// num poder que precisa das duas coisas — e é a que a pessoa pode resolver
// primeiro.
func useDecision(spec book.Activation, contexto useContext) (bool, string) {
	if costVariableEh(spec) {
		return false, "custo variável"
	}
	if spec.RequiresFlag != "" && !contexto.Flags[spec.RequiresFlag] {
		return false, "requer " + spec.RequiresFlag
	}
	switch chargedScope(spec) {
	case "scene":
		if contexto.UsadoNaCena >= 1 {
			return false, "limite por cena atingido"
		}
	case "day":
		if contexto.UsadoNoDia >= 1 {
			return false, "limite por dia atingido"
		}
	}
	if activationPm(spec) > contexto.PmAtual {
		return false, "PM insuficiente"
	}
	return true, ""
}

// useContext é o que a decisão precisa saber da ficha AGORA.
type useContext struct {
	PmAtual     int
	UsadoNaCena int
	UsadoNoDia  int
	Flags       map[string]bool
}

// ── a POSTURA de degraus ─────────────────────────────────────────────────────

// levelSteps são os degraus EXTRAS que o nível na classe concede.
//
// O nível é o da CLASSE e não o do personagem (p40): um bárbaro 5/ladino 5 tem
// a Fúria de um bárbaro de nível 5, e não a de um personagem de nível 10.
func levelSteps(escala book.ActivationScale, nivelNaClasse int) int {
	if escala.StepEveryLevels <= 0 || nivelNaClasse < escala.FirstStepLevel {
		return 0
	}
	return 1 + (nivelNaClasse-escala.FirstStepLevel)/escala.StepEveryLevels
}

// stanceCost é o que entrar custa com os degraus escolhidos.
func stanceCost(spec book.Activation, degraus int) int {
	if spec.Scaling == nil {
		return activationPm(spec)
	}
	return spec.Scaling.BasePm + degraus*spec.Scaling.StepPm
}

// stanceDecision responde se dá para entrar na postura com esses degraus.
func stanceDecision(spec book.Activation, degraus, maximo, pmAtual int) (bool, string) {
	if degraus < 0 || degraus > maximo {
		return false, "o nível permite até " + strconv.Itoa(maximo) + " degraus"
	}
	if custo := stanceCost(spec, degraus); custo > pmAtual {
		return false, "PM insuficiente"
	}
	return true, ""
}
