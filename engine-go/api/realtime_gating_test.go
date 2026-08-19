package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Quem pode COMANDAR o combate.
//
// Cada handler mutante repete `g.access(...)` + `g.requireGm(...)`, e o risco
// real não é a linha estar errada — é um handler NOVO esquecer dela. Sem isso um
// jogador reinicia o combate da mesa inteira, e nada acusa: os handlers só são
// alcançáveis por um socket, e não existe cliente socket.io em Go para chamá-los
// num teste.
//
// Então esta é a invariante lida do PRÓPRIO registro: para cada evento que o
// gateway escuta, o handler dele contém (ou não) a porta do mestre, e a tabela
// abaixo diz qual é qual. Um evento novo entra vermelho até alguém decidir de
// que lado ele fica — que é exatamente a decisão que não pode passar batida.

// gmGate declara, por evento, se ele é do MESTRE. As exceções são deliberadas e
// estão anotadas uma a uma.
var gmGate = map[string]bool{
	"join-session":             false, // qualquer membro entra na sala
	"leave-session":            false, // sair é de quem está
	"get-session-state":        false, // ler o estado é de todo mundo na mesa
	"initiative-add":           true,
	"initiative-self":          false, // o JOGADOR rola a PRÓPRIA iniciativa: exceção deliberada
	"initiative-update":        true,
	"initiative-remove":        true,
	"initiative-next-turn":     true,
	"initiative-previous-turn": true,
	"initiative-reset":         true,
	"initiative-populate":      true,
	"vitals-patch":             false, // regra mais fina: assertVitalsEditable (mestre em qualquer um, jogador no próprio)
	"vitals-delta":             false, // idem
	"session-rest":             true,
	"apply-effect":             true,
	"disconnect":               false, // do transporte, não da mesa
	// Tabuleiro (ALE-124): abrir, montar e esconder peça é do mestre. Mover tem
	// regra mais fina (assertMovable) e por isso NÃO leva a porta larga — igual
	// aos vitais: o mestre move qualquer peça, o jogador move a própria na vez
	// dela, e fora de combate cada um anda com a sua.
	"board-open":          true,
	"board-close":         true,
	"get-board-state":     false, // ler o tabuleiro é de todo mundo na mesa — REDIGIDO por papel
	"board-token-add":     true,
	"board-token-remove":  true,
	"board-token-update":  true,
	"board-populate":      true,
	"board-terrain-paint": true,  // o chão é da cena, e a cena é do mestre
	"board-move-propose":  false, // assertMovable: papel, posse e a VEZ
	"board-move-cancel":   false, // desfaz o próprio provisório; o mestre desfaz o de qualquer um
	"board-move-commit":   false, // idem
}

var (
	registration = regexp.MustCompile(`sock\.On\("([a-z-]+)",\s*func\([^)]*\)\s*\{\s*g\.(\w+)\(`)
	handlerStart = `func (g *realtimeGateway) %s(`
)

func gatewaySource(t *testing.T) string {
	t.Helper()
	var all strings.Builder
	for _, file := range []string{"realtime_gateway.go", "realtime_initiative.go", "realtime_vitals.go", "realtime_board.go"} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("ler %s: %v", file, err)
		}
		all.Write(raw)
	}
	return all.String()
}

// handlerBody devolve o corpo do handler — do cabeçalho até o próximo `\n}` de
// topo, que em Go é o fim da função.
func handlerBody(t *testing.T, source, handler string) string {
	t.Helper()
	start := strings.Index(source, strings.ReplaceAll(handlerStart, "%s", handler))
	if start < 0 {
		t.Fatalf("não achei o handler %q no gateway", handler)
	}
	end := strings.Index(source[start:], "\n}\n")
	if end < 0 {
		t.Fatalf("handler %q não termina", handler)
	}
	return source[start : start+end]
}

