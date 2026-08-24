package plataforma

// ptrInt64 acompanha o `OptInt`: sem ele o ajudante não consegue devolver
// "ausente" distinto de "zero", que é a razão de ele existir.
func ptrInt64(v int64) *int64 { return &v }

// Leitura dos campos do CORPO da requisição.
//
// Estes ajudantes vieram do gateway do socket.io quando ele foi apagado
// (ALE-253) e não mudaram uma linha: o corpo continua sendo um `map[string]any`
// vindo de JSON, então quem lê um campo tolerante a tipo (o JSON não distingue
// inteiro de float) continua fazendo falta. O que mudou é de ONDE o mapa vem —
// era o argumento do evento, agora é o `DecodeJSON` da rota.

func StringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

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

// OptInt returns a pointer to the body's value for key, or nil when absent (so a vitals
// patch/delta only touches the fields the client actually sent).
func OptInt(m map[string]any, key string) *int64 {
	if v, ok := IntField(m, key); ok {
		return ptrInt64(v)
	}
	return nil
}
