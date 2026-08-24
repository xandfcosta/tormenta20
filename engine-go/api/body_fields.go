package api

// Leitura dos campos do CORPO da requisição.
//
// Estes ajudantes vieram do gateway do socket.io quando ele foi apagado
// (ALE-253) e não mudaram uma linha: o corpo continua sendo um `map[string]any`
// vindo de JSON, então quem lê um campo tolerante a tipo (o JSON não distingue
// inteiro de float) continua fazendo falta. O que mudou é de ONDE o mapa vem —
// era o argumento do evento, agora é o `decodeJSON` da rota.

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

// intField reads an integer body field (JSON numbers arrive as float64).
func intField(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	switch v := m[key].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	}
	return 0, false
}

// optInt returns a pointer to the body's value for key, or nil when absent (so a vitals
// patch/delta only touches the fields the client actually sent).
func optInt(m map[string]any, key string) *int64 {
	if v, ok := intField(m, key); ok {
		return ptrInt64(v)
	}
	return nil
}

// parseBoardToken lê a peça do corpo da mensagem. Posição ausente vira o
// primeiro quadrado livre da fileira de entrada — antes era (0,0) fixo, e com o
// "+ Peça" da ALE-178 duas peças criadas seguidas nasciam UMA EM CIMA DA OUTRA
// (ALE-166).
func parseBoardToken(body map[string]any) (BoardToken, bool) {
	token := BoardToken{
		Label: stringField(body, "label"),
		Kind:  stringField(body, "kind"),
	}
	x, temX := intField(body, "x")
	y, temY := intField(body, "y")
	if temX {
		token.X = int(x)
	}
	if temY {
		token.Y = int(y)
	}
	if footprint, ok := intField(body, "footprint"); ok {
		token.Footprint = int(footprint)
	}
	if entryID := stringField(body, "entryId"); entryID != "" {
		token.EntryID = &entryID
	}
	if characterID, ok := intField(body, "characterId"); ok {
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
	if footprint, ok := intField(m, "footprint"); ok {
		side := int(footprint)
		patch.Footprint = &side
	}
	if x, ok := intField(m, "x"); ok {
		col := int(x)
		patch.X = &col
	}
	if y, ok := intField(m, "y"); ok {
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
