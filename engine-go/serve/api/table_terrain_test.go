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

	for i, pincel := range board.TerrainKinds {
		// O caminho é um TRAÇO, e um clique parado é um traço de uma casa: a mesma
		// casa nas duas pontas.
		rec := f.pede(t, f.mestre, "POST",
			f.tableUrl()+"/tabuleiro/terreno", stroke(string(pincel.ID), i, 0, i, 0))
		if rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", pincel.ID, rec.Code)
		}
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	for i, pincel := range board.TerrainKinds {
		casas := board.SquaresOf(b, pincel.ID)
		if len(casas) != 1 || casas[0].X != i {
			t.Errorf("%s ficou com %v, esperado só a casa %d", pincel.ID, casas, i)
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

	for _, especie := range []string{"dificil", "camuflagem"} {
		if rec := f.pede(t, f.mestre, "POST", base, stroke(especie, 3, 3, 3, 3)); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", especie, rec.Code)
		}
	}
	if rec := f.pede(t, f.mestre, "POST", base, strokeErasing("camuflagem", 3, 3, 3, 3)); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
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
	for i, pincel := range board.TerrainKinds {
		if rec := f.pede(t, f.mestre, "POST",
			f.tableUrl()+"/tabuleiro/terreno", stroke(string(pincel.ID), i, 0, i, 0)); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", pincel.ID, rec.Code)
		}
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: o tabuleiro desenhou. Sem ele, não achar as classes seria
	// verdade também sobre uma cena que não abriu.
	if !strings.Contains(tela, "board-plane") {
		t.Fatal("o tabuleiro não desenhou — o guarda mediria a tela errada")
	}
	for _, pincel := range board.TerrainKinds {
		if !strings.Contains(tela, "board-"+board.ClassOf(pincel.ID)) {
			t.Errorf("a espécie %s foi pintada e não tem desenho próprio na cena", pincel.ID)
		}
	}
}

// "Cobertura" sozinho não lembra ninguém de que são +5 na Defesa, e o mestre que
// precisa da regra sai da mesa para procurá-la no livro. É a mesma razão de o
// diálogo de abrir dizer que um quadrado são 1,5m.
func TestTheRailSaysTheEffectOfEachKind(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// "Ferramentas do mapa" e não "Pincel de terreno": o trilho carrega mais que
	// o pincel, e um grupo que se anuncia como pincel mente para quem navega por
	// leitor de tela.
	if !strings.Contains(tela, "Ferramentas do mapa") {
		t.Fatal("o mestre não tem trilho de ferramentas na cena aberta")
	}
	// O nome acessível da camada NÃO cita a espécie: ela sairia do sinal, que
	// guarda o ID, e o leitor de tela anunciaria "Pintar dificil" sem acento.
	// Quem diz qual é a espécie é o botão `aria-pressed` do trilho.
	//
	// O sinal chama-se `$tool`, e o nome importa aqui: asserção de AUSÊNCIA sobre
	// um nome de sinal MORTO passa verde sobre nada, e envelhece em silêncio.
	if strings.Contains(tela, "'Pintar ' + $tool") {
		t.Error("o nome acessível da camada monta o rótulo com o id da ferramenta")
	}
	for _, pincel := range board.TerrainKinds {
		if !strings.Contains(tela, pincel.Effect) {
			t.Errorf("o trilho não diz o que %s faz (%q)", pincel.ID, pincel.Effect)
		}
	}
	// A PÁGINA, e não a regra: a citação vai junto para conferir sem reabrir o
	// livro. O número é da Tabela 5-3.
	if !strings.Contains(tela, "p238") {
		t.Error("o trilho não cita a página da regra")
	}

	// E o pincel é do MESTRE: o jogador não pinta chão. A asserção é sobre os
	// PINCÉIS e não sobre o trilho, que existe para os dois papéis porque a régua
	// é de quem ataca.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	for _, pincel := range board.TerrainKinds {
		if strings.Contains(doJogador, pincel.Effect) {
			t.Errorf("o pincel %q apareceu na cena do jogador", pincel.ID)
		}
	}
	if strings.Contains(doJogador, "Borracha") {
		t.Error("a borracha do mestre apareceu na cena do jogador")
	}
}

// A trava do pincel é do SERVIDOR, e não o botão escondido.
func TestOnlyTheGmPaints(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 1, 1))
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pintou o chão: %d", rec.Code)
	}
	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if n := len(board.SquaresOf(b, board.TerrenoDificil)); n != 0 {
		t.Errorf("a pintura do jogador entrou mesmo assim (%d casas)", n)
	}
}

// Não é 500 nem silêncio: pintar chão de uma cena que não está na mesa não tem
// onde acontecer, e a recusa fala no `command_error` do rodapé do mestre.
func TestPaintingWithoutABoardRefusesWithASentence(t *testing.T) {
	f := newSceneFixture(t)
	corpo := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 1, 1, 1, 1)).Body.String()
	if !strings.Contains(corpo, "não há tabuleiro aberto") {
		t.Errorf("a recusa não explica o que faltou; sinais = %s", trechoDeSinais(corpo))
	}
}
