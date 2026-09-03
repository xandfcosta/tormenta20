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

	casos := []struct{ aba, esperado, porque string }{
		{"racas", "Graça de Glórienn", "a habilidade de raça vem do catálogo"},
		{"racas", "+2 Int", "o modificador de atributo é o que muda numa ficha"},
		{"classes", "poderes de classe", "a conta de poderes é derivada, não transcrita"},
		{"classes", "Treinado em", "as perícias saem de class-expertises"},
		{"deuses", "Arma preferida", "o clérigo saca a arma preferida na cena"},
		{"deuses", "Concede", "os poderes concedidos são o que o mestre consulta"},
	}
	for _, caso := range casos {
		corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/"+caso.aba, "").Body.String()
		if !strings.Contains(corpo, caso.esperado) {
			t.Errorf("a aba %q não traz %q — %s", caso.aba, caso.esperado, caso.porque)
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

	corpo := pedeNoMestre(t, s, eu, "GET", "/mestre/deuses", "").Body.String()
	if strings.Contains(corpo, "Arma preferida: </span>") {
		t.Error("um rótulo saiu sem valor — Lena e Marah não têm arma preferida")
	}
	// O CONTROLE: quem TEM arma preferida continua mostrando.
	if !strings.Contains(corpo, "Arma preferida: Martelo de guerra") {
		t.Error("o rótulo sumiu de quem tem valor — o guarda acima passaria por ausência de tudo")
	}
}
