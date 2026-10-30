package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// A SESSÃO PARALELA (ALE-473), pelo mockup aprovado `sessao.html`: o jogador
// com personagem abre a sessão com `?tempestade=1` e recebe o sistema novo —
// o cabeçalho com a faixa, a barra da sessão embaixo e a Ficha embutida já na
// ficha paralela, com a fileira de seções. O fluxo leva a marca, para o stream
// desenhar o cabeçalho novo; e o mestre segue com o palco de hoje.
func TestTheParallelSessionDressesThePlayerSheetInTempestade(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl()+"?tempestade=1", "").Body.String()

	if !strings.Contains(page, "scene-ds") {
		t.Fatal("a sessão paralela não veste o Tempestade")
	}
	if !regexp.MustCompile(`@get\((&#39;|')[^'&]*/fluxo\?tempestade=1`).MatchString(page) {
		t.Error("o fluxo da sessão paralela não leva a marca: o stream desenharia o cabeçalho de hoje")
	}
	if strings.Contains(page, "O que ver na sessão") {
		t.Error("a sessão paralela ainda desenha o seletor de superfície de hoje")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	surfaces := findNode(doc, "aria-label", "Superfícies da sessão")
	if surfaces == nil {
		t.Fatal("a sessão paralela não tem a barra da sessão")
	}
	for _, s := range []string{"Ações", "Ficha", "Mesa", "Tabuleiro"} {
		if !strings.Contains(textOf(surfaces), s) {
			t.Errorf("a barra da sessão não tem %s", s)
		}
	}
	sheet := findNode(doc, "id", "sheet-scene")
	if sheet == nil || findNode(sheet, "data-system", "tempestade") == nil {
		t.Fatal("a Ficha embutida não é a ficha paralela")
	}
	sheetHTML := renderNode(t, sheet)
	if strings.Contains(sheetHTML, "ds-sheet-head") {
		t.Error("a Ficha embutida repete o cabeçalho da ficha solta, e a sessão já tem o dela")
	}
	sections := findNode(sheet, "aria-label", "Seções da ficha")
	if sections == nil || !regexp.MustCompile(`embutida=1[^"']*tempestade=1|tempestade=1[^"']*embutida=1`).MatchString(renderNode(t, sections)) {
		t.Error("as seções da Ficha embutida não trocam de aba com as duas marcas")
	}
}

// O MESTRE com a marca segue com o palco de hoje: esta fatia é a visão do
// jogador.
func TestTheGmKeepsTodaysStageInTheParallelSession(t *testing.T) {
	f := newSceneFixture(t)
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl()+"?tempestade=1", "").Body.String()
	if strings.Contains(page, "scene-ds") {
		t.Error("o mestre recebeu a sessão paralela")
	}
}

// O STREAM MANTÉM O CABEÇALHO NOVO depois do primeiro quadro. Ele recarrega a
// mesa a cada mudança, e a marca da sessão paralela tem de valer em todo
// quadro — e não só no primeiro, que é como o cabeçalho de hoje voltava dois
// segundos depois de abrir a página.
func TestTheParallelSessionStreamKeepsTheNewHeaderAfterATurn(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+f.tableUrl()+"/fluxo?tempestade=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.player))
	req.Header.Set("Datastar-Request", "true")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("abrir o fluxo: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	reader := bufio.NewReader(resp.Body)

	nextHeader := func(when string) string {
		for range 30 {
			if frame := lerQuadro(t, reader); strings.Contains(frame, `id="table-header"`) {
				return frame
			}
		}
		t.Fatalf("o fluxo não mandou o cabeçalho %s", when)
		return ""
	}
	if first := nextHeader("no primeiro quadro"); !strings.Contains(first, "ds-session-head") {
		t.Fatal("o primeiro quadro já veio com o cabeçalho de hoje")
	}

	f.posts(t, f.gm, f.tableUrl()+"/iniciativa/proxima-vez", "{}")
	if after := nextHeader("depois da vez"); !strings.Contains(after, "ds-session-head") {
		t.Error("depois de a vez andar, o fluxo mandou o cabeçalho de hoje para a sessão paralela")
	}
}

// AS AÇÕES DA SESSÃO PARALELA (ALE-473), pelo mockup aprovado: a barra de alvo
// fixa no topo, que carrega de quem é a vez (`my_turn`), e as ações desenhadas
// no kit. O que gasta a vez lê o alvo E a vez; a reação, não.
func TestTheParallelSessionDrawsTheActionsInTempestade(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl()+"?tempestade=1", "").Body.String()
	if strings.Contains(page, "Ações ainda está na sessão de hoje") {
		t.Fatal("as Ações da sessão paralela ainda levam à sessão de hoje")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	target := findNode(doc, "id", "table-turn-target")
	if target == nil || !strings.Contains(attr(target, "data-signals"), "my_turn") {
		t.Fatal("a barra de alvo não carrega de quem é a vez")
	}
	if findNode(target, "data-bind", "turn_target") == nil {
		t.Error("a barra de alvo não escolhe o alvo")
	}
	actions := findNode(doc, "id", "actions-scene")
	if actions == nil || findNode(actions, "class", "ds-list") == nil {
		t.Fatal("as Ações não estão desenhadas no kit")
	}
	if !regexp.MustCompile(`data-attr:disabled="[^"]*\$turn_target === (&#39;|')(&#39;|')[^"]*!\$my_turn|data-attr:disabled="[^"]*!\$my_turn[^"]*\$turn_target`).MatchString(renderNode(t, actions)) {
		t.Error("o ataque não espera o alvo E a vez")
	}
}

// O GESTO DA MESA NA SESSÃO PARALELA responde com as regiões NOVAS. Os
// comandos da mesa redesenham a mesa na hora, e sem a marca devolviam o
// cabeçalho de hoje — o mesmo defeito do stream, por outro caminho.
func TestAParallelSessionGestureAnswersWithTheNewRegions(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	answer := f.posts(t, f.player, f.tableUrl()+"/ataque/cancelar?tempestade=1", "{}")
	if !strings.Contains(answer, "ds-session-head") {
		t.Errorf("o gesto da sessão paralela respondeu com o cabeçalho de hoje:\n%s", firstRows(answer, 6))
	}
}

// A MESA DA SESSÃO PARALELA (ALE-473), pelo mockup aprovado `mesa.html`: em
// combate, a fila com quem joga agora e o PV dos outros oculto; o registro
// encolhido para quem já está na fila; e o grupo.
func TestTheParallelSessionDrawsTheTable(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl()+"?tempestade=1", "").Body.String()
	if strings.Contains(page, "Mesa ainda está na sessão de hoje") {
		t.Fatal("a Mesa da sessão paralela ainda leva à sessão de hoje")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	for id, wants := range map[string][]string{
		"table-tracker":  {"Iniciativa", "Ogro cansado", "PV ocultos pelo mestre", "Arcanista", "você"},
		"table-register": {"Você está na fila com 12"},
		"table-party":    {"Grupo", "Arcanista"},
	} {
		region := findNode(doc, "id", id)
		if region == nil {
			t.Errorf("a Mesa paralela não tem a região %s", id)
			continue
		}
		if findNode(region, "class", "ds-list") == nil && id != "table-register" {
			t.Errorf("a região %s não está desenhada no kit", id)
		}
		text := textOf(region)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("a região %s não mostra %q", id, want)
			}
		}
	}
}