func TestGmGateOnEveryRegisteredEvent(t *testing.T) {
	source := gatewaySource(t)
	matches := registration.FindAllStringSubmatch(source, -1)
	if len(matches) == 0 {
		t.Fatal("nenhum `sock.On` reconhecido — o registro mudou de forma e este teste ficou cego")
	}

	seen := map[string]bool{}
	for _, match := range matches {
		event, handler := match[1], match[2]
		seen[event] = true
		wantGm, declared := gmGate[event]
		if !declared {
			t.Errorf("evento %q é novo: decida se é do mestre e declare em gmGate", event)
			continue
		}
		gated := strings.Contains(handlerBody(t, source, handler), "requireGm")
		if gated != wantGm {
			t.Errorf("%s (%s): porta do mestre=%v, declarado=%v", event, handler, gated, wantGm)
		}
	}

	for event := range gmGate {
		if !seen[event] {
			t.Errorf("gmGate declara %q, mas o gateway não escuta mais esse evento", event)
		}
	}
}

// stateExit é toda linha que ENTREGA estado a um socket: o `Emit` do broadcast e
// o `ack` que hidrata quem acabou de pedir. Ack de literal (`{"joined": room}`)
// não é estado e fica de fora.
//
// O `board-state` entrou aqui junto com o primeiro emit do tabuleiro (ALE-124):
// enquanto o regex só conhecia `session-state`, uma peça escondida podia sair
// sem redação nenhuma e este teste passava verde — é assim que uma rede fica
// cega, e o próprio arquivo avisa que é assim.
var stateExit = regexp.MustCompile(`(?m)^.*(Emit\("(session|board)-state"|ackOK\(ctx\.ack, [^m]).*$`)

// O PV oculto é do mestre, e o broadcast não é o único caminho do estado até a
// tela: o `ack` do `get-session-state` responde a quem pediu, inclusive jogador.
// Foi exatamente por aí que ele vazou (ALE-122) — a redação existia e o ack
// passava por fora dela.
//
// Este teste lê o CÓDIGO porque não há cliente socket.io em Go: qualquer saída
// nova de estado nasce vermelha até dizer para qual papel está saindo.
func TestStateLeavesTheServerFilteredByRole(t *testing.T) {
	lines := stateExit.FindAllString(gatewaySource(t), -1)
	if len(lines) == 0 {
		t.Fatal("nenhuma saída de estado reconhecida — a forma mudou e este teste ficou cego")
	}

	for _, line := range lines {
		filtered := strings.Contains(line, "stateForRole") ||
			strings.Contains(line, "redactForPlayers") ||
			strings.Contains(line, "boardForRole") ||
			strings.Contains(line, `roleRoomName(sessionID, "gm")`)
		if !filtered {
			t.Errorf("estado sai sem filtro de papel:\n%s\ndiga para quem: stateForRole(ctx.role, …)"+
				" no ack, ou a sala do mestre no emit", strings.TrimSpace(line))
		}
	}
}

// Quem MOVE peça resolve posse e orçamento contra o BANCO.
//
// O `gmGate` acima diz que os handlers de movimento não levam a porta larga do
// mestre — e é isso mesmo, porque a regra é mais fina. O risco que sobra é o
// inverso: um handler novo de movimento que acredita no papel que o CLIENTE
// mandou, ou no orçamento que ele mandou junto. As duas coisas se resolvem em
// `moverFor`, que lê o dono do personagem e o deslocamento da ficha computada.
//
// Como os outros invariantes deste arquivo, isto se lê do FONTE: não existe
// cliente socket.io em Go para chamar o handler num teste.
func TestMoveHandlersResolveOwnershipOnTheServer(t *testing.T) {
	source := gatewaySource(t)
	matches := registration.FindAllStringSubmatch(source, -1)

	found := 0
	for _, match := range matches {
		event, handler := match[1], match[2]
		if !strings.HasPrefix(event, "board-move") {
			continue
		}
		found++
		if !strings.Contains(handlerBody(t, source, handler), "g.moverFor(") {
			t.Errorf("%s (%s) move peça sem resolver quem é o autor no servidor:"+
				" chame g.moverFor(ctx, tokenId) em vez de confiar no corpo da mensagem", event, handler)
		}
	}
	if found == 0 {
		t.Fatal("nenhum handler de movimento reconhecido — o nome dos eventos mudou e este teste ficou cego")
	}
}

