package wire

// Leitura dos campos do CORPO da requisição.
//
// Estes ajudantes vieram do gateway do socket.io quando ele foi apagado
// (ALE-253) e não mudaram uma linha: o corpo continua sendo um `map[string]any`
// vindo de JSON, então quem lê um campo tolerante a tipo (o JSON não distingue
// inteiro de float) continua fazendo falta. O que mudou é de ONDE o mapa vem —
// era o argumento do evento, agora é o `DecodeJSON` da rota.

// IntField reads an integer body field (JSON numbers arrive as float64).
func IntField(m map[string]any, key string) (int64, bool) {
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
