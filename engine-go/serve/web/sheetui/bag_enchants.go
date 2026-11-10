package sheetui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O ENCANTO NA FOLHA DO ITEM: o que ele faz, com o gasto de PM e a escolha dele. Quem encanta é o mestre,
// porque encanto não se compra, se acha ou se fabrica (p333).

// appliedEnchants são os encantos que o item carrega, na ordem do livro, com o que cada um faz.
//
// Ordenados pelo NOME e não pela ordem de gravação: o `EnchantsOfItem` devolve
// por id, e "encanto-ameacadora" antes de "encanto-formidavel" é a ordem do
// slug, não a que se lê.
func appliedEnchants(item sheet.ItemDTO) []overlayRow {
	rows := []overlayRow{}
	for _, id := range item.Enchants {
		entry := book.ItemByID(id)
		if entry == nil {
			continue
		}
		rows = append(rows, overlayRow{Name: enchantName(*entry, item.EnchantChoices[id]), Effect: enchantEffect(*entry)})
	}
	sort.SliceStable(rows, func(a, b int) bool { return rows[a].Name < rows[b].Name })
	return rows
}

// enchantName é o nome do encanto com a escolha dele: "Anticriatura · contra morto-vivo".
func enchantName(entry book.Item, chosen string) string {
	if chosen == "" {
		return entry.Name
	}
	return entry.Name + " · contra " + book.ChosenKindName(chosen)
}

// enchantBadgeName é o nome do encanto no crachá da Mochila, com a escolha: "Anticriatura contra morto-vivo".
// Sem o "·" da folha, porque no crachá o "·" já separa um encanto do outro.
func enchantBadgeName(entry book.Item, chosen string) string {
	if chosen == "" {
		return entry.Name
	}
	return entry.Name + " contra " + book.ChosenKindName(chosen)
}

// enchantEffect é o que o encanto faz: as notas do verbete e o gasto de PM dele. Sem os dois, "sem efeito
// mecânico", que antes era o que a folha dizia também do encanto cujo efeito inteiro é o gasto.
func enchantEffect(entry book.Item) string {
	parts := []string{}
	if notes := modifierNotes(entry); notes != "" {
		parts = append(parts, notes)
	}
	if spend := enchantSpendWritten(entry.Boost); spend != "" {
		parts = append(parts, spend)
	}
	if len(parts) == 0 {
		return "sem efeito mecânico"
	}
	return strings.Join(parts, "; ")
}

// enchantSpendWritten é o gasto do encanto como a mesa lê: "2 PM no golpe, uma vez por rodada: +4d8 de dano
// contra o tipo escolhido", ou o que ele dispara em vez do golpe (a bola de fogo da Flamejante, p335).
func enchantSpendWritten(boost *engine.ItemBoost) string {
	if boost == nil {
		return ""
	}
	price := strconv.Itoa(boost.PM) + " PM"
	if boost.HitPointCost != "" {
		price += " e " + boost.HitPointCost + " PV"
	}
	if in := boost.Instead; in != nil {
		return fmt.Sprintf("%s em vez do golpe, uma vez por rodada: %s de %s de %s, alcance %s, %s %s",
			price, in.Name, in.Damage, in.DamageType, in.Range, in.Save, in.OnSave)
	}
	said := []string{}
	for _, m := range boost.Modifiers {
		switch m.Target.K {
		case "announcedCondition":
			line := "no acerto, " + book.ConditionName(m.Target.Name)
			if m.Note != "" {
				line += " (" + m.Note + ")"
			}
			said = append(said, line)
		case "announcedEffect":
			said = append(said, "no acerto, "+m.Note)
		default:
			if m.Note != "" {
				said = append(said, m.Note)
			}
		}
	}
	return price + " no golpe, uma vez por rodada: " + strings.Join(said, ", ")
}
