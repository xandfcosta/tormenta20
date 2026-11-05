package book

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/engine"
)

// O QUE O EDITOR DAS REGRAS DA MESA OFERECE: as espécies de condição, as opções de cada uma e os
// alvos da soma, e como a regra se escreve em português. O que vale sai do catálogo, e é por isso
// que mora aqui.

// RuleOption é uma opção de uma lista do editor.
type RuleOption struct {
	ID   string
	Name string
}

// MaxRuleConditions é quantas condições cabem numa regra.
const MaxRuleConditions = 5

// RuleConditionKinds são as espécies, na ordem do editor.
func RuleConditionKinds() []RuleOption {
	return []RuleOption{
		{engine.RuleOnItem, "Arma ou item"}, {engine.RuleOnImprovement, "Melhoria"}, {engine.RuleOnTrait, "Traço da arma"},
		{engine.RuleOnVested, "Está vestido"}, {engine.RuleOnRace, "Raça"}, {engine.RuleOnClass, "Classe"},
		{engine.RuleOnDevotion, "Devoção"}, {engine.RuleOnPower, "Poder"}, {engine.RuleOnStance, "Postura"},
	}
}

// RuleTargets são os alvos da soma, e a frase de cada um.
func RuleTargets() []RuleOption {
	return []RuleOption{
		{"attack", "Ataque"}, {"damage", "Dano"}, {"expertise", "Perícia"}, {"defense", "Defesa"},
		{"maxPv", "PV"}, {"maxPm", "PM"}, {"critRange", "Margem de ameaça"},
	}
}

var ruleTargetPhrases = map[string]string{
	"attack": "no ataque", "damage": "no dano", "defense": "na Defesa",
	"maxPv": "em PV", "maxPm": "em PM", "critRange": "na margem de ameaça",
}

var ruleTraitNames = map[string]string{
	"adaptavel": "Adaptável", "agil": "Ágil", "alongada": "Alongada",
	"desbalanceada": "Desbalanceada", "dupla": "Dupla", "versatil": "Versátil",
}

// RuleOptions são as opções de uma espécie de condição, em ordem de nome.
func RuleOptions(kind string) []RuleOption {
	out := []RuleOption{}
	switch kind {
	case engine.RuleOnItem, engine.RuleOnImprovement:
		for _, item := range Catalogs().Items {
			// Melhoria, material e encanto moram DENTRO de um item, e não viram linha da ficha:
			// como "Arma ou item", seriam uma condição que nunca vale.
			overlay := item.Category == "improvement" || item.Category == "material" ||
				item.Category == "weapon-enchant" || item.Category == "armor-enchant"
			if (kind == engine.RuleOnImprovement && item.Category == "improvement") || (kind == engine.RuleOnItem && !overlay) {
				out = append(out, RuleOption{item.ID, item.Name})
			}
		}
	case engine.RuleOnTrait:
		for id, name := range ruleTraitNames {
			out = append(out, RuleOption{id, name})
		}
	case engine.RuleOnRace:
		for key, race := range RaceTraitsByKey() {
			if key == race.ID {
				out = append(out, RuleOption{race.ID, RaceLabel(race.ID)})
			}
		}
	case engine.RuleOnClass:
		_, classes, _ := CharacterCatalogs()
		for _, class := range classes {
			out = append(out, RuleOption{class.Name, class.Name})
		}
	case engine.RuleOnDevotion:
		for _, god := range CatalogGods() {
			out = append(out, RuleOption{god.ID, god.Name})
		}
	case engine.RuleOnPower:
		for id, power := range GeneralPowers() {
			out = append(out, RuleOption{id, power.Name})
		}
		for id, power := range ClassPowers() {
			out = append(out, RuleOption{id, power.Name + " · " + power.ClassName})
		}
		for _, power := range ListOf[struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}]("granted-powers") {
			out = append(out, RuleOption{power.ID, power.Name})
		}
	case engine.RuleOnStance:
		for flag, stance := range StancesFromCatalog() {
			out = append(out, RuleOption{flag, stance.Name})
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Name != out[b].Name {
			return out[a].Name < out[b].Name
		}
		return out[a].ID < out[b].ID
	})
	return out
}

func ruleOptionName(kind, id string) (string, bool) {
	for _, option := range RuleOptions(kind) {
		if option.ID == id {
			return option.Name, true
		}
	}
	return "", false
}

