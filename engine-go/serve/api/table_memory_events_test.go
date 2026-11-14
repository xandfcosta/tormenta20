package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"t20engine/domain/board"
	"t20engine/domain/live"
	"testing"
	"time"
)

// O QUE MUDA SÓ EM MEMÓRIA CHEGA PELO EVENTO, e não pelo relógio.
//
// O fluxo da Mesa só redesenha no evento (CQRS puro): o que muda sem gravar —
// quem está com a Mesa aberta, a tela da mesa, o Grupo que cresce com a Mesa
// aberta — precisa publicar o seu, senão a tela dos outros fica parada sem erro.
// Cada caso abre o fluxo por HTTP de verdade e exige o quadro em 400ms: menos da
// metade do tique de 1s que existia, então um verde aqui nunca foi o relógio.

// eventReach é o prazo de cada caso: folgado para um round-trip local.
const eventReach = 400 * time.Millisecond

// liveStream é um fluxo aberto por HTTP, lido em quadros.
type liveStream struct {
	frames chan string
	cancel context.CancelFunc
}

// liveServer sobe o roteador num servidor de verdade e o fecha no t.Cleanup,
// DEPOIS dos fluxos: num `defer`, o `Close` esperaria pelos fluxos ainda abertos
// para sempre, e o caso travaria em vez de reprovar.
func liveServer(t *testing.T, f sceneFixture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(f.s.WebRouter())
	t.Cleanup(srv.Close)
	return srv
}

// opensStream abre o fluxo `path` como `userID` e começa a ler os quadros.
func opensStream(t *testing.T, srv *httptest.Server, f sceneFixture, userID int64, path string) *liveStream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("montar o pedido de %s: %v", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token(t, userID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("abrir o fluxo %s: %v", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("o fluxo %s respondeu %d", path, resp.StatusCode)
	}
	stream := &liveStream{frames: make(chan string, 64), cancel: cancel}
	go func() {
		defer close(stream.frames)
		defer func() { _ = resp.Body.Close() }()
		reader := bufio.NewScanner(resp.Body)
		reader.Buffer(make([]byte, 0, 64*1024), 4<<20)
		var frame strings.Builder
		for reader.Scan() {
			if row := reader.Text(); row != "" {
				frame.WriteString(row)
				frame.WriteByte('\n')
				continue
			}
			select {
			case stream.frames <- frame.String():
			case <-ctx.Done():
				return
			}
			frame.Reset()
		}
	}()
	return stream
}

// waitsFor lê quadros até um satisfazer `wanted`, ou reprova com `what` no prazo.
//
// PROCURA entre os quadros, porque a carga manda um por região: esperar no
// primeiro afirmaria a ordem do render, que não é promessa.
func (s *liveStream) waitsFor(t *testing.T, within time.Duration, what string, wanted func(frame string) bool) {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case frame, open := <-s.frames:
			if !open {
				t.Fatalf("o fluxo fechou antes de: %s", what)
			}
			if wanted(frame) {
				return
			}
		case <-deadline:
			t.Fatalf("em %v: %s", within, what)
		}
	}
}

func containing(text string) func(string) bool {
	return func(frame string) bool { return strings.Contains(frame, text) }
}

// ABRIR E FECHAR A MESA ACENDE E APAGA O CARTÃO no Grupo do mestre: a presença
// mora só em memória, e quem entra e sai é o próprio fluxo.
func TestOpeningAndClosingTheTableReachesTheGroupPage(t *testing.T) {
	f := newSceneFixture(t)
	srv := liveServer(t, f)
	group := opensStream(t, srv, f, f.gm, f.tableUrl()+"/fluxo?pagina=grupo")
	// O CONTROLE: o Grupo nasce com o jogador fora. Sem ele, "na mesa" depois
	// não distinguiria a chegada de uma frase que já estava lá.
	group.waitsFor(t, 3*time.Second, "o Grupo do mestre não desenhou o jogador fora da mesa", containing("fora da mesa"))

	player := opensStream(t, srv, f, f.player, f.tableUrl()+"/fluxo")
	group.waitsFor(t, eventReach, "o jogador abriu a Mesa e o cartão dele não acendeu no Grupo do mestre",
		func(frame string) bool {
			return strings.Contains(frame, "na mesa") && !strings.Contains(frame, "fora da mesa")
		})

	player.cancel()
	group.waitsFor(t, eventReach, "o jogador fechou a Mesa e o cartão dele continuou aceso", containing("fora da mesa"))
}

// ABRIR E FECHAR A TELA DA MESA ACENDE E APAGA o "ao vivo" no tabuleiro do mestre.
func TestOpeningAndClosingTheDisplayReachesTheGmBoard(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	srv := liveServer(t, f)
	table := opensStream(t, srv, f, f.gm, f.tableUrl()+"/fluxo")
	table.waitsFor(t, 3*time.Second, "o tabuleiro do mestre não desenhou", containing("table-board"))

	display := opensStream(t, srv, f, f.gm, f.displayUrl()+"/fluxo")
	table.waitsFor(t, eventReach, "a tela da mesa abriu e o tabuleiro do mestre não acendeu o ao vivo",
		containing("Tela da mesa ao vivo"))

	display.cancel()
	table.waitsFor(t, eventReach, "a tela da mesa fechou e o ao vivo continuou aceso", func(frame string) bool {
		return strings.Contains(frame, `id="table-board"`) && !strings.Contains(frame, "Tela da mesa ao vivo")
	})
}

