package engine

import "fmt"

// Regras puras, sem catálogo: montagem dos modificadores de raça e de origem,
// resolução de atributo, proficiência de item, a conta de Carisma da Tormenta e
// a posse de poder de classe. Todas operam sobre os valores recebidos — sem
// global.

// raceModifiers monta os bônus de atributo como mods `attribute`, e depois os
// modificadores de cada habilidade mais os da variante escolhida.
func raceModifiers(race *RaceDefinition, variantChoices map[string]bool) []Modifier {
	out := []Modifier{}
	for _, attr := range AttributeKeys {
		amount, ok := race.AttributeBonuses[attr]
		if !ok || amount == 0 {
			continue
		}
		out = append(out, Modifier{
			Target:    ModifierTarget{K: "attribute", Name: attr},
			Amount:    amount,
			BonusType: "untyped",
			Note:      race.Name,
		})
	}
	for _, ability := range race.Abilities {
		out = append(out, ability.Modifiers...)
		if len(ability.Variants) == 0 {
			continue
		}
		for _, v := range ability.Variants {
			if variantChoices[v.ID] {
				out = append(out, v.Modifiers...)
				break
			}
		}
	}
	return out
}

// originModifiers soma os modificadores dos benefícios escolhidos (benefícios e
// depois poderUnico), e a ORDEM importa: o oráculo compara byte a byte.
func originModifiers(
	origin *OriginDefinition, choiceSet map[string]bool,
	poderPorUid func(string) *GeneralPower,
) []Modifier {
	out := []Modifier{}
	all := make([]OriginBenefit, 0, len(origin.Benefits)+1)
	all = append(all, origin.Benefits...)
	all = append(all, origin.UniquePower)
	for _, benefit := range all {
		if !choiceSet[benefit.ID] {
			continue
		}
		// O benefício que APONTA não tem modificador próprio: a regra é a do
		// poder geral, uma vez só (ALE-401).
		if benefit.PowerUid != "" {
			if p := poderPorUid(benefit.PowerUid); p != nil {
				out = append(out, p.Modifiers...)
			}
			continue
		}
		out = append(out, benefit.Modifiers...)
	}
	return out
}

// resolveAttributeDeltas resolve o `atributoMod` de uma raça numa lista ORDENADA
// de deltas — a ordem em que o oráculo os compara byte a byte. Escolha inválida
// volta como erro, e quem chama o engole em "nenhum mod": coluna ruim degrada a
// ficha, nunca a quebra.
func resolveAttributeDeltas(race *RaceAttributeEntry, floatingPicks []string, ancestry string) ([]attrDelta, error) {
	mod := race.AttributeMod
	switch mod.Kind {
	case "fixed":
		return mod.Mods.pairs, nil
	case "floating":
		return resolveFloating(race, mod, floatingPicks)
	default: // subraca-gated
		variant, ok := mod.Variants[ancestry]
		if ancestry == "" || !ok {
			return nil, fmt.Errorf(
				"resolveAtributoMod: %s requires a valid ascendência, got %q", race.Name, ancestry)
		}
		return variant.pairs, nil
	}
}

func resolveFloating(race *RaceAttributeEntry, mod AttributeMod, picks []string) ([]attrDelta, error) {
	if len(picks) != mod.Count {
		return nil, fmt.Errorf(
			"resolveAtributoMod: %s requires exactly %d floating picks, got %d",
			race.Name, mod.Count, len(picks))
	}
	if hasDuplicates(picks) {
		return nil, fmt.Errorf("resolveAtributoMod: %s floating picks must be distinct", race.Name)
	}
	if mod.Exclude != "" && contains(picks, mod.Exclude) {
		return nil, fmt.Errorf(
			"resolveAtributoMod: %s cannot place +%d in %s", race.Name, mod.Value, mod.Exclude)
	}
	result := make([]attrDelta, 0, len(picks)+1)
	for _, a := range picks {
		result = append(result, attrDelta{attr: a, amount: mod.Value})
	}
	if mod.Penalty != nil {
		result = append(result, attrDelta{attr: mod.Penalty.Attribute, amount: mod.Penalty.Value})
	}
	return result, nil
}

