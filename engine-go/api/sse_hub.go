package api

import (
	"encoding/json"
	"fmt"
	"sync"
)

// O hub das conexões SSE — o que as SALAS do socket.io eram (ALE-253).
//
// A troca não é de biblioteca, é de FORMA: o socket existia para o cliente
// mandar coisa, e ele não precisava — as 37 mensagens que subiam eram todas
// mutação, e mutação é `POST`. O que sobra do tempo real é só o de descida, e
// para descida SSE basta.
//
// O que este arquivo substitui é exatamente o que o socket dava de útil:
// destinatários agrupados por sessão e por PAPEL, porque o estado sai duas
// vezes — inteiro para o mestre e redigido para os jogadores (ALE-122). O
// `redactForPlayers` não mudou uma linha; mudou quem o entrega.

// sseFrame é um evento já codificado no formato do fio. Codificar UMA vez por
// broadcast e não uma por conexão: numa mesa de seis, o estado da iniciativa
// seria serializado seis vezes por turno sem isto.
type sseFrame []byte

func encodeFrame(event string, payload any) (sseFrame, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return sseFrame(fmt.Sprintf("event: %s\ndata: %s\n\n", event, body)), nil
}

// sseConn é um leitor ligado. O `role` é o do momento da ENTRADA, como no
// socket: quem for promovido no meio da sessão reconecta (ALE-122).
type sseConn struct {
	id     string
	role   string
	frames chan sseFrame
}

// sseHub guarda quem está ouvindo cada sessão.
type sseHub struct {
	mu sync.Mutex
	// sessionID → connID → conexão
	conns map[int64]map[string]*sseConn
}

func newSSEHub() *sseHub {
	return &sseHub{conns: map[int64]map[string]*sseConn{}}
}

// A fila de cada conexão. Um leitor lento não pode segurar o broadcast — ver
// `emit`. Dez quadros é folga para uma rajada (mover peça, virar turno) sem
// virar memória parada por cliente.
const sseBuffer = 10

func (h *sseHub) add(sessionID int64, connID, role string) *sseConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	conn := &sseConn{id: connID, role: role, frames: make(chan sseFrame, sseBuffer)}
	if h.conns[sessionID] == nil {
		h.conns[sessionID] = map[string]*sseConn{}
	}
	h.conns[sessionID][connID] = conn
	return conn
}

func (h *sseHub) remove(sessionID int64, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sala := h.conns[sessionID]
	if sala == nil {
		return
	}
	if conn, ok := sala[connID]; ok {
		close(conn.frames)
		delete(sala, connID)
	}
	if len(sala) == 0 {
		delete(h.conns, sessionID)
	}
}

// emit entrega um evento aos ouvintes de uma sessão. `role` vazio é para todos;
// "gm" ou "player" recorta.
//
// NUNCA BLOQUEIA. Se a fila de um leitor está cheia, o quadro é DESCARTADO para
// ele e a conexão fica marcada para morrer — o `select` com `default` é o que
// impede um cliente lento de congelar o servidor inteiro, que é o modo de falha
// clássico de hub de broadcast. Quem perdeu quadro reconecta e busca o estado
// por HTTP, que é o mesmo caminho da primeira carga: nenhum estado vive só no
// fio.
func (h *sseHub) emit(sessionID int64, role string, event string, payload any) {
	frame, err := encodeFrame(event, payload)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, conn := range h.conns[sessionID] {
		if role != "" && conn.role != role {
			continue
		}
		select {
		case conn.frames <- frame:
		default:
			// Leitor lento: o quadro se perde de propósito.
		}
	}
}

// listeners conta quem está ouvindo — para teste e para o `liveSessions`.
func (h *sseHub) listeners(sessionID int64) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns[sessionID])
}
