package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// A VIRADA (ALE-269): entrar numa sessão passa a ser entrar na Mesa em Datastar.
//
// Este é o guarda de uma linha só de produção — quatro `href` — e ainda assim o
// mais fácil de perder: a Mesa nova FUNCIONAVA havia semanas e não era alcançável
// por gesto nenhum, só por URL digitada. Um caminho que existe e ninguém percorre
// é indistinguível de um que não existe.
//
// O que se prende é a NEGATIVA junto com a positiva: achar `/mesa/` numa
// página não prova que o link velho saiu — as duas rotas cabem no mesmo HTML, e
// foi assim que a tela conviveu com as duas durante toda a migração.

// asPortasParaASessao são as três cenas de onde se entra numa sessão. Uma
// esquecida é uma porta que continua levando para a tela antiga, e ninguém nota
// até tentar jogar por ela.
func asPortasParaASessao(campanha int64) []struct{ Nome, Caminho string } {
	id := strconv.FormatInt(campanha, 10)
	return []struct{ Nome, Caminho string }{
		{"o Hub", "/"},
		{"a lista de campanhas", "/campanhas"},
		{"a crônica da campanha", "/campanhas/" + id},
	}
}

// TestTODASasPortasLevamAMesaEmDatastar.
func TestTodasAsPortasLevamAMesaEmDatastar(t *testing.T) {
	s, dono := hubFixture(t)
	campanha := seedCampaign(t, s, dono)
	sessao := seedSession(t, s, campanha)
	if _, err := s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		UpdatedAt: plataforma.NowISO(), ID: sessao,
	}); err != nil {
		t.Fatalf("iniciar sessão: %v", err)
	}

	// Escrito à mão nos dois lados, e não derivado da produção: derivar o destino
	// novo faria o teste concordar com o defeito, e derivar o velho o faria
	// procurar uma string que ninguém escreve mais.
	daMesa := "/mesa/" + strconv.FormatInt(campanha, 10) + "/" + strconv.FormatInt(sessao, 10)
	daSPA := "/campaigns/" + strconv.FormatInt(campanha, 10) + "/sessions/" + strconv.FormatInt(sessao, 10)

	for _, porta := range asPortasParaASessao(campanha) {
		html := pedeHub(t, s, dono, http.MethodGet, porta.Caminho).Body.String()

		// O CONTROLE primeiro: a cena chegou e tem a sessão nela. Sem ele, "não
		// achei o link velho" seria verdade também numa página em branco, num
		// 403, ou numa cena que deixou de listar a sessão.
		if !strings.Contains(html, daMesa) {
			t.Errorf("%s não leva à Mesa em Datastar (%s)", porta.Nome, daMesa)
			continue
		}
		if strings.Contains(html, daSPA) {
			t.Errorf("%s ainda leva à tela antiga (%s)", porta.Nome, daSPA)
		}
	}
}

// TestALinhaDaCRONICAtambemLeva.
//
// A crônica tem DOIS caminhos para a sessão e eles são diferentes: o botão
// "Continuar a sessão" (só com uma viva) e a LINHA da linha do tempo (toda
// sessão, viva ou não). O de cima cobre o primeiro; este cobre o segundo, que é o
// único jeito de reabrir uma sessão encerrada.
func TestALinhaDaCronicaTambemLeva(t *testing.T) {
	s, dono := hubFixture(t)
	campanha := seedCampaign(t, s, dono)
	// Uma sessão PLANEJADA: sem `StartSessionFresh`, então não há "viva" e o
	// botão do cabeçalho não existe. O que sobra é a linha do tempo.
	sessao := seedSession(t, s, campanha)

	html := pedeHub(t, s, dono, http.MethodGet,
		"/campanhas/"+strconv.FormatInt(campanha, 10)).Body.String()

	daMesa := "/mesa/" + strconv.FormatInt(campanha, 10) + "/" + strconv.FormatInt(sessao, 10)
	if !strings.Contains(html, daMesa) {
		t.Errorf("a linha da crônica não leva à Mesa em Datastar (%s)", daMesa)
	}
	if strings.Contains(html, "/campaigns/") {
		t.Error("a crônica ainda tem um caminho para a tela antiga")
	}
}
