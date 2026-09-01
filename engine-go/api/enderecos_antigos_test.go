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

// TestTodoEnderecoAntigoLevaAoPiloto é guarda de varredura sobre a tabela.
//
// Ela é uma lista escrita à mão — não há de onde derivá-la, porque a fonte era o
// `beforeLoad` de cada rota da SPA, que some. O que este guarda impede é a
// forma de errar que sobra: uma entrada apontando para um endereço que o piloto
// não atende, ou um padrão que nunca casa.
func TestTodoEnderecoAntigoLevaAoPiloto(t *testing.T) {
	// Treze cascas da SPA na fatia 10a, mais os quatro endereços das telas que
	// só morreram com ela na 10c: a ficha, a forja (em dois formatos) e a mesa.
	if len(osEnderecosAntigos) < 17 {
		t.Fatalf("%d endereços na tabela, e são pelo menos dezessete", len(osEnderecosAntigos))
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
		if !strings.HasPrefix(destino, "/piloto/") {
			t.Errorf("%s levou para %q, que não é do piloto", caminho, destino)
		}
	}
}

// TestOEnderecoAntigoLevaOQueOFazValer: o parâmetro de busca é metade do
// endereço em quatro deles — um convite sem token é uma tela de erro.
func TestOEnderecoAntigoLevaOQueOFazValer(t *testing.T) {
	casos := []struct{ de, para string }{
		{"/campaigns/join?token=abc-123", "/piloto/campanhas/entrar?token=abc-123"},
		{"/join/abc-123", "/piloto/campanhas/entrar?token=abc-123"},
		{"/register?convite=xyz", "/piloto/criar-conta?convite=xyz"},
		{"/redefinir-senha?token=t-9", "/piloto/redefinir-senha?token=t-9"},
		{"/login?redirect=%2Fpiloto%2Fcampanhas", "/piloto/entrar?redirect=%2Fpiloto%2Fcampanhas"},
		// A SEÇÃO da crônica viaja junto: `?tab=` é endereço guardado, e é o que
		// o caso "encaminha COM a seção" do e2e afirma.
		{"/campaigns/12?tab=config", "/piloto/campanhas/12?tab=config"},
		{"/campaigns/12", "/piloto/campanhas/12"},
		{"/gm/condicoes", "/piloto/mestre/condicoes"},
		// A FICHA, a FORJA e a MESA — os três endereços que só puderam desviar
		// quando a tela antiga deles deixou de existir (fatia 10c).
		{"/characters/13", "/piloto/personagens/13"},
		{"/characters/13?tab=bag", "/piloto/personagens/13?tab=bag"},
		{"/characters/new", "/piloto/personagens/nova"},
		{"/characters/new/equipamento", "/piloto/personagens/nova"},
		{"/campaigns/1/sessions/4", "/piloto/mesa/1/4"},
		// Busca que a casca da SPA não preservava continua não passando: o que
		// não valia antes não passa a valer por acidente.
		{"/grimorio?ruido=1", "/piloto/grimorio"},
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
		"/campaigns/new":  "/piloto/campanhas/nova",
		"/campaigns/join": "/piloto/campanhas/entrar",
		"/campaigns/":     "/piloto/campanhas",
	}
	for de, para := range casos {
		if _, destino := oDesvioDe(t, de); destino != para {
			t.Errorf("%s levou para %q, esperado %q", de, destino, para)
		}
	}
}
