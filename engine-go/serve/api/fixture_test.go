package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"t20engine/app/session"
	"t20engine/domain/live"
	"testing"

	"t20engine/infra/db/sqlcgen"
)

// A BANCADA HTTP do `api`, lida por dezenas de arquivos de teste.
//
// Ela fica no pacote e não num `testdb` irmão porque o tipo `api.Server` a
// prende aqui: um pacote de bancada que o importasse seria importado de volta
// pelos testes dele, que é o ciclo que a divisão existe para evitar.

type sceneFixture struct {
	s          *Server
	gm         int64
	player     int64
	campaignID int64
	sessionID  int64
	charID     int64
}

// A mesa do app: mestre, um jogador com PC de nível 8 COM a perícia Iniciativa
// (sem ela o bônus cai em zero e o teste do d20 nasce vácuo), e um NPC para o
// mestre esconder.
func newSceneFixture(t *testing.T) sceneFixture {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)
	charID := seedCharacterAtLevel(t, s, player, "Arcanista", "Arcanista", 8, 10, 5)
	seedMember(t, s, campaignID, charID)
	if _, err := s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: charID, Name: "Iniciativa", Attribute: "dexterity", Trained: 0, Custom: 0,
	}); err != nil {
		t.Fatalf("semear perícia: %v", err)
	}
	return sceneFixture{s: s, gm: gm, player: player, campaignID: campaignID, sessionID: sessionID, charID: charID}
}

// token assina um JWT do usuário — o mesmo caminho do `authed` da casa.
func (f sceneFixture) token(t *testing.T, userID int64) string {
	t.Helper()
	user, err := f.s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", userID, err)
	}
	tok, err := f.s.accountGate().SignSession(user)
	if err != nil {
		t.Fatalf("assinar token: %v", err)
	}
	return tok
}

// pede manda uma requisição autenticada pelo MesaRouter — que é outro roteador
// que o `Router()` da API, e por isso o `authed` da casa não serve.
func (f sceneFixture) pede(t *testing.T, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.token(t, userID))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	return rec
}

func (f sceneFixture) tableUrl() string {
	return "/campanhas/" + strconv.FormatInt(f.campaignID, 10) +
		"/sessoes/" + strconv.FormatInt(f.sessionID, 10)
}

// posta manda a escrita por um servidor HTTP DE VERDADE, e não pelo par
// `httptest.NewRequest` + recorder.
//
// Isso não é preciosismo: o SDK do Datastar fecha o corpo do pedido ao criar o
// gerador SSE, então `ReadSignals` depois do `NewSSE` falha com "body already
// closed" — e o par de teste NÃO reproduz esse ciclo de vida. A ordem trocada
// passou verde na suíte inteira e quebrou toda escrita no servidor real; o
// defeito apareceu com um curl, não com um teste. Este helper existe para que
// não apareça assim de novo.
func (f sceneFixture) posta(t *testing.T, userID int64, path, body string) string {
	t.Helper()
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("montar pedido: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token(t, userID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("postar: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	read, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ler resposta: %v", err)
	}
	return string(read)
}

// cena põe a sessão em cena com um ogro de PV OCULTOS e o PC do jogador.
func (f sceneFixture) scene(t *testing.T) {
	t.Helper()
	hidden := true
	pv, pvMax := int64(12), int64(130)
	if _, err := f.s.sessions.StartScene(f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		Label: "Ogro cansado", Initiative: 19, Type: "npc",
		HpHidden: &hidden, HpCurrent: &pv, HpMax: &pvMax,
	}); err != nil {
		t.Fatalf("semear ogro: %v", err)
	}
	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &f.charID,
	}); err != nil {
		t.Fatalf("semear PC: %v", err)
	}
}

// seedClasse põe uma classe na ficha, e ela é a bancada de SEIS arquivos — por
// isso mora aqui. Ajudante compartilhado hospedado no arquivo de UM caso só
// aparece quando esse caso morre.
func seedClasse(t *testing.T, s *Server, characterID int64, name string, level int64) {
	t.Helper()
	err := s.queries.CreateClass(context.Background(), sqlcgen.CreateClassParams{
		Characterid: characterID, Classname: name, Level: level,
	})
	if err != nil {
		t.Fatalf("seed classe %q: %v", name, err)
	}
}

// stateOf lê a mesa e FALHA ALTO quando o banco recusa: um teste que seguisse
// com uma fila vazia afirmaria sobre nada (ALE-373).
func stateOf(t *testing.T, store *session.Store, sessionID int64) *live.SessionRuntimeState {
	t.Helper()
	state, err := store.State(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("ler a mesa da sessão %d: %v", sessionID, err)
	}
	return state
}
