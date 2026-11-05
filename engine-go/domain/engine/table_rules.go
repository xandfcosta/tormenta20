package engine

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// AS REGRAS DA MESA: o que o mestre monta no editor guiado, condições e um efeito.
//
// Elas viram `ActiveItem` como toda outra fonte, e por isso chegam ao número e à conta da ficha
// pelo mesmo cano. O silêncio da mesa as alcança nessa coleta, mas não no PV e no PM, que o máximo
// deriva à parte (`ruleVitalGrants`).

// As espécies de condição, como a coluna as grava.
const (
	RuleOnItem        = "item"
	RuleOnImprovement = "improvement"
	RuleOnTrait       = "trait"
	RuleOnVested      = "vested"
	RuleOnRace        = "race"
	RuleOnClass       = "class"
	RuleOnDevotion    = "devotion"
	RuleOnPower       = "power"
	RuleOnStance      = "stance"
)

// RuleAdds é o efeito de somar um número.
const RuleAdds = "adds"

// TableRuleSource é o prefixo que separa a regra de toda fonte do livro, como o da concessão.
const TableRuleSource = "campaign-rule:"

// RuleCondition é uma condição da regra.
type RuleCondition struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// RuleEffect é o que a regra muda.
type RuleEffect struct {
	Kind   string         `json:"kind"`
	Target ModifierTarget `json:"target"`
	Amount int            `json:"amount"`
	// Limit é até quantos itens a regra soma, quando ela soma por item. Zero é sem limite.
	Limit int `json:"limit,omitempty"`
}

// TableRule é uma regra da mesa ligada.
//
// @example engine.TableRule{ID: "r1", Name: "Espada da casa", Applies: engine.EveryoneIn(), When: …, Effect: …}
type TableRule struct {
	ID      string
	Name    string
	Applies Selector
	When    []RuleCondition
	Effect  RuleEffect
}

// ParseTableRule lê as duas colunas da regra.
func ParseTableRule(conditions, effect string) ([]RuleCondition, RuleEffect, error) {
	var when []RuleCondition
	if err := json.Unmarshal([]byte(conditions), &when); err != nil {
		return nil, RuleEffect{}, fmt.Errorf("as condições da regra estão ilegíveis: %w", err)
	}
	then, err := ParseRuleEffect(effect)
	if err != nil {
		return nil, RuleEffect{}, err
	}
	return when, then, nil
}

// ParseRuleEffect lê o efeito como a coluna o guarda: o da regra e o da melhoria têm a mesma forma.
func ParseRuleEffect(effect string) (RuleEffect, error) {
	var then RuleEffect
	if err := json.Unmarshal([]byte(effect), &then); err != nil {
		return RuleEffect{}, fmt.Errorf("o efeito está ilegível: %w", err)
	}
	return then, nil
}

// RuleReaches diz se a regra alcança esta ficha hoje: é a prévia do editor. A postura conta como
// alcançada, porque quem a tem pode ligá-la; quem pergunta troca a postura pelo poder que a dá
// (`book.RuleConditionsForReach`), já que o motor só conhece a flag dela.
func (r *Ruleset) RuleReaches(ch Character, rule TableRule) bool {
	return rule.Applies.Matches(ch) && r.characterFitsRule(ch, rule) && len(r.ruleModifiers(ch, rule)) > 0
}

func (r *Ruleset) tableRuleItems(ch Character) []ActiveItem {
	out := []ActiveItem{}
	for _, rule := range r.mesa.Rules {
		if rule.Effect.Kind != RuleAdds || !rule.Applies.Matches(ch) || !r.characterFitsRule(ch, rule) {
			continue
		}
		if mods := r.ruleModifiers(ch, rule); len(mods) > 0 {
			out = append(out, ActiveItem{
				SourceID: TableRuleSource + rule.ID, Source: rule.Name + " (regra da mesa)",
				Equipped: &vestedWear, Modifiers: mods,
			})
		}
	}
	return out
}

// ruleVitalGrants são as regras que somam em PV ou PM: o máximo é derivado à parte da coleta.
func (r *Ruleset) ruleVitalGrants(ch Character) []vitalGrant {
	out := []vitalGrant{}
	for _, item := range r.tableRuleItems(ch) {
		for _, modifier := range item.Modifiers {
			if modifier.Target.K == "maxPv" || modifier.Target.K == "maxPm" {
				out = append(out, vitalGrant{Modifier: modifier, Source: item.Source})
			}
		}
	}
	return out
}

// characterFitsRule confere o que é da ficha. Espécie desconhecida não alcança ninguém, pela razão
// do `Selector.Matches`: a coluna sobrevive ao rollback do binário.
func (r *Ruleset) characterFitsRule(ch Character, rule TableRule) bool {
	for _, condition := range rule.When {
		fits := false
		switch condition.Kind {
		case RuleOnItem, RuleOnImprovement, RuleOnTrait, RuleOnVested, RuleOnStance:
			fits = true // o item e a postura são de `ruleModifiers`
		case RuleOnRace:
			fits = slices.ContainsFunc(ch.Races, func(race CharacterRace) bool { return race.Race == condition.Value })
		case RuleOnClass:
			fits = slices.ContainsFunc(ch.Classes, func(class CharacterClass) bool { return class.ClassName == condition.Value })
		case RuleOnDevotion:
			fits = ch.God != nil && *ch.God == condition.Value
		case RuleOnPower:
			fits = r.ownsPower(ch, condition.Value)
		}
		if !fits {
			return false
		}
	}
	return true
}

