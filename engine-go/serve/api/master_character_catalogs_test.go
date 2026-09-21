package api

import (
	"strings"
	"testing"
)

// O guarda dos três catálogos DO PERSONAGEM (ALE-264).

// TestTheThreeNewTabsDrawWhatTheirEntryHas.
//
// INTEGRAÇÃO e não asserção de componente porque o que se protege é a
// composição: catálogo novo (`classes.json`), tipo novo, aba nova e cartão novo
// — quatro saltos, e a cena passaria a existir vazia se qualquer um falhasse,
// sem erro nenhum.
func TestTheThreeNewTabsDrawWhatTheirEntryHas(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	cases := []struct{ aba, want, reason string }{
		{"racas", "Graça de Glórienn", "a habilidade de raça vem do catálogo"},
		{"racas", "+2 Int", "o modificador de atributo é o que muda numa ficha"},
		{"classes", "poderes de classe", "a conta de poderes é derivada, não transcrita"},
		{"classes", "Treinado em", "as perícias saem de class-expertises"},
		{"deuses", "Arma preferida", "o clérigo saca a arma preferida na cena"},
		{"deuses", "Concede", "os poderes concedidos são o que o mestre consulta"},
	}
	for _, tc := range cases {
		body := pedeNoMestre(t, s, eu, "GET", "/mestre/"+tc.aba, "").Body.String()
		if !strings.Contains(body, tc.want) {
			t.Errorf("a aba %q não traz %q — %s", tc.aba, tc.want, tc.reason)
		}
	}
}

// TestAnEmptyLabelDoesNotComeOutAlone: Lena e Marah não têm arma preferida no livro.
//
// Provado VERMELHO antes do conserto: a cena escrevia "Arma preferida:" seguido
// de nada, que parece dado perdido em vez de ausência com significado.
func TestAnEmptyLabelDoesNotComeOutAlone(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/mestre/deuses", "").Body.String()
	if strings.Contains(body, "Arma preferida: </span>") {
		t.Error("um rótulo saiu sem valor — Lena e Marah não têm arma preferida")
	}
	// O CONTROLE: quem TEM arma preferida continua mostrando.
	if !strings.Contains(body, "Arma preferida: Martelo de guerra") {
		t.Error("o rótulo sumiu de quem tem valor — o guarda acima passaria por ausência de tudo")
	}
}
