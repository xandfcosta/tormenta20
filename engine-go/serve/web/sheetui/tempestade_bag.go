package sheetui

import (
	"strconv"
	"strings"
)

// A ABA MOCHILA DA FICHA PARALELA (ALE-473), pelo mockup aprovado. O dado é o
// da ficha de hoje (`bagPanel`); daqui sai só o que a aba lê diferente: as
// linhas do que está em uso, quantos lugares sobram, e os gestos que abrem os
// diálogos carregando o estado do item.

// inUseRow é um item equipado como linha da lista: o nome, onde ele está e o
// que concede.
type inUseRow struct {
	Name    string
	Meta    string
	Command string
}

// inUseRowsOf são as linhas de um teto (mãos ou vestidos), na ordem das
// posições. A arma de duas mãos é UMA linha, como era um cartão largo.
func inUseRowsOf(positions []equippedPosition) []inUseRow {
	var rows []inUseRow
	for _, p := range positions {
		if p.Card == nil {
			continue
		}
		parts := []string{p.Label}
		if p.Card.NoProficiency {
			parts = append(parts, "sem proficiência")
		}
		for _, chip := range p.Card.Chips {
			if !chip.Inactive {
				parts = append(parts, chip.Text)
			}
		}
		rows = append(rows, inUseRow{Name: p.Card.Name, Meta: strings.Join(parts, " · "), Command: p.Card.Command})
	}
	return rows
}

// freePlaces é a linha que diz quanto sobra no teto: "3 lugares livres".
//
//	freePlaces(1, 4, "lugar livre", "lugares livres") // "3 lugares livres"
func freePlaces(used, ceiling int, one, many string) string {
	free := ceiling - used
	switch {
	case free <= 0:
		return ""
	case free == 1:
		return "1 " + one
	}
	return strconv.Itoa(free) + " " + many
}

// itemSheetByCommand é a folha de um item, para a linha do guardado saber se
// ele se usa.
func itemSheetByCommand(v View, command string) *itemSheet {
	for i := range v.Bag.Sheets {
		if v.Bag.Sheets[i].Command == command {
			return &v.Bag.Sheets[i]
		}
	}
	return nil
}

// stowedMeta é a nota da linha do guardado: a quantidade e as melhorias.
func stowedMeta(tile stowedTile) string {
	var parts []string
	if tile.Quantity > 1 {
		parts = append(parts, "×"+strconv.FormatInt(tile.Quantity, 10))
	}
	parts = append(parts, tile.Overlays...)
	return strings.Join(parts, " · ")
}

func itemSheetID(command string) string        { return "item-" + command }
func itemEditID(command string) string         { return "item-edit-" + command }
func itemImprovementsID(command string) string { return "item-improvements-" + command }

// improvementsOpens carrega o que o item JÁ tem antes de abrir as melhorias —
// quem troca de item limpa, senão as melhorias marcadas numa espada
// apareceriam marcadas na armadura aberta em seguida.
func improvementsOpens(sheet itemSheet) string {
	return "$item_improvements = " + improvementWrittenList(sheet) + "; $item_material = '" + chosenMaterial(sheet) + "'"
}

// editOpens põe os valores atuais no formulário de editar.
func editOpens(sheet itemSheet) string {
	return "$item_name = " + strconv.Quote(sheet.Name) +
		"; $item_qty = " + strconv.FormatInt(sheet.Quantity, 10) +
		"; $item_slots = " + strings.Replace(sheet.Slots, ",", ".", 1)
}

// loadKind pinta a barra da carga: estourada, ela tem a cor do perigo.
func loadKind(load loadMeter) string {
	if load.Overloaded {
		return "danger"
	}
	return ""
}
