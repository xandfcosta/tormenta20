package sheetui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O IMPULSO DE PM NA LINHA DE ATAQUE (ALE-487).
//
// O Golpe Divino (p82) e o Ataque Especial (p65) são escolha DESTE golpe, e por
// isso moram na linha da arma, embaixo do número — e não na tira "com" do
// grupo, que é do que DURA (postura, situacional).
//
// OS NÚMEROS COM O IMPULSO VÊM PRONTOS DO SERVIDOR, um por degrau e por lado:
// a tela só escolhe qual mostrar. Somar no cliente seria a segunda
// implementação da regra, e o Carisma do Golpe Divino, que não cresce com o
// degrau, é exatamente o tipo de detalhe que a cópia erraria. Quem cobra e
// confere de verdade é o `combat`, com o motor.

// Os sinais do impulso. Toda escrita deles começa zerando os quatro, porque a
// Mesa não os declara: é a escrita que os cria (o mesmo do `stance_degrees`).
const (
	boostWeaponSignal = "boost_weapon"
	boostPowerSignal  = "boost_power"
	boostStepsSignal  = "boost_steps"
	boostSplitSignal  = "boost_split"
)

// attackBoost é um impulso oferecido numa arma.
type attackBoost struct {
	PowerID string
	Name    string
	// Can é ter PM para a base; Why é a frase quando não tem.
	Can bool
	Why string
	// MaxSteps é o teto do nível JÁ cortado pelo PM: o "+" não oferece o degrau
	// que a ficha não paga.
	MaxSteps int
	BasePM   int
	StepPM   int
	Split    bool
	// Buys é o rótulo de cada contagem de degraus ("+2d8"), do motor.
	Buys []string
	// Previews são os números da linha com o impulso, pela chave
	// `degraus|lado` ("1|" no que não divide, "2|half" no que divide).
	Previews map[string]boostPreview
}

// boostPreview é o que a linha mostra com um impulso escolhido.
type boostPreview struct {
	Attack string `json:"attack"`
	Detail string `json:"detail"`
}

// boostSplits são os lados de um impulso, na ordem da tela.
var boostSplits = []struct{ Value, Label string }{
	{engine.BoostToAttack, "no ataque"},
	{engine.BoostToDamage, "no dano"},
	{engine.BoostSplitEvenly, "metade"},
}

// attackBoostsFor são os impulsos de cada arma, pelo índice da carta.
//
// A arma à distância não recebe o que só vale corpo a corpo (o Golpe Divino),
// e a ficha sem impulso nenhum devolve nil — a linha não desenha a tira.
func attackBoostsFor(dto sheet.CharacterDTO, cards []engine.WeaponCard) [][]attackBoost {
	if dto.Ruleset == nil {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return nil
	}
	offers := dto.Ruleset.BoostOffers(ec)
	if len(offers) == 0 {
		return nil
	}
	active := sheet.ToStringSet(dto.Conditionals)
	mana := int(dto.ManaAvailable())
	out := make([][]attackBoost, len(cards))
	for _, offer := range offers {
		boost := attackBoost{
			PowerID: offer.PowerID, Name: offer.Name, BasePM: offer.BasePM, StepPM: offer.StepPM,
			Split: offer.Split, Buys: offer.Buys, Can: offer.BasePM <= mana, Previews: map[string]boostPreview{},
		}
		if !boost.Can {
			boost.Why = fmt.Sprintf("falta PM: %s pede %d", offer.Name, offer.BasePM)
		}
		boost.MaxSteps = offer.MaxSteps
		for boost.MaxSteps > 0 && offer.BasePM+boost.MaxSteps*offer.StepPM > mana {
			boost.MaxSteps--
		}
		previews := boostPreviewsOf(dto.Ruleset, ec, active, offer, boost.MaxSteps, cards)
		for i := range cards {
			if offer.MeleeOnly && !cards[i].IsMelee() {
				continue
			}
			weaponBoost := boost
			weaponBoost.Previews = previews[i]
			out[i] = append(out[i], weaponBoost)
		}
	}
	return out
}

// boostPreviewsOf computa as cartas com o impulso para cada degrau e lado, e
// devolve, por arma, a prévia de cada combinação.
func boostPreviewsOf(
	world *engine.Ruleset, ec engine.Character, active map[string]bool,
	offer engine.BoostOffer, maxSteps int, plain []engine.WeaponCard,
) []map[string]boostPreview {
	out := make([]map[string]boostPreview, len(plain))
	for i := range out {
		out[i] = map[string]boostPreview{}
	}
	sides := []string{""}
	if offer.Split {
		sides = []string{}
		for _, s := range boostSplits {
			sides = append(sides, s.Value)
		}
	}
	for steps := 0; steps <= maxSteps; steps++ {
		for _, side := range sides {
			cards, _, err := world.BoostedWeaponCards(ec, active,
				engine.BoostChoice{PowerID: offer.PowerID, Steps: steps, Split: side})
			if err != nil {
				continue
			}
			key := strconv.Itoa(steps) + "|" + side
			for i := range plain {
				if i >= len(cards) {
					break
				}
				out[i][key] = boostPreview{
					Attack: book.WithSign(cards[i].Attack),
					Detail: boostedWeaponDetail(cards[i], plain[i]),
				}
			}
		}
	}
	return out
}