// O ENQUADRAMENTO do mestre chega à tela da mesa.
func TestTheGmFramingReachesTheDisplay(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	srv := liveServer(t, f)
	display := opensStream(t, srv, f, f.gm, f.displayUrl()+"/fluxo")
	display.waitsFor(t, 3*time.Second, "a tela da mesa não desenhou", containing("table-display"))

	f.posts(t, f.gm, f.displayUrl()+"/enquadramento", `{"x":3,"y":4,"width":17,"height":9}`)
	// O SINAL do recorte, como o templ o escreve no atributo: `"` vira `&#34;`.
	display.waitsFor(t, eventReach, "o enquadramento do mestre não chegou à tela da mesa",
		containing("display_width&#34;:17"))
}

// O HERÓI QUE SENTA COM A MESA ABERTA aparece no Grupo de quem já está nela, e a
// ficha dele passa a ser ouvida: o fluxo assinou as fichas do Grupo de quando
// abriu, e o herói novo não estava lá.
func TestAHeroSeatedWithTheTableOpenIsHeardByIt(t *testing.T) {
	f := newSceneFixture(t)
	srv := liveServer(t, f)
	table := opensStream(t, srv, f, f.player, f.tableUrl()+"/fluxo")
	table.waitsFor(t, 3*time.Second, "a Mesa do jogador não desenhou o Grupo", containing("Arcanista"))

	newcomer := seedUser(t, f.s, "recem-chegado@t20.local")
	hero := seedCharacterAtLevel(t, f.s, newcomer, "Brisa Recém-Chegada", "Ladino", 3, 0, 0)
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE campaigns SET inviteToken = ? WHERE id = ?", "convite-aberto", f.campaignID); err != nil {
		t.Fatalf("abrir o convite: %v", err)
	}
	if err := f.s.campaignSeating().Seat(context.Background(), newcomer, f.campaignID, hero, "convite-aberto"); err != nil {
		t.Fatalf("sentar o herói novo: %v", err)
	}
	table.waitsFor(t, eventReach, "o herói sentou com a Mesa aberta e o Grupo não o mostrou",
		containing("Brisa Recém-Chegada"))

	copyID := memberCopyOf(t, f, "Brisa Recém-Chegada")
	if rec := f.requests(t, newcomer, http.MethodPost, sheetVitalPath(copyID, -3), "{}"); rec.Code != http.StatusOK {
		t.Fatalf("ferir a cópia deu %d", rec.Code)
	}
	table.waitsFor(t, eventReach, "a ficha do herói novo mudou e a Mesa, que não a assinava, não soube",
		containing(heroHPAfterHurt(t, f, copyID)))
}

// memberCopyOf é o id da CÓPIA que o convite fez do herói nesta campanha.
func memberCopyOf(t *testing.T, f sceneFixture, name string) int64 {
	t.Helper()
	var id int64
	if err := f.s.db.QueryRowContext(context.Background(), `SELECT c.id FROM characters c
		JOIN campaign_members m ON m.characterId = c.id WHERE m.campaignId = ? AND c.name = ?`,
		f.campaignID, name).Scan(&id); err != nil {
		t.Fatalf("achar a cópia de %q na campanha %d: %v", name, f.campaignID, err)
	}
	return id
}

func sheetVitalPath(characterID int64, delta int) string {
	return fmt.Sprintf("/personagens/%d/vitais/pv/%d", characterID, delta)
}

// heroHPAfterHurt é a linha de PV que o Grupo tem de desenhar para a ficha, lida
// do banco DEPOIS do gesto: o caso pergunta se a tela soube, e não a conta.
func heroHPAfterHurt(t *testing.T, f sceneFixture, characterID int64) string {
	t.Helper()
	pools := poolsOf(t, f.s, characterID)
	if pools.HpCurrent == pools.HpMax {
		t.Fatalf("o gesto não feriu a ficha %d: PV %d/%d", characterID, pools.HpCurrent, pools.HpMax)
	}
	return fmt.Sprintf("PV %d/%d", pools.HpCurrent, pools.HpMax)
}

// A FICHA DE QUEM ENTRA NA FILA com a tela da mesa aberta passa a ser ouvida por ela: a barra de PV da
// peça vem da ficha, e a tela abriu antes de o combatente existir.
func TestTheDisplayHearsTheSheetOfWhoJoinsTheQueue(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	srv := liveServer(t, f)
	display := opensStream(t, srv, f, f.gm, f.displayUrl()+"/fluxo")
	display.waitsFor(t, 3*time.Second, "a tela da mesa não desenhou", containing("table-display"))

	// EM CENA: fora dela a fila não chega ao papel de jogador, que é o da tela.
	if _, err := f.s.sessions.StartScene(context.Background(), f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("iniciar a cena: %v", err)
	}
	entryID := f.tracker(t)
	if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: "Arcanista", X: 2, Y: 2, EntryID: &entryID, CharacterID: &f.charID}); err != nil {
		t.Fatalf("pôr a peça do Arcanista: %v", err)
	}
	// O CONTROLE: a peça chegou com a barra. Sem ele, "a barra mudou" não
	// distinguiria a ficha ouvida de uma barra que nunca foi desenhada.
	bar := regexp.MustCompile(`board-vitals-full[^>]*width:(\d+)%`)
	var before string
	display.waitsFor(t, eventReach, "a peça do Arcanista não chegou à tela com a barra de PV", func(frame string) bool {
		if found := bar.FindStringSubmatch(frame); found != nil {
			before = found[1]
			return true
		}
		return false
	})

	if rec := f.requests(t, f.player, http.MethodPost, sheetVitalPath(f.charID, -5), "{}"); rec.Code != http.StatusOK {
		t.Fatalf("ferir o Arcanista pela ficha deu %d", rec.Code)
	}
	display.waitsFor(t, eventReach, "a ficha do Arcanista mudou e a tela da mesa não soube", func(frame string) bool {
		found := bar.FindStringSubmatch(frame)
		return found != nil && found[1] != before
	})
}
