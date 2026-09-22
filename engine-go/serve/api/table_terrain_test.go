package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"testing"
)

func TestTheBrushPaintsTheKindItAskedFor(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	for i, brush := range board.TerrainKinds {
		// O caminho é um TRAÇO, e um clique parado é um traço de uma casa: a mesma
		// casa nas duas pontas.
		rec := f.pede(t, f.gm, "POST",
			f.tableUrl()+"/tabuleiro/terreno", stroke(string(brush.ID), i, 0, i, 0))
		if rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", brush.ID, rec.Code)
		}
	}

	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	for i, brush := range board.TerrainKinds {
		squares := board.SquaresOf(b, brush.ID)
		if len(squares) != 1 || squares[0].X != i {
			t.Errorf("%s ficou com %v, esperado só a casa %d", brush.ID, squares, i)
		}
	}
}

// A borracha limpa SÓ a espécie escolhida.
//
// É por isso que ela é um MODO e não uma espécie: numa casa com duas, uma
// "espécie borracha" teria de decidir qual apagar, e a resposta certa — a que
// está selecionada — já é o que o modo faz. Folhagens são difícil E camuflagem
// (p267), então a casa com duas não é hipótese.
func TestTheEraserClearsOnlyTheChosenKind(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	base := f.tableUrl() + "/tabuleiro/terreno"

	for _, species := range []string{"dificil", "camuflagem"} {
		if rec := f.pede(t, f.gm, "POST", base, stroke(species, 3, 3, 3, 3)); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", species, rec.Code)
		}
	}
	if rec := f.pede(t, f.gm, "POST", base, strokeErasing("camuflagem", 3, 3, 3, 3)); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}

	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if n := len(board.SquaresOf(b, board.TerrenoCamuflagem)); n != 0 {
		t.Errorf("a camuflagem não foi apagada (%d casas)", n)
	}
	if n := len(board.SquaresOf(b, board.TerrenoDificil)); n != 1 {
		t.Errorf("a borracha levou o difícil junto (%d casas) — a casa tinha as duas", n)
	}
}

// O guarda de leiaute que esta casa cobra: um traço pintado que não vira classe
// própria some no desenho das outras, e o mestre lê a cena errada sem nada
// estourar. Amostragem sobre a lista.
func TestTheFourKindsAreDrawnDistinctly(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	for i, brush := range board.TerrainKinds {
		if rec := f.pede(t, f.gm, "POST",
			f.tableUrl()+"/tabuleiro/terreno", stroke(string(brush.ID), i, 0, i, 0)); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", brush.ID, rec.Code)
		}
	}

	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: o tabuleiro desenhou. Sem ele, não achar as classes seria
	// verdade também sobre uma cena que não abriu.
	if !strings.Contains(screen, "board-plane") {
		t.Fatal("o tabuleiro não desenhou — o guarda mediria a tela errada")
	}
	for _, brush := range board.TerrainKinds {
		if !strings.Contains(screen, "board-"+board.ClassOf(brush.ID)) {
			t.Errorf("a espécie %s foi pintada e não tem desenho próprio na cena", brush.ID)
		}
	}
}

// "Cobertura" sozinho não lembra ninguém de que são +5 na Defesa, e o mestre que
// precisa da regra sai da mesa para procurá-la no livro. É a mesma razão de o
// diálogo de abrir dizer que um quadrado são 1,5m.
func TestTheRailSaysTheEffectOfEachKind(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// "Ferramentas do mapa" e não "Pincel de terreno": o trilho carrega mais que
	// o pincel, e um grupo que se anuncia como pincel mente para quem navega por
	// leitor de tela.
	if !strings.Contains(screen, "Ferramentas do mapa") {
		t.Fatal("o mestre não tem trilho de ferramentas na cena aberta")
	}
	// O nome acessível da camada NÃO cita a espécie: ela sairia do sinal, que
	// guarda o ID, e o leitor de tela anunciaria "Pintar dificil" sem acento.
	// Quem diz qual é a espécie é o botão `aria-pressed` do trilho.
	//
	// O sinal chama-se `$tool`, e o nome importa aqui: asserção de AUSÊNCIA sobre
	// um nome de sinal MORTO passa verde sobre nada, e envelhece em silêncio.
	if strings.Contains(screen, "'Pintar ' + $tool") {
		t.Error("o nome acessível da camada monta o rótulo com o id da ferramenta")
	}
	for _, brush := range board.TerrainKinds {
		if !strings.Contains(screen, brush.Effect) {
			t.Errorf("o trilho não diz o que %s faz (%q)", brush.ID, brush.Effect)
		}
	}
	// A PÁGINA, e não a regra: a citação vai junto para conferir sem reabrir o
	// livro. O número é da Tabela 5-3.
	if !strings.Contains(screen, "p238") {
		t.Error("o trilho não cita a página da regra")
	}

	// E o pincel é do MESTRE: o jogador não pinta chão. A asserção é sobre os
	// PINCÉIS e não sobre o trilho, que existe para os dois papéis porque a régua
	// é de quem ataca.
	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	for _, brush := range board.TerrainKinds {
		if strings.Contains(forPlayer, brush.Effect) {
			t.Errorf("o pincel %q apareceu na cena do jogador", brush.ID)
		}
	}
	if strings.Contains(forPlayer, "Borracha") {
		t.Error("a borracha do mestre apareceu na cena do jogador")
	}
}

// A trava do pincel é do SERVIDOR, e não o botão escondido.
func TestOnlyTheGmPaints(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	rec := f.pede(t, f.player, "POST", f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 1, 1))
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pintou o chão: %d", rec.Code)
	}
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if n := len(board.SquaresOf(b, board.TerrenoDificil)); n != 0 {
		t.Errorf("a pintura do jogador entrou mesmo assim (%d casas)", n)
	}
}

// Não é 500 nem silêncio: pintar chão de uma cena que não está na mesa não tem
// onde acontecer, e a recusa fala no `command_error` do rodapé do mestre.
//
// A FRASE é a do `errNoBoard`, e ela passou a ser a única com a ALE-375: a cena
// tinha uma pré-conferência própria ("não há tabuleiro aberto para pintar") que
// repetia, com outras palavras, a recusa que a mutação já dava — e custava uma
// leitura a mais para dizer a mesma coisa. Escrita à mão aqui de propósito:
// importar a constante faria o guarda andar junto com o defeito.
func TestPaintingWithoutABoardRefusesWithASentence(t *testing.T) {
	f := newSceneFixture(t)
	body := f.pede(t, f.gm, "POST", f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 1, 1)).Body.String()
	if !strings.Contains(body, "esta sessão não tem tabuleiro aberto") {
		t.Errorf("a recusa não explica o que faltou; sinais = %s", trechoDeSinais(body))
	}
}
