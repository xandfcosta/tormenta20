package sheetui

import (
	"encoding/json"
	"fmt"
	"slices"
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
// degrau, é exatamente o tipo de detalhe que a cópia erraria. Quem confere de
// verdade é o `combat`, com o motor.

// Os sinais do impulso. Toda escrita deles começa zerando os quatro, porque a
// Mesa não os declara: é a escrita que os cria (o mesmo do `stance_degrees`).
const (
	boostWeaponSignal = "boost_weapon"
	boostPowerSignal  = "boost_power"
	boostStepsSignal  = "boost_steps"
	boostSplitSignal  = "boost_split"
	// boostAddOnSignal é o acréscimo pago escolhido junto do impulso (o Romper Resistências, p66), ou vazio.
	boostAddOnSignal = "boost_addon"
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
	// PMs é o preço NESTA arma por degraus a mais, do motor: a arma Harmonizada corta o do poder dela, e o
	// piso de 1 PM tira a conta da linha reta (p165, p226).
	PMs   []int
	Split bool
	// Buys é o rótulo de cada contagem de degraus ("+2d8"), do motor e DESTA arma: a Arma Sagrada rola d12 só
	// na arma preferida (p82).
	Buys []string
	// Previews são os números da linha com o impulso, pela chave
	// `degraus|lado` ("1|" no que não divide, "2|half" no que divide).
	Previews map[string]boostPreview
	// HitPointCost é o dado de PV que o gasto também cobra (o 1d8 da Tumular, p336), vazio no resto.
	HitPointCost string
	// OncePerRound diz que o gasto é "uma vez por rodada" (o do encanto, p335-336). Com a iniciativa
	// correndo a proposta recusa o segundo; fora dela não há rodada para contar, e a etiqueta é quem avisa.
	OncePerRound bool
	// AddOns são os acréscimos pagos que a ficha tem para este impulso (o Romper Resistências no Ataque
	// Especial, p66): um interruptor dentro do impulso escolhido, com o PM a mais.
	AddOns []engine.BoostAddOnOffer
	// AddOnPMs é o preço NESTA arma, por degrau, com cada acréscimo ligado — do motor, e não `PMs` mais o PM do
	// acréscimo: a Harmonizada corta o custo inteiro com o piso de 1, e 1 + 1 − 1 é 1, não 2.
	AddOnPMs map[string][]int
	// Foresees é o que o golpe pago prevê no acerto, escrito: "no acerto: Envenenado (1d12 PV por rodada,
	// por 3 rodadas)". Vazio no impulso que só soma número.
	Foresees string
}

// foreseenWritten escreve o que o gasto prevê no acerto, com o nome da condição em português e a nota do
// livro — as mesmas palavras da faixa do mestre.
func foreseenWritten(foresees []engine.AnnouncedCondition) string {
	said := []string{}
	for _, a := range foresees {
		if a.Condition == "" {
			said = append(said, a.Effect)
			continue
		}
		line := book.ConditionName(a.Condition)
		if a.Note != "" {
			line += " (" + a.Note + ")"
		}
		said = append(said, line)
	}
	if len(said) == 0 {
		return ""
	}
	return "no acerto: " + strings.Join(said, ", ")
}

// Price é o que o gasto custa, para a etiqueta da tira: "2 PM", ou "2 PM e 1d8 PV" na Tumular.
func (b attackBoost) Price() string {
	price := strconv.Itoa(b.PMs[0]) + " PM"
	if b.HitPointCost != "" {
		price += " e " + b.HitPointCost + " PV"
	}
	return price
}

// boostPreview é o que a linha mostra com um impulso escolhido.
type boostPreview struct {
	Attack string `json:"attack"`
	Detail string `json:"detail"`
	// OtherUse é o rótulo do botão do outro uso com o impulso, vazio quando o impulso
	// não vale nele: o Golpe Divino não acompanha a lança que voa.
	OtherUse string `json:"otherUse"`
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
		name := offer.Name
		if offer.Against != "" {
			name += " contra " + book.ChosenKindName(offer.Against)
		}
		boost := attackBoost{
			PowerID: offer.PowerID, Name: name, Split: offer.Split, Buys: offer.Buys,
			HitPointCost: offer.HitPointCost, OncePerRound: offer.ItemID != 0, Foresees: foreseenWritten(offer.Foresees),
			AddOns: offer.AddOns,
		}
		previews, prices, buys := boostPreviewsOf(dto.Ruleset, ec, active, offer, cards)
		addOnPrices := addOnPricesOf(dto.Ruleset, ec, active, offer, len(cards))
		for i := range cards {
			if offer.MeleeOnly && !cards[i].IsMelee() {
				continue
			}
			// O GASTO DE UM ENCANTO é da arma dele (p335-336), e só a linha dela o oferece.
			if offer.ItemID != 0 && cards[i].ItemID != offer.ItemID {
				continue
			}
			priced := pricedOn(boost, previews[i], prices[i], mana)
			// O QUE O DEGRAU COMPRA é desta arma: a Arma Sagrada rola d12 só na arma preferida (p82).
			priced.Buys = buys[i]
			priced.AddOnPMs = addOnPrices[i]
			out[i] = append(out[i], priced)
		}
	}
	return out
}

