package api

// Andaime dos testes deste pacote (ALE-254).
//
// Eram helpers do `session_state_test.go`, que mudou para `aovivo/` junto com o
// que ele testa. Teste não exporta para o vizinho, e helper compartilhado entre
// pacotes seria uma dependência que ninguém declarou — então cada lado fica com
// o seu. Duplicação de ANDAIME é barata; duplicação de REGRA não seria, e não é
// o caso aqui.

import "t20engine/aovivo"

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func labels(st *aovivo.SessionRuntimeState) []string {
	out := make([]string, len(st.Initiative))
	for i, e := range st.Initiative {
		out[i] = e.Label
	}
	return out
}

func npc(label string, init int) aovivo.InitiativeEntry {
	return aovivo.InitiativeEntry{Label: label, Initiative: init, Type: "npc"}
}
