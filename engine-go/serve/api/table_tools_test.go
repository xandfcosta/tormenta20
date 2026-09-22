package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/serve/web/table"
	"testing"
)

// A borracha limpa a CASA INTEIRA.
//
// Como MODO que inverte o pincel selecionado ela erra em silêncio: com
// `Cobertura` na mão, clicar num quadrado de `Difícil` manda
// `terreno/cobertura/…?apagar=1`, apaga a cobertura que não estava ali, e a tela
// não diz nada. A rota não tem espécie no caminho justamente por isso.
func TestTheEraserClearsTheWholeSquare(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	square := f.tableUrl() + "/tabuleiro/terreno"

	// Três espécies EMPILHADAS na mesma casa: é o caso que o modo antigo não
	// sabia resolver, porque ele tinha de escolher uma.
	for _, species := range []string{"dificil", "cobertura", "elevado"} {
		if rec := f.pede(t, f.gm, http.MethodPost, square, stroke(species, 4, 4, 4, 4)); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", species, rec.Code)
		}
	}
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if len(b.Difficult) != 1 || len(b.Cover) != 1 || len(b.Elevated) != 1 {
		t.Fatalf("as três não foram pintadas: %d/%d/%d — sem o caso positivo o resto não mede nada",
			len(b.Difficult), len(b.Cover), len(b.Elevated))
	}

	if rec := f.pede(t, f.gm, http.MethodPost, square+"/limpar", stroke("", 4, 4, 4, 4)); rec.Code != http.StatusOK {
		t.Fatalf("limpar deu %d", rec.Code)
	}
	b = boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if len(b.Difficult)+len(b.Cover)+len(b.Concealment)+len(b.Elevated) != 0 {
		t.Errorf("a borracha deixou terreno na casa: %d difícil, %d cobertura, %d camuflagem, %d elevado",
			len(b.Difficult), len(b.Cover), len(b.Concealment), len(b.Elevated))
	}
}

// A metade do defeito que um teste de "limpa a casa" sozinho não pega: com o
// pincel CERTO na mão a rota antiga funcionava perfeitamente, e o outro caso
// passava despercebido porque o servidor respondia 200.
//
// Aqui isso vira uma afirmação sobre a FORMA da rota: se a espécie voltar para o
// caminho, este teste cai.
func TestTheEraserDoesNotDependOnTheSelectedBrush(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(screen, "tabuleiro/terreno/limpar/") {
		t.Error("a borracha não usa a rota sem espécie")
	}
	if strings.Contains(screen, "apagar=1") {
		t.Error("a borracha voltou a ser um modo do pincel (`?apagar=1`)")
	}
	// E ela é FERRAMENTA: tem lugar no trilho, com tecla.
	if !strings.Contains(screen, "Borracha (tecla ") {
		t.Error("a borracha não é uma ferramenta do trilho")
	}
}

// A trava de verdade é do servidor (`gmBoardCommand`); isto é a
// cortesia de não oferecer o que seria recusado. Mas ela também é o que impede um
// gesto MUDO: a camada de pintura não existe na cena do jogador, então uma
// ferramenta oferecida a ele seria um modo que liga e não faz nada.
func TestThePlayerRailLacksWhatThePlayerCannotDo(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(screen, "Régua (tecla ") {
		t.Fatal("o jogador não recebeu o trilho — a página não é o que este teste pensa que é")
	}
	for _, f := range table.MapTools() {
		if f.GMOnly && strings.Contains(screen, f.Label+" (tecla ") {
			t.Errorf("o jogador recebeu %q, que é do mestre", f.Label)
		}
	}
}
