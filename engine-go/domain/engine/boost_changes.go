package engine

import (
	"strconv"
	"strings"
)

// O QUE MUDA UM IMPULSO DE FORA DELE (ALE-652, fatia C): a Arma Sagrada troca o dado do Golpe Divino na arma
// preferida da divindade (p82). O impulso continua sendo o mesmo poder; o que muda é o DADO que ele rola.
//
// A Arma Sagrada é do poder, e por isso mora no verbete dela (`FavoredWeaponDie`).

// ClassPowerFavoredWeaponDie é o dado que um poder põe no impulso `Boost` quando o golpe é com a arma preferida
// da divindade: a Arma Sagrada leva o Golpe Divino a d12 (p82).
type ClassPowerFavoredWeaponDie struct {
	Boost string `json:"boost"`
	Faces int    `json:"faces"`
}

// boostChanges é o que muda um impulso nesta ficha, agora.
type boostChanges struct {
	// favoredFaces é o dado do impulso na arma preferida (12 com a Arma Sagrada), ou zero; favoredWeapon é o
	// id dela no catálogo, ou vazio para quem não tem uma (Lena, Marah, e o "todas" de Nimb, p102).
	favoredFaces  int
	favoredWeapon string
}

func (r *Ruleset) boostChangesOf(ch Character, activeConditionals map[string]bool, powerID string) boostChanges {
	changes := boostChanges{}
	if r.book != nil {
		changes.favoredWeapon = r.book.favoredWeapons[devotionOf(ch)]
	}
	for _, power := range r.ownedPowersOfEveryClass(ch) {
		if die := power.FavoredWeaponDie; die != nil && die.Boost == powerID {
			changes.favoredFaces = die.Faces
		}
	}
	return changes
}

// ownedPowersOfEveryClass são os poderes de classe que a ficha tem, na ordem das classes.
func (r *Ruleset) ownedPowersOfEveryClass(ch Character) []*ClassPower {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	var out []*ClassPower
	for _, entry := range ch.Classes {
		out = append(out, r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName])...)
	}
	return out
}

// onDice devolve os modificadores do impulso com o dado trocado por `faces`, quando não é zero.
func (c boostChanges) onDice(mods []Modifier, faces int) []Modifier {
	out := make([]Modifier, len(mods))
	for i, m := range mods {
		if m.Dice != "" {
			count, die, _ := strings.Cut(m.Dice, "d")
			n, err := strconv.Atoi(count)
			if err == nil {
				if faces > 0 {
					die = strconv.Itoa(faces)
				}
				m.Dice = strconv.Itoa(n) + "d" + die
			}
		}
		out[i] = m
	}
	return out
}

// isTheFavoredWeapon diz se a carta é a arma preferida: pelo item, ou pela arma base do item específico, que é
// também a arma base dele (decisão do dono, como em `itemCard`).
func (r *Ruleset) isTheFavoredWeapon(card WeaponCard, favored string) bool {
	if favored == "" || card.CatalogID == "" {
		return false
	}
	if card.CatalogID == favored {
		return true
	}
	item := r.getCatalogItem(card.CatalogID)
	return item != nil && item.ComposedOf != nil && item.ComposedOf.Base == favored
}

// buysWritten é o que os modificadores do impulso compram, escrito: o dado ou o número do primeiro que cresce
// com o degrau ("+2d12", "+4").
func buysWritten(mods []Modifier) string {
	for _, m := range mods {
		if m.Scale != nil {
			continue
		}
		if m.Dice != "" {
			return "+" + m.Dice
		}
		return "+" + strconv.Itoa(m.Amount)
	}
	return ""
}
