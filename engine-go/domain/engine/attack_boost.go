package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// O IMPULSO DE PM NO ATAQUE (ALE-487).
//
// "Quando faz um ataque, você pode gastar N PM para …" é a forma do Golpe Divino
// (p82) e do Ataque Especial (p65): o PM de base compra o bônus, e "a cada
// quatro níveis, pode gastar +1 PM para aumentar" — o mesmo degrau das posturas
// (`StepsAt`). O verbete o escreve em `boost`, fora dos modificadores comuns,
// porque ele não vale na ficha: vale no golpe em que o jogador o paga.
//
// O motor confere e devolve; quem COBRA é a proposta, e o cancelamento do mestre
// devolve (decisão do dono) — um provisório cancelado não custa nada.

// ClassPowerBoost é a oferta do verbete: o preço, o teto que cresce com o nível,
// o que cada degrau soma, e as duas travas do livro.
type ClassPowerBoost struct {
	Degrees   BoostDegrees `json:"degrees"`
	Modifiers []Modifier   `json:"modifiers"`
	// MeleeOnly é o "quando faz um ataque CORPO A CORPO" do Golpe Divino: a
	// carta à distância fica sem o impulso.
	MeleeOnly bool `json:"meleeOnly,omitempty"`
	// Split é o "receber +4 no teste de ataque OU na rolagem de dano […] você
	// pode dividir os bônus igualmente" do Ataque Especial: o jogador diz para
	// onde vai, e o pedido sem isso é recusado.
	Split bool `json:"split,omitempty"`
}

// BoostDegrees é o preço e o teto: base, PM por degrau, e de quantos em quantos
// níveis um degrau novo abre.
type BoostDegrees struct {
	BasePM          int `json:"basePm"`
	StepPM          int `json:"stepPm"`
	FirstStepLevel  int `json:"firstStepLevel"`
	StepEveryLevels int `json:"stepEveryLevels"`
}

// StepsAt são os degraus A MAIS que o nível na classe abre: zero antes do
// primeiro, e um a cada `every` níveis dali em diante. É a conta das posturas
// (p40) e a do impulso, num lugar só.
//
//	StepsAt(5, 4, 9) // 2
func StepsAt(first, every, classLevel int) int {
	if every <= 0 || classLevel < first {
		return 0
	}
	return 1 + (classLevel-first)/every
}

// Para onde vai o bônus de um impulso que se divide.
const (
	BoostToAttack    = "attack"
	BoostToDamage    = "damage"
	BoostSplitEvenly = "half"
)

// BoostChoice é o que o jogador pediu: o poder, os degraus A MAIS, e, no
// Ataque Especial, para onde vai o bônus. O poder vazio é o ataque sem impulso.
type BoostChoice struct {
	PowerID string
	Steps   int
	Split   string
	// AddOn é o acréscimo pago pedido junto (o Romper Resistências no Ataque Especial, p66), ou vazio.
	AddOn string
}