// RaceAttributeChoiceIsComplete diz se a raça JÁ recebeu a escolha de atributo
// que ela pede — o `+1 ×3` do humano, a ascendência do suraggel.
//
// Exportada porque a ficha precisa mostrar essa pendência, e a forja promete por
// escrito "dá para criar assim e terminar na ficha". Ela PERGUNTA em vez de
// repetir a condição: quantas escolhas cada raça pede, que elas sejam distintas
// e qual atributo é proibido já está no `resolveFloating`, e uma segunda cópia
// divergiria no dia em que uma raça nova tivesse uma quarta condição.
//
// Raça desconhecida conta como completa: não dá para cobrar escolha de uma raça
// que o catálogo não tem.
func (c *Catalogs) RaceAttributeChoiceIsComplete(raceName, choicesJSON string) bool {
	race := c.raceEntryByName(raceName)
	if race == nil {
		return true
	}
	choice := parseRaceAttributeChoices(choicesJSON)
	_, err := resolveAttributeDeltas(race, choice.floatingPicks, choice.ancestry)
	return err == nil
}

// RequiredProficiency é a proficiência que um item exige para ser usado sem
// penalidade, ou "" quando ele não exige nenhuma.
//
// Exportada porque a Mochila marca o item equipado SEM proficiência, e essa
// marca tem de sair da MESMA tabela que decide a penalidade do motor. Uma
// segunda cópia daria uma tela que avisa sobre um item e um motor que penaliza
// outro.
func RequiredProficiency(item *CatalogItem) string { return requiredProficiency(item) }

// requiredProficiency é a tabela: a proficiência que um item exige para ser
// usado sem penalidade, ou "" para nenhuma.
func requiredProficiency(item *CatalogItem) string {
	switch item.Category {
	case "weapon-simple":
		return "armas-simples"
	case "weapon-martial":
		return "armas-marciais"
	case "weapon-exotic":
		return "armas-exoticas"
	case "weapon-firearm":
		return "armas-de-fogo"
	case "armor-light":
		return "armaduras-leves"
	case "armor-heavy":
		return "armaduras-pesadas"
	case "shield":
		return "escudos"
	}
	return ""
}

// carismaLossFromPowers mora no tormenta.go — a camada de coleta a reusa.

// OwnsClassPower é a REGRA de posse de um poder de classe: automático pelo
// nível, escolhido pelo id, ou concedido por uma escolha da classe (o caminho
// do arcanista, o deus do clérigo).
//
// Exportada porque a aba Poderes lista o que o personagem TEM, e essa lista
// precisa ser a mesma que a derivação soma. Uma segunda leitura daria uma tela
// mostrando um poder que a ficha não conta — ou o contrário, que é pior, porque
// o número aparece sem explicação.
func OwnsClassPower(
	power *ClassPower,
	classLevel int,
	chosen map[string]bool,
	choice ClassChoiceSelections,
) bool {
	return ownsClassPower(power, classLevel, chosen, choice)
}

// ownsClassPower: automático pelo nível, eletivo pelo id escolhido, ou concedido
// por um valor casado em `classChoices`.
func ownsClassPower(
	power *ClassPower,
	classLevel int,
	chosen map[string]bool,
	choice ClassChoiceSelections,
) bool {
	if power.GrantedAtLevel != nil && *power.GrantedAtLevel <= classLevel {
		return true
	}
	if chosen[power.ID] {
		return true
	}
	if power.GrantedByChoice == nil {
		return false
	}
	return choice.value(power.GrantedByChoice.Field) == power.GrantedByChoice.Value
}

// ClassChoiceSelections são as escolhas de devoto e de caminho de UMA classe.
type ClassChoiceSelections struct {
	Devotee string `json:"devoto"`
	Path    string `json:"caminho"`
}

func (s ClassChoiceSelections) value(field string) string {
	if field == "devoto" {
		return s.Devotee
	}
	if field == "caminho" {
		return s.Path
	}
	return ""
}
