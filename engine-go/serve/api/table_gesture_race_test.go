package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"t20engine/app/boards"
	"t20engine/domain/board"
	"t20engine/domain/live"
)

// DOIS GESTOS AO MESMO TEMPO decidem sobre o tabuleiro que cada um leu.
//
// O gesto que lê o tabuleiro, decide (a próxima letra, o oposto do que está lá, a parada a tirar) e
// SÓ DEPOIS pede a mutação deixa a decisão fora da trava do store: dois cliques quase juntos leem a
// mesma versão e gravam a mesma decisão. A decisão mora dentro da mutação, e estes casos a prendem.
//
// A INTERCALAÇÃO É FORÇADA, e não sorteada: o `readsMeet` segura a primeira leitura do tabuleiro feita
// por um dos dois gestos até a leitura do outro chegar. Só contam as leituras dos dois pedidos
// (`racingGesture` no contexto): um redesenho assíncrono da semeadura consumia o encontro, e o caso
// passava verde sobre uma corrida que existia. Com a decisão fora da trava, os dois gestos leem a mesma versão sempre. Com ela
// dentro, a segunda leitura espera a trava — a primeira desiste de esperar e segue, e o veredito não
// depende do relógio, só a duração do teste.

// racingGesture marca, no contexto do pedido, que a leitura vem de um dos dois gestos.
type racingGesture struct{}

// readsMeet é o retrato do tabuleiro que faz a primeira leitura de CADA gesto se encontrar: a
// primeira volta só quando a segunda também já voltou.
type readsMeet struct {
	boards.BoardSnapshots
	mu      sync.Mutex
	armed   bool
	waiting chan struct{}
}

// meetingWindow é quanto a primeira leitura espera a segunda. Só pesa no caminho certo, em que a
// segunda nunca chega a tempo porque está na fila da trava.
const meetingWindow = 300 * time.Millisecond

func (r *readsMeet) arm() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.armed = true
}

func (r *readsMeet) Read(ctx context.Context, sessionID int64) ([]*board.BoardState, error) {
	// LER ANTES DE ESPERAR: o encontro garante que as duas leituras já VOLTARAM antes de qualquer
	// gravação. Esperando antes de ler, um gesto acordava depois de o outro gravar e lia a versão nova.
	open, err := r.BoardSnapshots.Read(ctx, sessionID)
	r.mu.Lock()
	switch {
	case !r.armed || ctx.Value(racingGesture{}) == nil:
		r.mu.Unlock()
	case r.waiting == nil:
		ch := make(chan struct{})
		r.waiting = ch
		r.mu.Unlock()
		select {
		case <-ch:
		case <-time.After(meetingWindow):
		}
		r.mu.Lock()
		r.armed = false
		r.mu.Unlock()
	default:
		close(r.waiting)
		r.armed = false
		r.mu.Unlock()
	}
	return open, err
}

// withMeetingReads troca o store dos tabuleiros por um que lê pelo `readsMeet`. A cena da Mesa é
// montada uma vez e guarda o store que recebeu, então o `primeCatalogs` a remonta sobre o novo.
func withMeetingReads(f sceneFixture) *readsMeet {
	meet := &readsMeet{BoardSnapshots: boards.NewSnapshots(f.s.queries)}
	f.s.boards = boards.NewStore(meet, f.s.queries, live.NewUUID, f.s.bus)
	f.s.primeCatalogs(f.s.catalogs)
	return meet
}

// twiceAtOnce manda o MESMO gesto duas vezes ao mesmo tempo, e devolve os dois códigos.
//
// O token sai ANTES das goroutines: o `t.Fatalf` do `f.token` não pode rodar fora da goroutine do
// teste.
func twiceAtOnce(t *testing.T, f sceneFixture, userID int64, path, body string) [2]int {
	t.Helper()
	bearer := "Bearer " + f.token(t, userID)
	var codes [2]int
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			req = req.WithContext(context.WithValue(req.Context(), racingGesture{}, true))
			req.Header.Set("Authorization", bearer)
			if body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			f.s.WebRouter().ServeHTTP(rec, req)
			codes[i] = rec.Code
		}()
	}
	wg.Wait()
	return codes
}

