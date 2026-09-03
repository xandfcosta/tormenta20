package reader

import (
	"net/http/httptest"
	"testing"

	"t20engine/web/bookui"
)

// As duas REGRAS do leitor, presas onde elas moram (ALE-278).
//
// O endereço dele é COMPARTILHÁVEL — o mestre manda "olha na p289" no chat da
// mesa —, então ele se digita à mão e chega torto. As duas decisões abaixo são
// sobre o que fazer com o torto, e nenhuma delas precisa de servidor: elas são
// função de uma requisição para um view.
//
// A bancada equivalente do hospedeiro (`TestTheReaderPageRefusesGarbage`) mede a
// mesma coisa pelo HTML servido, e as duas convivem porque medem coisas
// diferentes: aquela prende que a rota está montada e desenha; esta prende o
// VALOR, e por isso pode cobrar os casos de borda um por um sem subir um banco.

// TestGarbageInThePageFallsBackToTheFirst.
//
// Nunca 404 e nunca erro: derrubar a cena por um número torto na URL trocaria um
// incômodo pequeno por uma tela quebrada, e a primeira página é uma resposta
// honesta a "não entendi qual você quis".
func TestGarbageInThePageFallsBackToTheFirst(t *testing.T) {
	livro := bookui.BookAddress{Base: "/livro?v=abc", Abertura: 6}
	for _, alvo := range []string{
		"/livro/ler?p=abacaxi",
		"/livro/ler?p=-3",
		"/livro/ler?p=0",
		"/livro/ler?p=",
		"/livro/ler",
	} {
		v := readerFromRequest(httptest.NewRequest("GET", alvo, nil), livro)
		if v.Page != 1 {
			t.Errorf("%s abriu na página %d, e o torto tem de cair na primeira", alvo, v.Page)
		}
	}
	// O CONTROLE: um número BOM passa. Sem ele, um `readerFromRequest` que
	// devolvesse 1 sempre passaria em todos os casos acima.
	v := readerFromRequest(httptest.NewRequest("GET", "/livro/ler?p=289", nil), livro)
	if v.Page != 289 {
		t.Fatalf("a página válida virou %d — o guarda acima passaria sobre um leitor travado na 1", v.Page)
	}
}

// TestAForeignRefererDoesNotBecomeABackLink.
//
// O link de VOLTAR sai do `Referer`, e aceitar qualquer um poria o endereço de
// um terceiro na barra de quem está na nossa mesa. Quem chegou por um link
// colado volta para o Hub, que é o destino honesto de "não sei de onde você
// veio".
func TestAForeignRefererDoesNotBecomeABackLink(t *testing.T) {
	livro := bookui.BookAddress{Base: "/livro?v=abc", Abertura: 6}
	for _, referer := range []string{
		"https://exemplo.invalido/pagina",
		"http://localhost:3001/mestre/bestiario", // absoluto, mesmo sendo nosso
		"",
	} {
		req := httptest.NewRequest("GET", "/livro/ler?p=10", nil)
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		if v := readerFromRequest(req, livro); v.Back != "/" {
			t.Errorf("Referer %q virou link de voltar %q", referer, v.Back)
		}
	}
	// O CONTROLE: um caminho RELATIVO nosso vira o voltar. Sem ele, um leitor que
	// devolvesse "/" sempre passaria nos três casos acima.
	req := httptest.NewRequest("GET", "/livro/ler?p=10", nil)
	req.Header.Set("Referer", "/mestre/bestiario")
	if v := readerFromRequest(req, livro); v.Back != "/mestre/bestiario" {
		t.Fatalf("o voltar de um endereço nosso virou %q", v.Back)
	}
}
