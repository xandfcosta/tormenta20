package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// APAGAR A CAMPANHA ESQUECE O ESTADO EM MEMÓRIA DAS SESSÕES DELA.
//
// # A invariante é de ORDEM, e ela é frágil por natureza
//
// Apagar a campanha leva as sessões por CASCATA. Depois disso não há mais como
// perguntar quais eram — então esquecer o estado em memória tem de acontecer
// ANTES da linha sumir. Invertida, a lista volta vazia, o laço não esquece
// ninguém, e o tabuleiro de cada sessão fica no mapa em memória batendo numa
// chave estrangeira que não existe mais: a mesa se declara suja para sempre.
//
// # Por que ele nasce agora
//
// A sequência estava certa e não tinha quem a prendesse: ela morava num handler,
// entre dois comentários, e a fatia que a move para o caso de uso é exatamente o
// tipo de mudança que a perde. Um teste que só aparece DEPOIS de o defeito
// acontecer chega tarde (ALE-359).
//
// # Como ele foi provado capaz de acusar
//
// Invertendo as duas linhas na origem: com o `DeleteCampaign` primeiro, a
// asserção abaixo reprova dizendo que a fila sobreviveu. É o controle que separa
// "a ordem está certa" de "este caso não mede ordem nenhuma".
func TestDeletingACampaignForgetsItsSessionsFirst(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	owner := seedUser(t, s, "dono@t.com")
	campaign := seedCampaign(t, s, owner)
	session := seedSession(t, s, campaign)

	if _, err := s.sessions.State(ctx, session); err != nil {
		t.Fatalf("hidratar a sessão: %v", err)
	}
	if _, err := s.sessions.AddInitiativeEntry(session, npc("Goblin", 12)); err != nil {
		t.Fatalf("pôr alguém na fila: %v", err)
	}
	// O CONTROLE: sem uma fila de verdade em memória, "a fila ficou vazia"
	// depois seria verdade desde o começo e não testemunharia nada.
	if n := len(stateOf(t, s.sessions, session).Initiative); n != 1 {
		t.Fatalf("a sessão tem %d na fila antes de apagar, e o caso precisa de 1", n)
	}

	rec := sceneFixture{s: s}.pede(t, owner, http.MethodPost,
		fmt.Sprintf("/campanhas/%d/excluir", campaign), "")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("excluir respondeu %d, queria 303", rec.Code)
	}

	// A LINHA SUMIU do banco com a cascata, então a leitura tem de FALHAR. Se ela
	// responder, respondeu de MEMÓRIA — e memória que sobrevive à exclusão é
	// exatamente o que este caso existe para pegar (ALE-373).
	survivor, err := s.sessions.State(ctx, session)
	if err == nil {
		t.Errorf("a fila da sessão %d respondeu com %d combatente(s) depois de a campanha "+
			"ser apagada.\nO estado em memória tem de ser esquecido ANTES da linha sumir: "+
			"depois da cascata\nnão há como perguntar quais sessões eram.",
			session, len(survivor.Initiative))
	}
}
