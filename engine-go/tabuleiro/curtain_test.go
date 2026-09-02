package tabuleiro

import (
	"encoding/json"
	"strings"
	"testing"
)

// A CORTINA esconde a CENA, não a peça (ALE-202). O teste afirma as duas
// direções do risco: o que a mesa NÃO pode receber (peça visível, marcador
// visível, nome do lugar — tudo que denuncia o que está sendo montado) e o que
// ela PRECISA receber (que existe uma cortina, senão o jogador não sabe que vem
// cena).
//
// Redigir peça por peça deixaria passar tudo que não está marcado como
// escondido, que é exatamente a taverna que o mestre monta enquanto a mesa olha
// a cripta — por isso a cortina vem ANTES da redação, e não dentro dela.
func TestTheCurtainHidesTheWholeSceneFromTheTable(t *testing.T) {
	b := openBoard(t)
	id := boardCounter()
	_ = AddToken(b, BoardToken{Label: "Taverneiro", X: 1, Y: 1}, id)
	if err := AddMarker(b, BoardMarker{X: 2, Y: 2, Text: "A", Color: "ouro"}, id); err != nil {
		t.Fatalf("marcar: %v", err)
	}
	b.Curtained = true

	daMesa := BoardForRole("player", b)

	if daMesa == nil {
		t.Fatal("a mesa recebeu `nil`, que significa \"não há tabuleiro\" — outra coisa")
	}
	if len(daMesa.Tokens) != 0 {
		t.Errorf("a peça visível atravessou a cortina: %+v", daMesa.Tokens)
	}
	if len(daMesa.Markers) != 0 {
		t.Errorf("o marcador visível atravessou a cortina: %+v", daMesa.Markers)
	}
	if daMesa.Place != "" {
		t.Errorf("o nome do lugar atravessou a cortina e entrega a cena: %q", daMesa.Place)
	}
	// O jogador precisa saber que vem cena sem ver qual (decisão do dono).
	if !daMesa.Curtained {
		t.Error("a mesa não soube que há uma cortina, e a tela não tem como desenhá-la")
	}
	// O mestre continua vendo o que ele está montando — é o ponto todo.
	if doMestre := BoardForRole("gm", b); len(doMestre.Tokens) != 1 || doMestre.Place == "" {
		t.Errorf("o mestre perdeu a própria cena: %+v", doMestre)
	}
}

// A cortina ABERTA não muda nada: sem esta afirmação, um `BoardForRole` que
// redigisse sempre passaria no teste de cima e esconderia o tabuleiro da mesa
// para sempre. É o controle do outro teste, não um caso a mais.
func TestWithoutTheCurtainTheTableKeepsSeeingTheBoard(t *testing.T) {
	b := openBoard(t)
	_ = AddToken(b, BoardToken{Label: "Taverneiro", X: 1, Y: 1}, boardCounter())

	daMesa := BoardForRole("player", b)

	if daMesa.Curtained {
		t.Error("o tabuleiro nasceu sob cortina sem ninguém fechá-la")
	}
	if len(daMesa.Tokens) != 1 || daMesa.Place == "" {
		t.Errorf("a mesa perdeu a cena sem cortina nenhuma: %+v", daMesa)
	}
}

// A lista de peças vai VAZIA e não nula (ALE-202). Fatia nil vira `null` no
// JSON, e o cabeçalho do cliente indexa `tokens.length`: a cortina derrubaria a
// tela da mesa em vez de escondê-la — um defeito que aparece como página branca
// e não como cena escondida, e ninguém ia ligar uma coisa à outra.
//
// Isto se afirma no FIO e não no struct porque é o fio que o cliente lê; um
// teste sobre o valor Go passaria verde com `nil` e `[]` indistintos.
func TestTheCurtainDoesNotSendANullListOnTheWire(t *testing.T) {
	b := &BoardState{Version: 3, Curtained: true}

	fio, err := json.Marshal(BoardForRole("player", b))
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}

	if strings.Contains(string(fio), `"tokens":null`) {
		t.Errorf("a cortina mandou lista nula e o cliente indexa `tokens.length`: %s", fio)
	}
}
