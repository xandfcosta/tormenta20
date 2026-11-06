package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"t20engine/domain/board"
)

// A TELA DA MESA (ALE-608): o tabuleiro que o mestre põe no display grande que
// a mesa inteira vê. Ela não tem ação nenhuma, mostra o que os JOGADORES veem,
// segue o tabuleiro e o enquadramento do mestre, e congela quando ele quer
// mexer escondido.

func (f sceneFixture) displayUrl() string { return f.tableUrl() + "/tela" }

// seedAmbush põe no tabuleiro aberto uma peça que a mesa vê e uma escondida.
func (f sceneFixture) seedAmbush(t *testing.T) {
	t.Helper()
	f.seedOpenBoard(t, "crypt")
	for _, tok := range []board.BoardToken{
		{Label: "Taverneiro", X: 1, Y: 1},
		{Label: "Ogro emboscado", X: 4, Y: 4, Hidden: true},
	} {
		if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab, tok); err != nil {
			t.Fatalf("pôr %q: %v", tok.Label, err)
		}
	}
}

func TestTheDisplayIsTheGmsAlone(t *testing.T) {
	f := newSceneFixture(t)
	for _, path := range []string{"", "/fluxo"} {
		if rec := f.requests(t, f.player, http.MethodGet, f.displayUrl()+path, ""); rec.Code != http.StatusForbidden {
			t.Errorf("o jogador abriu %s (%d)", f.displayUrl()+path, rec.Code)
		}
	}
	for _, path := range []string{"/enquadramento", "/congelar", "/soltar"} {
		if rec := f.requests(t, f.player, http.MethodPost, f.displayUrl()+path, `{}`); rec.Code != http.StatusForbidden {
			t.Errorf("o jogador postou em %s (%d)", path, rec.Code)
		}
	}
}

// O display mostra a cena COMO A MESA a vê — é o mesmo gargalo da lente —, e
// não tem ferramenta, barra nem verbo: ninguém joga por ele.
func TestTheDisplayDrawsTheBoardAsTheTableSeesItWithNoActions(t *testing.T) {
	f := newSceneFixture(t)
	f.seedAmbush(t)
	page := gmRequest(t, f, f.displayUrl()).Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	region := findNode(doc, "id", "table-display")
	if region == nil || findNode(region, "id", "board-scene") == nil {
		t.Fatal("a tela da mesa não desenha o mapa")
	}
	text := renderNode(t, region)
	if !strings.Contains(text, "Taverneiro") || !strings.Contains(text, "Taverna do Javali") {
		t.Error("a tela da mesa não mostra o que a mesa vê")
	}
	if strings.Contains(page, "Ogro emboscado") {
		t.Error("a tela da mesa mostrou a peça escondida")
	}
	for _, chrome := range []string{"Ferramentas do mapa", "Fechar a cortina", "Pôr no mapa", "Encerrar o tabuleiro", "table-tracker"} {
		if strings.Contains(page, chrome) {
			t.Errorf("a tela da mesa tem %q, e ela não tem ação nenhuma", chrome)
		}
	}
	if !strings.Contains(page, f.displayUrl()+"/fluxo") {
		t.Error("a tela da mesa não abre o fluxo dela: ela não se atualizaria")
	}
}

// Ela segue a ABA que o mestre está olhando, e não a de cada jogador.
func TestTheDisplayFollowsTheGmsTab(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	crypt := f.openSecond(t, "Cripta")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("trocar de aba deu %d", rec.Code)
	}
	region := displayRegion(t, f)
	if !strings.Contains(region, "Cripta") || strings.Contains(region, "Taverna do Javali") {
		t.Errorf("a tela da mesa não foi para a aba do mestre: %q", region)
	}
}

// O ENQUADRAMENTO do mestre chega à tela em QUADRADOS — o meio e o tamanho do
// recorte —, e não em pixels: o display tem outro tamanho, e quem encaixa o
// recorte nele é ele.
func TestTheDisplayFollowsTheGmsFraming(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	if rec := f.requests(t, f.gm, http.MethodPost, f.displayUrl()+"/enquadramento",
		`{"x":3.5,"y":-2,"width":20,"height":12}`); rec.Code >= 300 {
		t.Fatalf("mandar o enquadramento deu %d", rec.Code)
	}
	framing := displaySignals(t, f)
	for _, want := range []string{`"display_x":3.5`, `"display_y":-2`, `"display_width":20`, `"display_height":12`} {
		if !strings.Contains(framing, want) {
			t.Errorf("a tela da mesa não recebeu %s: %q", want, framing)
		}
	}
}