// TableRuleProblem diz o que está errado na regra, com o valor e o que se esperava.
func TableRuleProblem(name string, when []engine.RuleCondition, effect engine.RuleEffect) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("dê um nome à regra: é como ela aparece na conta da ficha")
	}
	if len(when) > MaxRuleConditions {
		return fmt.Errorf("a regra tem %d condições, e cabem %d", len(when), MaxRuleConditions)
	}
	stances := 0
	for _, condition := range when {
		if !slices.ContainsFunc(RuleConditionKinds(), func(o RuleOption) bool { return o.ID == condition.Kind }) {
			return fmt.Errorf("a condição %q não existe: escolha uma da lista", condition.Kind)
		}
		if condition.Kind == engine.RuleOnStance {
			stances++
		}
		if condition.Kind == engine.RuleOnVested {
			continue
		}
		if _, known := ruleOptionName(condition.Kind, condition.Value); !known {
			return fmt.Errorf("%q não é opção de %s: escolha uma da lista", condition.Value, ruleKindName(condition.Kind))
		}
	}
	if stances > 1 {
		return fmt.Errorf("a regra olha %d posturas, e cabe uma por regra", stances)
	}
	ignored := map[string][]string{
		engine.RuleCountsAsWielded: {engine.RuleOnStance},
		engine.RuleBorrowsPowers:   {engine.RuleOnItem, engine.RuleOnImprovement, engine.RuleOnTrait, engine.RuleOnVested, engine.RuleOnStance},
	}[effect.Kind]
	for _, condition := range when {
		if slices.Contains(ignored, condition.Kind) {
			return fmt.Errorf("\"%s\" não olha %s: tire essa condição, ou mude o efeito", ruleEffectName(effect.Kind), strings.ToLower(ruleKindName(condition.Kind)))
		}
	}
	switch effect.Kind {
	case engine.RuleCountsAsWielded:
		if !slices.ContainsFunc(when, func(c engine.RuleCondition) bool {
			return slices.Contains([]string{engine.RuleOnItem, engine.RuleOnImprovement, engine.RuleOnTrait}, c.Kind)
		}) {
			return fmt.Errorf("\"vale como empunhado\" precisa dizer qual item: ponha uma condição de arma ou item, melhoria ou traço")
		}
		return nil
	case engine.RuleBorrowsPowers:
		if len(GrantedPowersOf(effect.God)) == 0 {
			return fmt.Errorf("%s não concede poderes: escolha um deus que conceda", DevotionWritten(effect.God))
		}
		return nil
	}
	if err := effectTargetProblem(effect); err != nil {
		return err
	}
	if stances > 0 && (effect.Target.K == "maxPv" || effect.Target.K == "maxPm") {
		return fmt.Errorf("PV e PM máximos não dependem de postura: tire a condição de postura, ou mude o alvo")
	}
	if effect.Limit < 0 {
		return fmt.Errorf("o limite é %d, e tem de ser um número de itens: deixe em branco para não ter limite", effect.Limit)
	}
	if effect.Amount == 0 {
		return fmt.Errorf("a soma é zero, e a regra não mudaria nada: escreva um número diferente de zero")
	}
	return nil
}

func effectTargetProblem(effect engine.RuleEffect) error {
	if effect.Kind != engine.RuleAdds {
		return fmt.Errorf("o efeito %q não existe", effect.Kind)
	}
	if !slices.ContainsFunc(RuleTargets(), func(o RuleOption) bool { return o.ID == effect.Target.K }) {
		return fmt.Errorf("a soma em %q não existe: escolha um alvo da lista", effect.Target.K)
	}
	if effect.Target.K == "expertise" && !slices.ContainsFunc(Expertises(), func(e Expertise) bool { return e.Name == effect.Target.Name }) {
		return fmt.Errorf("%q não é perícia do livro", effect.Target.Name)
	}
	return nil
}

func ruleKindName(kind string) string {
	for _, option := range RuleConditionKinds() {
		if option.ID == kind {
			return option.Name
		}
	}
	return kind
}