// ownsPower diz se a ficha tem o poder, de classe, geral, da raça, da origem ou da devoção.
func (r *Ruleset) ownsPower(ch Character, id string) bool {
	if r.chosenGeneralPowers(ch).has[id] || slices.Contains(r.originPickedPowerIds(ch), id) {
		return true
	}
	chosen, choices := parseChoiceSet(ch.ClassPowers), parseClassChoices(ch.ClassChoices)
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			if power.ID == id {
				return true
			}
		}
	}
	for _, name := range parseChoiceSet(ch.GodPowers).list {
		if power := r.grantedPowerByName(name); power != nil && power.ID == id {
			return true
		}
	}
	return false
}

// ruleModifiers são os modificadores da regra nesta ficha. Sem condição de item, é um. Com ela, um
// por item em uso que passa em todas, porque elas olham o MESMO item; dois iguais na mão contam uma
// vez. O limite corta o que soma na ficha; o que é preso a uma arma é de cada arma, e não acumula.
func (r *Ruleset) ruleModifiers(ch Character, rule TableRule) []Modifier {
	onItem := slices.DeleteFunc(slices.Clone(rule.When), func(c RuleCondition) bool {
		return !slices.Contains([]string{RuleOnItem, RuleOnImprovement, RuleOnTrait, RuleOnVested}, c.Kind)
	})
	modifier := Modifier{
		Target: rule.Effect.Target, Amount: rule.Effect.Amount, BonusType: "untyped", Note: rule.Name,
		Condition: ruleStanceCondition(rule),
	}
	if len(onItem) == 0 {
		// Sem item, o número de golpe vale em toda arma: o escopo vazio não tem leitor na carta.
		if aimsAWeapon(modifier.Target.K) && modifier.Target.Scope == "" {
			modifier.Target.Scope = "all"
		}
		return []Modifier{modifier}
	}
	out := []Modifier{}
	seen := map[string]bool{}
	counted := 0
	for _, item := range ch.Items {
		if item.Equipped == nil || item.CatalogID == nil || seen[*item.CatalogID] {
			continue
		}
		catalog := r.getCatalogItem(*item.CatalogID)
		if catalog == nil || !itemFitsRule(item, catalog, onItem) {
			continue
		}
		seen[*item.CatalogID] = true
		onThisItem := modifierOnItem(modifier, catalog)
		if !isWeaponScoped(onThisItem) {
			if rule.Effect.Limit > 0 && counted == rule.Effect.Limit {
				continue
			}
			counted++
		}
		out = append(out, onThisItem)
	}
	return out
}

func itemFitsRule(item CharacterItem, catalog *CatalogItem, conditions []RuleCondition) bool {
	for _, condition := range conditions {
		fits := false
		switch condition.Kind {
		case RuleOnItem:
			fits = catalog.ID == condition.Value
		case RuleOnImprovement:
			fits = slices.Contains(parseStringArray(item.Improvements), condition.Value)
		case RuleOnTrait:
			fits = catalog.Weapon != nil && slices.Contains(catalog.Weapon.Traits, condition.Value)
		case RuleOnVested:
			fits = *item.Equipped == "vested"
		}
		if !fits {
			return false
		}
	}
	return true
}

// modifierOnItem prende à arma o que é número de golpe: "+2 no ataque com o machado" não sobe a
// outra mão, e a margem também não. O resto, como o +1 em Luta do medalhão, é da ficha.
func modifierOnItem(modifier Modifier, catalog *CatalogItem) Modifier {
	switch {
	case modifier.Target.Scope != "":
	case catalog.Weapon != nil && (aimsAWeapon(modifier.Target.K) || modifier.Target.K == "critRange"):
		modifier.Target.Scope = WeaponDamageScope(catalog.ID)
	case aimsAWeapon(modifier.Target.K):
		modifier.Target.Scope = "all"
	}
	return modifier
}

func isWeaponScoped(modifier Modifier) bool {
	return strings.HasPrefix(modifier.Target.Scope, WeaponDamageScope(""))
}

// aimsAWeapon diz se o alvo só tem leitor na carta com escopo. A margem não entra aqui: sem escopo,
// `threatRangeOf` já a lê em toda arma.
func aimsAWeapon(target string) bool { return target == "attack" || target == "damage" }

// ruleStanceCondition faz a regra seguir a postura, pelo mesmo interruptor dos poderes
// (`flagOn`). O interruptor olha uma flag só, então o editor deixa uma postura por regra.
func ruleStanceCondition(rule TableRule) *ModifierCondition {
	at := slices.IndexFunc(rule.When, func(c RuleCondition) bool { return c.Kind == RuleOnStance })
	if at < 0 {
		return nil
	}
	return &ModifierCondition{C: "flagOn", Flag: rule.When[at].Value, Label: rule.Name}
}
