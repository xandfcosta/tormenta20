package httpio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O guarda da POLÍTICA de cache por URL versionada.
//
// O defeito que ele prende é de EXPERIÊNCIA e volta em SILÊNCIO: sem validador
// nem `Cache-Control`, o navegador rebaixa a folha bloqueante de renderização a
// cada troca de página, o documento novo não pinta até ela chegar, e aparece o
// branco entre as duas telas. Nada estoura.
//
// Por que HANDLER e não e2e: a garantia é sobre CABEÇALHO, que é a camada mais
// barata que a segura. Um e2e que medisse "não piscou" seria caro, intermitente,
// e passaria verde num dia de máquina rápida.

// testVersion é escrito à mão de propósito: esta política não conhece os
// estáticos, e derivar o esperado de quem produz o dígito de verdade seria
// espelhar a implementação na asserção.
const testVersion = "a1b2c3d4e5f6"

func requestWithCache(t *testing.T, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	WithVersionedCache(testVersion, "public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("/* folha */"))
	})).ServeHTTP(rec, req)
	return rec
}

// Um ano e `immutable`: é o caminho que TIRA a ida à rede, e revalidar ainda
// atrasaria a primeira pintura porque a folha bloqueia a renderização.
func TestTheVersionedAddressDoesNotComeBack(t *testing.T) {
	rec := requestWithCache(t, "/static/app.css?v="+testVersion, nil)

	cache := rec.Header().Get("Cache-Control")
	if !strings.Contains(cache, "immutable") {
		t.Errorf("o endereço versionado não é imutável: %q — cada troca de página rebaixa a folha e a tela pisca", cache)
	}
	if !strings.Contains(cache, "max-age=31536000") {
		t.Errorf("Cache-Control = %q", cache)
	}
	if etag := rec.Header().Get("ETag"); etag != `"`+testVersion+`"` {
		t.Errorf("ETag = %q, esperado o dígito entre aspas", etag)
	}
}

// A outra metade, deliberadamente o pior caso: um endereço sem versão pode ter
// sido guardado antes de um deploy, e servi-lo como eterno prenderia a pessoa
// numa folha velha sem nenhum gesto que a resgate — nem recarregar.
func TestTheUnversionedAddressIsNotEternal(t *testing.T) {
	rec := requestWithCache(t, "/static/app.css", nil)

	if cache := rec.Header().Get("Cache-Control"); strings.Contains(cache, "immutable") {
		t.Errorf("endereço sem versão servido como imutável (%q): uma folha velha ficaria presa para sempre", cache)
	}
	// Mas com ETag, senão ele não tem nem como revalidar.
	if rec.Header().Get("ETag") == "" {
		t.Error("sem ETag não há 304: o navegador rebaixa a folha inteira toda vez")
	}
}

// É o que o `embed` não dá sozinho: arquivo embutido tem modtime ZERO, e o
// `http.ServeContent` não emite `Last-Modified` de um tempo nulo nem inventa
// `ETag`.
func TestWhoeverAlreadyHasTheStylesheetGets304(t *testing.T) {
	rec := requestWithCache(t, "/static/app.css",
		map[string]string{"If-None-Match": `"` + testVersion + `"`})

	if rec.Code != http.StatusNotModified {
		t.Errorf("quem já tem a folha recebeu %d em vez de 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("o 304 veio com %d bytes de corpo", rec.Body.Len())
	}

	// O CONTROLE: com um ETag ANTIGO o corpo VEM. Sem isto, "recebeu 304" seria
	// verdade também sobre um handler que responde 304 para todo mundo — e o
	// sintoma seria a folha nova nunca chegando depois de um deploy.
	stale := requestWithCache(t, "/static/app.css",
		map[string]string{"If-None-Match": `"digito-de-outro-binario"`})
	if stale.Code != http.StatusOK || stale.Body.Len() == 0 {
		t.Errorf("com ETag velho veio %d e %d bytes — a folha nova não chegaria", stale.Code, stale.Body.Len())
	}
}

// Versão VAZIA não pode responder 304 para todo mundo: `strings.Contains(x, "")`
// é verdadeiro, e sem esta guarda o corpo nunca sairia.
func TestAnEmptyVersionStillSendsTheBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	req.Header.Set("If-None-Match", `"qualquer-coisa"`)
	rec := httptest.NewRecorder()
	WithVersionedCache("", "public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("/* folha */"))
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Errorf("com versão vazia veio %d e %d bytes — o corpo nunca sairia", rec.Code, rec.Body.Len())
	}
}
