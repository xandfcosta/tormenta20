package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// O PREFIXO `/piloto` SAIU DO ENDEREÇO, e o velho não responde.
//
// # Por que 404 e não desvio
//
// Decisão do dono: CORTE SECO. O app nunca foi usado numa mesa real, não há
// link de jogador a proteger, e um desvio a menos é uma exceção a menos no mux.
//
// # Por que este guarda existe
//
// Uma decisão que se cumpre por AUSÊNCIA não deixa rastro no código: não há
// linha nenhuma escrita para produzir o 404, ele vem de o roteador simplesmente
// não conhecer aquele caminho. Sem este caso, alguém que amanhã acrescente um
// desvio "por segurança" não encontra nada que discorde — e o mux volta a ter a
// exceção que esta issue tirou.
//
// A varredura é por CENA e não um caminho de exemplo, porque o erro que ela
// pega é o desvio ressuscitar numa só.
func TestTheOldPilotPrefixIsGone(t *testing.T) {
	s := newTestServer(t)
	roteador := s.WebRouter()

	cenas := []string{
		"/", "/campanhas", "/personagens", "/grimorio", "/admin",
		"/mestre/bestiario", "/entrar", "/criar-conta", "/mesa/1/4",
	}
	for _, scene := range cenas {
		antigo := "/piloto" + scene
		rec := httptest.NewRecorder()
		roteador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, antigo, nil))

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s respondeu %d, e o corte foi SECO — o prefixo velho não é endereço deste app",
				antigo, rec.Code)
		}
		if destino := rec.Header().Get("Location"); destino != "" {
			t.Errorf("%s desviou para %q: a decisão foi 404, e um desvio aqui é a exceção que a ALE-280 tirou do mux",
				antigo, destino)
		}
	}
}

// E o CONTROLE, sem o qual o caso acima é verde sobre nada: as mesmas cenas, no
// endereço NOVO, não podem dar 404.
//
// Sem ele um roteador quebrado — ou um `newTestServer` que não montasse rota
// nenhuma — passaria no guarda de cima com louvor, porque "tudo dá 404" atende
// perfeitamente a "o endereço velho dá 404".
func TestTheNewAddressesAnswer(t *testing.T) {
	s := newTestServer(t)
	roteador := s.WebRouter()

	for _, scene := range []string{"/", "/entrar", "/criar-conta"} {
		rec := httptest.NewRecorder()
		roteador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, scene, nil))

		// Anônimo: a porta responde 200 e o resto manda para ela com 303. O que
		// não pode acontecer é 404.
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s deu 404 — o controle deste arquivo caiu, e o guarda de cima passou a medir nada", scene)
		}
	}
}

// TUDO O QUE O PROCESSO ATENDE sai de um roteador só.
//
// Este caso era IMPOSSÍVEL de escrever antes: as quatro rotas abaixo moravam num
// roteador da biblioteca padrão montado no `cmd/api`, e nenhum teste deste
// pacote alcançava aquele mux — a raiz de composição do binário não tinha teste
// nenhum, que é o que permite uma rota sumir na montagem sem ninguém ver.
//
// Ele afirma ROTEAMENTO e não conteúdo: o que importa é o roteador conhecer o
// caminho. Por isso o alvo protegido é medido por "não é 404" — 401 é a resposta
// certa de uma rota que existe e exige sessão, e foi ela que provou o `Mount`
// do `/api` funcionando com o prefixo removido.
func TestTheProcessServesEverythingFromOneRouter(t *testing.T) {
	roteador := newTestServer(t).WebRouter()

	for _, caso := range []struct {
		alvo   string
		quero  int
		porque string
	}{
		{"/health", http.StatusOK, "a sonda do compose e o `-health` do binário perguntam na RAIZ"},
		{"/api/health", http.StatusOK, "a mesma saúde sob o prefixo da API"},
		{"/favicon.svg", http.StatusOK, "o layout pede o ícone por caminho absoluto"},
		{"/fonts/cinzel-latin.woff2", http.StatusOK, "a folha pede a Cinzel por caminho absoluto"},
		{"/static/app.css", http.StatusOK, "a folha e o bundle do Datastar são anônimos"},
	} {
		rec := httptest.NewRecorder()
		roteador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, caso.alvo, nil))
		if rec.Code != caso.quero {
			t.Errorf("%s respondeu %d, quero %d — %s", caso.alvo, rec.Code, caso.quero, caso.porque)
		}
	}

	// A API sob `/api`, com o prefixo TIRADO pelo `Mount`: se ele não tirasse, o
	// handler receberia `/api/campanhas` e devolveria 404 em vez de 401.
	rec := httptest.NewRecorder()
	roteador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/campanhas", nil))
	if rec.Code == http.StatusNotFound {
		t.Error("/api/campanhas respondeu 404: o `Mount` não está tirando o prefixo, e a API inteira está fora do ar")
	}
}
