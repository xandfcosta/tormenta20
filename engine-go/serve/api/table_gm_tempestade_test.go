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

// O TABULEIRO DO MESTRE (ALE-522, fatia 2): o mapa de hoje com o que é do
// mestre a mais — as ferramentas de montar depois de um vinco, o Terreno que
// abre as quatro espécies, e os verbos da sessão numa barra só. Nada do
// grimório entra na região: os diálogos de abrir, encerrar, pôr no mapa e
// acervo vestem o kit.
func TestTheParallelGmTableDrawsTheBoard(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	closed := gmBoardRegion(t, f)
	if !strings.Contains(textOf(closed), "Nenhum tabuleiro aberto") || findNode(closed, "aria-label", "Abrir tabuleiro") == nil {
		t.Errorf("sem tabuleiro, a mesa do mestre não oferece abrir um: %q", textOf(closed))
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/abrir",
		`{"new_place":"Taverna do Javali","new_ground":"tavern"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir o tabuleiro deu %d", rec.Code)
	}
	board := gmBoardRegion(t, f)
	if findNode(board, "id", "board-scene") == nil {
		t.Fatal("a mesa do mestre não desenha o mapa")
	}
	tools := findNode(board, "aria-label", "Ferramentas do mapa")
	if tools == nil {
		t.Fatal("o tabuleiro do mestre não tem a barra das ferramentas")
	}
	for _, want := range []string{"Mover", "Vista", "Régua", "Gabarito", "Marcar", "Terreno", "Borracha", "Peça"} {
		if !strings.Contains(textOf(tools), want) {
			t.Errorf("a barra do mestre não escreve %q: %q", want, textOf(tools))
		}
	}
	for _, verb := range []string{"Fechar a cortina", "Ver a cena como a mesa vê", "Pôr no mapa", "Abrir o tabuleiro numa janela", "Encerrar o tabuleiro"} {
		if findNode(board, "aria-label", verb) == nil {
			t.Errorf("o tabuleiro do mestre não tem o verbo %q", verb)
		}
	}
	rendered := renderNode(t, board)
	for _, old := range []string{"scene-grimorio", "grimorio-iron", "grimorio-gold", "board-rail"} {
		if strings.Contains(rendered, old) {
			t.Errorf("a região do tabuleiro do mestre ainda leva %q, do grimório", old)
		}
	}
}

// A JANELA À PARTE desenha só o mapa: é para a segunda tela ou a TV da mesa.
func TestTheGmBoardOpensInAWindowOfItsOwn(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/abrir",
		`{"new_place":"Taverna do Javali","new_ground":"tavern"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir o tabuleiro deu %d", rec.Code)
	}
	page := gmRequest(t, f, f.tableUrl()+"?tempestade=1&janela=tabuleiro", nil).Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	if findNode(doc, "id", "board-scene") == nil {
		t.Fatal("a janela do tabuleiro não desenha o mapa")
	}
	if findNode(doc, "id", "table-tracker") != nil {
		t.Error("a janela do tabuleiro desenhou a fila, e ela é só o mapa")
	}
	if !strings.Contains(page, "/fluxo?janela=tabuleiro") {
		t.Error("o fluxo da janela não leva a marca da janela: ele remendaria regiões que não existem nela")
	}
}

// gmBoardRegion é a região do tabuleiro na mesa nova do mestre.
func gmBoardRegion(t *testing.T, f sceneFixture) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(gmRequest(t, f, f.tableUrl()+"?tempestade=1", nil).Body.String()))
	if err != nil {
		t.Fatal(err)
	}
	region := findNode(doc, "id", "table-board")
	if region == nil {
		t.Fatal("a mesa do mestre não tem a região do tabuleiro")
	}
	return region
}

// AS CONSULTAS DO MESTRE (ALE-522, fatia 3): quatro abas no lugar do aviso — as
// notas, o grupo com a ficha completa, o elenco com o Novo e o bestiário. As
// rotas são as de hoje; muda onde cada coisa mora.
func TestTheParallelGmTableDrawsTheConsults(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.posts(t, f.gm, f.tableUrl()+"/elenco/npc/do-verbete", `{"creature":"ogro","npc_name":"Capitão Varek"}`)
	page := gmRequest(t, f, f.tableUrl()+"?tempestade=1", nil).Body.String()
	if strings.Contains(page, "As Consultas ainda estão no palco de hoje") {
		t.Fatal("as Consultas ainda levam ao palco de hoje")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	tabs := findNode(doc, "aria-label", "Consultas do mestre")
	if tabs == nil {
		t.Fatal("as Consultas não têm as abas")
	}
	for _, want := range []string{"Notas", "Grupo", "Elenco", "Bestiário"} {
		if !strings.Contains(textOf(tabs), want) {
			t.Errorf("as abas das Consultas não têm %s: %q", want, textOf(tabs))
		}
	}
	if findNode(doc, "aria-label", "Notas da sessão") == nil {
		t.Error("a aba Notas não tem o caderno")
	}
	if findNode(doc, "aria-label", "Abrir a ficha completa de Arcanista") == nil {
		t.Error("o Grupo não leva à ficha completa do Arcanista")
	}
	cast := findNode(doc, "id", "table-npcs")
	if cast == nil || !strings.Contains(textOf(cast), "Capitão Varek") {
		t.Fatal("o Elenco não mostra o NPC guardado")
	}
	if findNode(cast, "aria-label", "Editar o bloco de Capitão Varek") == nil || !strings.Contains(textOf(cast), "Pôr na fila") {
		t.Errorf("a linha do NPC não tem o lápis e o Pôr na fila: %q", textOf(cast))
	}
	if findNode(doc, "aria-label", "Novo no elenco") == nil {
		t.Error("o Elenco não tem o Novo")
	}
	bestiary := findNode(doc, "id", "table-bestiary")
	if bestiary == nil || !strings.Contains(textOf(bestiary), "Ogro") {
		t.Fatal("a aba Bestiário não lista as criaturas")
	}
	if strings.Contains(renderNode(t, bestiary), "scene-grimorio") {
		t.Error("o bestiário do mestre no Tempestade ainda veste o grimório")
	}
}

// A CRIATURA ABERTA no bestiário da mesa nova volta no kit, com o ajuste e os
// dois verbos: o remendo tem de ser o desenho da aba, e não o diálogo de hoje.
func TestTheGmBestiaryOpensTheCreatureInTheKit(t *testing.T) {
	f := newSceneFixture(t)
	mark := &http.Cookie{Name: table.GmTableCookie, Value: "1"}
	body := gmRequest(t, f, f.tableUrl()+"/bestiario?criatura=ogro&abrir=1", mark).Body.String()
	if strings.Contains(body, "scene-grimorio") {
		t.Error("com a mesa nova, o bestiário remendou o diálogo do grimório")
	}
	for _, want := range []string{"Guardar no elenco", "na fila"} {
		if !strings.Contains(body, want) {
			t.Errorf("a criatura aberta não tem %q", want)
		}
	}
	if old := gmRequest(t, f, f.tableUrl()+"/bestiario?criatura=ogro&abrir=1", nil).Body.String(); !strings.Contains(old, "scene-grimorio") {
		t.Error("sem a marca, o bestiário deixou de remendar o diálogo de hoje")
	}
}
