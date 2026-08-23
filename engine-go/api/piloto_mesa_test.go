package api

import (
	"bufio"
	"compress/gzip"
	"context"
	"io"
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
	// O gateway do socket, como em produção: é ele que preenche o `s.rt`, e sem
	// ele a escrita pela página grava mas recusa dizendo que a mesa não foi
	// avisada. Montá-lo aqui é o que põe a PONTE entre os dois transportes
	// debaixo do teste — ela é o custo central do piloto e não pode ficar de fora.
	_ = s.SocketHandler()
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
	http.StripPrefix("/piloto", f.s.PilotoRouter()).ServeHTTP(rec, req)
	return rec
}

func (f pilotoFixture) urlDaMesa() string {
	return "/piloto/mesa/" + strconv.FormatInt(f.campaignID, 10) + "/" + strconv.FormatInt(f.sessionID, 10)
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
	srv := httptest.NewServer(http.StripPrefix("/piloto", f.s.PilotoRouter()))
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

	corpo := f.posta(t, f.jogador, f.urlDaMesa()+"/iniciativa", `{"d20":47}`)

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

	// O corpo é conferido, e não só o código: com a ordem trocada o servidor
	// devolve 200 com um erro DENTRO do sinal, e "deu 200" não é resposta.
	if resposta := f.posta(t, f.jogador, f.urlDaMesa()+"/iniciativa", `{"d20":14}`); !strings.Contains(resposta, `{"erro":""}`) {
		t.Fatalf("a escrita não foi aceita, respondeu:\n%s", resposta)
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

// A COMPRESSÃO do stream — o passo (a) da ordem combinada, e o ganho que domina
// todos os outros: medido neste piloto, 52.332 bytes crus de três remendos viram
// 2.513 em gzip e 1.827 em brotli. Ela é invisível na tela, então quem apagar o
// `WithCompression()` multiplica a banda por vinte sem nada parecer errado — é
// exatamente por isso que ela precisa de guarda.
//
// Servidor de verdade e `Accept-Encoding` escrito à mão de propósito: o
// `http.Client` do Go põe o cabeçalho sozinho e descomprime por baixo do pano,
// escondendo o `Content-Encoding` que este teste existe para ver.
func TestMesaStreamComprime(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)
	srv := httptest.NewServer(http.StripPrefix("/piloto", f.s.PilotoRouter()))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+f.urlDaMesa()+"/stream", nil)
	if err != nil {
		t.Fatalf("montar pedido: %v", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.jogador))
	// O stream não termina sozinho; o contexto é o que devolve o controle depois
	// do primeiro quadro.
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("abrir stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, queria gzip — o stream saiu cru", got)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("o corpo não é gzip: %v", err)
	}
	// Lê UM evento — até a linha em branco que o fecha. Ler um buffer de tamanho
	// fixo não serve: o fragmento tem ~17KB e a fila fica depois da faixa e do
	// Grupo, então um buffer curto corta justamente o que interessa, e um longo
	// bloquearia esperando quadros que só o batimento traria.
	leitor := bufio.NewScanner(zr)
	leitor.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var quadro strings.Builder
	for leitor.Scan() {
		linha := leitor.Text()
		if linha == "" {
			break
		}
		quadro.WriteString(linha)
		quadro.WriteString("\n")
	}

	// As duas metades: veio no evento do Datastar E carrega a fila de verdade.
	// Só a primeira passaria verde com um stream que comprime silêncio.
	texto := quadro.String()
	if !strings.Contains(texto, "datastar-patch-elements") {
		t.Errorf("o primeiro quadro não é um patch do Datastar:\n%s", texto)
	}
	if !strings.Contains(texto, "Ogro cansado") {
		t.Errorf("o primeiro quadro não trouxe a fila:\n%s", texto)
	}
}

// O aviso do store — o passo (b). O que este teste afirma é a INVARIANTE que
// torna o aviso confiável: `apply` é o funil das treze mutações da fila,
// então nenhuma delas pode escapar sem cutucar quem escuta.
func TestMesaAvisaAssinantesEmCadaMutacao(t *testing.T) {
	f := novoPiloto(t)
	aviso, parar := f.s.sessions.assinar(f.sessionID)

	if _, err := f.s.sessions.startScene(f.sessionID); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	select {
	case <-aviso:
	default:
		t.Fatal("a mutação passou sem avisar quem escuta")
	}

	// Baixar a assinatura tem de PARAR o aviso: sem isto cada aba fechada deixa
	// um canal para sempre, e o `avisarLocked` passa a percorrer uma lista que só
	// cresce escrevendo em canais que ninguém lê.
	parar()
	if _, err := f.s.sessions.endScene(f.sessionID); err != nil {
		t.Fatalf("encerrar cena: %v", err)
	}
	select {
	case <-aviso:
		t.Fatal("o ouvinte baixado continuou recebendo — a lista vaza")
	default:
	}
	if n := len(f.s.sessions.ouvintes[f.sessionID]); n != 0 {
		t.Errorf("sobraram %d ouvintes registrados depois da baixa", n)
	}
}

// O campo vazio não pode virar um total (ALE-236).
//
// MEDIDO no navegador: o `data-bind` do Datastar escreve ZERO no sinal quando
// um `<input type=number>` esvazia — digitar 7, apagar, e o sinal vai a 0. Sem
// guarda, apagar para redigitar mostra "Total previsto 8" com bônus 8 e dado
// nenhum: um total que não existe, lido no instante da decisão. Mesma família
// da ALE-224, onde a prévia era o que impedia o erro silencioso.
//
// Este guarda pina a EXPRESSÃO, e digo isso em vez de fingir que ele pina o
// comportamento. O teste comportamental exigiria a cena EM JOGO com o jogador
// tendo personagem nela, e montar esse estado no e2e é entrar exatamente na
// armadilha que a ALE-238 documenta: asserção que depende do estado do combate
// mede o banco, não o app. A prova do comportamento foi a medição no navegador,
// que está descrita acima e é reproduzível em três linhas.
func TestPreviaDoD20NaoMenteComOCampoVazio(t *testing.T) {
	bonus := int64(8)
	html, err := renderFragmento(t.Context(), mesa(mesaView{
		CampaignID: 7, SessionID: 42, SceneActive: true,
		Turn: mesaTurn{Kind: "idle"},
		Eu:   &mesaEu{CharacterID: 1, Nome: "Samira", Bonus: bonus},
	}))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// LITERAL e não escapado: o atributo é CONSTANTE, e o templ só escapa os
	// dinâmicos — foi o que a ALE-227 mediu ao comparar as duas saídas byte a
	// byte. Escrevi a forma escapada primeiro e o guarda nasceu vermelho por
	// isso, o que ao menos provou que ele lê o HTML de verdade.
	const faixa = "$d20 >= 1 && $d20 <= 20"
	if !strings.Contains(html, faixa) {
		t.Errorf("a prévia não é condicionada à faixa do dado — campo vazio vira um total inventado")
	}
	if !strings.Contains(html, "informe o dado") {
		t.Error("sem dado, a linha não diz o que falta")
	}
	if !strings.Contains(html, "$registrando || !(") {
		t.Error("o botão continua oferecendo uma ação que o servidor vai recusar")
	}
}
