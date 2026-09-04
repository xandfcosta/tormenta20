package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/web/table"
	"testing"
)

// TestTheEraserClearsTheWholeSquare — o conserto do defeito que o dono achou.
//
// Ela era um MODO que invertia o pincel selecionado: com `Cobertura` na mão,
// clicar num quadrado de `Difícil` mandava `terreno/cobertura/…?apagar=1`,
// apagava a cobertura que não estava ali, e a tela não dizia nada. Medido na
// bancada, clique a clique, antes de virar este teste.
//
// Agora a rota não tem espécie no caminho — não há como errar qual.
func TestTheEraserClearsTheWholeSquare(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	casa := f.tableUrl() + "/tabuleiro/terreno"

	// Três espécies EMPILHADAS na mesma casa: é o caso que o modo antigo não
	// sabia resolver, porque ele tinha de escolher uma.
	for _, especie := range []string{"dificil", "cobertura", "elevado"} {
		if rec := f.pede(t, f.mestre, http.MethodPost, casa+"/"+especie+"/4/4/ate/4/4", ""); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", especie, rec.Code)
		}
	}
	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(b.Difficult) != 1 || len(b.Cover) != 1 || len(b.Elevated) != 1 {
		t.Fatalf("as três não foram pintadas: %d/%d/%d — sem o caso positivo o resto não mede nada",
			len(b.Difficult), len(b.Cover), len(b.Elevated))
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, casa+"/limpar/4/4/ate/4/4", ""); rec.Code != http.StatusOK {
		t.Fatalf("limpar deu %d", rec.Code)
	}
	b = f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(b.Difficult)+len(b.Cover)+len(b.Concealment)+len(b.Elevated) != 0 {
		t.Errorf("a borracha deixou terreno na casa: %d difícil, %d cobertura, %d camuflagem, %d elevado",
			len(b.Difficult), len(b.Cover), len(b.Concealment), len(b.Elevated))
	}
}

// TestTheEraserDoesNotDependOnTheSelectedBrush.
//
// É a metade do defeito que um teste de "limpa a casa" sozinho não pegaria: a
// rota antiga funcionava perfeitamente quando o pincel na mão era o certo. O que
// quebrava era o outro caso, e ele passava despercebido porque o servidor
// respondia 200.
//
// Aqui isso vira uma afirmação sobre a FORMA da rota: se a espécie voltar para o
// caminho, este teste cai.
func TestTheEraserDoesNotDependOnTheSelectedBrush(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(tela, "tabuleiro/terreno/limpar/") {
		t.Error("a borracha não usa a rota sem espécie")
	}
	if strings.Contains(tela, "apagar=1") {
		t.Error("a borracha voltou a ser um modo do pincel (`?apagar=1`)")
	}
	// E ela é FERRAMENTA: tem lugar no trilho, com tecla.
	if !strings.Contains(tela, "Borracha (tecla ") {
		t.Error("a borracha não é uma ferramenta do trilho")
	}
}

// TestThePlayerRailLacksWhatThePlayerCannotDo.
//
// A trava de verdade é do servidor (`gmBoardCommand`); isto é a
// cortesia de não oferecer o que seria recusado. Mas ela também é o que impede um
// gesto MUDO: a camada de pintura não existe na cena do jogador, então uma
// ferramenta oferecida a ele seria um modo que liga e não faz nada.
func TestThePlayerRailLacksWhatThePlayerCannotDo(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	tela := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(tela, "Régua (tecla ") {
		t.Fatal("o jogador não recebeu o trilho — a página não é o que este teste pensa que é")
	}
	for _, f := range table.MapTools() {
		if f.SoMestre && strings.Contains(tela, f.Rotulo+" (tecla ") {
			t.Errorf("o jogador recebeu %q, que é do mestre", f.Rotulo)
		}
	}
}
