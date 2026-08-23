package api

import "testing"

// O REPETIDO NA FILA GANHA NÚMERO (ALE-208).
//
// Quatro ogros davam quatro linhas chamadas "Ogro", e a fila nomeia os botões
// dela pelo rótulo — "Remover Ogro" quatro vezes. Nem o mestre nem um leitor de
// tela separam isso. O defeito apareceu ao dar quantidade ao diálogo do
// bestiário, mas já existia: mandar quatro ogros do Montar encontro fazia igual.
//
// Quem numera é o SERVIDOR, pela mesma razão do tabuleiro (ALE-192): achar o
// próximo livre é decisão sobre o estado, e duas telas adivinhando produziriam
// dois "Ogro 2".

func addNpcs(t *testing.T, labels ...string) *SessionRuntimeState {
	t.Helper()
	st := &SessionRuntimeState{TurnIndex: -1}
	n := 0
	novoID := func() string { n++; return string(rune('a' + n)) }
	for _, label := range labels {
		if err := addEntry(st, InitiativeEntry{Label: label, Type: "npc", Initiative: 10}, novoID); err != nil {
			t.Fatalf("adicionar %q: %v", label, err)
		}
	}
	return st
}

func labelsOf(st *SessionRuntimeState) []string {
	out := make([]string, 0, len(st.Initiative))
	for _, e := range st.Initiative {
		out = append(out, e.Label)
	}
	return out
}

// O PRIMEIRO de cada espécie fica sem número: numerar desde o começo encheria a
// fila de "1" que não distinguem nada quando há um só.
func TestOPrimeiroNaoGanhaNumeroEOsSeguintesSim(t *testing.T) {
	st := addNpcs(t, "Ogro", "Ogro", "Ogro")

	got := labelsOf(st)
	want := []string{"Ogro", "Ogro 2", "Ogro 3"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rótulos = %v, queria %v", got, want)
		}
	}
}

// Espécies diferentes não disputam numeração.
func TestEspeciesDiferentesNaoSeMisturam(t *testing.T) {
	st := addNpcs(t, "Ogro", "Goblin", "Ogro", "Goblin")

	got := labelsOf(st)
	esperado := map[string]bool{"Ogro": true, "Ogro 2": true, "Goblin": true, "Goblin 2": true}
	for _, label := range got {
		if !esperado[label] {
			t.Fatalf("rótulos = %v — %q não é um dos esperados", got, label)
		}
	}
}

// O MENOR número livre, e não "o maior mais um": tirado o Ogro 2, o próximo
// volta a ser o Ogro 2. Uma fila com "Ogro, Ogro 3, Ogro 7" faz o mestre
// procurar os que não existem — a mesma razão escrita no `nextInstanceLabel`.
func TestOBuracoEReaproveitado(t *testing.T) {
	st := addNpcs(t, "Ogro", "Ogro", "Ogro")
	if err := removeEntry(st, st.Initiative[1].ID); err != nil {
		t.Fatalf("remover: %v", err)
	}

	novoID := func() string { return "z" }
	if err := addEntry(st, InitiativeEntry{Label: "Ogro", Type: "npc", Initiative: 10}, novoID); err != nil {
		t.Fatalf("readicionar: %v", err)
	}

	got := labelsOf(st)
	tem2 := false
	for _, l := range got {
		if l == "Ogro 2" {
			tem2 = true
		}
		if l == "Ogro 4" {
			t.Fatalf("rótulos = %v — pulou para o 4 em vez de reaproveitar o buraco", got)
		}
	}
	if !tem2 {
		t.Fatalf("rótulos = %v — o Ogro 2 não voltou", got)
	}
}

// A ficha do jogador não é numerada por este caminho, e não pode ser: dois
// personagens homônimos na mesma mesa são escolha das pessoas, e renomear a
// ficha de alguém por baixo faria a fila e a ficha discordarem sobre quem é.
// O que este teste prende é que o rótulo VAZIO passa reto — quem cobra a
// obrigatoriedade é quem materializa a linha, e inventar um número aqui
// esconderia o erro dele.
func TestRotuloVazioPassaReto(t *testing.T) {
	st := &SessionRuntimeState{TurnIndex: -1}
	if err := addEntry(st, InitiativeEntry{Label: "", Type: "npc"}, func() string { return "a" }); err != nil {
		t.Fatalf("adicionar: %v", err)
	}

	if st.Initiative[0].Label != "" {
		t.Errorf("rótulo = %q, queria vazio", st.Initiative[0].Label)
	}
}
