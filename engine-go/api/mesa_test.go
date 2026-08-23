package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// Os guardas do piloto Datastar (ALE-219).
//
// O que vale a pena proteger aqui NÃO é o HTML: é que a página do jogador
// obedece à mesma redação que o socket obedece. A tela é nova, o transporte é
// novo, e a tentação é justamente reimplementar a regra do lado novo — o teste
// que importa é o que pega isso.

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
	catalogs, err := engine.PrimeEngineCatalogs([]byte(`{"items":[]}`))
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
	return pilotoFixture{s: s, mestre: mestre, jogador: jogador, campaignID: campaignID, sessionID: sessionID, charID: charID}
}

// pede manda uma requisição autenticada pelo MesaRouter — que é outro roteador
// que o `Router()` da API, e por isso o `authed` da casa não serve.
func (f pilotoFixture) pede(t *testing.T, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	user, err := f.s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", userID, err)
	}
	token, err := f.s.signToken(user)
	if err != nil {
		t.Fatalf("assinar token: %v", err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	http.StripPrefix("/mesa", f.s.MesaRouter()).ServeHTTP(rec, req)
	return rec
}

func (f pilotoFixture) urlDaMesa() string {
	return "/mesa/" + strconv.FormatInt(f.campaignID, 10) + "/" + strconv.FormatInt(f.sessionID, 10)
}

// cena põe a sessão em cena com um ogro de PV OCULTOS e o PC do jogador.
func (f pilotoFixture) cena(t *testing.T) {
	t.Helper()
	oculto := true
	pv, pvMax := int64(12), int64(130)
	if _, err := f.s.sessions.startScene(f.sessionID); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	if _, err := f.s.sessions.addInitiativeEntry(f.sessionID, InitiativeEntry{
		Label: "Ogro cansado", Initiative: 19, Type: "npc",
		HpHidden: &oculto, HpCurrent: &pv, HpMax: &pvMax,
	}); err != nil {
		t.Fatalf("semear ogro: %v", err)
	}
	if _, err := f.s.sessions.addInitiativeEntry(f.sessionID, InitiativeEntry{
		Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &f.charID,
	}); err != nil {
		t.Fatalf("semear PC: %v", err)
	}
}

// O guarda que justifica o piloto reusar `stateForRole` em vez de montar a
// própria leitura: a PÁGINA obedece à mesma redação que o socket.
//
// Provado VERMELHO trocando `stateForRole(role, ...)` por `s.sessions.getState(...)`
// no `loadMesaView` — o HTML passou a carregar "12/130", os PV que o mestre
// escondeu, para dentro da tela do jogador.
func TestMesaNaoVazaPVOculto(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	corpo := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(corpo, "Ogro cansado") {
		t.Fatal("o ogro sumiu da fila do jogador — ele deve ver QUEM está lá")
	}
	if strings.Contains(corpo, "130") {
		t.Errorf("os PV ocultos do mestre vazaram para o HTML do jogador")
	}
	// A flag sobrevive à redação de propósito: "sem barra" e "escondido" são
	// coisas diferentes, e a segunda é informação (ALE-210).
	if !strings.Contains(corpo, "PV ocultos pelo mestre") {
		t.Errorf("a linha oculta não DISSE que está oculta — vira 'sem vida' na tela")
	}
}

// A outra metade da ALE-210: fora de cena o jogador não recebe fila NENHUMA.
// Não desenhar seria UX; não mandar é a trava.
//
// Provado VERMELHO com o mesmo desvio do teste acima: sem cena o HTML passou a
// listar os dois combatentes que o mestre está montando às escondidas.
func TestMesaForaDeCenaNaoMandaFila(t *testing.T) {
	f := novoPiloto(t)
	// Fila CHEIA e cena DESLIGADA: é o mestre montando a briga antes de começar.
	if _, err := f.s.sessions.addInitiativeEntry(f.sessionID, InitiativeEntry{
		Label: "Chefe secreto", Initiative: 22, Type: "npc",
	}); err != nil {
		t.Fatalf("semear chefe: %v", err)
	}

	corpo := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if strings.Contains(corpo, "Chefe secreto") {
		t.Error("a fila de fora de cena vazou para o jogador")
	}
	if !strings.Contains(corpo, "Fora de cena") {
		t.Error("a tela não explicou o vazio")
	}
	// O mestre, na MESMA página, continua vendo o que montou — sem esta metade
	// o teste passaria com um `redactForPlayers` aplicado a todo mundo.
	corpoDoMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(corpoDoMestre, "Chefe secreto") {
		t.Error("o mestre perdeu a própria fila — a redação está pegando o papel errado")
	}
}

// A recusa tem de CHEGAR NA TELA, e é isto que o piloto ganha de graça sobre o
// socket: a ALE-213 deixou anotado que o cliente não escuta o `exception`, então
// lá um d20 fora da faixa some em silêncio.
//
// Provado VERMELHO devolvendo `http.Error` no lugar do patch de sinal: o corpo
// virou texto solto que o Datastar descarta, e a tela não muda.
func TestMesaRecusaD20ForaDaFaixaEDiz(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	rec := f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/iniciativa", `{"d20":47}`)
	corpo := rec.Body.String()

	if !strings.HasPrefix(corpo, "event: datastar-patch-signals") {
		t.Fatalf("a recusa não saiu como evento do Datastar, saiu como:\n%s", corpo)
	}
	// A mensagem carrega o valor ofendido, que é a regra da casa para exceção.
	if !strings.Contains(corpo, "47") {
		t.Errorf("a recusa não disse qual valor foi recusado:\n%s", corpo)
	}
	if !strings.Contains(corpo, `\"erro\"`) && !strings.Contains(corpo, `"erro"`) {
		t.Errorf("a recusa não veio no sinal `erro`, então nada acende na tela:\n%s", corpo)
	}
}

// O caminho feliz, e a metade que importa é a SEGUNDA: o total é do servidor.
//
// O bônus do fixture é 3 (nível 8, perícia Iniciativa treinada em Destreza),
// então mandar d20=14 tem de gravar 17. Sem a perícia semeada o bônus seria
// zero e 14 == 14 — o teste passaria verde sobre uma tela que somou sozinha,
// que é exatamente o defeito que ele mira (a armadilha da ALE-213).
func TestMesaRegistraIniciativaComTotalDoServidor(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)
	bonus, err := f.s.initiativeBonus(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("bônus: %v", err)
	}
	if bonus == 0 {
		t.Fatal("bônus zero: o fixture nasceu vácuo e este teste não provaria nada")
	}

	if rec := f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/iniciativa", `{"d20":14}`); rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}

	estado := f.s.sessions.getState(f.sessionID)
	for i := range estado.Initiative {
		e := &estado.Initiative[i]
		if e.CharacterID != nil && *e.CharacterID == f.charID {
			if querido := 14 + int(bonus); e.Initiative != querido {
				t.Fatalf("iniciativa gravada = %d, queria %d (14 + bônus %d)", e.Initiative, querido, bonus)
			}
			return
		}
	}
	t.Fatal("a linha do jogador não entrou na fila")
}

