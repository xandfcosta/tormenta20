package api

import "testing"

// entryIDByLabel devolve o ID que `addEntry` sorteou para uma linha. A escolha
// viaja por ID e nunca por rótulo (ALE-204/192): dois goblins têm o mesmo nome
// até o servidor numerá-los.
func entryIDByLabel(t *testing.T, st *SessionRuntimeState, label string) string {
	t.Helper()
	for _, entry := range st.Initiative {
		if entry.Label == label {
			return entry.ID
		}
	}
	t.Fatalf("não há linha %q na iniciativa", label)
	return ""
}

// A EMBOSCADA (ALE-204).
//
// "Trazer a iniciativa" punha no tabuleiro a fila inteira, e a fila inclui quem
// o mestre montou para aparecer no terceiro turno: num clique a surpresa virava
// peça na tela da mesa, e desfazer era peça por peça. Quem não foi escolhido
// não nasce — nem escondido, porque peça que não existe não vaza por bug de
// redação.
func TestPopulateBringsOnlyTheChosen(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, charEntry("Sílfide", 18, 7), id)
	_ = addEntry(st, charEntry("Paladino", 15, 8), id)
	_ = addEntry(st, npc("Assassino", 20), id)
	b := newBoard("Cripta", "pedra")

	escolhidos := entrySelection{
		entryIDByLabel(t, st, "Sílfide"):  true,
		entryIDByLabel(t, st, "Paladino"): true,
	}
	if placed := populateBoard(b, st, boardCounter(), escolhidos); placed != 2 {
		t.Fatalf("colocou %d peças, esperado 2", placed)
	}
	for _, token := range b.Tokens {
		if token.Label == "Assassino" {
			t.Fatal("a peça que o mestre NÃO escolheu foi para o tabuleiro — a emboscada foi revelada")
		}
	}
}

// "Não escolhi" não é "escolhi ninguém", e a diferença é o que separa uma aba
// aberta antes da ALE-204 (que manda `board-populate` pelado e espera a fila
// inteira) de um mestre que desmarcou todo mundo no diálogo.
func TestChosenEntriesTellsAbsentFromEmpty(t *testing.T) {
	if ausente := chosenEntries(map[string]any{}, "entryIds"); ausente != nil {
		t.Errorf("corpo sem entryIds virou seleção %v — o cliente antigo pararia de trazer alguém", ausente)
	}
	if !chosenEntries(map[string]any{}, "entryIds").wants("e1") {
		t.Error("seleção ausente recusou uma linha — ausente é TODAS")
	}

	vazia := chosenEntries(map[string]any{"entryIds": []any{}}, "entryIds")
	if vazia == nil {
		t.Fatal("lista vazia virou 'todas' — o mestre pediu ninguém e receberia a fila inteira")
	}
	if vazia.wants("e1") {
		t.Error("lista vazia aceitou uma linha")
	}

	uma := chosenEntries(map[string]any{"entryIds": []any{"e2", 7, nil}}, "entryIds")
	if !uma.wants("e2") {
		t.Error("a linha nomeada ficou de fora")
	}
	if uma.wants("e1") {
		t.Error("uma linha que ninguém nomeou entrou na escolha")
	}
}
