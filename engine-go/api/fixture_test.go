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
	"t20engine/aovivo"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// A BANCADA HTTP do `api`, lida por 49 arquivos de teste.
//
// Ela morava em `fixture_test.go` — o arquivo da CENA — e isso não era
// arrumação: era a fatia 2 da ALE-278 acontecendo pela metade. O molde do
// BANCO virou pacote (`db/testdb`) antes da primeira cena sair; esta bancada,
// que monta um `Server` de verdade e fala HTTP, ficou onde estava porque o
// tipo `api.Server` a prende aqui — um pacote de bancada que o importasse
// seria importado de volta pelos testes dele, que é o ciclo que a divisão
// existe para evitar.
//
// O que a mudança de arquivo conserta é o nome: quando a Mesa virou
// `web/table` (ALE-278), a bancada do repositório INTEIRO teria ido junto por
// acidente de prefixo. É a mesma família do guarda que media o próprio
// diretório e encolheu ao mudar de casa.

type pilotoFixture struct {
	s          *Server
	mestre     int64
	jogador    int64
	campaignID int64
	sessionID  int64
	charID     int64
}

// A mesa do piloto: mestre, um jogador com PC de nível 8 COM a perícia
// Iniciativa (sem ela o bônus cai em zero e o teste do d20 nasce vácuo — a
// armadilha que a ALE-213 registrou), e um NPC para o mestre esconder.
func novoPiloto(t *testing.T) pilotoFixture {
	t.Helper()
	s := newTestServer(t)
	// O CATÁLOGO É O DE VERDADE, e não um `{"items":[]}`.
	//
	// Ele era vazio, e isso fazia regra sumir do TESTE sem sumir da produção: a
	// fatia 7 mediu um escudo sendo VESTIDO porque o eixo de equipar não achava
	// o item, e a fatia 8 mediu a distribuição de atributo do humano passando com
	// três vezes o mesmo, porque a raça não estava no catálogo primado. Um
	// fixture que desliga validação em silêncio é pior que um fixture lento.
	bruto, err := os.ReadFile(filepath.Join("..", "parity", "_catalogs.json"))
	if err != nil {
		t.Fatalf("ler catálogos: %v (gere com `go run ./cmd/genoracle`)", err)
	}
	catalogs, err := engine.PrimeEngineCatalogs(bruto)
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.catalogs = catalogs

	mestre := seedUser(t, s, "mestre@t.com")
	jogador := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, mestre)
	sessionID := seedSession(t, s, campaignID)
	charID := seedCharacterAtLevel(t, s, jogador, "Arcanista", 8, 20, 30, 5, 10)
	seedMember(t, s, campaignID, charID, "player")
	if _, err := s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: charID, Name: "Iniciativa", Attribute: "dexterity", Trained: 0, Custom: 0,
	}); err != nil {
		t.Fatalf("semear perícia: %v", err)
	}
	// Aqui havia um `_ = s.SocketHandler()`, e o comentário dele dizia que
	// montar o gateway punha "a PONTE entre os dois transportes debaixo do
	// teste, porque ela é o custo central do piloto". A ALE-253 tirou o socket
	// do projeto e a ponte junto: há um caminho de publicação só, o hub SSE, e
	// ele existe desde o `newServer`. O custo central do piloto deixou de
	// existir em vez de deixar de ser testado.
	return pilotoFixture{s: s, mestre: mestre, jogador: jogador, campaignID: campaignID, sessionID: sessionID, charID: charID}
}

// token assina um JWT do usuário — o mesmo caminho do `authed` da casa.
func (f pilotoFixture) token(t *testing.T, userID int64) string {
	t.Helper()
	user, err := f.s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", userID, err)
	}
	tok, err := f.s.signToken(user)
	if err != nil {
		t.Fatalf("assinar token: %v", err)
	}
	return tok
}

// pede manda uma requisição autenticada pelo MesaRouter — que é outro roteador
// que o `Router()` da API, e por isso o `authed` da casa não serve.
func (f pilotoFixture) pede(t *testing.T, userID int64, method, path, body string) *httptest.ResponseRecorder {
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

func (f pilotoFixture) tableUrl() string {
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
func (f pilotoFixture) posta(t *testing.T, userID int64, caminho, corpo string) string {
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
func (f pilotoFixture) scene(t *testing.T) {
	t.Helper()
	oculto := true
	pv, pvMax := int64(12), int64(130)
	if _, err := f.s.sessions.StartScene(f.sessionID); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, aovivo.InitiativeEntry{
		Label: "Ogro cansado", Initiative: 19, Type: "npc",
		HpHidden: &oculto, HpCurrent: &pv, HpMax: &pvMax,
	}); err != nil {
		t.Fatalf("semear ogro: %v", err)
	}
	if _, err := f.s.sessions.AddInitiativeEntry(f.sessionID, aovivo.InitiativeEntry{
		Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &f.charID,
	}); err != nil {
		t.Fatalf("semear PC: %v", err)
	}
}
