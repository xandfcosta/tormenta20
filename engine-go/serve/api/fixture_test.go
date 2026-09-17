package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"t20engine/domain/live"
	"testing"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// A BANCADA HTTP do `api`, lida por dezenas de arquivos de teste.
//
// Ela fica no pacote e não num `testdb` irmão porque o tipo `api.Server` a
// prende aqui: um pacote de bancada que o importasse seria importado de volta
// pelos testes dele, que é o ciclo que a divisão existe para evitar.

type sceneFixture struct {
	s          *Server
	mestre     int64
	jogador    int64
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
	// O CATÁLOGO É O DE VERDADE, e não um `{"items":[]}`.
	//
	// Catálogo vazio faz regra sumir do TESTE sem sumir da produção: um escudo
	// passa a ser VESTIDO porque o eixo de equipar não acha o item, e a
	// distribuição de atributo do humano aceita três vezes o mesmo porque a raça
	// não está primada. Fixture que desliga validação em silêncio é pior que
	// fixture lento.
	bruto, err := os.ReadFile(filepath.Join("..", "..", "parity", "_catalogs.json"))
	if err != nil {
		t.Fatalf("ler catálogos: %v (gere com `go run ./cmd/genoracle`)", err)
	}
	catalogs, err := engine.PrimeEngineCatalogs(bruto)
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.primeCatalogs(catalogs)

	mestre := seedUser(t, s, "mestre@t.com")
	jogador := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, mestre)
	sessionID := seedSession(t, s, campaignID)
	charID := seedCharacterAtLevel(t, s, jogador, "Arcanista", 8, 20, 30, 5, 10)
	seedMember(t, s, campaignID, charID)
	if _, err := s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: charID, Name: "Iniciativa", Attribute: "dexterity", Trained: 0, Custom: 0,
	}); err != nil {
		t.Fatalf("semear perícia: %v", err)
	}
	return sceneFixture{s: s, mestre: mestre, jogador: jogador, campaignID: campaignID, sessionID: sessionID, charID: charID}
}

// token assina um JWT do usuário — o mesmo caminho do `authed` da casa.
func (f sceneFixture) token(t *testing.T, userID int64) string {
	t.Helper()
	user, err := f.s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", userID, err)
	}
	tok, err := f.s.accountRules().signToken(user)
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
	return "/mesa/" + strconv.FormatInt(f.campaignID, 10) + "/" + strconv.FormatInt(f.sessionID, 10)
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
func (f sceneFixture) posta(t *testing.T, userID int64, caminho, corpo string) string {
	t.Helper()
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+caminho, strings.NewReader(corpo))
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
	lido, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ler resposta: %v", err)
	}
	return string(lido)
}

// cena põe a sessão em cena com um ogro de PV OCULTOS e o PC do jogador.
func (f sceneFixture) scene(t *testing.T) {
	t.Helper()
	oculto := true
	pv, pvMax := int64(12), int64(130)
	if _, err := f.s.sessions.StartScene(f.sessionID); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		Label: "Ogro cansado", Initiative: 19, Type: "npc",
		HpHidden: &oculto, HpCurrent: &pv, HpMax: &pvMax,
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
func seedClasse(t *testing.T, s *Server, characterID int64, nome string, nivel int64) {
	t.Helper()
	err := s.queries.CreateClass(context.Background(), sqlcgen.CreateClassParams{
		Characterid: characterID, Classname: nome, Level: nivel,
	})
	if err != nil {
		t.Fatalf("seed classe %q: %v", nome, err)
	}
}
