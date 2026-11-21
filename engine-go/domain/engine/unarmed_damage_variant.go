package engine

import "fmt"

// AS LINHAS DO DANO DESARMADO (ALE-655, desenho do dono).
//
// A Arma Improvisada e a Chave (p76) sobem um passo o dano desarmado numa situação que só a mesa conhece: atacar
// com uma cadeira, esmagar quem se agarra. Interruptor na ficha foi recusado, porque a tela já tem muitos: a carta
// desarmada leva uma LINHA por situação, com o dado já um passo acima, e quem atacou escolhe a linha no passo do
// dano, depois do "acertou" do mestre. O que é da mesa continua da mesa — achar o objeto, a arma que quebra no
// erro com d20 ímpar, o teste de agarrar que causa dano (p234).

// ClassPowerUnarmedDamageVariant é a linha que o poder dá: o rótulo da situação e quantos passos o dado sobe.
type ClassPowerUnarmedDamageVariant struct {
	Label string `json:"label"`
	Steps int    `json:"steps"`
}

// DamageVariant é uma linha do dano desarmado: o poder que a dá (`ID`, `Source`, `BookPage`), a situação
// (`Label`) e o dado já com o passo (`Damage`).
type DamageVariant struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Source   string `json:"source"`
	BookPage int    `json:"bookPage,omitempty"`
	Damage   string `json:"damage"`
	Steps    int    `json:"steps"`
}

// unarmedDamageVariants são as linhas que os poderes da ficha dão, na ordem das classes, já com o dado da carta.
func (r *Ruleset) unarmedDamageVariants(ch Character, damage string) []DamageVariant {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	var out []DamageVariant
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			grant := power.UnarmedDamageVariant
			if grant == nil {
				continue
			}
			out = append(out, DamageVariant{
				ID: power.ID, Label: grant.Label, Source: power.Name, BookPage: power.BookPage,
				Damage: DamageStep(damage, grant.Steps), Steps: grant.Steps,
			})
		}
	}
	return out
}

// WithDamageVariant é a carta com a linha escolhida: o dado da linha no lugar do da carta, e o resto da conta igual.
// Sem linha (`id` vazio), a carta como está.
//
//	held, err := engine.WithDamageVariant(card, "class.lutador.chave")
func WithDamageVariant(card WeaponCard, id string) (WeaponCard, error) {
	if id == "" {
		return card, nil
	}
	for _, variant := range card.DamageVariants {
		if variant.ID == id {
			card.Damage = DamageStep(card.Damage, variant.Steps)
			return card, nil
		}
	}
	return card, fmt.Errorf("a carta %s não tem a linha de dano %q; as que ela tem: %v", card.Name, id, variantIDs(card.DamageVariants))
}

func variantIDs(variants []DamageVariant) []string {
	ids := make([]string, 0, len(variants))
	for _, v := range variants {
		ids = append(ids, v.ID)
	}
	return ids
}
