package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"t20engine/serve/web/table"

	"golang.org/x/net/html"
)

// A MESA DO MESTRE EM PARALELO (ALE-522): com `?tempestade=1` o mestre recebe
// a mesa nova, e a marca vira um COOKIE — o stream e os comandos a leem sem que
// cada endereço precise carregá-la. `?tempestade=0` volta ao palco de hoje.
func TestTheGmOpensTheParallelTableAndTheMarkSticks(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	first := gmRequest(t, f, f.tableUrl()+"?tempestade=1", nil)
	if !strings.Contains(first.Body.String(), "scene-ds") {
		t.Fatal("com a marca, o mestre não recebeu a mesa no Tempestade")
	}
	var mark *http.Cookie
	for _, c := range first.Result().Cookies() {
		if c.Name == table.GmTableCookie {
			mark = c
		}
	}
	if mark == nil || mark.Value != "1" {
		t.Fatalf("a marca não virou cookie: %+v", first.Result().Cookies())
	}
	if again := gmRequest(t, f, f.tableUrl(), mark).Body.String(); !strings.Contains(again, "scene-ds") {
		t.Error("com o cookie e sem a marca no endereço, o mestre voltou ao palco de hoje")
	}
	if off := gmRequest(t, f, f.tableUrl()+"?tempestade=0", mark).Body.String(); strings.Contains(off, "scene-ds") {
		t.Error("`?tempestade=0` não devolveu o palco de hoje")
	}
	if plain := gmRequest(t, f, f.tableUrl(), nil).Body.String(); strings.Contains(plain, "scene-ds") {
		t.Error("sem marca nem cookie, o mestre recebeu a mesa nova")
	}
}

// A FILA DO MESTRE, pelo mockup aprovado `mestre.html`: as três superfícies na
// barra, cada linha com o − e o + de um PV e a folha do combatente, e a linha
// do turno com o avanço.
func TestTheParallelGmTableDrawsTheQueue(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	page := gmRequest(t, f, f.tableUrl()+"?tempestade=1", nil).Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	nav := findNode(doc, "aria-label", "Superfícies do mestre")
	if nav == nil {
		t.Fatal("a mesa do mestre não tem a barra das superfícies")
	}
	for _, s := range []string{"Fila", "Tabuleiro", "Consultas"} {
		if !strings.Contains(textOf(nav), s) {
			t.Errorf("a barra do mestre não tem %s", s)
		}
	}
	queue := findNode(doc, "id", "table-tracker")
	if queue == nil {
		t.Fatal("a mesa do mestre não desenhou a fila")
	}
	text := textOf(queue)
	for _, want := range []string{"Ogro cansado", "Arcanista", "PV 12/130"} {
		if !strings.Contains(text, want) {
			t.Errorf("a fila do mestre não mostra %q: %q", want, text)
		}
	}
	rendered := renderNode(t, queue)
	for _, want := range []string{`aria-label="Tirar 1 PV de Ogro cansado"`, `aria-label="Devolver 1 PV a Ogro cansado"`, `aria-label="Abrir Ogro cansado"`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("a linha do ogro não tem %s", want)
		}
	}
	if findNode(doc, "aria-label", "Adicionar à fila") == nil {
		t.Error("a fila não tem o Adicionar")
	}
	turn := findNode(doc, "id", "table-commands")
	// Na rodada zero o avanço COMEÇA a fila, e o botão diz por quem.
	if turn == nil || !strings.Contains(textOf(turn), "Começar: Ogro cansado") {
		t.Errorf("a linha do turno não tem o avanço: %q", textOf(turn))
	}
}

// O DANO DE QUALQUER VALOR: a folha do combatente escreve "Dano N", e a rota
// do passo, que só aceitava 1 e 5, passa a aceitar o número digitado.
func TestTheGmWoundsARowByAnyAmount(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	var ogre string
	for _, e := range stateOf(t, f.s.sessions, f.sessionID).Initiative {
		if e.Label == "Ogro cansado" {
			ogre = e.ID
		}
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+ogre+"/vitais/hp/ferir/7", ""); rec.Code != http.StatusOK {
		t.Fatalf("ferir 7 deu %d", rec.Code)
	}
	for _, e := range stateOf(t, f.s.sessions, f.sessionID).Initiative {
		if e.ID == ogre && e.HpCurrent != nil && *e.HpCurrent != 5 {
			t.Errorf("o ogro tinha 12 PV e levou 7, e ficou com %d", *e.HpCurrent)
		}
	}
}

// gmRequest é o pedido do mestre, com o cookie da mesa nova quando há um.
func gmRequest(t *testing.T, f sceneFixture, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	return rec
}