// BoostedWeaponCards são as cartas de arma com o impulso pedido, cada uma com o PM que ele custa NELA
// (`BoostPM`): o mesmo em toda arma, salvo a Harmonizada, que o corta (p165). O pedido que o livro não
// permite volta como erro que diz o porquê.
//
//	cards, err := world.BoostedWeaponCards(ch, nil,
//		BoostChoice{PowerID: "class.paladino.golpe-divino-1d8", Steps: 1})
func (r *Ruleset) BoostedWeaponCards(ch Character, activeConditionals map[string]bool, choice BoostChoice) ([]WeaponCard, error) {
	plain := r.ComputeWeaponCards(ch, activeConditionals)
	if choice.PowerID == "" {
		return plain, nil
	}
	if _, enchant, isEnchant := enchantBoostOf(choice.PowerID); isEnchant {
		cards, pm, err := r.enchantBoostedCards(ch, activeConditionals, choice.PowerID)
		for i := range cards {
			pricedAt(&cards[i], r.boostCostWith(ch, cards[i].ItemID, enchant, pm))
		}
		return cards, err
	}
	power, classLevel := r.ownedBoostPower(ch, choice.PowerID)
	if power == nil {
		return nil, fmt.Errorf("o personagem não tem um poder que se paga no ataque com o id %q", choice.PowerID)
	}
	boost := power.Boost
	maxSteps := StepsAt(boost.Degrees.FirstStepLevel, boost.Degrees.StepEveryLevels, classLevel)
	if choice.Steps < 0 || choice.Steps > maxSteps {
		return nil, fmt.Errorf("%s: %d degraus a mais pedidos, e o %dº nível de %s abre %d",
			power.Name, choice.Steps, classLevel, power.ClassName, maxSteps)
	}
	if boost.Split && choice.Split != BoostToAttack && choice.Split != BoostToDamage && choice.Split != BoostSplitEvenly {
		return nil, fmt.Errorf("%s: diga para onde vai o bônus (%q, %q ou %q), e veio %q",
			power.Name, BoostToAttack, BoostToDamage, BoostSplitEvenly, choice.Split)
	}
	mods := boostModifiers(boost, 1+choice.Steps, choice.Split, classLevel)
	item := ActiveItem{SourceID: power.ID, Source: power.Name, Equipped: &vestedWear, Modifiers: mods}
	items := []ActiveItem{item}
	addOnPM := 0
	if choice.AddOn != "" {
		addOn := r.ownedBoostAddOn(ch, choice.AddOn, power.ID)
		if addOn == nil {
			return nil, fmt.Errorf("%s: o acréscimo %q não é de um poder que a ficha tenha para este impulso", power.Name, choice.AddOn)
		}
		items = append(items, ActiveItem{SourceID: addOn.ID, Source: addOn.Name, Equipped: &vestedWear, Modifiers: addOn.BoostAddOn.Modifiers})
		addOnPM = addOn.BoostAddOn.PM
	}
	boosted := r.weaponCardsWith(ch, activeConditionals, items)
	if boost.MeleeOnly {
		for i := range boosted {
			if i < len(plain) {
				boosted[i] = refusingRangedUses(boosted[i], plain[i])
			}
		}
	}
	pm := boost.Degrees.BasePM + choice.Steps*boost.Degrees.StepPM + addOnPM
	for i := range boosted {
		pricedAt(&boosted[i], r.boostCostWith(ch, boosted[i].ItemID, power.ID, pm))
	}
	return boosted, nil
}

// pricedAt escreve o custo do impulso na carta e no outro uso dela: a azagaia arremessada paga o mesmo que a
// azagaia que bate.
func pricedAt(card *WeaponCard, pm int) {
	card.BoostPM = pm
	if card.OtherUse != nil {
		card.OtherUse.BoostPM = pm
	}
}

// boostCostWith é o custo do impulso `powerID` pago com a arma `itemID`: o `pm` de toda arma, menos o corte
// que ela leva para aquele poder (a Harmonizada, p165). No gasto de um encanto, `powerID` é o id do encanto,
// que é o que a Harmonizada guarda: a arma já é a dela.
//
// Os cortes NÃO SE ACUMULAM e o custo não desce de 1 PM — a regra do custo da magia (p226), que o livro dá
// para o custo de qualquer habilidade. Vale o maior corte, e não a soma, porque duas fontes de "–1 PM" na
// mesma habilidade são o caso que a p226 nomeia.
func (r *Ruleset) boostCostWith(ch Character, itemID int64, powerID string, pm int) int {
	if itemID == 0 {
		return pm
	}
	cut := 0
	for _, it := range ch.Items {
		if it.ID != itemID || it.CatalogID == nil {
			continue
		}
		for _, m := range r.ownItemMods(it, r.getCatalogItem(*it.CatalogID), devotionOf(ch)) {
			if m.Target.K == "boostCost" && m.Target.Name == powerID {
				cut = min(cut, m.Amount)
			}
		}
	}
	if cut == 0 {
		return pm
	}
	return max(1, pm+cut)
}

// BoostOffer é um impulso que a ficha pode pagar num golpe, com o teto de
// degraus que o nível na classe abre — o que a tela precisa para oferecê-lo.
type BoostOffer struct {
	PowerID   string
	Name      string
	BasePM    int
	StepPM    int
	MaxSteps  int
	Split     bool
	MeleeOnly bool
	// Buys é o que o impulso compra com N degraus a mais, de zero ao teto:
	// "+1d8", "+2d8"… no Golpe Divino, "+4", "+8"… no Ataque Especial. É o
	// rótulo do contador, saído dos mesmos modificadores que a carta soma.
	Buys []string
	// ItemID é a arma do gasto de um ENCANTO (p335-336): ele só vale nela. Zero no impulso de poder, que
	// vale em toda arma que o livro deixa. HitPointCost é o dado de PV que o portador perde ao gastar.
	ItemID       int64
	HitPointCost string
	// Against é o tipo de criatura (ou a raça) contra o qual o gasto vale, escolhido no encanto da arma (a
	// Anticriatura, p335). Vazio no gasto que vale contra qualquer alvo.
	Against string
	// Foresees é o que o golpe pago prevê no acerto (a condição da Venenosa, o efeito da Corrosiva), para a
	// tela dizer antes de o jogador pagar.
	Foresees []AnnouncedCondition
	// AddOns são os acréscimos pagos que a ficha tem para este impulso (o Romper Resistências no Ataque
	// Especial, p66).
	AddOns []BoostAddOnOffer
}

