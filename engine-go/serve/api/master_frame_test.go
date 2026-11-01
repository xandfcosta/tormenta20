package api

import (
	"strings"
	"testing"
)

// A MOLDURA DA MESA DO MESTRE leva a todas as treze telas, nas duas larguras
// (ALE-538): o menu do desktop tem as duas ferramentas e os onze catálogos, e no
// celular a barra de baixo tem as duas ferramentas e o botão dos Catálogos, que
// abre a folha com os onze. Um destino que sumisse de um dos dois caminhos só
// seria notado por quem estivesse naquela largura.
//
// Os endereços estão escritos à mão de propósito: eles são a promessa para quem
// usa, e montá-los do trilho do código provaria só que o trilho se repete.
var masterTools = []string{"encontros", "improviso"}

var masterCatalogs = []string{
	"bestiario", "classes", "condicoes", "deuses", "efeitos", "escolas",
	"itens", "magias", "pericias", "poderes", "racas",
}

// sliceBetween devolve o trecho de `body` que começa no marcador e termina no
// próximo `end`.
func sliceBetween(t *testing.T, body, start, end string) string {
	t.Helper()
	i := strings.Index(body, start)
	if i < 0 {
		t.Fatalf("a página não tem %q", start)
	}
	rest := body[i:]
	j := strings.Index(rest, end)
	if j < 0 {
		t.Fatalf("o trecho de %q não fecha em %q", start, end)
	}
	return rest[:j]
}

func TestTheMasterFrameReachesEveryTool(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	body := pedeNoMestre(t, s, gm, "GET", "/mestre/bestiario", "").Body.String()

	if !strings.Contains(body, "scene-ds") {
		t.Fatal("o Bestiário não veste o Tempestade")
	}
	menu := sliceBetween(t, body, `class="ds-master-menu"`, "</nav>")
	for _, slug := range append(append([]string{}, masterTools...), masterCatalogs...) {
		if !strings.Contains(menu, `href="/mestre/`+slug+`"`) {
			t.Errorf("o menu do desktop não leva a /mestre/%s", slug)
		}
	}
	current := sliceBetween(t, menu, `href="/mestre/bestiario"`, ">")
	if !strings.Contains(current, `aria-current="page"`) {
		t.Errorf("o menu do desktop não marca o Bestiário como a tela atual: %s", current)
	}
	if strings.Count(menu, `aria-current="page"`) != 1 {
		t.Errorf("o menu do desktop marca %d telas como atuais", strings.Count(menu, `aria-current="page"`))
	}

	bottom := sliceBetween(t, body, `class="ds-bottom-nav"`, "</nav>")
	for _, slug := range masterTools {
		if !strings.Contains(bottom, `href="/mestre/`+slug+`"`) {
			t.Errorf("a barra do celular não leva a /mestre/%s", slug)
		}
	}
	if !strings.Contains(bottom, `commandfor="master-catalogs"`) {
		t.Fatal("a barra do celular não abre a folha dos Catálogos")
	}
	sheet := sliceBetween(t, body, `id="master-catalogs"`, "</dialog>")
	for _, slug := range masterCatalogs {
		if !strings.Contains(sheet, `href="/mestre/`+slug+`"`) {
			t.Errorf("a folha dos Catálogos não leva a /mestre/%s", slug)
		}
	}
	for _, slug := range masterTools {
		if strings.Contains(sheet, `href="/mestre/`+slug+`"`) {
			t.Errorf("a folha dos Catálogos repete a ferramenta /mestre/%s, que já está na barra", slug)
		}
	}
}