// pricedOn é o impulso na linha de UMA arma, com o preço dela: o que a ficha não paga fica recusado, e o teto
// de degraus desce até o que ela paga.
func pricedOn(boost attackBoost, previews map[string]boostPreview, prices []int, mana int) attackBoost {
	boost.Previews, boost.PMs = previews, prices
	boost.Can = prices[0] <= mana
	if !boost.Can {
		boost.Why = fmt.Sprintf("falta PM: %s pede %d", boost.Name, prices[0])
	}
	boost.MaxSteps = len(prices) - 1
	for boost.MaxSteps > 0 && prices[boost.MaxSteps] > mana {
		boost.MaxSteps--
	}
	return boost
}

// addOnPricesOf é, por arma, o preço de cada degrau com cada acréscimo da oferta ligado. O lado não muda o
// preço, e por isso se pergunta por um só.
func addOnPricesOf(world *engine.Ruleset, ec engine.Character, active map[string]bool, offer engine.BoostOffer, weapons int) []map[string][]int {
	out := make([]map[string][]int, weapons)
	if len(offer.AddOns) == 0 {
		return out
	}
	side := ""
	if offer.Split {
		side = engine.BoostToAttack
	}
	for i := range out {
		out[i] = map[string][]int{}
	}
	for _, addOn := range offer.AddOns {
		for i := range out {
			out[i][addOn.PowerID] = make([]int, offer.MaxSteps+1)
		}
		for steps := 0; steps <= offer.MaxSteps; steps++ {
			cards, err := world.BoostedWeaponCards(ec, active,
				engine.BoostChoice{PowerID: offer.PowerID, Steps: steps, Split: side, AddOn: addOn.PowerID})
			if err != nil {
				continue
			}
			for i := 0; i < weapons && i < len(cards); i++ {
				out[i][addOn.PowerID][steps] = cards[i].BoostPM
			}
		}
	}
	return out
}

// boostPreviewsOf computa as cartas com o impulso para cada degrau e lado, e
// devolve, por arma, a prévia de cada combinação, o preço de cada degrau e o que ele compra.
func boostPreviewsOf(
	world *engine.Ruleset, ec engine.Character, active map[string]bool,
	offer engine.BoostOffer, plain []engine.WeaponCard,
) ([]map[string]boostPreview, [][]int, [][]string) {
	maxSteps := offer.MaxSteps
	out := make([]map[string]boostPreview, len(plain))
	prices := make([][]int, len(plain))
	buys := make([][]string, len(plain))
	for i := range out {
		out[i] = map[string]boostPreview{}
		prices[i] = make([]int, maxSteps+1)
		buys[i] = slices.Clone(offer.Buys)
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
			cards, err := world.BoostedWeaponCards(ec, active,
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
					Attack:   book.WithSign(cards[i].Attack),
					Detail:   boostedWeaponDetail(cards[i]),
					OtherUse: otherUseLabel(cards[i].OtherUse),
				}
				prices[i][steps] = cards[i].BoostPM
				if steps < len(buys[i]) && cards[i].BoostBuys != "" {
					buys[i][steps] = cards[i].BoostBuys
				}
			}
		}
	}
	return out, prices, buys
}

// otherUseVerb é o nome do gesto. Ele sai da perícia, porque o único outro uso em
// corpo a corpo é bater com a azagaia.
func otherUseVerb(other *engine.WeaponCard) string {
	if other.IsMelee() {
		return "Bater"
	}
	return "Arremessar"
}

// otherUseLabel é o botão do outro uso, com o ataque dele: "Arremessar +12".
func otherUseLabel(other *engine.WeaponCard) string {
	if other == nil || other.RefusesBoost {
		return ""
	}
	return otherUseVerb(other) + " " + book.WithSign(other.Attack)
}

