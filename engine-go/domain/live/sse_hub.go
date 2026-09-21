package live

import (
	"encoding/json"
	"fmt"
	"sync"
)

// O hub das conexões SSE: destinatários agrupados por sessão e por PAPEL,
// porque o estado sai duas vezes — inteiro para o mestre e redigido para os
// jogadores, pelo `RedactForPlayers`.
//
// SSE e não socket: tudo o que sobe é mutação, e mutação é `POST`. Do tempo
// real sobra só a descida, e para descida SSE basta.

// SSEFrame é um evento já codificado no formato do fio. Codificar UMA vez por
// broadcast e não uma por conexão: numa mesa de seis, o estado da iniciativa
// seria serializado seis vezes por turno sem isto.
type SSEFrame []byte

func encodeFrame(event string, payload any) (SSEFrame, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return SSEFrame(fmt.Sprintf("event: %s\ndata: %s\n\n", event, body)), nil
}

// SSEConn é um leitor ligado. O `role` é o do momento da ENTRADA: quem for
// promovido no meio da sessão reconecta.
type SSEConn struct {
	id     string
	role   string
	Frames chan SSEFrame
}

// destination identifica um fluxo ordenado: os quadros de um mesmo evento para um
// mesmo papel de uma mesma sessão chegam na ordem em que as mutações
// aconteceram, e não na ordem em que as goroutines conseguiram emitir.
type destination struct {
	sessionID int64
	role      string
	event     string
}

// SSEHub guarda quem está ouvindo cada sessão.
type SSEHub struct {
	Mu sync.Mutex
	// sessionID → connID → conexão
	conns map[int64]map[string]*SSEConn
	// lastSeq guarda a maior ordem já emitida por destino, para reconhecer
	// quadro atrasado. Ver `EmitOrdered`.
	lastSeq map[destination]uint64
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		conns:   map[int64]map[string]*SSEConn{},
		lastSeq: map[destination]uint64{},
	}
}

// A fila de cada conexão. Um leitor lento não pode segurar o broadcast — ver
// `Emit`. Dez quadros é folga para uma rajada (mover peça, virar turno) sem
// virar memória parada por cliente.
const sseBuffer = 10

func (h *SSEHub) Add(sessionID int64, connID, role string) *SSEConn {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	conn := &SSEConn{id: connID, role: role, Frames: make(chan SSEFrame, sseBuffer)}
	if h.conns[sessionID] == nil {
		h.conns[sessionID] = map[string]*SSEConn{}
	}
	h.conns[sessionID][connID] = conn
	return conn
}

func (h *SSEHub) Remove(sessionID int64, connID string) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	room := h.conns[sessionID]
	if room == nil {
		return
	}
	if conn, ok := room[connID]; ok {
		close(conn.Frames)
		delete(room, connID)
	}
	if len(room) == 0 {
		delete(h.conns, sessionID)
	}
}

// Emit entrega um evento aos ouvintes de uma sessão. `role` vazio é para todos;
// "gm" ou "player" recorta.
//
// NUNCA BLOQUEIA. Se a fila de um leitor está cheia, o quadro é DESCARTADO para
// ele e a conexão fica marcada para morrer — o `select` com `default` é o que
// impede um cliente lento de congelar o servidor inteiro, que é o modo de falha
// clássico de hub de broadcast. Quem perdeu quadro reconecta e busca o estado
// por HTTP, que é o mesmo caminho da primeira carga: nenhum estado vive só no
// fio.
func (h *SSEHub) Emit(sessionID int64, role string, event string, payload any) {
	frame, err := encodeFrame(event, payload)
	if err != nil {
		return
	}
	h.Mu.Lock()
	defer h.Mu.Unlock()
	h.entregaLocked(sessionID, role, frame)
}

// entregaLocked enfileira o quadro nos destinatários. Exige `h.Mu`.
func (h *SSEHub) entregaLocked(sessionID int64, role string, frame SSEFrame) {
	for _, conn := range h.conns[sessionID] {
		if role != "" && conn.role != role {
			continue
		}
		select {
		case conn.Frames <- frame:
		default:
			// Leitor lento: o quadro se perde de propósito.
		}
	}
}

// EmitOrdered entrega um evento DESCARTANDO quadro atrasado.
//
// A mutação do estado é serializada por uma trava, mas a EMISSÃO acontece
// depois de a trava cair. Duas mutações concorrentes emitem então fora de
// ordem, e o cliente aplica o estado INTEIRO: o quadro velho vence, e a entrada
// que só existia no quadro novo some da tela do mestre e da mesa.
// `TestTheFrameFollowsTheOrderOfTheMutation` reproduz sob `-race`.
//
// POR QUE AQUI E NÃO EM TODO STREAM DA CASA: a ordem importa neste hub porque o
// que viaja é o CLONE do estado, capturado no instante da mutação, e um clone
// velho que chegue depois de um novo é uma verdade antiga sobrescrevendo uma
// recente. O stream do app (Datastar) é imune por construção — ele manda um
// AVISO sem estado, e o leitor relê o estado de agora; dois avisos fora de
// ordem mandam reler duas vezes, e não há o que reordenar.
//
// A guarda mora AQUI, e não numa trava em volta de cada publicação, porque
// publicação nova é fácil de esquecer. O hub é o funil por onde tudo passa, e o
// que ele não deixa acontecer ninguém precisa lembrar de evitar — é a mesma
// razão de o guarda de papel viver no `Emit`.
//
// `Seq == 0` significa SEM ORDEM: o quadro passa e o destino reinicia. É o que
// eventos sem contador usam (presença, aviso de persistência) e é o que o
// fechamento de tabuleiro usa — depois dele, o próximo tabuleiro começa da
// versão 1, e insistir na ordem antiga descartaria o mapa novo inteiro.
func (h *SSEHub) EmitOrdered(sessionID int64, role, event string, Seq uint64, payload any) {
	frame, err := encodeFrame(event, payload)
	if err != nil {
		return
	}
	key := destination{sessionID: sessionID, role: role, event: event}

	// A trava é UMA e cobre decidir E entregar. Decidir sob trava e entregar
	// fora dela não conserta nada: outra goroutine se enfia entre as duas e
	// enfileira o quadro dela primeiro — a contabilidade fica certa e a fila
	// errada.
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if Seq == 0 {
		delete(h.lastSeq, key) // sem ordem: passa e reinicia o destino
	} else {
		if Seq < h.lastSeq[key] {
			return // quadro atrasado: a tela já tem coisa mais nova
		}
		h.lastSeq[key] = Seq
	}
	h.entregaLocked(sessionID, role, frame)
}

// listeners conta quem está ouvindo — para teste e para o `liveSessions`.
func (h *SSEHub) listeners(sessionID int64) int {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	return len(h.conns[sessionID])
}
