package api

import (
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
	// Tabuleiro (ALE-124): abrir, montar e esconder peça é do mestre. Quem move a
	// PRÓPRIA peça no próprio turno entra na fatia do movimento, e virá com regra
	// mais fina (assertMovable), como os vitais.
	"board-open":         true,
	"board-close":        true,
	"get-board-state":    false, // ler o tabuleiro é de todo mundo na mesa — REDIGIDO por papel
	"board-token-add":    true,
	"board-token-remove": true,
	"board-token-update": true,
	"board-populate":     true,
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