// boostedWeaponDetail é a linha pequena sob o nome da arma — "1d8+3 · 20/x3 · Luta" —, com a carta
// impulsionada: os dados que o impulso pôs saem junto dos da arma, com o tipo (`damageLabel`).
func boostedWeaponDetail(card engine.WeaponCard) string {
	return damageLabel(card) + " · " + critLabel(card) + " · " + card.Skill
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
	return fmt.Sprintf("if (%s) { $%s = '' } else { $%s = %d; $%s = %q; $%s = 0; $%s = %q; $%s = '' }",
		boostChosenExpr(weapon, boost.PowerID), boostPowerSignal,
		boostWeaponSignal, weapon, boostPowerSignal, boost.PowerID,
		boostStepsSignal, boostSplitSignal, splitDefault(boost), boostAddOnSignal)
}

// boostAddOnIsExpr diz se o acréscimo `id` está escolhido.
func boostAddOnIsExpr(id string) string {
	return fmt.Sprintf("$%s === %q", boostAddOnSignal, id)
}

// boostAddOnCommand liga o acréscimo, ou o desliga se já estava.
func boostAddOnCommand(id string) string {
	return fmt.Sprintf("$%s = (%s) ? '' : %q", boostAddOnSignal, boostAddOnIsExpr(id), id)
}

// addOnLabel é o rótulo do acréscimo: o que ele compra e o PM a mais — "ignora 10 de RD · +1 PM".
func addOnLabel(addOn engine.BoostAddOnOffer) string {
	price := "+" + strconv.Itoa(addOn.PM) + " PM"
	if addOn.Buys == "" {
		return price
	}
	return addOn.Buys + " · " + price
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
// total — "+2d8 · 3 PM", "+8 · 2 PM". O gasto que só anuncia (a Venenosa) não
// compra número, e fica só com o preço; o da Tumular diz também o PV.
func boostStepLabelExpr(boost attackBoost) string {
	suffix := ""
	if boost.HitPointCost != "" {
		suffix = " e " + boost.HitPointCost + " PV"
	}
	if boost.OncePerRound {
		suffix += " · uma vez por rodada"
	}
	return fmt.Sprintf("((%s) ? %s + ' · ' : '') + %s + ' PM%s'",
		boostAmountExpr(boost), boostAmountExpr(boost), boostPriceExpr(boost), suffix)
}

// boostPriceExpr é o PM do degrau escolhido, pela tabela de preços desta arma — a do acréscimo escolhido, se
// houver um ligado.
func boostPriceExpr(boost attackBoost) string {
	blob, _ := json.Marshal(boost.PMs)
	table := string(blob)
	for _, addOn := range boost.AddOns {
		withAddOn, _ := json.Marshal(boost.AddOnPMs[addOn.PowerID])
		table = fmt.Sprintf("(%s ? %s : %s)", boostAddOnIsExpr(addOn.PowerID), withAddOn, table)
	}
	return fmt.Sprintf("%s[$%s]", table, boostStepsSignal)
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
			switch field {
			case "attack":
				table[key] = preview.Attack
			case "otherUse":
				table[key] = preview.OtherUse
			default:
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
		parts = append(parts, fmt.Sprintf("(%s ? %s : 0)", boostChosenExpr(weapon, boost.PowerID), boostPriceExpr(boost)))
	}
	suffix := []string{}
	for _, boost := range boosts {
		if boost.HitPointCost != "" {
			suffix = append(suffix, fmt.Sprintf("(%s ? ' e %s PV' : '')", boostChosenExpr(weapon, boost.PowerID), boost.HitPointCost))
		}
	}
	if len(suffix) == 0 {
		return "(" + strings.Join(parts, " + ") + ") + ' PM'"
	}
	// O PV da Tumular vai junto do PM (p336): só um impulso se escolhe por vez, e o sufixo é o dele.
	return "(" + strings.Join(parts, " + ") + ") + ' PM' + " + strings.Join(suffix, " + ")
}

// boostSplitCommand escolhe o lado.
func boostSplitCommand(side string) string {
	return fmt.Sprintf("$%s = %q", boostSplitSignal, side)
}

// boostSplitIsExpr diz se o lado escolhido é este.
func boostSplitIsExpr(side string) string {
	return fmt.Sprintf("$%s === %q", boostSplitSignal, side)
}

// ariaBoolExpr escreve a condição como "true"/"false": o `data-attr` põe o
// verdadeiro como atributo VAZIO, e `aria-pressed=""` não diz nada a ninguém.
func ariaBoolExpr(when string) string {
	return "String(" + when + ")"
}

// rollClearsTheBoost é o que o "Rolar" do ataque faz depois de mandar o golpe:
// a escolha volta a "sem", e o próximo golpe paga de novo se quiser.
func rollClearsTheBoost() string {
	return "; $" + boostPowerSignal + " = ''; $" + boostAddOnSignal + " = ''"
}
