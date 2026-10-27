package engine

// AS ESCOLHAS DAS HABILIDADES DE RAÇA no motor (ALE-493, ALE-502, ALE-504): o
// que se escolheu mora em `abilityPicks`, no blob da raça, pelo id da
// habilidade — perícia (o +2 da Deformidade, o Ofício do Vanguardista), poder
// (a troca da Deformidade, o poder geral da Memória Póstuma) ou habilidade de
// outra raça (a Memória Póstuma do osteon).

// abilityChoiceModifiers são os modificadores das habilidades que pedem
// escolha, um por perícia escolhida. O poder escolhido — o da Tormenta que a
// Deformidade troca por um bônus, o geral da Memória Póstuma — não é perícia,
// e fica de fora (`RaceHeldPowers`).
func (r *Ruleset) abilityChoiceModifiers(race *RaceDefinition, picks map[string][]string) []Modifier {
	out := []Modifier{}
	for _, ability := range race.Abilities {
		if ability.Choice == nil {
			continue
		}
		skills := []string{}
		for _, pick := range picks[ability.ID] {
			if !r.isPower(pick) && r.borrowable(pick) == nil {
				skills = append(skills, pick)
			}
		}
		out = append(out, expandChoiceModifiers(ability.Modifiers, skills, nil)...)
	}
	return out
}

// RaceHeldPowers são os poderes que uma habilidade de raça deu à ficha: o da
// Tormenta que a Deformidade trocou por um bônus (p24), o geral da Memória
// Póstuma (p29).
//
//	r.RaceHeldPowers(lefou) // ["olhos-vermelhos"]
func (r *Ruleset) RaceHeldPowers(ch Character) []string {
	out := []string{}
	for _, held := range r.raceHeld(ch) {
		out = append(out, held.id)
	}
	return out
}

// heldPower é um poder dado por habilidade de raça, e se ele está isento da
// perda de Carisma — só a troca da Deformidade está (p24).
type heldPower struct {
	id             string
	charismaExempt bool
}

func (r *Ruleset) raceHeld(ch Character) []heldPower {
	out := []heldPower{}
	for _, applied := range r.appliedRaces(ch) {
		for _, ability := range applied.race.Abilities {
			if ability.Choice == nil {
				continue
			}
			for _, pick := range applied.choice.abilityPicks[ability.ID] {
				if r.isPower(pick) {
					out = append(out, heldPower{id: pick, charismaExempt: ability.isTormentaSwap()})
				}
			}
		}
	}
	return out
}

// isPower diz se a escolha é um poder geral ou da Tormenta, e não perícia.
func (r *Ruleset) isPower(id string) bool {
	return r.isTormentaPower(id) || r.getGeneralPower(id) != nil
}

// borrowedAbility é uma habilidade de OUTRA raça que a ficha tomou — a
// Memória Póstuma do osteon (p29, ALE-504).
type borrowedAbility struct {
	race    *RaceDefinition
	ability RaceAbility
}

// borrowable acha a habilidade de raça pelo id, ou nil.
func (r *Ruleset) borrowable(id string) *borrowedAbility {
	for _, race := range r.book.racesByID {
		for _, ability := range race.Abilities {
			if ability.ID == id {
				return &borrowedAbility{race: race, ability: ability}
			}
		}
	}
	return nil
}

// borrowedOf são as habilidades que as escolhas da raça tomaram de outra raça.
func (r *Ruleset) borrowedOf(race *RaceDefinition, picks map[string][]string) []borrowedAbility {
	out := []borrowedAbility{}
	for _, ability := range race.Abilities {
		if ability.Choice == nil {
			continue
		}
		for _, pick := range picks[ability.ID] {
			if b := r.borrowable(pick); b != nil && b.race.ID != race.ID {
				out = append(out, *b)
			}
		}
	}
	return out
}

// borrowedModifiers são os modificadores das habilidades tomadas: os dela, a
// variante escolhida, ou a escolha dela — guardadas pela chave da habilidade
// emprestada. O ATRIBUTO fica de fora: o livro dá "uma habilidade dessa raça",
// e não os atributos dela.
func (r *Ruleset) borrowedModifiers(race *RaceDefinition, picks map[string][]string) []Modifier {
	out := []Modifier{}
	for _, b := range r.borrowedOf(race, picks) {
		own := b.ability.Modifiers
		if b.ability.Choice != nil {
			own = expandChoiceModifiers(b.ability.Modifiers, picks[b.ability.ID], nil)
		}
		for _, v := range b.ability.Variants {
			if contains(picks[b.ability.ID], v.ID) {
				own = append(append([]Modifier{}, own...), v.Modifiers...)
			}
		}
		for _, m := range own {
			if m.Target.K != "attribute" {
				out = append(out, m)
			}
		}
	}
	return out
}

// borrowedSizeRace é a raça cujo tamanho o osteon tomou: "se a raça era de
// tamanho diferente de Médio, você também possui sua categoria de tamanho"
// (p29). Vazia quando não há empréstimo.
func (r *Ruleset) borrowedSizeRace(ch Character) string {
	for _, applied := range r.appliedRaces(ch) {
		for _, b := range r.borrowedOf(applied.race, applied.choice.abilityPicks) {
			return b.race.ID
		}
	}
	return ""
}
