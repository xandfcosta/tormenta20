package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"testing"
)

func onLens(t *testing.T, f sceneFixture) string {
	t.Helper()
	rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/lente", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("acender a lente deu %d", rec.Code)
	}
	return f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
}

// O caso que a lente existe para resolver.
//
// Sem ela, conferir a emboscada exigia dois navegadores com dois logins. O
// que se afirma é que a peça escondida SAI da tela do mestre enquanto a lente
// está ligada — e o controle positivo é a mesma tela sem a lente, onde ela está.
func TestTheLensHidesFromTheGmWhatIsHiddenFromTheTable(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "emboscada", Label: "Ogro emboscado", X: 4, Y: 4, Hidden: true}); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "visivel", Label: "Taverneiro", X: 1, Y: 1}); err != nil {
		t.Fatalf("pôr a peça visível: %v", err)
	}

	// O CONTROLE: sem a lente, o mestre vê as duas.
	noLens := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(noLens, "Ogro emboscado") {
		t.Fatal("o mestre não via a própria peça escondida — o caso não mede nada")
	}

	withLens := onLens(t, f)
	if strings.Contains(withLens, "Ogro emboscado") {
		t.Error("a peça escondida continuou na tela do mestre com a lente ligada")
	}
	if !strings.Contains(withLens, "Taverneiro") {
		t.Error("a lente escondeu também o que a mesa VÊ")
	}
}

// É a pergunta que trouxe o mestre até aqui — "a emboscada está mesmo
// invisível?" —, e contar o que sobrou na tela não a responde: ele não sabe o
// que não está vendo.
func TestTheLensSaysHowManyVanished(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	for _, id := range []string{"a", "b"} {
		if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
			board.BoardToken{ID: id, Label: "Emboscado " + id, X: 4, Y: 4, Hidden: true}); err != nil {
			t.Fatalf("pôr a peça %q: %v", id, err)
		}
	}
	if !strings.Contains(onLens(t, f), "2 peças escondidas não aparecem") {
		t.Error("a tira não disse quantas peças a mesa não vê")
	}
}

// "Ele confere a emboscada sem parar de montá-la": a lente é sobre a CENA e não
// sobre as ferramentas. Se ela trocasse o papel de quem olha em vez de trocar só
// o tabuleiro, o mestre perderia o pincel, o acervo e a própria saída — e ficaria
// preso na vista da mesa.
func TestTheLensDoesNotTakeTheGmControlsAway(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := onLens(t, f)

	for _, control := range []string{"Borracha", "Encerrar o tabuleiro", "Voltar à vista do mestre"} {
		if !strings.Contains(screen, control) {
			t.Errorf("a lente tirou %q do mestre", control)
		}
	}
}

// "Você está vendo a cena como a mesa" sobre uma tela sem tabuleiro faz o mestre
// concluir que o mapa sumiu PARA OS JOGADORES — a resposta errada exatamente à
// pergunta que a lente existe para responder.
func TestTheLensDiesWithTheScene(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	if !strings.Contains(onLens(t, f), "Voltar à vista do mestre") {
		t.Fatal("a lente não acendeu — o resto não mede nada")
	}

	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	f.seedOpenBoard(t, "tavern")
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(screen, "Voltar à vista do mestre") {
		t.Error("a lente sobreviveu ao fim da cena e acendeu sobre a cena seguinte")
	}
}

// Ela é um modo de conferência de uma pessoa, e não um estado da mesa: acender a
// do mestre não pode mudar nada do que o jogador vê — nem, o que seria pior,
// revelar-lhe que alguém está conferindo.
func TestTheLensBelongsToWhoeverLitIt(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{ID: "visivel", Label: "Taverneiro", X: 1, Y: 1}); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	before := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	onLens(t, f)
	after := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(after, "vendo a cena como a mesa") {
		t.Error("a tira da lente do mestre apareceu na tela do jogador")
	}
	if strings.Contains(before, "Taverneiro") != strings.Contains(after, "Taverneiro") {
		t.Error("acender a lente do mestre mudou o que o jogador vê")
	}
}

// A trava é do servidor, e não o botão escondido.
func TestOnlyTheGmLightsTheLens(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	rec := f.pede(t, f.player, http.MethodPost, f.tableUrl()+"/tabuleiro/lente", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador acendeu a lente: %d", rec.Code)
	}
}