func TestTwoMarksAtOnceGetDifferentLetters(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	withMeetingReads(f).arm()

	codes := twiceAtOnce(t, f, f.gm, f.tableUrl()+"/tabuleiro/marcadores/novo", `{"from":{"X":2,"Y":3}}`)

	if codes != [2]int{http.StatusOK, http.StatusOK} {
		t.Fatalf("marcar deu %v", codes)
	}
	markers := mapMarkers(t, f)
	if len(markers) != 2 {
		t.Fatalf("o mapa ficou com %d marcadores, esperado 2", len(markers))
	}
	if markers[0].Text == markers[1].Text {
		t.Errorf("dois cliques juntos deram dois %q: a letra foi decidida fora da trava", markers[0].Text)
	}
}

func TestTwoRevealsAtOnceHideTheMarkerAgain(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	base := f.tableUrl() + "/tabuleiro/marcadores"
	if rec := f.requests(t, f.gm, http.MethodPost, base+"/novo", `{"from":{"X":1,"Y":1}}`); rec.Code != http.StatusOK {
		t.Fatalf("marcar deu %d", rec.Code)
	}
	id := mapMarkers(t, f)[0].ID
	withMeetingReads(f).arm()

	codes := twiceAtOnce(t, f, f.gm, base+"/"+id+"/revelar", "")

	if codes != [2]int{http.StatusOK, http.StatusOK} {
		t.Fatalf("revelar deu %v", codes)
	}
	if !mapMarkers(t, f)[0].Hidden {
		t.Error("dois toques juntos deixaram o marcador À MOSTRA: os dois gravaram o oposto da mesma leitura")
	}
}

func TestTwoHideTogglesAtOnceShowTheTokenAgain(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	id := mapToken(t, f, "Ogro", 4, 4)
	withMeetingReads(f).arm()

	codes := twiceAtOnce(t, f, f.gm, f.tableUrl()+"/tabuleiro/pecas/"+id+"/visibilidade", "")

	if codes != [2]int{http.StatusOK, http.StatusOK} {
		t.Fatalf("esconder deu %v", codes)
	}
	if board.FindToken(nowBoard(t, f), id).Hidden {
		t.Error("dois toques juntos deixaram a peça ESCONDIDA: os dois gravaram o oposto da mesma leitura")
	}
}

func TestTwoMovingTogglesAtOnceLeaveTheObjectStill(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "crypt")
	placed, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab, board.BoardToken{
		Label: "Carroça", X: 3, Y: 3, Kind: "object", Size: "Grande", Material: "Madeira", HpMax: 20, HpCurrent: 20,
	})
	if err != nil {
		t.Fatalf("pôr a carroça: %v", err)
	}
	id := placed.Tokens[len(placed.Tokens)-1].ID
	withMeetingReads(f).arm()

	codes := twiceAtOnce(t, f, f.gm, f.tableUrl()+"/tabuleiro/pecas/"+id+"/movimento", "")

	if codes != [2]int{http.StatusOK, http.StatusOK} {
		t.Fatalf("pôr em movimento deu %v", codes)
	}
	if board.FindToken(nowBoard(t, f), id).Moving {
		t.Error("dois toques juntos deixaram a carroça EM MOVIMENTO: os dois gravaram o oposto da mesma leitura")
	}
}

func TestTwoUndosAtOnceTakeTwoLegs(t *testing.T) {
	f := newSceneFixture(t)
	tokenID := f.onBoard(t)
	base := f.tableUrl() + "/tabuleiro/" + tokenID
	for _, square := range []string{`{"from":{"X":2,"Y":0}}`, `{"from":{"X":2,"Y":2}}`, `{"from":{"X":4,"Y":2}}`} {
		if rec := f.requests(t, f.gm, http.MethodPost, base+"/parada", square); rec.Code != http.StatusOK {
			t.Fatalf("a parada %s deu %d", square, rec.Code)
		}
	}
	if stops := boardStops(t, f); len(stops) != 4 {
		t.Fatalf("as três pernas deram %d paradas — sem o caso positivo o desfazer não mede nada", len(stops))
	}
	withMeetingReads(f).arm()

	codes := twiceAtOnce(t, f, f.gm, base+"/desfazer-parada", "")

	if codes != [2]int{http.StatusOK, http.StatusOK} {
		t.Fatalf("desfazer deu %v", codes)
	}
	if stops := boardStops(t, f); len(stops) != 2 {
		t.Errorf("dois desfazer juntos deixaram %d paradas, esperado 2 (a origem e a primeira)", len(stops))
	}
}
