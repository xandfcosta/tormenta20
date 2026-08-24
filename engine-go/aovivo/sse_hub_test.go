package aovivo

import (
	"strings"
	"testing"
)

// O hub SSE (ALE-253). O que se prende aqui é o que o socket dava e que não
// pode ser perdido na troca: recorte por PAPEL, e um leitor lento não derrubar
// a mesa.

func recebe(t *testing.T, conn *SSEConn) string {
	t.Helper()
	select {
	case frame := <-conn.Frames:
		return string(frame)
	default:
		return ""
	}
}

func TestOQuadroSaiNoFormatoDoFio(t *testing.T) {
	h := NewSSEHub()
	conn := h.Add(7, "c1", "gm")

	h.Emit(7, "", "session-state", map[string]any{"turnIndex": 2})

	got := recebe(t, conn)
	if !strings.HasPrefix(got, "event: session-state\n") {
		t.Fatalf("quadro = %q, queria começar com o nome do evento", got)
	}
	if !strings.Contains(got, `"turnIndex":2`) {
		t.Fatalf("quadro = %q, queria o corpo em JSON", got)
	}
	// Duas quebras fecham o quadro; sem elas o navegador espera para sempre.
	if !strings.HasSuffix(got, "\n\n") {
		t.Fatalf("quadro = %q, queria terminar em linha em branco", got)
	}
}

// O ESTADO SAI DUAS VEZES, e o recorte é por papel (ALE-122): inteiro para o
// mestre, redigido para os jogadores. Um hub que ignorasse o papel entregaria a
// fila inteira à mesa e passaria por todo teste que só conta mensagens.
func TestOPapelRecortaODestinatario(t *testing.T) {
	h := NewSSEHub()
	mestre := h.Add(7, "c1", "gm")
	jogador := h.Add(7, "c2", "player")

	h.Emit(7, "gm", "session-state", map[string]any{"segredo": true})

	if recebe(t, mestre) == "" {
		t.Error("o mestre não recebeu o que era dele")
	}
	if got := recebe(t, jogador); got != "" {
		t.Errorf("o jogador recebeu o quadro do mestre: %q", got)
	}
}

func TestPapelVazioVaiParaTodos(t *testing.T) {
	h := NewSSEHub()
	mestre := h.Add(7, "c1", "gm")
	jogador := h.Add(7, "c2", "player")

	h.Emit(7, "", "presence", map[string]any{"users": []string{}})

	if recebe(t, mestre) == "" || recebe(t, jogador) == "" {
		t.Error("papel vazio tinha de alcançar os dois")
	}
}

// Sessão vizinha não escuta a sala alheia.
func TestOutraSessaoNaoRecebe(t *testing.T) {
	h := NewSSEHub()
	deOutraMesa := h.Add(8, "c1", "gm")

	h.Emit(7, "", "session-state", map[string]any{})

	if got := recebe(t, deOutraMesa); got != "" {
		t.Errorf("vazou para a sessão 8: %q", got)
	}
}

// UM LEITOR LENTO NÃO CONGELA A MESA, e este é o guarda que justifica o
// `default` no `select`.
//
// Sem ele, um cliente que parou de ler (aba suspensa, rede ruim) encheria a
// fila e o `Emit` bloquearia segurando a trava do hub — o servidor inteiro
// pararia de transmitir por causa de um navegador. O quadro se perde de
// propósito: quem perdeu reconecta e busca o estado por HTTP, que é o mesmo
// caminho da primeira carga.
func TestLeitorLentoNaoBloqueiaOBroadcast(t *testing.T) {
	h := NewSSEHub()
	lento := h.Add(7, "c1", "gm")
	rapido := h.Add(7, "c2", "gm")
	// Enche a fila do lento sem ninguém consumir.
	for i := 0; i < sseBuffer+5; i++ {
		h.Emit(7, "", "session-state", map[string]any{"i": i})
	}

	// Se o `Emit` bloqueasse, o teste não chegaria aqui. E o rápido tem de ter
	// recebido — a perda é DELE, não da mesa.
	if len(lento.Frames) != sseBuffer {
		t.Errorf("fila do lento = %d, queria estar cheia em %d", len(lento.Frames), sseBuffer)
	}
	if len(rapido.Frames) != sseBuffer {
		t.Errorf("o leitor rápido também perdeu quadros: %d", len(rapido.Frames))
	}
}

func TestSairFechaAFilaESomeDaSala(t *testing.T) {
	h := NewSSEHub()
	conn := h.Add(7, "c1", "gm")

	h.Remove(7, "c1")

	if h.listeners(7) != 0 {
		t.Errorf("ouvintes = %d, queria zero", h.listeners(7))
	}
	// A fila fechada é o que faz o laço do handler terminar; sem isso a
	// goroutine da requisição vazaria a cada reconexão.
	if _, aberta := <-conn.Frames; aberta {
		t.Error("a fila continuou aberta depois de sair")
	}
}

// Sair duas vezes acontece de verdade: o cliente manda `Leave` e a conexão cai
// logo depois. A segunda não pode entrar em pânico fechando canal fechado.
func TestSairDuasVezesNaoDerruba(t *testing.T) {
	h := NewSSEHub()
	h.Add(7, "c1", "gm")

	h.Remove(7, "c1")
	h.Remove(7, "c1")
}
