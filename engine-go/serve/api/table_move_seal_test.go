package api

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O SELO DO MOVIMENTO se lê "Movimento proposto", com o espaço.
//
// O rabo do selo (" proposto") mora num `<span>` próprio porque some no celular, e o espaço do começo
// dele é o que o templ apaga na geração: o `_templ.go` saía com `>proposto<`, e a tela dizia
// "MOVIMENTOPROPOSTO". O que se lê é o TEXTO do selo, como o leitor de tela e o olho o leem, e não a
// fonte do template.
func TestTheMoveSealReadsMovimentoProposto(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":2,"Y":0}}`); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}

	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	seal, found := textOfFirst(screen, "ds-board-seal")
	if !found {
		t.Fatal("não há selo no cartão do movimento — o caso não mede nada")
	}
	if seal != "Movimento proposto" {
		t.Errorf("o selo do movimento se lê %q, esperado \"Movimento proposto\"", seal)
	}
}

// textOfFirst devolve o texto do primeiro elemento que tem a classe, com os filhos, como o navegador o
// concatena — sem aparar, porque o espaço entre as partes é justamente o que se mede.
func textOfFirst(screen, class string) (string, bool) {
	z := html.NewTokenizer(strings.NewReader(screen))
	depth := 0
	var text strings.Builder
	for {
		switch z.Next() {
		case html.ErrorToken:
			return text.String(), depth > 0
		case html.StartTagToken:
			if depth > 0 {
				depth++
				continue
			}
			for more := true; more; {
				var key, value []byte
				key, value, more = z.TagAttr()
				if string(key) == "class" && strings.Contains(" "+string(value)+" ", " "+class+" ") {
					depth = 1
				}
			}
		case html.EndTagToken:
			if depth == 0 {
				continue
			}
			if depth--; depth == 0 {
				return text.String(), true
			}
		case html.TextToken:
			if depth > 0 {
				text.Write(z.Text())
			}
		}
	}
}
