package engine

// expertiseAttributeFor é o atributo com que a perícia rola: o dela, ou o que um verbete autoriza
// no lugar dele (o Pequeno e Rechonchudo, p28), pela regra do `licensedAttribute`.
func expertiseAttributeFor(ch Character, state CharacterExpertise, e ItemEffects) (string, string) {
	attribute, license := licensedAttribute(ch, state.Attribute, e, func(attribute string) ModifierTarget {
		return ModifierTarget{K: "expertiseAttribute", Name: state.Name, Attribute: attribute}
	})
	return attribute, license.Source
}

// LicensedAttribute é o atributo que vale numa conta no lugar do que ela pede, e quem o autorizou.
type LicensedAttribute struct {
	Attribute string `json:"attribute"`
	Source    string `json:"source"`
}

// licensedAttribute é o atributo que vale numa conta: o que ela pede, ou um que um verbete AUTORIZA
// no lugar dele, quando é MAIOR (decisão do dono — "pode usar" não obriga a piorar a conta). Volta
// também a parcela da licença que autorizou (a fonte e a nota dela), vazia quando nada trocou, para a
// conta dizer por que o atributo é outro. É a regra de toda licença de atributo — a perícia do Pequeno
// e Rechonchudo (p28), a CD do item alquímico do Granadeiro e a escala do Estilo de Disparo com a
// Balística (p69) —, e só o ALVO da licença muda de uma para outra.
func licensedAttribute(ch Character, base string, e ItemEffects, license func(attribute string) ModifierTarget) (string, Contribution) {
	best, by := base, Contribution{}
	value := effectiveAttribute(ch, best, e)
	for _, key := range AttributeKeys {
		allowed := StatFor(e, license(key))
		if len(allowed.Contributions) == 0 {
			continue
		}
		if amount := effectiveAttribute(ch, key, e); amount > value {
			best, value, by = key, amount, allowed.Contributions[0]
		}
	}
	return best, by
}
