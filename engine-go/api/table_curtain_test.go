package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestTheGmHasAWayToDrawTheCurtain(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "Fechar a cortina") {
		t.Fatal("o mestre não tem gesto para fechar a cortina — a feature volta a ser invisível")
	}

	base := f.tableUrl() + "/tabuleiro/cortina"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/fechar", ""); rec.Code != http.StatusOK {
		t.Fatalf("fechar deu %d", rec.Code)
	}
	if b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab); !b.Curtained {
		t.Fatal("o gesto não fechou a cortina")
	}

	// Fechada, a tela do mestre oferece o CAMINHO DE VOLTA — e em dois lugares,
	// porque a tira de aviso é onde ele percebe e o cabeçalho é onde ele procura.
	fechada := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if n := strings.Count(fechada, "Abrir a cortina"); n < 2 {
		t.Errorf("com a cortina fechada há %d caminhos de volta, esperado 2 (a tira e o cabeçalho)", n)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/abrir", ""); rec.Code != http.StatusOK {
		t.Fatalf("abrir deu %d", rec.Code)
	}
	if b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab); b.Curtained {
		t.Error("o gesto não abriu a cortina de volta")
	}
}

// TestThePlayerDoesNotDrawTheCurtain — a trava é do servidor.
//
// Ela é o ponto INTEIRO desta feature: a cortina existe para o jogador não ver o
// que o mestre está montando, e um jogador que a abre pela mão vê a emboscada.
func TestThePlayerDoesNotDrawTheCurtain(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	base := f.tableUrl() + "/tabuleiro/cortina"
	if rec := f.pede(t, f.mestre, http.MethodPost, base+"/fechar", ""); rec.Code != http.StatusOK {
		t.Fatalf("fechar deu %d", rec.Code)
	}

	if rec := f.pede(t, f.jogador, http.MethodPost, base+"/abrir", ""); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador abriu a cortina: %d", rec.Code)
	}
	if b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab); !b.Curtained {
		t.Error("a cortina abriu apesar do 403")
	}
	// E o gesto nem aparece para ele: cortesia, não a trava.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(doJogador, "Fechar a cortina") {
		t.Error("o jogador recebeu o botão da cortina")
	}
}

// NÃO HÁ GUARDA AQUI PARA A ECONOMIA DE PUBLICAÇÃO ("sem mudança não se
// transmite"), e a ausência é deliberada — eu tinha escrito um e o cortei.
//
// Ele media a VERSÃO do tabuleiro, que é comportamento do `SetCurtain` e já tem
// dono no `tabuleiro/curtain_test.go`. Sabotei a economia no comando daqui e o
// caso continuou VERDE: ele afirmava uma garantia que não segurava, o que é pior
// que teste nenhum porque parece cobertura.
//
// Medir de verdade custaria observar o FLUXO SSE e contar quadros — caro, e a
// consequência de errar é um remendo a mais numa tela, não um defeito que
// alguém note. Se um dia doer, o lugar é um teste de stream, não este arquivo.
