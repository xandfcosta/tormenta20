package api

import (
	"html"
	"net/http"
	"strings"
	"testing"
)

// O JOGADOR LÊ A DECISÃO DA MANOBRA, como lê o erro do golpe (decisão do dono). A decisão
// limpava a carta na hora, e quem propôs nunca via se o agarrar venceu. Agora a manobra
// decidida fica na carta com "O mestre disse" e um Fechar — e ela está ENCERRADA: não
// trava o golpe seguinte, e a vez mudando a tira da mesa.
func TestThePlayerReadsTheManeuverRuling(t *testing.T) {
	for _, c := range []struct{ path, said string }{
		{"/ataque/julgar/sucesso", "Sucesso"},
		{"/ataque/julgar/falha", "Falha"},
	} {
		t.Run(c.said, func(t *testing.T) {
			f, goblin := attackOnTurn(t)
			if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/manobra/derrubar", ""); rec.Code != http.StatusOK {
				t.Fatalf("propor a manobra deu %d", rec.Code)
			}
			if refusal := gmRefusal(t, f, c.path); refusal != "" {
				t.Fatalf("a decisão foi recusada: %q", refusal)
			}

			page := html.UnescapeString(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())
			if !strings.Contains(page, "O mestre disse: <b>"+c.said+"</b>") {
				t.Errorf("a carta do jogador não diz a decisão do mestre (%s)", c.said)
			}
			if !strings.Contains(page, ">Fechar<") {
				t.Error("a carta da manobra decidida não tem o Fechar")
			}
			// A CARTA DO MESTRE sai, como no erro: não há mais nada a pedir dele.
			if gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(gm, `id="gm-attack-details"`) {
				t.Error("a carta do mestre continua na mesa depois de ele decidir a manobra")
			}

			// O FECHAR do jogador tira a carta, e não devolve a ação: a manobra aconteceu.
			if refusal := tableRefusal(t, f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "").Body.String()); refusal != "" {
				t.Fatalf("o jogador não conseguiu fechar a manobra decidida: %q", refusal)
			}
			if pendingOf(t, f) != nil {
				t.Error("o Fechar não tirou a manobra decidida da mesa")
			}
			if sceneOf(t, f).StandardLeft {
				t.Error("fechar a leitura devolveu a ação padrão, e a manobra já aconteceu")
			}
		})
	}
}

// A MANOBRA DECIDIDA NÃO TRAVA a mesa: a vez mudando a tira, como tira o erro do golpe.
func TestTheTurnChangeDropsTheRuledManeuver(t *testing.T) {
	f, goblin := attackOnTurn(t)
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/manobra/derrubar", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a manobra deu %d", rec.Code)
	}
	if refusal := gmRefusal(t, f, "/ataque/julgar/sucesso"); refusal != "" {
		t.Fatalf("o Sucesso foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil {
		t.Fatal("a manobra decidida saiu da mesa antes de o jogador a ler")
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("passar a vez deu %d", rec.Code)
	}
	if pa := pendingOf(t, f); pa != nil {
		t.Errorf("a vez mudou e a manobra decidida continuou na mesa: %+v", pa)
	}
}
