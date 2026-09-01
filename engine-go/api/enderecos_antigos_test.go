package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// muxComOsEnderecosAntigos monta só os desvios, sem o resto do servidor: o que
// se afirma aqui é a TABELA e o casamento de padrões, e um mux completo traria
// junto a autenticação e o banco, que não têm nada a ver com desviar endereço.
func muxComOsEnderecosAntigos() *http.ServeMux {
	mux := http.NewServeMux()
	MontaEnderecosAntigos(mux)
	return mux
}

func oDesvioDe(t *testing.T, caminho string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	muxComOsEnderecosAntigos().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, caminho, nil))
	return rec.Code, rec.Header().Get("Location")
}

// TestEveryLegacyAddressLandsOnAScene é guarda de varredura sobre a tabela.
//
// Ela é uma lista escrita à mão — não há de onde derivá-la, porque a fonte era o
// `beforeLoad` de cada rota da SPA, que some. O que este guarda impede é a
// forma de errar que sobra: uma entrada apontando para um endereço que o piloto
// não atende, ou um padrão que nunca casa.
func TestEveryLegacyAddressLandsOnAScene(t *testing.T) {
	// Treze cascas da SPA na fatia 10a, mais os quatro endereços das telas que só
	// morreram com ela na 10c — menos as três que a raiz absorveu na ALE-280.
	if len(osEnderecosAntigos) < 14 {
		t.Fatalf("%d endereços na tabela, e são pelo menos catorze", len(osEnderecosAntigos))
	}
	for _, endereco := range osEnderecosAntigos {
		// O padrão vira um caminho de exemplo: `{id}` e `{tool}` recebem um
		// valor, e `{$}` (fim exato) vira a barra que ele exige.
		caminho := strings.NewReplacer(
			"{id}", "7", "{sid}", "9", "{tool}", "condicoes", "{token}", "abc-123",
			"{passo}", "raca", "{$}", "",
		).Replace(endereco.Padrao)

		status, destino := oDesvioDe(t, caminho)
		if status != http.StatusFound {
			t.Errorf("%s: status %d, esperado 302", caminho, status)
		}
		// O DESTINO NÃO PODE SER A ORIGEM, e esta asserção substitui uma que
		// deixou de medir. Ela era `HasPrefix(destino, "/piloto")`, e quando as
		// cenas subiram para a raiz (ALE-280) ela virou "começa com barra" — que
		// todo caminho começa. Verde sobre nada.
		//
		// O que ficou no lugar é o defeito que aquela mudança de fato produziu:
		// três endereços que a SPA e o piloto escreviam IGUAIS passaram a apontar
		// para si mesmos, e no mux o padrão literal ganha do `"/"` das cenas — o
		// desvio responderia 302 para si mesmo para sempre.
		if destino == caminho {
			t.Errorf("%s desvia para SI MESMO: a cena já é dona deste endereço,\n"+
				"e a entrada na tabela vira um laço de redirecionamento.", caminho)
		}
		if !strings.HasPrefix(destino, "/") {
			t.Errorf("%s levou para %q, que não é um endereço deste servidor", caminho, destino)
		}
	}
}

// TestOEnderecoAntigoLevaOQueOFazValer: o parâmetro de busca é metade do
// endereço em quatro deles — um convite sem token é uma tela de erro.
func TestOEnderecoAntigoLevaOQueOFazValer(t *testing.T) {
	casos := []struct{ de, para string }{
		{"/campaigns/join?token=abc-123", "/campanhas/entrar?token=abc-123"},
		{"/join/abc-123", "/campanhas/entrar?token=abc-123"},
		{"/register?convite=xyz", "/criar-conta?convite=xyz"},
		{"/login?redirect=%2Fcampanhas", "/entrar?redirect=%2Fcampanhas"},
		// A SEÇÃO da crônica viaja junto: `?tab=` é endereço guardado, e é o que
		// o caso "encaminha COM a seção" do e2e afirma.
		{"/campaigns/12?tab=config", "/campanhas/12?tab=config"},
		{"/campaigns/12", "/campanhas/12"},
		{"/gm/condicoes", "/mestre/condicoes"},
		// A FICHA, a FORJA e a MESA — os três endereços que só puderam desviar
		// quando a tela antiga deles deixou de existir (fatia 10c).
		{"/characters/13", "/personagens/13"},
		{"/characters/13?tab=bag", "/personagens/13?tab=bag"},
		{"/characters/new", "/personagens/nova"},
		{"/characters/new/equipamento", "/personagens/nova"},
		{"/campaigns/1/sessions/4", "/mesa/1/4"},
		// Busca que a casca da SPA não preservava continua não passando: o que
		// não valia antes não passa a valer por acidente.
	}
	for _, caso := range casos {
		_, destino := oDesvioDe(t, caso.de)
		if destino != caso.para {
			t.Errorf("%s levou para %q, esperado %q", caso.de, destino, caso.para)
		}
	}
}

// TestOLiteralGanhaDoCuringaNoEnderecoAntigo: `/campaigns/new` e
// `/campaigns/join` são irmãos de `/campaigns/{id}` no mesmo mux, e casar pelo
// curinga mandaria quem clica em "nova campanha" para a crônica de uma campanha
// chamada "new".
func TestOLiteralGanhaDoCuringaNoEnderecoAntigo(t *testing.T) {
	casos := map[string]string{
		"/campaigns/new":  "/campanhas/nova",
		"/campaigns/join": "/campanhas/entrar",
		"/campaigns/":     "/campanhas",
	}
	for de, para := range casos {
		if _, destino := oDesvioDe(t, de); destino != para {
			t.Errorf("%s levou para %q, esperado %q", de, destino, para)
		}
	}
}

// TestASaudeRespondeNaRaizEnaAPI: dois endereços para a mesma resposta, e o da
// raiz é o que a infraestrutura conhece.
//
// Ele nasceu VERMELHO no CI: quando a API saiu da raiz (fatia 10c), a sonda de
// prontidão ficou esperando trinta segundos por um `/health` que respondia 404
// num servidor que já escutava — e a mensagem dizia "o servidor não subiu". O
// `healthcheck` do compose bate no mesmo lugar, então em produção o contêiner
// seria marcado insalubre sem nada de errado com ele.
func TestASaudeRespondeNaRaizEnaAPI(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	mux.Handle("/health", s.HealthProbe())
	mux.Handle("/api/", http.StripPrefix("/api", s.Router()))

	for _, caminho := range []string{"/health", "/api/health"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, caminho, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d, esperado 200", caminho, rec.Code)
		}
		// O corpo diz `degraded` aqui porque o servidor de teste não prima
		// catálogo, e isso é o comportamento certo (ALE-155): o que este caso
		// afirma é que o ENDEREÇO responde, não que a mesa está inteira.
		if !strings.Contains(rec.Body.String(), `"status"`) {
			t.Errorf("%s: corpo %q não é a resposta da sonda", caminho, rec.Body.String())
		}
	}
}
