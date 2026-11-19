package sheet

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// WrittenPoison é o que o diálogo do veneno diz (p161-162): a inoculação com a CD de quem aplica, e o
// efeito com o que passar no teste faz. A CD é 10 + metade do nível + a Inteligência (p161, p227),
// mais o que o próprio veneno soma e o que a ficha dá (o Veneno Potente, p74) — decomposta, para quem
// lê saber de onde vem cada parcela. O app não aplica o veneno: quem lê a CD e aplica é o mestre.
//
//	WrittenPoison("beladona", 8, computed) // "Ingestão · Fortitude CD 27 (+5 Beladona, +5 Veneno Potente)", "fica paralisada por 3 rodadas; se passar, fica lenta por 3 rodadas", true
func WrittenPoison(catalogID string, level int64, computed engine.ComputedSheet) (headline, effect string, ok bool) {
	item := book.ItemByID(catalogID)
	if item == nil || item.Consumable == nil || item.Consumable.Poison == nil {
		return "", "", false
	}
	poison := item.Consumable.Poison
	effect = poison.Effect
	if poison.OnSave != "" {
		effect += "; se passar, " + poison.OnSave
	}
	dc, known := itemSaveDc(level, computed, "intelligence")
	if !known {
		return poison.Inoculation + " · Fortitude", effect, true
	}
	dc += poison.DcBonus
	parts := []string{}
	if poison.DcBonus != 0 {
		parts = append(parts, book.WithSign(poison.DcBonus)+" "+item.Name)
	}
	if computed.PoisonDc != nil {
		dc += computed.PoisonDc.Total
		parts = append(parts, namedParts(computed.PoisonDc)...)
	}
	save := "Fortitude CD " + strconv.Itoa(dc)
	if len(parts) > 0 {
		save += " (" + strings.Join(parts, ", ") + ")"
	}
	return poison.Inoculation + " · " + save, effect, true
}
