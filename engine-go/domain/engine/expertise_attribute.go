package engine

// expertiseAttributeFor é o atributo com que a perícia rola: o dela, ou o que um verbete autoriza
// no lugar dele, quando é MAIOR (decisão do dono — "pode usar" não obriga a piorar a conta). Volta
// também quem autorizou, para a conta dizer por que o atributo é outro.
func expertiseAttributeFor(ch Character, state CharacterExpertise, e ItemEffects) (string, string) {
	best, source := state.Attribute, ""
	value := effectiveAttribute(ch, best, e)
	for _, key := range AttributeKeys {
		allowed := StatFor(e, ModifierTarget{K: "expertiseAttribute", Name: state.Name, Attribute: key})
		if len(allowed.Contributions) == 0 {
			continue
		}
		if amount := effectiveAttribute(ch, key, e); amount > value {
			best, value, source = key, amount, allowed.Contributions[0].Source
		}
	}
	return best, source
}
