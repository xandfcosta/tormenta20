package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O PREFIXO `/piloto` SAIU DO ENDEREÇO, e o velho não responde (ALE-280).
//
// Ele era o nome de uma migração — a SPA de um lado, as cenas em Datastar do
// outro — e ficou onde mais custa: no endereço que o jogador favorita. A SPA
// saiu na ALE-272 e o prefixo perdeu a razão de existir.
//
// # Por que 404 e não desvio
//
// Decisão do dono: CORTE SECO. O app nunca foi usado numa mesa real, não há
// link de jogador a proteger, e um desvio a menos é uma exceção a menos no mux.
// Os dezessete endereços da SPA em `legacy_addresses.go` continuam desviando —
// aqueles foram publicados de verdade.
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

// A TABELA dos endereços antigos não pode ganhar uma entrada com o prefixo.
//
// É a outra metade da decisão: `/piloto/x` não vira linha de desvio. Um `grep`
// não serviria — o que se quer proibir é o VALOR em tempo de execução, e ele
// pode ser montado por concatenação.
func TestNoLegacyAddressMentionsThePilot(t *testing.T) {
	for _, endereco := range legacyAddresses {
		if strings.Contains(endereco.Padrao, "/piloto") {
			t.Errorf("a tabela tem %q: o prefixo velho não é endereço, é 404", endereco.Padrao)
		}
		destino := endereco.Destino(httptest.NewRequest(http.MethodGet, "/x", nil))
		if strings.Contains(destino, "/piloto") {
			t.Errorf("%s desvia para %q, que não existe mais", endereco.Padrao, destino)
		}
	}
}
