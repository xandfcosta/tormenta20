package api

import (
	"crypto/sha256"
	"net/http"
	"time"
)

// O stream da Mesa — o piloto Datastar (ALE-219).
//
// A cadência é um TIQUE e não um gatilho por evento, e isso é escolha, não
// preguiça: o gatilho exigiria que a página se pendurasse no barramento do
// socket.io, o que ataria o piloto ao transporte que ele existe para comparar.
// O tique lê a MESMA fonte que o socket lê (`sessionStore`), então os dois
// caminhos nunca podem discordar sobre o estado — só sobre o atraso.
//
// O custo do tique é uma renderização a cada 200ms por jogador conectado. Numa
// mesa de seis isso é 30 renderizações por segundo de um HTML de poucos KB, e
// nada sai do servidor enquanto ninguém mexe: o comparador abaixo é o que
// transforma "re-renderizar sempre" em "mandar quando muda".
func (s *Server) handleMesaStream(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado por este servidor", http.StatusInternalServerError)
		return
	}
	user := currentUser(r)
	// Uma primeira leitura ANTES dos cabeçalhos: sem acesso, o jogador merece um
	// 403 legível e não um stream aberto que nunca manda nada.
	view, status, err := s.loadMesaView(r.Context(), user, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	// O socket.io já dispensa buffer de proxy; isto diz o mesmo para o SSE.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ultimo := s.escreveMesa(w, flusher, view, [32]byte{})
	ticker := time.NewTicker(mesaTick)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			// O jogador fechou a aba, trocou de superfície ou perdeu a rede. Sair
			// aqui é o que impede a goroutine de sobreviver ao leitor.
			return
		case <-ticker.C:
			view, _, err := s.loadMesaView(r.Context(), user, campaignID, sessionID)
			if err != nil {
				// Um erro passageiro (banco ocupado) não derruba o stream: o
				// próximo tique tenta de novo, e a tela continua mostrando o
				// último estado bom em vez de piscar.
				continue
			}
			ultimo = s.escreveMesa(w, flusher, view, ultimo)
		}
	}
}

// escreveMesa manda o fragmento SÓ quando o HTML mudou, e devolve a impressão
// digital do que foi mandado.
//
// Comparar o HTML RENDERIZADO e não o estado é de propósito: o `refreshCharacterMaxes`
// devolve struct nova a cada leitura, então comparar estado por igualdade
// mandaria tudo a cada tique; e comparar campo a campo seria uma lista que
// envelhece — exatamente o defeito que o `cloneState` documenta ter tido
// quando o `TurnsTaken` entrou. O HTML é o que o jogador vê, e é a única coisa
// cuja mudança justifica um byte no fio.
func (s *Server) escreveMesa(w http.ResponseWriter, flusher http.Flusher, view mesaView, anterior [32]byte) [32]byte {
	fragmento, err := renderMesaFragment(view)
	if err != nil {
		return anterior
	}
	digital := sha256.Sum256(fragmento)
	if digital == anterior {
		return anterior
	}
	if _, err := w.Write([]byte(patchElementsEvent(fragmento))); err != nil {
		return anterior
	}
	flusher.Flush()
	return digital
}
