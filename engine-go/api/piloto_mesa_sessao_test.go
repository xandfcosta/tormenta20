package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// O CICLO DA SESSÃO na Mesa (ALE-269, superfícies 3, 4 e 11).
//
// A regra tem guarda no `session_lifecycle_test.go`. O que estes casos protegem é a
// AFORDÂNCIA: que a tela ofereça o verbo certo para cada estado, que o jogador
// não receba nenhum, e que excluir leve o mestre para algum lugar.

// TestTheScreenOffersTheVerbForTheState, e só ele.
//
// O servidor recusa encerrar o que nunca começou, e um botão que existe para
// levar recusa é um erro desenhado. Por isso a tela é travada pelo estado — e
// não é redundância com o guarda do servidor: são camadas diferentes da mesma
// decisão, e esta é a que o mestre vê.
func TestTheScreenOffersTheVerbForTheState(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()

	// PLANEJADA: iniciar sim, encerrar não.
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(tela, "Iniciar sessão") {
		t.Error("a sessão planejada não oferece iniciar")
	}
	if strings.Contains(tela, "Encerrar sessão") {
		t.Error("a sessão planejada oferece encerrar — o servidor recusaria")
	}

	// ATIVA: o contrário.
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/sessao/iniciar", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar deu %d", rec.Code)
	}
	ativa := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(ativa, "Encerrar sessão") {
		t.Error("a sessão ao vivo não oferece encerrar")
	}
	if strings.Contains(ativa, "Iniciar sessão") {
		t.Error("a sessão ao vivo ainda oferece iniciar")
	}

	// ENCERRADA: o verbo muda de PALAVRA, porque o gesto mudou de sentido —
	// "Reabrir" e não "Iniciar", que é o que o servidor faz de verdade.
	sess, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if _, err := f.s.EndSession(ctx, sess); err != nil {
		t.Fatalf("encerrar: %v", err)
	}
	encerrada := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(encerrada, "Reabrir") {
		t.Error("a sessão encerrada não oferece reabrir")
	}
}

// TestThePlayerHasNoLifecycleButHasTheWayOut.
//
// Sair não é do mestre: quem entrou numa mesa precisa poder sair dela.
func TestThePlayerHasNoLifecycleButHasTheWayOut(t *testing.T) {
	f := novoPiloto(t)
	tela := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if strings.Contains(tela, "Configurações da sessão") {
		t.Error("o jogador recebeu as configurações da sessão")
	}
	if strings.Contains(tela, "Excluir sessão") {
		t.Error("o jogador recebeu o excluir")
	}
	if !strings.Contains(tela, "Sair da sessão") {
		t.Error("o jogador não tem como sair da mesa")
	}

	// E a trava é do SERVIDOR, não do desenho.
	for _, gesto := range []string{"iniciar", "encerrar", "reiniciar", "titulo", "excluir"} {
		rec := f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/sessao/"+gesto, `{"titulodasessao":"x"}`)
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador passou em %q: %d", gesto, rec.Code)
		}
	}
}

// TestTheTitleSavesAndMayStayBlank.
//
// Vazio é legítimo: a identidade da sessão é o NÚMERO, e o título é o apelido da
// noite. Obrigar a um faria o mestre inventar texto para poder salvar.
func TestTheTitleSavesAndMayStayBlank(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()

	f.posta(t, f.mestre, f.urlDaMesa()+"/sessao/titulo", `{"titulodasessao":"A cripta do rio"}`)
	sess, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if !sess.Title.Valid || sess.Title.String != "A cripta do rio" {
		t.Fatalf("o título não foi salvo: %+v", sess.Title)
	}

	f.posta(t, f.mestre, f.urlDaMesa()+"/sessao/titulo", `{"titulodasessao":"   "}`)
	sess, _ = f.s.queries.GetSession(ctx, f.sessionID)
	if sess.Title.Valid && strings.TrimSpace(sess.Title.String) != "" {
		t.Errorf("o título em branco não virou nulo: %+v", sess.Title)
	}
}

// TestRestartingFromTheScreenEmptiesTheLiveTracker.
//
// O guarda da regra já cobre o `Forget`; este cobre o CAMINHO — que o comando da
// Mesa recarrega a fila depois, senão o `GetState` recria um estado vazio sem
// passar pelo banco e a próxima carga fria discordaria desta.
func TestRestartingFromTheScreenEmptiesTheLiveTracker(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n < 2 {
		t.Fatalf("a cena montou %d combatentes — não há o que reiniciar", n)
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/sessao/reiniciar", ""); rec.Code != http.StatusOK {
		t.Fatalf("reiniciar deu %d", rec.Code)
	}

	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("a fila ao vivo continua com %d combatentes", n)
	}
}

// TestDeletingErasesAndSendsTheGmToTheCampaign.
//
// O destino importa: voltar para a mesa apagada seria mandar o mestre para uma
// porta que não existe mais.
func TestDeletingErasesAndSendsTheGmToTheCampaign(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()

	rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/sessao/excluir", "")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("excluir deu %d, esperado 303", rec.Code)
	}
	if destino := rec.Header().Get("Location"); !strings.HasPrefix(destino, "/campanhas/") {
		t.Errorf("o destino depois de excluir é %q", destino)
	}
	if _, err := f.s.queries.GetSession(ctx, f.sessionID); err == nil {
		t.Error("a sessão continua no banco depois de excluída")
	}
}
