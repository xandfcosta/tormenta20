package sheetui

import (
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// itemSheetPoison é o veneno no diálogo do item: a inoculação com a CD, e o efeito. A frase é da
// `sheet.WrittenPoison`.
type itemSheetPoison struct {
	Headline string
	Effect   string
}

func poisonOnTheItemSheet(dto sheet.CharacterDTO, item sheet.ItemDTO, computed engine.ComputedSheet) *itemSheetPoison {
	if item.CatalogID == nil {
		return nil
	}
	headline, effect, ok := sheet.WrittenPoison(*item.CatalogID, dto.Level, computed)
	if !ok {
		return nil
	}
	return &itemSheetPoison{Headline: headline, Effect: effect}
}