// A política de origem do socket é a MESMA do HTTP (ALE-158).
//
// O `Router()` tinha o guarda do ALE-119 e o socket não tinha nenhum: o
// `SetCors` reflete qualquer origem, e refletir COM credenciais deixa um site
// de terceiros abrir o handshake do engine.io com o cookie do usuário
// (cross-site WebSocket hijacking). Numa LAN doméstica o risco prático é baixo,
// mas o binário não pode ter duas políticas.
func TestSocketOriginFollowsTheHttpPolicy(t *testing.T) {
	producao := &Server{cfg: Config{}} // serve a própria SPA
	// Atrás do proxy do Vite, e com os TRÊS apelidos da mesma origem: para o
	// browser eles são origens diferentes, e a que ficar de fora perde o socket
	// (ALE-185).
	dev := &Server{cfg: Config{CORSOrigins: splitOrigins(devCORSOrigins)}}

	casos := []struct {
		nome   string
		s      *Server
		origin string
		host   string
		quer   bool
	}{
		{"produção, mesma origem", producao, "http://192.168.1.5:3001", "192.168.1.5:3001", true},
		{"produção, site de terceiro", producao, "https://site-do-mal.example", "192.168.1.5:3001", false},
		{"produção, mesmo host em outra porta", producao, "http://192.168.1.5:9999", "192.168.1.5:3001", false},
		{"dev, a origem declarada", dev, "http://localhost:5173", "localhost:3001", true},
		{"dev, o loopback IPv6 (que é o que o Vite escuta)", dev, "http://[::1]:5173", "localhost:3001", true},
		{"dev, o loopback IPv4", dev, "http://127.0.0.1:5173", "localhost:3001", true},
		{"dev, qualquer outra", dev, "http://localhost:4444", "localhost:3001", false},
		// A mesa na LAN não precisa de CORS_ORIGIN nenhum: em produção o binário
		// serve a própria SPA, então quem abre pelo IP da rede é MESMA ORIGEM —
		// é o primeiro caso desta tabela, e é a resposta para "quando eu abrir
		// a LAN, o CORS vai barrar os outros?" (ALE-185).
		{"dev, a LAN que não foi declarada", dev, "http://192.168.1.5:5173", "192.168.1.5:3001", false},
		// Sem `Origin` passa de propósito: o navegador não manda esse cabeçalho
		// em GET de mesma origem, que é o transporte de polling em produção.
		// Quem guarda a sala aí é o JWT do handshake.
		{"sem Origin (polling de mesma origem)", producao, "", "192.168.1.5:3001", true},
	}
	for _, caso := range casos {
		if got := caso.s.socketOriginAllowed(caso.origin, caso.host); got != caso.quer {
			t.Errorf("%s: origem %q contra host %q deu %v, queria %v",
				caso.nome, caso.origin, caso.host, got, caso.quer)
		}
	}
}

// E o guarda roda ANTES do engine.io: o handshake de terceiro nem chega lá.
func TestAThirdPartyHandshakeIsRefused(t *testing.T) {
	s := newTestServer(t)
	s.cfg.CORSOrigins = nil // produção: mesma origem

	handler := s.SocketHandler()
	req := httptest.NewRequest(http.MethodGet, "/socket.io/?EIO=4&transport=polling", nil)
	req.Host = "192.168.1.5:3001"
	req.Header.Set("Origin", "https://site-do-mal.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("handshake de terceiro respondeu %d, esperava 403", rec.Code)
	}
}

// O caminho normal continua funcionando — um guarda que fecha a porta da mesa
// seria o defeito seguinte.
func TestTheOwnAppStillHandshakes(t *testing.T) {
	s := newTestServer(t)
	s.cfg.CORSOrigins = nil

	req := httptest.NewRequest(http.MethodGet, "/socket.io/?EIO=4&transport=polling", nil)
	req.Host = "192.168.1.5:3001"
	req.Header.Set("Origin", "http://192.168.1.5:3001")
	rec := httptest.NewRecorder()

	s.SocketHandler().ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Errorf("o próprio app foi recusado no handshake: %d %s", rec.Code, rec.Body.String())
	}
}
