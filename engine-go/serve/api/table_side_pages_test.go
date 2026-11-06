package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"t20engine/serve/web/table"
)

// AS PÁGINAS À PARTE DO MESTRE (ALE-608): a mesa enxuta fica com a fila, o
// tabuleiro e a linha do turno, e o que era a coluna das Consultas abre cada um
// numa aba do navegador — Notas, Grupo, Elenco e Bestiário.

// sidePages são as páginas que o cabeçalho abre, com o rótulo do botão e o
// pedaço do endereço depois da sessão.
var sidePages = []struct{ Label, Path string }{
	{"Notas", "notas"},
	{"Grupo", "grupo"},
	{"Elenco", "elenco"},
	{"Bestiário", "bestiario"},
}

func gmTableDoc(t *testing.T, f sceneFixture) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(gmRequest(t, f, f.tableUrl()).Body.String()))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestTheGmTableHasNoConsultsColumnAnymore(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	doc := gmTableDoc(t, f)
	nav := findNode(doc, "aria-label", "Superfícies do mestre")
	// O CONTROLE: a mesa do mestre chegou. Sem ele, "não há Consultas" seria
	// verdade numa página de erro.
	if nav == nil || !strings.Contains(textOf(nav), "Fila") || !strings.Contains(textOf(nav), "Tabuleiro") {
		t.Fatal("a mesa do mestre não tem a barra das superfícies com a Fila e o Tabuleiro")
	}
	if strings.Contains(textOf(nav), "Consultas") {
		t.Error("o celular ainda tem a superfície das Consultas")
	}
	if findNode(doc, "data-gm-col", "consults") != nil || findNode(doc, "aria-label", "Consultas do mestre") != nil {
		t.Error("a mesa ainda desenha a coluna das Consultas")
	}
	if findNode(doc, "data-gm-rail", "consults") != nil {
		t.Error("a mesa ainda tem o trilho de reabrir as Consultas")
	}
}

// O cabeçalho escreve cada página DUAS vezes — os botões soltos do desktop e o
// menu "Ferramentas" das telas estreitas —, e o ALVO tem nome: clicar de novo
// traz a aba que já está aberta em vez de abrir outra.
func TestTheGmHeaderOpensEverySidePageInATabOfItsOwn(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	doc := gmTableDoc(t, f)
	header := findNode(doc, "id", "table-header")
	if header == nil {
		t.Fatal("a mesa do mestre não tem o cabeçalho")
	}
	menu := findNode(header, "id", "gm-side-menu")
	if menu == nil {
		t.Fatal("o cabeçalho não tem o menu das telas estreitas")
	}
	if !strings.Contains(textOf(header), "Ferramentas") {
		t.Error("o menu das telas estreitas não diz Ferramentas")
	}
	for _, p := range sidePages {
		address := f.tableUrl() + "/" + p.Path
		links := findAll(header, "href", address)
		if len(links) != 2 {
			t.Errorf("o cabeçalho tem %d elos para %s (%s); são dois, o botão e o item do menu", len(links), p.Label, address)
			continue
		}
		for _, a := range links {
			if target := attr(a, "target"); target != fmt.Sprintf("t20-%s-%d", p.Path, f.sessionID) {
				t.Errorf("o elo de %s abre em %q: sem um nome por sessão, cada clique abre outra aba", p.Label, target)
			}
		}
		if findNode(menu, "href", address) == nil {
			t.Errorf("o menu Ferramentas não tem %s", p.Label)
		}
		if rec := gmRequest(t, f, address); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "scene-ds") {
			t.Errorf("a página de %s respondeu %d sem a cena", p.Label, rec.Code)
		}
	}
}

// As páginas são do MESTRE, e a trava é do servidor: o Grupo diz o PV de todo
// mundo, o Elenco e o Bestiário dizem os números de quem a mesa vai enfrentar.
func TestEverySidePageIsTheGmsAlone(t *testing.T) {
	f := newSceneFixture(t)
	for _, p := range sidePages {
		if rec := f.requests(t, f.player, http.MethodGet, f.tableUrl()+"/"+p.Path, ""); rec.Code != http.StatusForbidden {
			t.Errorf("o jogador abriu a página de %s (%d)", p.Label, rec.Code)
		}
	}
}

func TestTheGroupPageDrawsThePartyAndKeepsItLive(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	page := gmRequest(t, f, f.tableUrl()+"/grupo").Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	group := findNode(doc, "id", "table-group")
	if group == nil {
		t.Fatal("a página do Grupo não tem a região do grupo")
	}
	if findNode(group, "aria-label", "Abrir a ficha completa de Arcanista") == nil {
		t.Errorf("o Grupo não leva à ficha completa do Arcanista: %q", textOf(group))
	}
	minus := findNode(group, "aria-label", "Tirar 1 PV de Arcanista")
	if minus == nil || !strings.Contains(attr(minus, "data-on:click"), "?pagina=grupo") {
		t.Error("o − do Grupo não leva a marca da página: a resposta remendaria a mesa inteira numa página que não a tem")
	}
	if findNode(doc, "id", "table-tracker") != nil {
		t.Error("a página do Grupo desenhou a fila")
	}
	if !strings.Contains(page, f.tableUrl()+"/fluxo?pagina=grupo") {
		t.Error("a página do Grupo não abre o fluxo com a marca dela: ela não se atualizaria sozinha")
	}
}

