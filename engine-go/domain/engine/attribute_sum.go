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
//
// A DEFESA DO ESCUDO tem a mesma forma, e por isso mora na mesma passada. A
// Solidez soma "o bônus na Defesa recebido pelo escudo em testes de resistência"
// (p55, p66), e esse bônus é o do ITEM resolvido — o escudo, e a melhoria e o
// encanto postos nele ("o bônus na Defesa ... do item aumenta em 1", p165); o
// que está na mochila não dá nada. O número sai das parcelas da Defesa que vêm
// de um escudo, depois da primeira passada — e a soma mira a RESISTÊNCIA, que a
// Defesa não lê, então a segunda passada não muda o que a primeira mediu.

// EffectsOf são os efeitos do personagem: todas as fontes que a coleta conhece, com os interruptores que
// ele ligou (pelo endereço do termo ou do grupo da flag) e as somas de atributo resolvidas. É a ÚNICA
// entrada para "o que vale para este personagem agora", e toda decomposição da ficha a lê. A lista dos
// interruptores que ele pode ligar vem junto, em `Conditional`, ligados e desligados.
//
// Havia três respostas para essa pergunta, cada uma com outras fontes, e a que não via a Forma Selvagem
// mandava a manobra da Mesa sem o tamanho da forma.
//
//	effects := ruleset.EffectsOf(ch, sheet.ToStringSet(dto.Conditionals))
func (r *Ruleset) EffectsOf(ch Character, switches map[string]bool) ItemEffects {
	return r.sheetEffectsWith(ch, switches, nil)
}

// sheetEffectsWith são os efeitos da ficha com fontes A MAIS que não moram nela
// — o impulso de PM de UM ataque (`BoostedWeaponCards`), que vale para aquele
// golpe e some depois.
func (r *Ruleset) sheetEffectsWith(ch Character, activeConditionals map[string]bool, extra []ActiveItem) ItemEffects {
	items := append(r.ActiveItemsFor(ch), extra...)
	plain, sums := splitSecondPassSums(items)
	first := resolveInWorld(plain, activeConditionals)
	if len(sums) == 0 {
		return first
	}
	return resolveInWorld(append(plain, r.resolveSecondPassSums(ch, sums, first)...), activeConditionals)
}

// shieldDefenseScale é a escala da Solidez: o modificador vale o que o escudo empunhado dá à Defesa.
const shieldDefenseScale = "shieldDefense"

// isSecondPassSum diz se o modificador é uma soma que a ficha resolve na
// segunda passada — a de atributo, que não seja a escala de um vital (essa é do
// `ComputeVitals`), ou a da Defesa do escudo.
func isSecondPassSum(m Modifier) bool {
	if m.Scale == nil {
		return false
	}
	switch m.Scale.Per {
	case "attribute":
		return m.Target.K != "maxPv" && m.Target.K != "maxPm" && m.Target.K != "attribute"
	case shieldDefenseScale:
		return true
	}
	return false
}

// splitSecondPassSums separa, item a item, os modificadores que são soma de
// segunda passada dos outros. O item que só tinha somas some da primeira lista.
func splitSecondPassSums(items []ActiveItem) (plain, sums []ActiveItem) {
	for _, it := range items {
		var rest, scaled []Modifier
		for _, m := range it.Modifiers {
			if isSecondPassSum(m) {
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

// resolveSecondPassSums põe em cada soma o número dela: o atributo efetivo, com
// o teto quando o verbete o escreve, ou a Defesa do escudo empunhado. A soma
// do escudo que dá zero — não há escudo na mão — sai, em vez de virar uma linha
// "+0" na conta da resistência.
func (r *Ruleset) resolveSecondPassSums(ch Character, sums []ActiveItem, effects ItemEffects) []ActiveItem {
	out := make([]ActiveItem, len(sums))
	for i, it := range sums {
		resolved := it
		resolved.Modifiers = make([]Modifier, 0, len(it.Modifiers))
		for _, m := range it.Modifiers {
			if m.Scale.Per == shieldDefenseScale {
				m.Amount *= r.wieldedShieldDefense(ch, effects)
				m.Scale = nil
				if m.Amount != 0 {
					resolved.Modifiers = append(resolved.Modifiers, m)
				}
				continue
			}
			amount := effectiveAttribute(ch, m.Scale.Attribute, effects)
			if m.Scale.Cap == "classLevel" {
				amount = min(amount, m.classLevel)
			}
			// O PISO vem antes da parcela: "Sabedoria (mínimo +1)", e depois o +2
			// de cada repetição (p50).
			if m.Scale.Min != 0 {
				amount = max(amount, m.Scale.Min)
			}
			m.Amount = amount + m.Scale.plus
			m.Scale = nil
			resolved.Modifiers = append(resolved.Modifiers, m)
		}
		out[i] = resolved
	}
	return out
}

// wieldedShieldDefense é o que os escudos em uso põem na Defesa, pelas parcelas
// que sobreviveram ao empilhamento. A parcela se reconhece pelo verbete de onde
// veio (`SourceID`): a do escudo, a da melhoria e a do encanto postos nele
// chegam todas com o id do escudo.
func (r *Ruleset) wieldedShieldDefense(ch Character, effects ItemEffects) int {
	shields := map[string]bool{}
	for _, it := range ch.Items {
		if it.Equipped == nil || it.CatalogID == nil {
			continue
		}
		if catalog := r.getCatalogItem(*it.CatalogID); catalog != nil && catalog.Category == "shield" {
			shields[*it.CatalogID] = true
		}
	}
	total := 0
	for _, c := range StatFor(effects, ModifierTarget{K: "defense"}).Contributions {
		if shields[c.SourceID] {
			total += c.Amount
		}
	}
	return total
}

// HandsNotFreeFlag diz que as mãos NÃO estão como o Estilo de Uma Arma pede: "uma arma corpo a corpo em uma
// das mãos e nada na outra" (p128).
//
// É NEGATIVA de propósito, com o `Unless` do lado do poder, e mora nas `bodyStateFlags`: ela cala o
// número e não tira o interruptor da oferta. Assim ele fica na tela e avisa (quem decide é o mestre), e o
// número é que não se mexe.
const HandsNotFreeFlag = "maos-sem-arma-sozinha"

var wieldedInOneHand = "wielded"

// HoldsALoneWeapon diz se há uma arma corpo a corpo em uma das mãos, e nada na outra.
func (r *Ruleset) HoldsALoneWeapon(ch Character) bool {
	inHand := 0
	fits := false
	for _, it := range ch.Items {
		if it.Equipped == nil || (*it.Equipped != "wielded" && *it.Equipped != "wielded2") {
			continue
		}
		inHand++
		if it.CatalogID == nil {
			continue
		}
		catalog := r.getCatalogItem(*it.CatalogID)
		fits = *it.Equipped == "wielded" && catalog != nil && catalog.Weapon != nil &&
			catalog.Weapon.Purpose == "melee" && catalog.Weapon.Hand != "two"
	}
	return inHand == 1 && fits
}

// handsAsAnItem levanta a flag como fonte da ficha, e não num item: o `ActiveItemsFor` lista o que o
// catálogo e as melhorias dão, e ela não é de item nenhum.
func (r *Ruleset) handsAsAnItem(ch Character) []ActiveItem {
	if r.HoldsALoneWeapon(ch) {
		return nil
	}
	return []ActiveItem{{Source: "as mãos", Equipped: &wieldedInOneHand, Modifiers: []Modifier{{
		Target: ModifierTarget{K: "flag", Name: HandsNotFreeFlag}, Amount: 1, BonusType: "untyped",
	}}}}
}