// O formato do fio, que é onde um erro custa caro e não aparece: o Datastar
// corta cada linha do `data:` no primeiro espaço e reagrupa por chave, então um
// fragmento com quebra de linha mandado num `data:` só chegaria TRUNCADO — e a
// tela mostraria meia página sem nenhum erro em lugar nenhum.
//
// Provado VERMELHO trocando o laço por um `data: elements ` único com o HTML
// inteiro: a segunda linha deixou de ter prefixo.
func TestPatchElementsQuebraCadaLinha(t *testing.T) {
	evento := patchElementsEvent([]byte("<div id=\"x\">\n  <p>oi</p>\n</div>\n"))

	querido := "event: datastar-patch-elements\n" +
		"data: elements <div id=\"x\">\n" +
		"data: elements   <p>oi</p>\n" +
		"data: elements </div>\n" +
		"\n"
	if evento != querido {
		t.Errorf("evento errado:\n--- veio ---\n%s\n--- queria ---\n%s", evento, querido)
	}
}

// A vez é MINHA quando a linha na vez é de um personagem meu — e é "de outro"
// quando não é. Tradução literal do `playerTurnState` da SPA; duas escadas
// divergiriam em silêncio.
func TestMesaTurnOf(t *testing.T) {
	meu, alheio := int64(7), int64(9)
	fila := []InitiativeEntry{
		{Label: "Ogro", Initiative: 19, Type: "npc"},
		{Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &meu},
	}
	meus := map[int64]bool{meu: true}

	casos := []struct {
		nome      string
		turnIndex int
		kind      string
		label     string
	}{
		{"fora de combate ninguém está na vez", -1, "idle", ""},
		{"a vez do ogro é de outro", 0, "other", "Ogro"},
		{"a vez do meu personagem é minha", 1, "mine", ""},
		{"índice além da fila não inventa uma vez", 5, "idle", ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := mesaTurnOf(&SessionRuntimeState{Initiative: fila, TurnIndex: c.turnIndex}, meus)
			if got.Kind != c.kind || got.Label != c.label {
				t.Errorf("veio {%s %q}, queria {%s %q}", got.Kind, got.Label, c.kind, c.label)
			}
		})
	}
	// O personagem alheio não acende a faixa de ninguém.
	outro := mesaTurnOf(&SessionRuntimeState{
		Initiative: []InitiativeEntry{{Label: "Colega", Type: "character", CharacterID: &alheio}},
		TurnIndex:  0,
	}, meus)
	if outro.Kind != "other" {
		t.Errorf("a vez de um PC alheio virou %q", outro.Kind)
	}
}

// Os LIMIARES da cor, e só eles: a tabela inteira de porcentagens seria a
// implementação reescrita. 25 e 50 são os mesmos do `hpFillVar` da SPA, e é a
// divergência entre os dois que este teste existe para tornar barulhenta.
func TestHpTomDeNosLimiares(t *testing.T) {
	casos := []struct {
		pct int
		tom string
	}{
		{0, "bg-hp-critical"},
		{25, "bg-hp-critical"},
		{26, "bg-hp-hurt"},
		{50, "bg-hp-hurt"},
		{51, "bg-hp-full"},
		{100, "bg-hp-full"},
	}
	for _, c := range casos {
		if got := hpTomDe(c.pct); got != c.tom {
			t.Errorf("hpTomDe(%d) = %q, queria %q", c.pct, got, c.tom)
		}
	}
}
