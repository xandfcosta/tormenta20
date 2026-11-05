package engine

import (
	"encoding/json"
	"fmt"
	"slices"
)

// AS REGRAS DA MESA: o que o mestre monta no editor guiado, condições e um efeito.
//
// Elas viram `ActiveItem` como toda outra fonte, e por isso chegam ao número e à conta da ficha
// pelo mesmo cano, e o silêncio da mesa também as alcança.

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
	var then RuleEffect
	if err := json.Unmarshal([]byte(effect), &then); err != nil {
		return nil, RuleEffect{}, fmt.Errorf("o efeito da regra está ilegível: %w", err)
	}
	return when, then, nil
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
// por item em uso que passa em todas, porque elas olham o MESMO item; dois iguais na mão contam uma vez.
func (r *Ruleset) ruleModifiers(ch Character, rule TableRule) []Modifier {
	onItem := slices.DeleteFunc(slices.Clone(rule.When), func(c RuleCondition) bool {
		return !slices.Contains([]string{RuleOnItem, RuleOnImprovement, RuleOnTrait, RuleOnVested}, c.Kind)
	})
	modifier := Modifier{
		Target: rule.Effect.Target, Amount: rule.Effect.Amount, BonusType: "untyped", Note: rule.Name,
		Condition: ruleStanceCondition(rule),
	}
	if len(onItem) == 0 {
		return []Modifier{modifier}
	}
	out := []Modifier{}
	seen := map[string]bool{}
	for _, item := range ch.Items {
		if item.Equipped == nil || item.CatalogID == nil || seen[*item.CatalogID] {
			continue
		}
		catalog := r.getCatalogItem(*item.CatalogID)
		if catalog == nil || !itemFitsRule(item, catalog, onItem) {
			continue
		}
		seen[*item.CatalogID] = true
		out = append(out, modifierOnItem(modifier, catalog))
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
// outra mão. O resto, como o +1 em Luta do medalhão, é da ficha.
func modifierOnItem(modifier Modifier, catalog *CatalogItem) Modifier {
	aimsTheWeapon := modifier.Target.K == "attack" || modifier.Target.K == "damage" || modifier.Target.K == "critRange"
	if catalog.Weapon != nil && aimsTheWeapon && modifier.Target.Scope == "" {
		modifier.Target.Scope = WeaponDamageScope(catalog.ID)
	}
	return modifier
}

// ruleStanceCondition faz a regra seguir a postura, pelo mesmo interruptor dos poderes
// (`flagOn`). O interruptor olha uma flag só, então o editor deixa uma postura por regra.
func ruleStanceCondition(rule TableRule) *ModifierCondition {
	at := slices.IndexFunc(rule.When, func(c RuleCondition) bool { return c.Kind == RuleOnStance })
	if at < 0 {
		return nil
	}
	return &ModifierCondition{C: "flagOn", Flag: rule.When[at].Value, Label: rule.Name}
}