// CONGELADA, a tela fica com o tabuleiro do instante em que congelou: o
// mestre mexe escondido, e a mesa só vê quando ele soltar.
func TestAFrozenDisplayKeepsTheBoardOfTheMomentItFroze(t *testing.T) {
	f := newSceneFixture(t)
	f.seedAmbush(t)
	f.posts(t, f.gm, f.displayUrl()+"/enquadramento", `{"x":1,"y":1,"width":10,"height":8}`)
	f.posts(t, f.gm, f.displayUrl()+"/congelar", "")
	f.seedToken(t, "Goblin novo", 6, 6)
	f.posts(t, f.gm, f.displayUrl()+"/enquadramento", `{"x":40,"y":40,"width":10,"height":8}`)
	// Congelar de novo não tira outro retrato: o segundo clique veria o
	// goblin que o mestre acabou de pôr.
	f.posts(t, f.gm, f.displayUrl()+"/congelar", "")

	frozen := displayRegion(t, f)
	if !strings.Contains(frozen, "Taverneiro") {
		t.Fatal("a tela congelada perdeu o tabuleiro — a ausência abaixo não provaria nada")
	}
	if strings.Contains(frozen, "Goblin novo") {
		t.Error("a tela congelada mostrou a peça posta depois de congelar")
	}
	if !strings.Contains(displaySignals(t, f), `"display_x":1,`) {
		t.Error("a tela congelada seguiu o enquadramento novo do mestre")
	}

	f.posts(t, f.gm, f.displayUrl()+"/soltar", "")
	thawed := displayRegion(t, f)
	if !strings.Contains(thawed, "Goblin novo") || !strings.Contains(displaySignals(t, f), `"display_x":40`) {
		t.Error("solta, a tela não voltou ao tabuleiro de agora")
	}
}

// O INTERRUPTOR do mestre só aparece com uma tela da mesa aberta: sem
// display, "ao vivo · segue este tabuleiro" seria uma frase falsa sobre o
// mapa dele. E ele diz o estado, ao vivo ou congelada.
func TestTheGmSeesTheFreezeSwitchOnlyWithADisplayOpen(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	if strings.Contains(gmRequest(t, f, f.tableUrl()).Body.String(), "Congelar") {
		t.Error("o interruptor apareceu sem tela da mesa aberta")
	}

	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+f.displayUrl()+"/fluxo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.gm))
	// O corpo fica ABERTO até o fim do caso: fechá-lo é fechar a tela, e o
	// interruptor some junto.
	go func() {
		if resp, err := http.DefaultClient.Do(req); err == nil {
			<-ctx.Done()
			_ = resp.Body.Close()
		}
	}()

	page := ""
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if page = gmRequest(t, f, f.tableUrl()).Body.String(); strings.Contains(page, "Congelar") {
			break
		}
	}
	if !strings.Contains(page, "Tela da mesa ao vivo") || !strings.Contains(page, "Congelar") {
		t.Fatal("com a tela da mesa aberta, o mestre não vê o interruptor")
	}
	if !strings.Contains(page, f.displayUrl()+"/enquadramento") {
		t.Error("com a tela da mesa aberta, o tabuleiro do mestre não manda o enquadramento")
	}
	f.posts(t, f.gm, f.displayUrl()+"/congelar", "")
	if frozen := gmRequest(t, f, f.tableUrl()).Body.String(); !strings.Contains(frozen, "Tela da mesa congelada") || !strings.Contains(frozen, "Soltar") {
		t.Error("congelada, o interruptor não diz que congelou nem oferece soltar")
	}
	if strings.Contains(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), "Tela da mesa") {
		t.Error("o jogador soube da tela da mesa")
	}
}

// displaySignals é o recorte que a região da tela leva, já decodificado.
func displaySignals(t *testing.T, f sceneFixture) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(gmRequest(t, f, f.displayUrl()).Body.String()))
	if err != nil {
		t.Fatal(err)
	}
	region := findNode(doc, "id", "table-display")
	if region == nil {
		t.Fatal("a tela da mesa não tem a região dela")
	}
	return attr(region, "data-signals")
}

// displayRegion é o HTML da região da tela da mesa, como a página a desenha.
func displayRegion(t *testing.T, f sceneFixture) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(gmRequest(t, f, f.displayUrl()).Body.String()))
	if err != nil {
		t.Fatal(err)
	}
	region := findNode(doc, "id", "table-display")
	if region == nil {
		t.Fatal("a tela da mesa não tem a região dela")
	}
	return renderNode(t, region)
}
