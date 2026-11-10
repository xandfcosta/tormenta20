package engine

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// O GASTO DE 2 PM DOS ENCANTOS (p335-336).
//
// "Uma vez por rodada, quando ataca, você pode gastar 2 PM. Se fizer isso…" é o impulso do Golpe Divino
// com outra origem: a ARMA que tem o encanto. Por isso ele entra pelo mesmo caminho — a oferta na linha
// da arma (`BoostOffers`), a carta paga (`BoostedWeaponCards`) e a cobrança na proposta —, e não por um
// parâmetro novo da resolução do ataque. O que muda é de onde ele vem e em que carta vale: só na da arma
// que tem o encanto.

// ItemBoost é o gasto do verbete do encanto: o PM, o que ele dá no golpe pago, e o dado de PV que o
// portador perde ao gastar (a Tumular).
type ItemBoost struct {
	PM           int        `json:"pm"`
	Modifiers    []Modifier `json:"modifiers"`
	HitPointCost string     `json:"hitPointCost,omitempty"`
}

// enchantBoostPrefix marca o id de um gasto de encanto no lugar do id do poder de classe.
const enchantBoostPrefix = "encanto:"

// EnchantBoostID é o id do gasto de um encanto numa arma: a mesma escolha (`BoostChoice.PowerID`) que
// carrega o poder do Golpe Divino carrega "o encanto X da arma Y". A arma vai no id porque duas armas
// podem ter o mesmo encanto, e o gasto é de uma só.
//
//	EnchantBoostID(7, "encanto-tumular") // "encanto:7:encanto-tumular"
func EnchantBoostID(itemID int64, enchant string) string {
	return enchantBoostPrefix + strconv.FormatInt(itemID, 10) + ":" + enchant
}

// EnchantBoostItem é a arma de um gasto de encanto, e se o id é de um encanto: quem propõe o golpe confere
// que ele é com essa arma.
func EnchantBoostItem(id string) (int64, bool) {
	itemID, _, isEnchant := enchantBoostOf(id)
	return itemID, isEnchant
}

// enchantBoostOf lê o id de volta: a arma e o encanto, e se o id é de um encanto.
func enchantBoostOf(id string) (int64, string, bool) {
	rest, isEnchant := strings.CutPrefix(id, enchantBoostPrefix)
	if !isEnchant {
		return 0, "", false
	}
	item, enchant, found := strings.Cut(rest, ":")
	itemID, err := strconv.ParseInt(item, 10, 64)
	if !found || err != nil {
		return 0, "", false
	}
	return itemID, enchant, true
}

// enchantBoostOffers são os gastos que as armas EMPUNHADAS oferecem, um por encanto com gasto.
func (r *Ruleset) enchantBoostOffers(ch Character) []BoostOffer {
	out := []BoostOffer{}
	for _, it := range ch.Items {
		if it.Equipped == nil || (*it.Equipped != "wielded" && *it.Equipped != "wielded2") {
			continue
		}
		for _, enchant := range it.Enchants {
			catalog := r.getCatalogItem(enchant)
			if catalog == nil || catalog.Boost == nil {
				continue
			}
			out = append(out, BoostOffer{
				PowerID: EnchantBoostID(it.ID, enchant), Name: catalog.Name, BasePM: catalog.Boost.PM,
				ItemID: it.ID, HitPointCost: catalog.Boost.HitPointCost,
				Buys: []string{enchantBoostBuys(catalog.Boost)},
			})
		}
	}
	return out
}

// enchantBoostBuys é o que o gasto compra, escrito para o rótulo: o dado a mais ("+1d8"), ou vazio
// quando ele só anuncia.
func enchantBoostBuys(boost *ItemBoost) string {
	for _, m := range boost.Modifiers {
		if m.Dice != "" {
			return "+" + m.Dice
		}
	}
	return ""
}

// enchantBoostedCards são as cartas com o gasto do encanto pago na arma dele.
//
// Os modificadores entram como uma fonte a mais, presos à arma do encanto pelo escopo dela: o dado da
// Tumular soma na espada, e não na adaga da outra mão. O anúncio não tem escopo — a carta de qualquer
// arma o leria —, e quem garante que o golpe pago é o da arma do encanto é a proposta.
func (r *Ruleset) enchantBoostedCards(ch Character, activeConditionals map[string]bool, id string) ([]WeaponCard, int, error) {
	itemID, enchant, _ := enchantBoostOf(id)
	for _, it := range ch.Items {
		if it.ID != itemID || it.CatalogID == nil {
			continue
		}
		if !slices.Contains(it.Enchants, enchant) {
			break
		}
		catalog := r.getCatalogItem(enchant)
		if catalog == nil || catalog.Boost == nil {
			break
		}
		scope := WeaponDamageScope(*it.CatalogID)
		mods := make([]Modifier, 0, len(catalog.Boost.Modifiers))
		for _, m := range catalog.Boost.Modifiers {
			if m.Target.K == "damage" && (m.Target.Scope == "" || m.Target.Scope == "this") {
				m.Target.Scope = scope
			}
			mods = append(mods, m)
		}
		source := ActiveItem{SourceID: enchant, Source: catalog.Name, Equipped: &vestedWear, Modifiers: mods}
		return r.weaponCardsWith(ch, activeConditionals, []ActiveItem{source}), catalog.Boost.PM, nil
	}
	return nil, 0, fmt.Errorf("a arma %d não tem um encanto que se paga no ataque com o id %q", itemID, enchant)
}
