package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/app"
)

func TestTheScreenOffersTheVerbForTheState(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()

	// PLANEJADA: iniciar sim, encerrar não.
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, "Iniciar sessão") {
		t.Error("a sessão planejada não oferece iniciar")
	}
	if strings.Contains(screen, "Encerrar sessão") {
		t.Error("a sessão planejada oferece encerrar — o servidor recusaria")
	}

	// ATIVA: o contrário. O pedido é um REMENDO no recurso — o status que se
	// quer —, e não uma rota com o verbo no caminho.
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar deu %d", rec.Code)
	}
	active := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(active, "Encerrar sessão") {
		t.Error("a sessão ao vivo não oferece encerrar")
	}
	if strings.Contains(active, "Iniciar sessão") {
		t.Error("a sessão ao vivo ainda oferece iniciar")
	}

	// ENCERRADA: o verbo muda de PALAVRA, porque o gesto mudou de sentido —
	// "Reabrir" e não "Iniciar", que é o que o servidor faz de verdade.
	//
	// Encerrada pelo CASO DE USO e não pela rota: o que este teste mede é o
	// DESENHO em cada status, e chegar ao status pela tela faria a montagem do
	// caso depender do gesto que vem logo abaixo.
	if _, err := f.s.sessionLifecycle().SetStatus(
		ctx, app.Caller{ID: f.gm}, f.campaignID, f.sessionID, "ended",
	); err != nil {
		t.Fatalf("encerrar: %v", err)
	}
	ended := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(ended, "Reabrir") {
		t.Error("a sessão encerrada não oferece reabrir")
	}
}

// Sair não é do mestre: quem entrou numa mesa precisa poder sair dela.
func TestThePlayerHasNoLifecycleButHasTheWayOut(t *testing.T) {
	f := newSceneFixture(t)
	screen := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(screen, "Configurações da sessão") {
		t.Error("o jogador recebeu as configurações da sessão")
	}
	if strings.Contains(screen, "Excluir sessão") {
		t.Error("o jogador recebeu o excluir")
	}
	if !strings.Contains(screen, "Sair da sessão") {
		t.Error("o jogador não tem como sair da mesa")
	}

	// E a trava é do SERVIDOR, não do desenho — agora ela mora no CASO DE USO, e
	// os quatro gestos passam pela mesma linha dele. São os quatro, e não uma
	// amostra: a trava é por gesto, e um gesto novo que esquecesse a chamada
	// passaria despercebido se este laço fosse menor que a família.
	for _, gesture := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"iniciar", http.MethodPatch, f.tableUrl(), `{"status":"active"}`},
		{"encerrar", http.MethodPatch, f.tableUrl(), `{"status":"ended"}`},
		{"renomear", http.MethodPatch, f.tableUrl(), `{"session_title":"x"}`},
		{"reiniciar o combate", http.MethodPost, f.tableUrl() + "/combate/reiniciar", ""},
		{"excluir", http.MethodDelete, f.tableUrl(), ""},
	} {
		rec := f.requests(t, f.player, gesture.method, gesture.path, gesture.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador passou em %q: %d", gesture.name, rec.Code)
		}
	}
}

// Vazio é legítimo: a identidade da sessão é o NÚMERO, e o título é o apelido da
// noite. Obrigar a um faria o mestre inventar texto para poder salvar.
func TestTheTitleSavesAndMayStayBlank(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()

	f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"session_title":"A cripta do rio"}`)
	sess, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if !sess.Title.Valid || sess.Title.String != "A cripta do rio" {
		t.Fatalf("o título não foi salvo: %+v", sess.Title)
	}

	f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"session_title":"   "}`)
	sess, _ = f.s.queries.GetSession(ctx, f.sessionID)
	if sess.Title.Valid && strings.TrimSpace(sess.Title.String) != "" {
		t.Errorf("o título em branco não virou nulo: %+v", sess.Title)
	}
}

// Um remendo que não pede NADA é recusado, e não aceito em silêncio.
//
// Sem esta linha, um botão que deixasse de mandar o campo responderia 200 e não
// faria nada — o sintoma seria "cliquei e a tela não mudou", que é o mais caro
// de investigar porque não deixa erro em lugar nenhum.
func TestAPatchThatAsksForNothingIsRefused(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"outra_coisa":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("o remendo vazio deu %d, e ele tem de ser recusado", rec.Code)
	}
}

// O caminho: o comando recarrega a fila depois de limpá-la, senão o `State`
// devolve a cópia em memória sem passar pelo banco e a próxima carga fria
// discordaria desta.
func TestRestartingFromTheScreenEmptiesTheLiveTracker(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n < 2 {
		t.Fatalf("a cena montou %d combatentes — não há o que reiniciar", n)
	}

	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/combate/reiniciar", ""); rec.Code != http.StatusOK {
		t.Fatalf("reiniciar deu %d", rec.Code)
	}

	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n != 0 {
		t.Errorf("a fila ao vivo continua com %d combatentes", n)
	}
}

// O destino importa: voltar para a mesa apagada seria mandar o mestre para uma
// porta que não existe mais.
//
// A navegação vem pelo FIO e não num cabeçalho `Location`: com `DELETE` não há
// formulário para navegar sozinho, e quem leva o mestre é o `sse.Redirect`.
func TestDeletingErasesAndSendsTheGmToTheCampaign(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()

	rec := f.requests(t, f.gm, http.MethodDelete, f.tableUrl(), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("excluir deu %d", rec.Code)
	}
	destination := "/campanhas/" + strconv.FormatInt(f.campaignID, 10)
	if !strings.Contains(rec.Body.String(), destination) {
		t.Errorf("a resposta não manda o mestre para %q:\n%s", destination, rec.Body.String())
	}
	if _, err := f.s.queries.GetSession(ctx, f.sessionID); err == nil {
		t.Error("a sessão continua no banco depois de excluída")
	}
}
