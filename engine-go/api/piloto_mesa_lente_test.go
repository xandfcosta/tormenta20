package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// Os guardas de VER COMO JOGADOR (ALE-193, superfície 7 da ALE-269).
//
// A REDAÇÃO por papel tem guarda no `tabuleiro` (`BoardForRole`) e no
// `piloto_mesa_tabuleiro_test`. O que se prende aqui é o que só existe desde a
// lente: que ela usa a redação em vez de reescrevê-la, que a CONTAGEM responde a
// pergunta do mestre, que os controles continuam dele, e que ela não sobrevive
// ao fim da cena.

// onLens acende a lente do mestre e devolve a cena que ele passa a ver.
func onLens(t *testing.T, f pilotoFixture) string {
	t.Helper()
	rec := f.pede(t, f.mestre, http.MethodPost, f.tableUrl()+"/tabuleiro/lente", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("acender a lente deu %d", rec.Code)
	}
	return f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
}

// TestTheLensHidesFromTheGmWhatIsHiddenFromTheTable — o caso que ela existe para
// resolver.
//
// Antes dela, conferir a emboscada exigia dois navegadores com dois logins. O
// que se afirma é que a peça escondida SAI da tela do mestre enquanto a lente
// está ligada — e o controle positivo é a mesma tela sem a lente, onde ela está.
func TestTheLensHidesFromTheGmWhatIsHiddenFromTheTable(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "emboscada", Label: "Ogro emboscado", X: 4, Y: 4, Hidden: true}, true); err != nil {
		t.Fatalf("pôr a peça escondida: %v", err)
	}
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "visivel", Label: "Taverneiro", X: 1, Y: 1}, true); err != nil {
		t.Fatalf("pôr a peça visível: %v", err)
	}

	// O CONTROLE: sem a lente, o mestre vê as duas.
	semLente := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(semLente, "Ogro emboscado") {
		t.Fatal("o mestre não via a própria peça escondida — o caso não mede nada")
	}

	comLente := onLens(t, f)
	if strings.Contains(comLente, "Ogro emboscado") {
		t.Error("a peça escondida continuou na tela do mestre com a lente ligada")
	}
	if !strings.Contains(comLente, "Taverneiro") {
		t.Error("a lente escondeu também o que a mesa VÊ")
	}
}

// TestTheLensSaysHowManyVanished.
//
// É a pergunta que trouxe o mestre até aqui — "a emboscada está mesmo
// invisível?" —, e contar o que sobrou na tela não a responde: ele não sabe o
// que não está vendo.
func TestTheLensSaysHowManyVanished(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	for _, id := range []string{"a", "b"} {
		if _, err := f.s.boards.AddToken(context.Background(), f.sessionID, defaultTab,
			tabuleiro.BoardToken{ID: id, Label: "Emboscado " + id, X: 4, Y: 4, Hidden: true}, true); err != nil {
			t.Fatalf("pôr a peça %q: %v", id, err)
		}
	}
	if !strings.Contains(onLens(t, f), "2 peças escondidas não aparecem") {
		t.Error("a tira não disse quantas peças a mesa não vê")
	}
}

// TestTheLensCountComesFromTheDifference, e não de uma varredura por `Hidden`.
//
// A CORTINA esvazia a cena inteira sem marcar peça nenhuma como escondida: uma
// contagem por campo diria "nenhuma peça escondida" sobre um mapa que a mesa
// simplesmente não vê. Comparar os dois retratos cobre tudo o que a redação tira,
// inclusive o que ela vier a tirar depois.
func TestTheLensCountComesFromTheDifference(t *testing.T) {
	doMestre := &tabuleiro.BoardState{
		Curtained: true,
		Tokens: []tabuleiro.BoardToken{
			{ID: "a", Label: "Taverneiro"}, {ID: "b", Label: "Ogro"},
		},
	}
	daMesa, escondidas := seesTableHowScene(doMestre)
	if escondidas != 2 {
		t.Errorf("com a cortina fechada a lente contou %d escondidas, esperado 2", escondidas)
	}
	if daMesa != nil && len(daMesa.Tokens) != 0 {
		t.Errorf("a cortina deixou %d peças na cena da mesa", len(daMesa.Tokens))
	}
}

// TestTheLensDoesNotTakeTheGmControlsAway.
//
// "Ele confere a emboscada sem parar de montá-la": a lente é sobre a CENA e não
// sobre as ferramentas. Se ela trocasse o papel de quem olha em vez de trocar só
// o tabuleiro, o mestre perderia o pincel, o acervo e a própria saída — e ficaria
// preso na vista da mesa.
func TestTheLensDoesNotTakeTheGmControlsAway(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	tela := onLens(t, f)

	for _, controle := range []string{"Borracha", "Encerrar o tabuleiro", "Voltar à vista do mestre"} {
		if !strings.Contains(tela, controle) {
			t.Errorf("a lente tirou %q do mestre", controle)
		}
	}
}

// TestTheLensDiesWithTheScene.
//
// "Você está vendo a cena como a mesa" sobre uma tela sem tabuleiro faz o mestre
// concluir que o mapa sumiu PARA OS JOGADORES — a resposta errada exatamente à
// pergunta que a lente existe para responder.
func TestTheLensDiesWithTheScene(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	if !strings.Contains(onLens(t, f), "Voltar à vista do mestre") {
		t.Fatal("a lente não acendeu — o resto não mede nada")
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	f.seedOpenBoard(t, "taverna")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(tela, "Voltar à vista do mestre") {
		t.Error("a lente sobreviveu ao fim da cena e acendeu sobre a cena seguinte")
	}
}

// TestTheLensBelongsToWhoeverLitIt.
//
// Ela é um modo de conferência de uma pessoa, e não um estado da mesa: acender a
// do mestre não pode mudar nada do que o jogador vê — nem, o que seria pior,
// revelar-lhe que alguém está conferindo.
func TestTheLensBelongsToWhoeverLitIt(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "cripta")
	if _, err := f.s.boards.AddToken(context.Background(), f.sessionID, defaultTab,
		tabuleiro.BoardToken{ID: "visivel", Label: "Taverneiro", X: 1, Y: 1}, true); err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	antes := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	onLens(t, f)
	depois := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(depois, "vendo a cena como a mesa") {
		t.Error("a tira da lente do mestre apareceu na tela do jogador")
	}
	if strings.Contains(antes, "Taverneiro") != strings.Contains(depois, "Taverneiro") {
		t.Error("acender a lente do mestre mudou o que o jogador vê")
	}
}

// TestOnlyTheGmLightsTheLens: a trava é do servidor, e não o botão escondido.
func TestOnlyTheGmLightsTheLens(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	rec := f.pede(t, f.jogador, http.MethodPost, f.tableUrl()+"/tabuleiro/lente", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador acendeu a lente: %d", rec.Code)
	}
}

// TestTheLensSentenceAgreesInNumber.
//
// "1 peças escondidas" é o defeito que passa por todo teste que compara com um
// `fmt.Sprintf` do mesmo jeito — o teste re-derivaria o erro. Os três casos são
// escritos por extenso.
func TestTheLensSentenceAgreesInNumber(t *testing.T) {
	casos := map[int]string{
		0: "Nenhuma peça escondida nesta cena.",
		1: "1 peça escondida não aparece.",
		3: "3 peças escondidas não aparecem.",
	}
	for quantas, esperado := range casos {
		if frase := lensPhrase(quantas); !strings.HasSuffix(frase, esperado) {
			t.Errorf("com %d escondidas a tira disse %q, esperado terminar em %q", quantas, frase, esperado)
		}
	}
}
