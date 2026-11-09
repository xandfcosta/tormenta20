package engine

// A MELHORIA APRIMORADO (p164): "só pode ser aplicada a uma ferramenta ou vestuário que modifique uma perícia
// (reduza uma penalidade ou forneça um bônus) e fornece um bônus de +1 nessa perícia". O que ela soma depende
// do item em que entra, e por isso a linha mora aqui e não no verbete dela.

// ImprovedItemImprovement é o id da melhoria.
const ImprovedItemImprovement = "melhoria-aprimorado"

// SkillTie é a perícia a que o item está ligado sem somar número, e a situação a que o livro a limita.
type SkillTie struct {
	Skill string `json:"skill"`
	When  string `json:"when,omitempty"`
}

// CraftExpertise é a perícia de Ofício sem a especialidade. A ficha guarda "Ofício (ferreiro)", e o catálogo
// tem um item só para "cada perícia de Ofício" (p158): o que mira esta soma em todas as da ficha.
const CraftExpertise = "Ofício"

// ImprovedSkillsOf são os alvos do +1 da melhoria naquele item: o bônus que ele já dá e as perícias ligadas.
// Vazio quer dizer que o item não recebe a melhoria. Onde o item só vale numa situação, o +1 vale nela: a
// condição é a mesma do bônus (a luneta), ou a da penalidade que ele tira (a gazua).
func ImprovedSkillsOf(modifiers []Modifier, ties []SkillTie) []Modifier {
	out := []Modifier{}
	for _, m := range modifiers {
		if m.Target.K == "expertise" && m.Amount > 0 {
			out = append(out, improvedSkill(m.Target.Name, m.Condition))
		}
	}
	for _, tie := range ties {
		condition := &ModifierCondition{C: "inUse"}
		if tie.When != "" {
			condition = &ModifierCondition{C: "context", Note: tie.When}
		}
		out = append(out, improvedSkill(tie.Skill, condition))
	}
	return out
}

func improvedSkill(skill string, condition *ModifierCondition) Modifier {
	return Modifier{
		Target: ModifierTarget{K: "expertise", Name: skill}, Amount: 1, BonusType: "untyped",
		Note: "Aprimorado", Condition: condition,
	}
}
