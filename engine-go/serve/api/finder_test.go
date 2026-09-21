package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// OS DOIS GUARDAS DE COMPOSIÇÃO DO BUSCADOR.
//
// Os casos da REGRA moram em `web/finder`, com as funções puras de
// ranqueamento. Estes dois ficam aqui porque precisam do servidor montado: um
// prova que a rota lê o termo do SINAL do Datastar, o outro que a porta (a tela
// de entrar) não desenha o buscador, porque a casca dela declara não ter estado
// de cliente.
//
// O endereço é escrito à mão. A constante dele é interna à cena e inalcançável
// daqui — o que é sorte, porque importar o valor de quem está sendo testado faz
// o teste andar junto com o defeito.
const finderAddress = "/buscador"

// O caminho que o navegador usa de verdade.
//
// Pelo SINAL e não por `?busca=`: é assim que o `@get` do Datastar manda o que
// foi digitado, e a URL é só o caminho de quem abre o endereço à mão. Um guarda
// que só medisse a URL passaria verde com o sinal quebrado.
func TestTheFinderRouteReadsTheSignal(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	body := askTheFinder(t, s, eu, `{"finder":"abalado"}`)
	if !strings.Contains(body, "datastar-patch-elements") {
		t.Fatal("a rota não devolveu remendo nenhum — o resto do guarda mediria a resposta errada")
	}
	if !strings.Contains(body, "Abalado") {
		t.Error("o remendo não traz a condição buscada")
	}
	if !strings.Contains(body, `id="finder-found"`) {
		t.Error("o remendo não traz o id que ele substitui — o Datastar não teria onde aplicá-lo")
	}
}

// A caixa liga um sinal, e sinal é estado de cliente que viaja em TODA
// requisição seguinte — na porta, junto com a senha. O
// `TestTheDoorPutsNothingInADatastarSignal` cobra a regra geral; este cobra que
// esta caixa em particular ficou de fora.
//
// O controle é o segundo caso: a MESMA casca, numa tela com sessão, desenha.
func TestTheDoorDoesNotDrawTheFinder(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	door := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(door, httptest.NewRequest(http.MethodGet, "/entrar", nil))
	if strings.Contains(door.Body.String(), `id="finder"`) {
		t.Error("a porta desenhou a caixa do buscador, e com ela um sinal que viaja com a senha")
	}

	inside := pedeNoMestre(t, s, eu, "GET", "/mestre/bestiario", "")
	if !strings.Contains(inside.Body.String(), `id="finder"`) {
		t.Error("a caixa sumiu da cena com sessão — o guarda acima passaria por ausência de tudo")
	}
}

func askTheFinder(t *testing.T, s *Server, userID int64, signals string) string {
	t.Helper()
	u, err := s.queries.GetUserByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.accountGate().SignSession(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, finderAddress+"?datastar="+url.QueryEscape(signals), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("datastar-request", "true")
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("o buscador respondeu %d", rec.Code)
	}
	return rec.Body.String()
}
