package aovivo

// O laço do fluxo SSE (ALE-253), agora no pacote do regime (ALE-254).
//
// Ele saiu do `sse_events.go` porque não é transporte: o handler HTTP é que
// decide cabeçalhos e autorização, e isso ficou em `api/`. O que está aqui é a
// entrega em si — quem escreve quadro, quando bate o comentário-batida, e por
// quais três caminhos o fluxo termina.

import (
	"context"
	"io"
	"net/http"
	"time"
)

// O intervalo do comentário-batida. Ele NÃO é para detectar queda — disso o
// contexto cuida —, é para atravessar intermediário que fecha conexão ociosa: o
// proxy do Vite em desenvolvimento e qualquer coisa entre o mestre e a mesa
// numa rede doméstica. 25s fica confortavelmente abaixo dos 60s que é o padrão
// mais comum.
const Heartbeat = 25 * time.Second

// StreamFrames é o laço do fluxo, separado do handler para o teste exercitar o
// código DE VERDADE em vez de uma imitação com a mesma forma.
//
// Sai por três caminhos, e os três são normais: a fila fechou (o hub tirou a
// conexão), a escrita falhou (o cliente sumiu no meio de um quadro), ou o
// contexto foi cancelado (ele foi embora). Nenhum é erro a registrar.
func StreamFrames(
	ctx context.Context,
	w io.Writer,
	flusher http.Flusher,
	conn *SSEConn,
	batidaCada time.Duration,
) {
	batida := time.NewTicker(batidaCada)
	defer batida.Stop()

	for {
		select {
		case frame, aberta := <-conn.Frames:
			if !aberta {
				return
			}
			if _, err := w.Write(frame); err != nil {
				return
			}
			flusher.Flush()
		case <-batida.C:
			// Comentário SSE: o cliente ignora, o intermediário vê tráfego.
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}
