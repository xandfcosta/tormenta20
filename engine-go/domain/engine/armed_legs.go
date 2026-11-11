package engine

import "slices"

// AS PATAS ARMADAS da Membros Extras (p137): "Se possuir Ambidestria ou Estilo de Duas Armas, pode empunhar
// armas leves em suas patas insetoides (mas ainda precisa pagar 2 PM para atacar com elas e sofre a
// penalidade de –2 em todos os ataques)."
//
// A pata está armada quando tem arma nela, e não por um interruptor (decisão do dono): o –2 vale enquanto
// houver arma numa pata, e sai quando ela é guardada. A arma na pata ataca pelo mesmo ataque extra das
// patas naturais — a mesma fonte, o mesmo pagamento de 2 PM e o mesmo limite de dois por rodada.

// LegSlot é a pata insetoide como lugar de empunhar.
const LegSlot = "leg"

// ExtraLimbsPowerID é o poder da Tormenta que dá as patas.
const ExtraLimbsPowerID = "membros-extras"

// LegsLimit é quantas patas a Membros Extras dá.
const LegsLimit = 2

// armedLegPenalty é o –2 "em todos os ataques" de quem tem arma numa pata.
const armedLegPenalty = -2

type armedLeg struct {
	item    CharacterItem
	catalog *CatalogItem
}

// CanArmTheLegs diz se a ficha pode empunhar nas patas: a Membros Extras e, junto, a Ambidestria ou o
// Estilo de Duas Armas. É a pergunta da Mochila, que oferece a pata e recusa o gesto.
//
//	world.CanArmTheLegs(ch) // true com `membros-extras` e `estilo-de-duas-armas`
func (r *Ruleset) CanArmTheLegs(ch Character) bool {
	return r.canArmTheLegs(ch, grantsFlag(r.ActiveItemsFor(ch), "ambidestria"))
}

// canArmTheLegs é a mesma pergunta com o "possui Ambidestria" já respondido: a flag `ambidestria`, que
// os verbetes de classe escrevem. Quem já tem as fontes ou os efeitos não as coleta de novo.
func (r *Ruleset) canArmTheLegs(ch Character, ambidextrous bool) bool {
	powers := r.chosenGeneralPowers(ch)
	hasLimbs := powers.has[ExtraLimbsPowerID] || slices.Contains(r.originPickedPowerIds(ch), ExtraLimbsPowerID)
	return hasLimbs && (powers.has[TwoWeaponStylePowerID] || ambidextrous)
}

// FitsALeg diz se a arma cabe numa pata: corpo a corpo e leve.
//
//	FitsALeg("melee", "light") // true: a adaga
func FitsALeg(purpose, hand string) bool {
	return purpose == "melee" && hand == "light"
}

// armedLegs são as armas nas patas que valem, até duas. A que não serve — arma que não é leve, ou a
// ficha que perdeu o estilo — fica guardada para a regra, e a pata continua natural.
func (r *Ruleset) armedLegs(ch Character, ambidextrous bool) []armedLeg {
	if !r.canArmTheLegs(ch, ambidextrous) {
		return nil
	}
	out := []armedLeg{}
	for _, it := range ch.Items {
		if it.Equipped == nil || *it.Equipped != LegSlot || it.CatalogID == nil {
			continue
		}
		catalog := r.getCatalogItem(*it.CatalogID)
		if catalog == nil || catalog.Weapon == nil || !FitsALeg(catalog.Weapon.Purpose, catalog.Weapon.Hand) {
			continue
		}
		out = append(out, armedLeg{item: it, catalog: catalog})
		if len(out) == LegsLimit {
			break
		}
	}
	return out
}

// armedLegsAsAnItem é o –2 como fonte da ficha, como o `handsAsAnItem`: ele não é de item nenhum, e sim do
// estado das patas. `items` são as fontes já coletadas, de onde sai o "possui Ambidestria".
func (r *Ruleset) armedLegsAsAnItem(ch Character, items []ActiveItem) []ActiveItem {
	if len(r.armedLegs(ch, grantsFlag(items, "ambidestria"))) == 0 {
		return nil
	}
	return []ActiveItem{{SourceID: ExtraLimbsPowerID, Source: "Membros Extras (patas armadas)", Equipped: &vestedWear,
		Modifiers: []Modifier{{Target: ModifierTarget{K: "attack", Scope: "all"}, Amount: armedLegPenalty, BonusType: "untyped"}}}}
}

// grantsFlag diz se alguma fonte levanta a flag sem condição.
func grantsFlag(items []ActiveItem, flag string) bool {
	for _, it := range items {
		for _, m := range it.Modifiers {
			if m.Target.K == "flag" && m.Target.Name == flag && m.Amount > 0 && m.Condition == nil {
				return true
			}
		}
	}
	return false
}

// legCards são as cartas das armas nas patas. Elas atacam pelo extra das patas naturais: a mesma
// fonte, que divide o pagamento e o limite.
func (r *Ruleset) legCards(ch Character, b cardBasis, limbs NaturalStrike) []WeaponCard {
	cards := []WeaponCard{}
	for _, leg := range r.armedLegs(ch, b.effects.Flags["ambidestria"]) {
		card := r.itemCard(ch, leg.item, leg.catalog, LegSlot, b.effects, b)
		strike := limbs
		card.Natural = &strike
		cards = append(cards, card)
	}
	return cards
}
