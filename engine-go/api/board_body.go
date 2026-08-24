package api

import "t20engine/plataforma"

// Leitura dos campos do corpo que são do TABULEIRO (ALE-254).
//
// Saíram do `body_fields.go` quando o `plataforma/` nasceu: aqueles leem campo
// genérico e não sabem o que é peça; estes leem PEÇA, e peça é conceito do
// jogo. A linha do glossário sobre `plataforma` diz exatamente isto — se um
// conceito do jogo entrar lá, a fronteira está errada.

// parseBoardToken lê a peça do corpo da mensagem. Posição ausente vira o
// primeiro quadrado livre da fileira de entrada — antes era (0,0) fixo, e com o
// "+ Peça" da ALE-178 duas peças criadas seguidas nasciam UMA EM CIMA DA OUTRA
// (ALE-166).
func parseBoardToken(body map[string]any) (BoardToken, bool) {
	token := BoardToken{
		Label: plataforma.StringField(body, "label"),
		Kind:  plataforma.StringField(body, "kind"),
	}
	x, temX := plataforma.IntField(body, "x")
	y, temY := plataforma.IntField(body, "y")
	if temX {
		token.X = int(x)
	}
	if temY {
		token.Y = int(y)
	}
	if footprint, ok := plataforma.IntField(body, "footprint"); ok {
		token.Footprint = int(footprint)
	}
	if entryID := plataforma.StringField(body, "entryId"); entryID != "" {
		token.EntryID = &entryID
	}
	if characterID, ok := plataforma.IntField(body, "characterId"); ok {
		token.CharacterID = &characterID
	}
	if hidden, ok := body["hidden"].(bool); ok {
		token.Hidden = hidden
	}
	return token, temX && temY
}

// parseTokenPatch lê só os campos PRESENTES: ausente é "não mexa", não "zere".
func parseTokenPatch(raw any) tokenPatch {
	patch := tokenPatch{}
	m, ok := raw.(map[string]any)
	if !ok {
		return patch
	}
	if label, ok := m["label"].(string); ok {
		patch.Label = &label
	}
	if hidden, ok := m["hidden"].(bool); ok {
		patch.Hidden = &hidden
	}
	if footprint, ok := plataforma.IntField(m, "footprint"); ok {
		side := int(footprint)
		patch.Footprint = &side
	}
	if x, ok := plataforma.IntField(m, "x"); ok {
		col := int(x)
		patch.X = &col
	}
	if y, ok := plataforma.IntField(m, "y"); ok {
		row := int(y)
		patch.Y = &row
	}
	return patch
}

// parseMarkerPatch lê só os campos PRESENTES.
func parseMarkerPatch(raw any) markerPatch {
	patch := markerPatch{}
	m, ok := raw.(map[string]any)
	if !ok {
		return patch
	}
	if text, ok := m["text"].(string); ok {
		patch.Text = &text
	}
	if color, ok := m["color"].(string); ok {
		patch.Color = &color
	}
	if hidden, ok := m["hidden"].(bool); ok {
		patch.Hidden = &hidden
	}
	return patch
}