// boostedWeaponDetail é a linha pequena sob o nome da arma — "1d8+3 · 20/x3 · Luta" —,
// com os dados que o impulso pôs além dos da carta sem ele.
func boostedWeaponDetail(card, plain engine.WeaponCard) string {
	damage := damageLabel(card)
	if len(card.ExtraDamage) > len(plain.ExtraDamage) {
		for _, extra := range card.ExtraDamage[len(plain.ExtraDamage):] {
			damage += " +" + extra.Dice
		}
	}
	return damage + " · " + critLabel(card) + " · " + card.Skill
}

// boostChosenExpr é a expressão que diz se ESTE impulso está escolhido nesta
// arma.
func boostChosenExpr(weapon int, powerID string) string {
	return fmt.Sprintf("$%s === %d && $%s === %q", boostWeaponSignal, weapon, boostPowerSignal, powerID)
}

// boostToggleCommand liga o impulso nesta arma, ou o desliga se já estava.
// Ligar zera o degrau e põe o lado em "no ataque", que é o que o motor recusaria
// vazio.
func boostToggleCommand(weapon int, boost attackBoost) string {
	return fmt.Sprintf("if (%s) { $%s = '' } else { $%s = %d; $%s = %q; $%s = 0; $%s = %q }",
		boostChosenExpr(weapon, boost.PowerID), boostPowerSignal,
		boostWeaponSignal, weapon, boostPowerSignal, boost.PowerID,
		boostStepsSignal, boostSplitSignal, splitDefault(boost))
}

func splitDefault(boost attackBoost) string {
	if boost.Split {
		return engine.BoostToAttack
	}
	return ""
}

// boostStepCommand anda o degrau em `delta`, dentro de zero e do teto.
func boostStepCommand(boost attackBoost, delta int) string {
	return fmt.Sprintf("$%s = Math.max(0, Math.min(%d, $%s + %d))",
		boostStepsSignal, boost.MaxSteps, boostStepsSignal, delta)
}

// boostStepLabelExpr é o que o contador mostra: o que o degrau compra e o PM
// total — "+2d8 · 3 PM", "+8 · 2 PM".
func boostStepLabelExpr(boost attackBoost) string {
	return fmt.Sprintf("%s + ' · ' + (%d + %d * $%s) + ' PM'",
		boostAmountExpr(boost), boost.BasePM, boost.StepPM, boostStepsSignal)
}

// boostAmountExpr escreve o que o degrau escolhido compra, pela tabela que o
// motor mandou.
func boostAmountExpr(boost attackBoost) string {
	blob, _ := json.Marshal(boost.Buys)
	return fmt.Sprintf("%s[$%s]", blob, boostStepsSignal)
}

// boostPreviewExpr é a expressão do texto da linha: a prévia do impulso
// escolhido nesta arma, ou o texto sem impulso.
func boostPreviewExpr(weapon int, boosts []attackBoost, field, plain string) string {
	expr := strconv.Quote(plain)
	for _, boost := range boosts {
		table := map[string]string{}
		for key, preview := range boost.Previews {
			if field == "attack" {
				table[key] = preview.Attack
			} else {
				table[key] = preview.Detail
			}
		}
		blob, _ := json.Marshal(table)
		expr = fmt.Sprintf("(%s ? (%s[$%s + '|' + $%s] ?? %s) : %s)",
			boostChosenExpr(weapon, boost.PowerID), blob,
			boostStepsSignal, boostSplitSignal, strconv.Quote(plain), expr)
	}
	return expr
}

// boostOnExpr diz se ALGUM impulso está escolhido nesta arma.
func boostOnExpr(weapon int) string {
	return fmt.Sprintf("$%s === %d && $%s !== ''", boostWeaponSignal, weapon, boostPowerSignal)
}

// boostCostExpr é o PM do impulso escolhido nesta arma, para a etiqueta ao lado
// do número.
func boostCostExpr(weapon int, boosts []attackBoost) string {
	parts := []string{}
	for _, boost := range boosts {
		parts = append(parts, fmt.Sprintf("(%s ? %d + %d * $%s : 0)",
			boostChosenExpr(weapon, boost.PowerID), boost.BasePM, boost.StepPM, boostStepsSignal))
	}
	return "(" + strings.Join(parts, " + ") + ") + ' PM'"
}

// boostSplitCommand escolhe o lado.
func boostSplitCommand(side string) string {
	return fmt.Sprintf("$%s = %q", boostSplitSignal, side)
}

// boostSplitIsExpr diz se o lado escolhido é este.
func boostSplitIsExpr(side string) string {
	return fmt.Sprintf("$%s === %q", boostSplitSignal, side)
}

// chosenLookExpr é o `data-class` de objeto que alterna as classes de ligado e
// de desligado. As DUAS alternam: com as de desligado fixas no `class`, a
// cascata as fazia vencer e o chip ligado continuava cinza (visto na tela). E é
// de objeto porque a forma de chave única não aceita o `/` de
// `bg-grimorio-gold/10`.
func chosenLookExpr(on, off, when string) string {
	return fmt.Sprintf("{%q: %s, %q: !(%s)}", on, when, off, when)
}

// ariaBoolExpr escreve a condição como "true"/"false": o `data-attr` põe o
// verdadeiro como atributo VAZIO, e `aria-pressed=""` não diz nada a ninguém.
func ariaBoolExpr(when string) string {
	return "String(" + when + ")"
}

// rollClearsTheBoost é o que o "Rolar" do ataque faz depois de mandar o golpe:
// a escolha volta a "sem", e o próximo golpe paga de novo se quiser.
func rollClearsTheBoost() string {
	return "; $" + boostPowerSignal + " = ''"
}
