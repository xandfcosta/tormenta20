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
}

var (
	registration = regexp.MustCompile(`sock\.On\("([a-z-]+)",\s*func\([^)]*\)\s*\{\s*g\.(\w+)\(`)
	handlerStart = `func (g *realtimeGateway) %s(`
)

func gatewaySource(t *testing.T) string {
	t.Helper()
	var all strings.Builder
	for _, file := range []string{"realtime_gateway.go", "realtime_initiative.go", "realtime_vitals.go"} {
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
