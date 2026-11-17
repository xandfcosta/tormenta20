package sheet

// ReceivedFrom diz se a ficha tem um efeito ativo dado pelo poder `catalogID` — a
// reserva do Estandarte que o mestre já entregou nesta cena, por exemplo. A faixa da
// mesa e a recusa do gesto perguntam aqui, para "já recebeu" ser uma resposta só.
//
//	sheet.ReceivedFrom(dto, book.BannerPowerID)
func ReceivedFrom(dto CharacterDTO, catalogID string) bool {
	for _, e := range dto.ActiveEffects {
		if e.CatalogID == catalogID {
			return true
		}
	}
	return false
}