// RuleCountsPerItem diz se a regra soma uma vez por item, e é onde o limite tem o que limitar. O item
// nomeado é um só, e o que é número de golpe é de cada arma.
func RuleCountsPerItem(when []engine.RuleCondition, target string) bool {
	if aimsTheStrike(target) {
		return false
	}
	byKind := func(kinds ...string) bool {
		return slices.ContainsFunc(when, func(c engine.RuleCondition) bool { return slices.Contains(kinds, c.Kind) })
	}
	return !byKind(engine.RuleOnItem) && byKind(engine.RuleOnImprovement, engine.RuleOnTrait, engine.RuleOnVested)
}

// WrittenTableRule é a regra em português: "com Machado táurico e a melhoria Equilibrada: +2 no ataque".
func WrittenTableRule(when []engine.RuleCondition, effect engine.RuleEffect) string {
	parts := []string{}
	for _, condition := range when {
		name, known := ruleOptionName(condition.Kind, condition.Value)
		if !known {
			name = condition.Value
		}
		switch condition.Kind {
		case engine.RuleOnItem:
			parts = append(parts, "com "+name)
		case engine.RuleOnImprovement:
			parts = append(parts, "com a melhoria "+name)
		case engine.RuleOnTrait:
			parts = append(parts, "com o traço "+name)
		case engine.RuleOnVested:
			parts = append(parts, "vestido")
		case engine.RuleOnRace:
			parts = append(parts, "da raça "+name)
		case engine.RuleOnClass:
			parts = append(parts, "da classe "+name)
		case engine.RuleOnDevotion:
			parts = append(parts, "devoto de "+name)
		case engine.RuleOnPower:
			parts = append(parts, "com o poder "+name)
		case engine.RuleOnStance:
			parts = append(parts, "em "+name)
		}
	}
	sum := WrittenSum(effect)
	switch effect.Kind {
	case engine.RuleCountsAsWielded:
		sum = "vestido, vale como empunhado"
	case engine.RuleBorrowsPowers:
		names := []string{}
		for _, power := range GrantedPowersOf(effect.God) {
			names = append(names, power.Name)
		}
		sum = "escolhe poderes de " + DevotionName(effect.God) + " (" + WrittenList(names) + ")"
	}
	if effect.Kind == engine.RuleAdds && RuleCountsPerItem(when, effect.Target.K) && effect.Limit != 1 {
		sum += " por item"
		if effect.Limit > 1 {
			sum += fmt.Sprintf(", até %+d", effect.Amount*effect.Limit)
		}
	}
	if len(parts) == 0 {
		return sum
	}
	return strings.Join(parts, " e ") + ": " + sum
}

// WrittenSum é a soma em português: "+1 em Luta".
func WrittenSum(effect engine.RuleEffect) string {
	amount := strconv.Itoa(effect.Amount)
	if effect.Amount > 0 {
		amount = "+" + amount
	}
	phrase := ruleTargetPhrases[effect.Target.K]
	if effect.Target.K == "expertise" {
		phrase = "em " + effect.Target.Name
	}
	return amount + " " + phrase
}

// RuleEffectKinds são os efeitos, na ordem do editor.
func RuleEffectKinds() []RuleOption {
	return []RuleOption{
		{engine.RuleAdds, "Soma em"}, {engine.RuleCountsAsWielded, "Vale como empunhado"}, {engine.RuleBorrowsPowers, "Escolhe poderes de"},
	}
}

// LendingGods são os deuses que têm poderes para emprestar: os que concedem algum.
func LendingGods() []RuleOption {
	out := []RuleOption{}
	for _, god := range CatalogGods() {
		if len(GrantedPowersOf(god.ID)) > 0 {
			out = append(out, RuleOption{god.ID, god.Name})
		}
	}
	return out
}

func ruleEffectName(kind string) string {
	for _, option := range RuleEffectKinds() {
		if option.ID == kind {
			return option.Name
		}
	}
	return kind
}

// WrittenList é a lista em português: "a, b e c".
func WrittenList(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " e " + names[len(names)-1]
}

// RuleConditionsForReach troca a postura pelo poder que a dá, para a pergunta "quem esta regra
// alcança hoje": a condição guarda a flag, que toda ficha pode ligar, e só o poder diz quem a tem.
func RuleConditionsForReach(when []engine.RuleCondition) []engine.RuleCondition {
	out := slices.Clone(when)
	for i, condition := range out {
		if stance, known := StancesFromCatalog()[condition.Value]; condition.Kind == engine.RuleOnStance && known {
			out[i] = engine.RuleCondition{Kind: engine.RuleOnPower, Value: stance.ID}
		}
	}
	return out
}
