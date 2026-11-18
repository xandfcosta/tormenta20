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

	doc, err := html.Parse(strings.NewReader(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()))
	if err != nil {
		t.Fatalf("ler a mesa: %v", err)
	}

	seals := nodesWithClassToken(doc, "ds-board-seal")
	if len(seals) == 0 {
		t.Fatal("não há selo no cartão do movimento — o caso não mede nada")
	}
	// O texto como o navegador o concatena, sem aparar: o espaço entre as partes é o que se mede.
	if seal := textOf(seals[0]); seal != "Movimento proposto" {
		t.Errorf("o selo do movimento se lê %q, esperado \"Movimento proposto\"", seal)
	}
}