// BoostOffers são os impulsos de PM que a ficha tem, na ordem das classes.
//
//	world.BoostOffers(paladin) // [{PowerID: "class.paladino.golpe-divino-1d8", BasePM: 2, …}]
func (r *Ruleset) BoostOffers(ch Character) []BoostOffer {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	out := []BoostOffer{}
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			if power.Boost == nil {
				continue
			}
			d := power.Boost.Degrees
			offer := BoostOffer{
				PowerID: power.ID, Name: power.Name, BasePM: d.BasePM, StepPM: d.StepPM,
				MaxSteps:  StepsAt(d.FirstStepLevel, d.StepEveryLevels, entry.Level),
				Split:     power.Boost.Split,
				MeleeOnly: power.Boost.MeleeOnly,
			}
			for steps := 0; steps <= offer.MaxSteps; steps++ {
				offer.Buys = append(offer.Buys, boostBuys(power.Boost, 1+steps))
			}
			for _, addOn := range r.ownedBoostAddOns(ch, power.ID) {
				offer.AddOns = append(offer.AddOns, BoostAddOnOffer{
					PowerID: addOn.ID, Name: addOn.Name, PM: addOn.BoostAddOn.PM, Buys: addOnBuys(addOn.BoostAddOn),
				})
			}
			out = append(out, offer)
		}
	}
	return append(out, r.enchantBoostOffers(ch)...)
}

// ownedBoostPower é o poder de impulso que a ficha tem com esse id, e o nível na
// classe dele.
func (r *Ruleset) ownedBoostPower(ch Character, id string) (*ClassPower, int) {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			if power.ID == id && power.Boost != nil {
				return power, entry.Level
			}
		}
	}
	return nil, 0
}

// boostModifiers são os modificadores do impulso com `degrees` degraus PAGOS
// (o de base conta como um).
//
// O número e o dado multiplicam pelo degrau — +4 vira +20 no 5º, 1d8 vira 5d8;
// a soma de atributo (o Carisma do Golpe Divino) não, porque o livro a dá uma
// vez. No impulso que se divide, o bônus vai inteiro para um lado ou metade
// para cada.
func boostModifiers(boost *ClassPowerBoost, degrees int, split string, classLevel int) []Modifier {
	out := []Modifier{}
	for _, m := range boost.Modifiers {
		m.classLevel = classLevel
		if m.Scale != nil && m.Scale.Per == "attribute" {
			out = append(out, m)
			continue
		}
		share := degrees
		if boost.Split {
			switch {
			case split == BoostSplitEvenly:
				m.Amount = m.Amount * degrees / 2
				out = append(out, m)
				continue
			case (split == BoostToAttack) != (m.Target.K == "attack"):
				continue
			}
		}
		m.Amount *= share
		if m.Dice != "" {
			m.Dice = MultiplyDice(m.Dice, share)
		}
		out = append(out, m)
	}
	return out
}

// boostBuys é o que `degrees` degraus pagos compram, escrito: o dado ou o
// número do primeiro modificador que cresce com o degrau. No impulso que se
// divide, é o bônus inteiro, antes de escolher o lado.
func boostBuys(boost *ClassPowerBoost, degrees int) string {
	for _, m := range boostModifiers(boost, degrees, BoostToAttack, 0) {
		if m.Scale != nil {
			continue
		}
		if m.Dice != "" {
			return "+" + m.Dice
		}
		return "+" + strconv.Itoa(m.Amount)
	}
	return ""
}

// MultiplyDice escreve "NdF" com N vezes `by` — 1d8 com 3 degraus é 3d8.
func MultiplyDice(dice string, by int) string {
	count, faces, ok := strings.Cut(dice, "d")
	n, err := strconv.Atoi(count)
	if !ok || err != nil {
		return dice
	}
	return strconv.Itoa(n*by) + "d" + faces
}
