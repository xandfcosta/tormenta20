package engine

// AS HABILIDADES QUE SOMAM UM ATRIBUTO, e por que elas pedem uma segunda passada.
//
// "Você soma seu Carisma na Defesa, limitado pelo seu nível" (Insolência, p47) é
// a mesma forma em oito verbetes do capítulo 1, e o catálogo a escreve uma vez:
//
//	{target: defense, scale: {per: attribute, attribute: charisma, cap: classLevel}}
//
// O número não existe na coleta. Ele é o atributo EFETIVO — o da ficha mais o
// que os itens e os condicionais lhe somam (o `effectiveAttribute`) —, e esse só
// se conhece depois da resolução. Por isso a ficha resolve duas vezes: a
// primeira sem estes modificadores, para saber os atributos; a segunda com o
// número já posto. Nenhum deles mira um atributo, então a primeira passada não
// muda pelo que a segunda acrescenta — e o `TestNoAttributeSumTargetsAnAttribute`
// prende isso no catálogo.
//
// O ATRIBUTO NEGATIVO SUBTRAI (decisão do dono, ALE-485). O livro não tem regra
// própria para isso, e o uso dele em todo lugar é somar o valor como é: "Sua
// Defesa é 10 + sua Destreza" (p106). O teto da p226 só limita por cima.
//
// PV e PM ficam de fora: o `scale` por atributo deles é do `ComputeVitals`, que
// o avalia com a conta própria dos vitais.

// sheetEffects são os efeitos da ficha sob os condicionais ligados, com as
// somas de atributo já resolvidas. É o que toda decomposição da ficha lê.
//
//	effects := r.sheetEffects(ch, activeConditionals)
func (r *Ruleset) sheetEffects(ch Character, activeConditionals map[string]bool) ItemEffects {
	return r.sheetEffectsWith(ch, activeConditionals, nil)
}

// sheetEffectsWith são os efeitos da ficha com fontes A MAIS que não moram nela
// — o impulso de PM de UM ataque (`BoostedWeaponCards`), que vale para aquele
// golpe e some depois.
func (r *Ruleset) sheetEffectsWith(ch Character, activeConditionals map[string]bool, extra []ActiveItem) ItemEffects {
	items := append(r.ActiveItemsFor(ch), extra...)
	plain, sums := splitAttributeSums(items)
	first := ApplyActiveConditionals(ComputeItemEffects(plain), activeConditionals)
	if len(sums) == 0 {
		return first
	}
	return ApplyActiveConditionals(ComputeItemEffects(append(plain, resolveAttributeSums(ch, sums, first)...)), activeConditionals)
}

// isAttributeSum diz se o modificador é uma soma de atributo que a ficha
// resolve — e não a escala de um vital, que é do `ComputeVitals`.
func isAttributeSum(m Modifier) bool {
	if m.Scale == nil || m.Scale.Per != "attribute" {
		return false
	}
	return m.Target.K != "maxPv" && m.Target.K != "maxPm" && m.Target.K != "attribute"
}

// splitAttributeSums separa, item a item, os modificadores que são soma de
// atributo dos outros. O item que só tinha somas some da primeira lista.
func splitAttributeSums(items []ActiveItem) (plain, sums []ActiveItem) {
	for _, it := range items {
		var rest, scaled []Modifier
		for _, m := range it.Modifiers {
			if isAttributeSum(m) {
				scaled = append(scaled, m)
			} else {
				rest = append(rest, m)
			}
		}
		if len(scaled) == 0 {
			plain = append(plain, it)
			continue
		}
		if len(rest) > 0 {
			kept := it
			kept.Modifiers = rest
			plain = append(plain, kept)
		}
		summed := it
		summed.Modifiers = scaled
		sums = append(sums, summed)
	}
	return plain, sums
}

// resolveAttributeSums põe em cada soma o número dela: o atributo efetivo, com
// o teto quando o verbete o escreve.
func resolveAttributeSums(ch Character, sums []ActiveItem, effects ItemEffects) []ActiveItem {
	out := make([]ActiveItem, len(sums))
	for i, it := range sums {
		resolved := it
		resolved.Modifiers = make([]Modifier, len(it.Modifiers))
		for j, m := range it.Modifiers {
			amount := effectiveAttribute(ch, m.Scale.Attribute, effects)
			if m.Scale.Cap == "classLevel" {
				amount = min(amount, m.classLevel)
			}
			m.Amount = amount
			m.Scale = nil
			resolved.Modifiers[j] = m
		}
		out[i] = resolved
	}
	return out
}
