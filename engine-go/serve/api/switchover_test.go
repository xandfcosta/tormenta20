package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// Entrar numa sessão leva à CENA DA SESSÃO.
//
// É o guarda de uma linha só de produção — quatro `href` — e ainda assim o mais
// fácil de perder: um caminho que existe e que gesto nenhum percorre é
// indistinguível de um que não existe.
//
// # A metade negativa saiu com o terreno dela (ALE-345)
//
// Aqui havia uma segunda asserção: que nenhuma porta escrevesse
// `/campanhas/{c}/sessoes/{s}`, que era o endereço da SPA. Ele é o endereço da
// CENA desde a ALE-345 — o `/mesa/` foi embora porque inventava uma entidade
// que o modelo não tem, e o caminho da SPA estava vago desde que ela foi
// apagada (ALE-314). Manter a negativa seria proibir o destino certo.
//
// A negativa que SOBRA é a do outro teste, e ela ainda discrimina:
// `/campaigns/` em inglês nunca foi rota deste servidor.

// asPortasParaASessao são as três cenas de onde se entra numa sessão. Uma
// esquecida é uma porta que continua levando para lugar nenhum, e ninguém nota
// até tentar jogar por ela.
func asPortasParaASessao(campanha int64) []struct{ Nome, Caminho string } {
	id := strconv.FormatInt(campanha, 10)
	return []struct{ Nome, Caminho string }{
		{"o Hub", "/"},
		{"a lista de campanhas", "/campanhas"},
		{"a crônica da campanha", "/campanhas/" + id},
	}
}

func TestEveryDoorLeadsToTheSessionScene(t *testing.T) {
	s, dono := hubFixture(t)
	campanha := seedCampaign(t, s, dono)
	sessao := seedSession(t, s, campanha)
	if _, err := s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		UpdatedAt: dbvalue.NowISO(), ID: sessao,
	}); err != nil {
		t.Fatalf("iniciar sessão: %v", err)
	}

	// Escrito à mão e não derivado da produção: derivar o destino faria o teste
	// concordar com o defeito.
	daSessao := "/campanhas/" + strconv.FormatInt(campanha, 10) +
		"/sessoes/" + strconv.FormatInt(sessao, 10)

	for _, porta := range asPortasParaASessao(campanha) {
		html := pedeHub(t, s, dono, http.MethodGet, porta.Caminho).Body.String()

		if !strings.Contains(html, daSessao) {
			t.Errorf("%s não leva à cena da sessão (%s)", porta.Nome, daSessao)
		}
	}
}

// A crônica tem DOIS caminhos para a sessão e eles são diferentes: o botão
// "Continuar a sessão" (só com uma viva) e a LINHA da linha do tempo (toda
// sessão, viva ou não). O de cima cobre o primeiro; este cobre o segundo, que é o
// único jeito de reabrir uma sessão encerrada.
func TestTheCampaignRowLeadsThereToo(t *testing.T) {
	s, dono := hubFixture(t)
	campanha := seedCampaign(t, s, dono)
	// Uma sessão PLANEJADA: sem `StartSessionFresh`, então não há "viva" e o
	// botão do cabeçalho não existe. O que sobra é a linha do tempo.
	sessao := seedSession(t, s, campanha)

	html := pedeHub(t, s, dono, http.MethodGet,
		"/campanhas/"+strconv.FormatInt(campanha, 10)).Body.String()

	daSessao := "/campanhas/" + strconv.FormatInt(campanha, 10) +
		"/sessoes/" + strconv.FormatInt(sessao, 10)
	if !strings.Contains(html, daSessao) {
		t.Errorf("a linha da crônica não leva à cena da sessão (%s)", daSessao)
	}
	if strings.Contains(html, "/campaigns/") {
		t.Error("a crônica ainda tem um caminho para a tela antiga")
	}
}
