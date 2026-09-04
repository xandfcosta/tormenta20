package tabuleiro

import "t20engine/plataforma"

// Leitura dos campos do corpo que são do TABULEIRO (ALE-254).
//
// Saíram do `body_fields.go` quando o `plataforma/` nasceu: aqueles leem campo
// genérico e não sabem o que é peça; estes leem PEÇA, e peça é conceito do
// jogo. A linha do glossário sobre `plataforma` diz exatamente isto — se um
// conceito do jogo entrar lá, a fronteira está errada.

// Aqui morava o `ParseBoardToken`, que lia a peça do CORPO da mensagem — e que
// nunca teve chamador, nem de produção nem de teste (ALE-291).
//
// Ele foi escrito para o "+ Peça" da ALE-178, e o gesto não veio. Quando ele
// finalmente chegou, veio com outra forma: a posição vai no CAMINHO e não no
// corpo, pela razão que o `quadradoDoCaminho` registra — coordenada negativa é
// lugar legítimo, e o valor tem de ser o do clique que aconteceu, não o de um
// estado que outro gesto poderia ter mexido. O resto (nome, tamanho, aparência)
// chega pelos sinais da tira, já tipado.
//
// Ressuscitá-lo exigiria montar um `map[string]any` sintético só para tornar a
// devolver os mesmos campos — cerimônia sobre uma função escrita para um pedido
// que este desenho não faz.

// ParseTokenPatch lê só os campos PRESENTES: ausente é "não mexa", não "zere".
func ParseTokenPatch(raw any) tokenPatch {
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

// ParseMarkerPatch lê só os campos PRESENTES.
func ParseMarkerPatch(raw any) markerPatch {
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
