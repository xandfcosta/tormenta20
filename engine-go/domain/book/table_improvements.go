package book

import (
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/engine"
)

// O QUE O EDITOR DE MELHORIA OFERECE, e o que ele recusa. As famílias são as de `ItemFamily`.

// TableImprovementFamilies são os lugares em que a melhoria da mesa cabe, na ordem do editor.
func TableImprovementFamilies() []RuleOption {
	return []RuleOption{
		{"", "Qualquer item"}, {"weapon", "Arma"}, {"armor", "Armadura"}, {"shield", "Escudo"},
		{"apparel", "Vestuário e esotérico"},
	}
}

// WrittenImprovementPlace é "cabe em": "vestuário e esotérico".
func WrittenImprovementPlace(family string) string {
	for _, option := range TableImprovementFamilies() {
		if option.ID == family {
			return strings.ToLower(option.Name)
		}
	}
	return family
}

// TableImprovementFits diz se a melhoria cabe neste item.
func TableImprovementFits(improvement engine.TableImprovement, item Item) bool {
	return improvement.Family == "" || improvement.Family == ItemFamily(item)
}

func aimsTheStrike(target string) bool {
	return target == "attack" || target == "damage" || target == "critRange"
}

// TableImprovementProblem diz o que está errado na melhoria, com o valor e o que se esperava.
func TableImprovementProblem(name, family string, effect engine.RuleEffect) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("dê um nome à melhoria: é como ela aparece na lista do item e na conta da ficha")
	}
	if !slices.ContainsFunc(TableImprovementFamilies(), func(o RuleOption) bool { return o.ID == family }) {
		return fmt.Errorf("%q não é lugar em que uma melhoria caiba: escolha um da lista", family)
	}
	if err := effectTargetProblem(effect); err != nil {
		return err
	}
	// Em qualquer outro item, o número de golpe não teria carta que o lesse, e a melhoria não faria nada.
	if aimsTheStrike(effect.Target.K) && family != "weapon" {
		return fmt.Errorf("ataque, dano e margem de ameaça só valem numa arma: em \"Cabe em\", escolha Arma")
	}
	if effect.Amount == 0 {
		return fmt.Errorf("a soma é zero, e a melhoria não mudaria nada: escreva um número diferente de zero")
	}
	return nil
}

// WrittenImprovementPreview é a frase da prévia: "Colocada num escudo, dá +1 na Defesa enquanto o item está em uso."
func WrittenImprovementPreview(family string, effect engine.RuleEffect) string {
	place := map[string]string{
		"": "em qualquer item", "weapon": "numa arma", "armor": "numa armadura", "shield": "num escudo",
		"apparel": "num item de vestuário ou esotérico",
	}[family]
	if aimsTheStrike(effect.Target.K) {
		return fmt.Sprintf("Colocada %s, dá %s daquela arma.", place, WrittenSum(effect))
	}
	return fmt.Sprintf("Colocada %s, dá %s enquanto o item está em uso.", place, WrittenSum(effect))
}