func TestTheCastPageDrawsTheCastWithItsVerbs(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.posts(t, f.gm, f.tableUrl()+"/elenco/npc/do-verbete", `{"creature":"ogro","npc_name":"Capitão Varek"}`)
	page := gmRequest(t, f, f.tableUrl()+"/elenco").Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	cast := findNode(doc, "id", "table-npcs")
	if cast == nil || !strings.Contains(textOf(cast), "Capitão Varek") {
		t.Fatal("a página do Elenco não mostra o NPC guardado")
	}
	edit := findNode(cast, "aria-label", "Editar o bloco de Capitão Varek")
	if edit == nil || !strings.Contains(textOf(cast), "Pôr na fila") {
		t.Errorf("a linha do NPC não tem o lápis e o Pôr na fila: %q", textOf(cast))
	}
	if edit != nil && !strings.Contains(attr(edit, "data-on:click"), "?pagina=elenco") {
		t.Error("o lápis do Elenco não leva a marca da página")
	}
	if findNode(doc, "aria-label", "Novo no elenco") == nil {
		t.Error("o Elenco não tem o Novo")
	}
	if findNode(doc, "aria-label", "Bloco do NPC") == nil {
		t.Error("a página do Elenco não tem o editor do bloco")
	}
	if findNode(cast, "href", f.tableUrl()+"/bestiario") == nil {
		t.Error("o Do bestiário não leva à página do Bestiário")
	}
	if !strings.Contains(page, f.tableUrl()+"/fluxo?pagina=elenco") {
		t.Error("a página do Elenco não abre o fluxo com a marca dela")
	}
}

// O BESTIÁRIO é a mesma rota que o painel já remendava: o navegador que a
// abre recebe a PÁGINA, e o Datastar continua recebendo o remendo.
func TestTheBestiaryPageListsTheCreaturesAndSendsToTheQueue(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	page := gmRequest(t, f, f.tableUrl()+"/bestiario?criatura=ogro&abrir=1").Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	bestiary := findNode(doc, "id", "table-bestiary")
	if bestiary == nil || !strings.Contains(textOf(bestiary), "Ogro") {
		t.Fatal("a página do Bestiário não lista as criaturas")
	}
	// A criatura aberta pelo ENDEREÇO nasce com o ajuste do livro: sem a
	// semente, o PV do ajuste seria 0 e o Pôr na fila seria recusado.
	if root := findNode(doc, "class", "ds-gm-page"); root == nil || !strings.Contains(attr(root, "data-signals"), `"entry_hp":130`) {
		t.Error("a criatura aberta pelo endereço não semeou o PV do livro no ajuste")
	}
	if !strings.Contains(page, f.tableUrl()+"/bestiario/enviar?pagina=bestiario") {
		t.Error("o Pôr na fila não leva a marca da página")
	}
	sent := f.posts(t, f.gm, f.tableUrl()+"/bestiario/enviar?pagina=bestiario",
		`{"creature":"ogro","entry_hp":30,"entry_initiative":7,"entry_copies":2}`)
	if strings.Contains(sent, `id="table-tracker"`) {
		t.Error("o Pôr na fila remendou a fila numa página que não a tem")
	}
	if !strings.Contains(sent, "entry_sent") {
		t.Error("o Bestiário não diz que a criatura entrou na fila: com a fila noutra aba, o clique não teria resposta")
	}
	ogres := 0
	for _, e := range stateOf(t, f.s.sessions, f.sessionID).Initiative {
		if strings.HasPrefix(e.Label, "Ogro") && e.Label != "Ogro cansado" {
			ogres++
		}
	}
	if ogres != 2 {
		t.Errorf("o Pôr 2 na fila pôs %d ogros", ogres)
	}
}

// O FLUXO DE CADA PÁGINA manda só as regiões que ela tem, cada uma uma vez
// na página: as duas pontas são escritas em lugares diferentes, e discordar não
// dá erro em lugar nenhum — o remendo simplesmente não pousa.
func TestEachSidePageStreamSendsOnlyTheRegionsItHas(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	for _, page := range []string{"grupo", "elenco"} {
		doc := gmRequest(t, f, f.tableUrl()+"/"+page).Body.String()
		view, _, err := f.s.tableScene.LoadView(t.Context(), f.gm, f.campaignID, f.sessionID)
		if err != nil {
			t.Fatalf("montar a view: %v", err)
		}
		view.Page = table.SidePageFor(page)
		regions := table.TableRegions(view)
		// O CONTROLE: sem região nenhuma, o laço abaixo é verde sobre nada.
		if len(regions) == 0 {
			t.Fatalf("o fluxo da página %s não manda região nenhuma", page)
		}
		for _, r := range regions {
			if n := strings.Count(doc, `id="`+r.ID+`"`); n != 1 {
				t.Errorf("o fluxo da página %s manda %q, e a página a tem %d vezes", page, r.ID, n)
			}
		}
	}
}

// findAll devolve todo nó com o atributo igual ao valor.
func findAll(n *html.Node, key, value string) []*html.Node {
	var found []*html.Node
	if n.Type == html.ElementNode && attr(n, key) == value {
		found = append(found, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		found = append(found, findAll(c, key, value)...)
	}
	return found
}
