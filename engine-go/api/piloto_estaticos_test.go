package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O guarda do CACHE dos estáticos (ALE-264).
//
// O defeito que ele prende é de EXPERIÊNCIA e volta em SILÊNCIO: sem validador
// nem `Cache-Control`, o navegador rebaixa 113KB de CSS bloqueante de
// renderização a cada troca de página, o documento novo não pinta até a folha
// chegar, e o navegador mostra o branco entre as duas telas. Nada estoura. O
// dono viu antes de qualquer teste, e é assim que esta classe é descoberta.
//
// Por que HANDLER e não e2e: a garantia é sobre CABEÇALHO, e cabeçalho é a
// camada mais barata que a segura. Um e2e que medisse "não piscou" seria caro,
// intermitente, e ainda passaria verde num dia de máquina rápida.

func pedeEstatico(t *testing.T, alvo string, cabecalhos map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, alvo, nil)
	for k, v := range cabecalhos {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	comCacheVersionado(versaoDosEstaticos, "public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("/* folha */"))
	})).ServeHTTP(rec, req)
	return rec
}

// TestOEnderecoVERSIONADOnaoVolta: um ano e `immutable`.
//
// É o caminho que TIRA a ida à rede, e é ele que conserta o clarão — revalidar
// ainda atrasaria a primeira pintura, porque a folha bloqueia a renderização.
func TestOEnderecoVersionadoNaoVolta(t *testing.T) {
	rec := pedeEstatico(t, "/piloto/static/piloto.css?v="+versaoDosEstaticos, nil)

	cache := rec.Header().Get("Cache-Control")
	if !strings.Contains(cache, "immutable") {
		t.Errorf("o endereço versionado não é imutável: %q — cada troca de página rebaixa a folha e a tela pisca", cache)
	}
	if !strings.Contains(cache, "max-age=31536000") {
		t.Errorf("Cache-Control = %q", cache)
	}
	if etag := rec.Header().Get("ETag"); etag != `"`+versaoDosEstaticos+`"` {
		t.Errorf("ETag = %q, esperado o dígito entre aspas", etag)
	}
}

// TestOEnderecoSEMversaoNAOeEterno.
//
// A outra metade, e ela é deliberadamente o pior caso: um endereço sem versão
// pode ter sido guardado antes de um deploy, e servi-lo como eterno prenderia a
// pessoa numa folha velha sem nenhum gesto que a resgate — nem recarregar.
func TestOEnderecoSemVersaoNaoEEterno(t *testing.T) {
	rec := pedeEstatico(t, "/piloto/static/piloto.css", nil)

	if cache := rec.Header().Get("Cache-Control"); strings.Contains(cache, "immutable") {
		t.Errorf("endereço sem versão servido como imutável (%q): uma folha velha ficaria presa para sempre", cache)
	}
	// Mas com ETag, senão ele não tem nem como revalidar — que era o estado
	// anterior a esta fatia, e a causa do clarão.
	if rec.Header().Get("ETag") == "" {
		t.Error("sem ETag não há 304: o navegador rebaixa a folha inteira toda vez")
	}
}

// TestQuemJAtemAfolhaRECEBE304.
//
// É o que o `embed` não podia dar sozinho: arquivo embutido tem modtime ZERO, e
// o `http.ServeContent` não emite `Last-Modified` de um tempo nulo nem inventa
// `ETag`. O `http.FileServer` estava certo; o sistema de arquivos por baixo é
// que não tinha o que datar.
func TestQuemJaTemAFolhaRecebe304(t *testing.T) {
	rec := pedeEstatico(t, "/piloto/static/piloto.css",
		map[string]string{"If-None-Match": `"` + versaoDosEstaticos + `"`})

	if rec.Code != http.StatusNotModified {
		t.Errorf("quem já tem a folha recebeu %d em vez de 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("o 304 veio com %d bytes de corpo", rec.Body.Len())
	}

	// O CONTROLE: com um ETag ANTIGO o corpo VEM. Sem isto, "recebeu 304" seria
	// verdade também sobre um handler que responde 304 para todo mundo — e o
	// sintoma seria a folha nova nunca chegando depois de um deploy.
	velho := pedeEstatico(t, "/piloto/static/piloto.css",
		map[string]string{"If-None-Match": `"digito-de-outro-binario"`})
	if velho.Code != http.StatusOK || velho.Body.Len() == 0 {
		t.Errorf("com ETag velho veio %d e %d bytes — a folha nova não chegaria", velho.Code, velho.Body.Len())
	}
}

// TestODIGITOeESTAVELentreLeituras.
//
// Ele decide invalidação de cache: se variasse entre dois boots do MESMO
// binário, todo reinício jogaria fora o cache de todo mundo — e o clarão
// voltaria uma vez por deploy sem ninguém entender por quê.
func TestODigitoEEstavelEntreLeituras(t *testing.T) {
	if a, b := digitoDosEstaticos(), digitoDosEstaticos(); a != b {
		t.Errorf("o dígito mudou entre duas leituras: %q e %q", a, b)
	}
	if len(versaoDosEstaticos) == 0 {
		t.Fatal("o dígito saiu vazio: toda URL versionada viraria `?v=` e o cache nunca casaria")
	}
	// E ele tem de VIR do conteúdo: um dígito constante casaria com ele mesmo
	// para sempre e serviria folha velha como imutável depois de um deploy.
	if strings.Trim(versaoDosEstaticos, "0") == "" {
		t.Errorf("o dígito é %q, que não parece hash de conteúdo", versaoDosEstaticos)
	}
}

// TestTODOenderecoEstaticoDaPaginaEVERSIONADO.
//
// A regressão silenciosa desta fatia: caminho cru continua funcionando, e é
// servido SEM cache. A página que escrever um à mão volta a piscar, sozinha, e
// ninguém liga uma coisa à outra — o sintoma aparece em UMA tela e a causa está
// noutro arquivo.
func TestTodoEnderecoEstaticoDaPaginaEVersionado(t *testing.T) {
	f := novoPiloto(t)
	tela := f.pede(t, f.mestre, http.MethodGet, "/piloto/", "").Body.String()

	// O CONTROLE: a página REFERENCIA estáticos. Sem ele, "nenhum endereço cru"
	// seria verdade também sobre uma página que não carregou.
	if !strings.Contains(tela, "/piloto/static/") {
		t.Fatal("a página não referencia estático nenhum — o guarda mediria a tela errada")
	}
	if strings.Contains(tela, `"/piloto/static/piloto.css"`) {
		t.Error("a folha entrou sem versão: ela volta a ser rebaixada a cada troca de página")
	}
	for _, pedaco := range strings.Split(tela, "/piloto/static/")[1:] {
		fim := strings.IndexAny(pedaco, `"'`)
		if fim < 0 {
			continue
		}
		if !strings.Contains(pedaco[:fim], "?v=") {
			t.Errorf("endereço estático sem versão: /piloto/static/%s", pedaco[:fim])
		}
	}
}
