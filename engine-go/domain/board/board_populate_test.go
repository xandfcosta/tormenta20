package board

import "t20engine/domain/live"

import "testing"

// entryIDByLabel devolve o ID que `live.AddEntry` sorteou para uma linha. A
// escolha viaja por ID e nunca por rótulo: dois goblins têm o mesmo nome até o
// servidor numerá-los.
func entryIDByLabel(t *testing.T, st *live.SessionRuntimeState, label string) string {
	t.Helper()
	for _, entry := range st.Initiative {
		if entry.Label == label {
			return entry.ID
		}
	}
	t.Fatalf("não há linha %q na iniciativa", label)
	return ""
}

// A EMBOSCADA.
//
// A fila inclui quem o mestre montou para aparecer no terceiro turno: trazê-la
// inteira num clique põe a surpresa na tela da mesa, e desfazer é peça por
// peça. Quem não foi escolhido não nasce — nem escondido, porque peça que não
// existe não vaza por bug de redação.
func TestPopulateBringsOnlyTheChosen(t *testing.T) {
	st := live.EmptyRuntimeState()
	id := ContadorDeIds()
	_ = live.AddEntry(st, combatenteDeFicha("Sílfide", 18, 7), id)
	_ = live.AddEntry(st, combatenteDeFicha("Paladino", 15, 8), id)
	_ = live.AddEntry(st, npc("Assassino", 20), id)
	b := NewBoard("t1", "Cripta", "stone")

	escolhidos := EntrySelection{
		entryIDByLabel(t, st, "Sílfide"):  true,
		entryIDByLabel(t, st, "Paladino"): true,
	}
	if placed := PopulateBoard(b, st, boardCounter(), escolhidos); placed != 2 {
		t.Fatalf("colocou %d peças, esperado 2", placed)
	}
	for _, token := range b.Tokens {
		if token.Label == "Assassino" {
			t.Fatal("a peça que o mestre NÃO escolheu foi para o tabuleiro — a emboscada foi revelada")
		}
	}
}

// "Não escolhi" não é "escolhi ninguém", e a diferença separa um cliente antigo
// (que manda `board-Populate` pelado e espera a fila inteira) de um mestre que
// desmarcou todo mundo no diálogo.
func TestChosenEntriesTellsAbsentFromEmpty(t *testing.T) {
	if ausente := ChosenEntries(map[string]any{}, "entryIds"); ausente != nil {
		t.Errorf("corpo sem entryIds virou seleção %v — o cliente antigo pararia de trazer alguém", ausente)
	}
	if !ChosenEntries(map[string]any{}, "entryIds").wants("e1") {
		t.Error("seleção ausente recusou uma linha — ausente é TODAS")
	}

	vazia := ChosenEntries(map[string]any{"entryIds": []any{}}, "entryIds")
	if vazia == nil {
		t.Fatal("lista vazia virou 'todas' — o mestre pediu ninguém e receberia a fila inteira")
	}
	if vazia.wants("e1") {
		t.Error("lista vazia aceitou uma linha")
	}

	uma := ChosenEntries(map[string]any{"entryIds": []any{"e2", 7, nil}}, "entryIds")
	if !uma.wants("e2") {
		t.Error("a linha nomeada ficou de fora")
	}
	if uma.wants("e1") {
		t.Error("uma linha que ninguém nomeou entrou na escolha")
	}
}

// A PEÇA NÃO NASCE DEBAIXO DO CROMO.
//
// A janela do mapa nasce com o quadrado (0,0) na quina de cima da tela, e o
// painel de verbos da cena flutua ali. Medido a 390×844: uma peça em (3,0) fica
// com 82% da área sob o painel, e o clique direito nela vai para o botão
// "Afastar o mapa" em vez de abrir o menu dela. Qualquer gesto naquela faixa é
// do painel — pintar terreno, largar marcador, pegar a peça.
//
// A regra prende as DUAS pontas: nenhuma peça na faixa do cromo, e nenhuma
// longe demais para a janela mostrar. Nascer na fileira 40 também resolveria a
// primeira e deixaria o mestre procurando o próprio grupo.
func TestPopulateIsBornBelowTheTopChrome(t *testing.T) {
	st := live.EmptyRuntimeState()
	id := ContadorDeIds()
	_ = live.AddEntry(st, combatenteDeFicha("Sílfide", 18, 7), id)
	_ = live.AddEntry(st, npc("Ogro", 12), id)
	b := NewBoard("t1", "Cripta", "stone")

	if placed := PopulateBoard(b, st, boardCounter(), nil); placed != 2 {
		t.Fatalf("colocou %d peças, esperado 2: o que vem abaixo não mediria nada", placed)
	}
	for _, token := range b.Tokens {
		if token.Y < TopChromeRows {
			t.Errorf("%s nasceu na fileira %d, debaixo do painel de verbos — o clique nela vai para o botão de afastar",
				token.Label, token.Y)
		}
		if token.Y > TopChromeRows+2 {
			t.Errorf("%s nasceu na fileira %d, longe demais da faixa que a janela mostra ao abrir", token.Label, token.Y)
		}
	}
}
