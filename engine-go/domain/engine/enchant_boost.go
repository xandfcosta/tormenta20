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
	// AgainstOneOf são as opções do tipo de criatura que o encanto escolhe (a Anticriatura, p335): o gasto
	// só vale contra o tipo escolhido, e quem confere o alvo é o mestre.
	AgainstOneOf []string `json:"againstOneOf,omitempty"`
	// Instead é o que o gasto faz EM VEZ do golpe (a bola de fogo da Flamejante, p335): ele não é oferecido
	// na linha de ataque, e vira um gesto próprio (`EnchantBlasts`).
	Instead *BoostInstead `json:"instead,omitempty"`
}

// BoostInstead é o efeito que o gasto dispara no lugar do golpe: o dano, o alcance e a resistência do alvo,
// com a CD feita do maior atributo da lista (p227).
type BoostInstead struct {
	Name           string   `json:"name"`
	Damage         string   `json:"damage"`
	DamageType     string   `json:"damageType"`
	Range          string   `json:"range"`
	Save           string   `json:"save"`
	SaveAttributes []string `json:"saveAttributes"`
	OnSave         string   `json:"onSave"`
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

// EnchantBoostParts lê o id de um gasto de encanto: a arma e o encanto, e se o id é de um encanto. Quem
// propõe o golpe confere a arma; a faixa do mestre escreve o nome do encanto.
func EnchantBoostParts(id string) (int64, string, bool) { return enchantBoostOf(id) }

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
		if !inAHand(it) {
			continue
		}
		for _, enchant := range it.Enchants {
			catalog := r.getCatalogItem(enchant)
			if catalog == nil || catalog.Boost == nil || catalog.Boost.Instead != nil {
				continue
			}
			out = append(out, BoostOffer{
				PowerID: EnchantBoostID(it.ID, enchant), Name: catalog.Name, BasePM: catalog.Boost.PM,
				ItemID: it.ID, HitPointCost: catalog.Boost.HitPointCost, Against: it.EnchantChoices[enchant],
				Buys: []string{enchantBoostBuys(catalog.Boost)}, Foresees: enchantBoostForesees(catalog.Boost),
			})
		}
	}
	return out
}

// enchantBoostForesees é o que o golpe pago anuncia no acerto, pelas mesmas formas da colheita da carta
// (`harvestWeaponRiders`): a condição com a nota, ou o efeito com a fórmula.
func enchantBoostForesees(boost *ItemBoost) []AnnouncedCondition {
	out := []AnnouncedCondition{}
	for _, m := range boost.Modifiers {
		switch m.Target.K {
		case "announcedCondition":
			out = append(out, AnnouncedCondition{Condition: m.Target.Name, Note: m.Note})
		case "announcedEffect":
			out = append(out, AnnouncedCondition{Effect: m.Note})
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

// BoostHitPointCost é o dado de PV que o gasto pedido tira de quem ataca — o 1d8 da Tumular (p336) —, ou
// vazio. Só o gasto de encanto tem custo de PV hoje.
func (r *Ruleset) BoostHitPointCost(id string) string {
	_, enchant, isEnchant := enchantBoostOf(id)
	if !isEnchant {
		return ""
	}
	if catalog := r.getCatalogItem(enchant); catalog != nil && catalog.Boost != nil {
		return catalog.Boost.HitPointCost
	}
	return ""
}

// EnchantBlast é o gesto que um encanto dispara EM VEZ do golpe (a bola de fogo da Flamejante, p335), numa arma
// empunhada. O ID é o do gasto do encanto (`EnchantBoostID`): a vez da rodada é a mesma.
type EnchantBlast struct {
	BoostInstead
	ID      string
	ItemID  int64
	Weapon  string
	Enchant string
	PM      int
}

// EnchantBlasts são os gestos que as armas empunhadas disparam em vez do golpe, um por encanto que tem um.
func (r *Ruleset) EnchantBlasts(ch Character) []EnchantBlast {
	out := []EnchantBlast{}
	for _, it := range ch.Items {
		if !inAHand(it) {
			continue
		}
		for _, enchant := range it.Enchants {
			catalog := r.getCatalogItem(enchant)
			if catalog == nil || catalog.Boost == nil || catalog.Boost.Instead == nil {
				continue
			}
			out = append(out, EnchantBlast{
				BoostInstead: *catalog.Boost.Instead, ID: EnchantBoostID(it.ID, enchant), ItemID: it.ID,
				Weapon: it.Name, Enchant: catalog.Name, PM: catalog.Boost.PM,
			})
		}
	}
	return out
}

// inAHand diz se o item está numa das mãos: o gasto do encanto é da arma em punho.
func inAHand(it CharacterItem) bool {
	return it.Equipped != nil && (*it.Equipped == "wielded" || *it.Equipped == "wielded2")
}
