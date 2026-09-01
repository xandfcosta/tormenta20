package api

import (
	"net/http"
	"t20engine/plataforma"

	"t20engine/catalog"
)

// handleHealth diz se o servidor está VIVO e se está INTEIRO — duas coisas
// diferentes, e o app já subiu no estado do meio sem ninguém saber (ALE-155).
//
// O boot é best-effort de propósito: sem os catálogos, autenticação, leitura e
// vitais continuam funcionando, e derrubar a mesa por causa disso seria pior.
// Mas até agora essa degradação só existia numa linha de log, enquanto o
// `/health` respondia "ok" — e os handlers que precisam do catálogo devolvem
// 503 lá na frente, no meio de uma jogada.
//
// Continua 200 mesmo degradado, e isso é decisão: nada aqui se conserta
// reiniciando o processo (falta um ARQUIVO), então responder 503 só criaria um
// laço de reinício em quem monitora. Quem quer saber, lê o corpo.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	degraded := []string{}
	if s.catalogs == nil {
		degraded = append(degraded, "catalogs")
	}
	if !catalog.ActivationsLoaded() {
		degraded = append(degraded, "activations")
	}
	if len(degraded) == 0 {
		plataforma.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"status": "degraded", "degraded": degraded})
}

// SondaDeSaude é o `/health` na RAIZ, ao lado do `/api/health`.
//
// São dois endereços para a mesma resposta, e isso é deliberado: quem pergunta
// da raiz não é o app, é a INFRAESTRUTURA — o `healthcheck` do compose, o
// `-health` do próprio binário, um monitor externo —, e nenhum deles sabe que a
// API mora sob um prefixo. Quando a API saiu da raiz (ALE-272, fatia 10c) o
// `/health` foi junto sem que ninguém pensasse nele, e o sintoma foi o CI
// esperando trinta segundos por uma sonda que respondia 404 num servidor que já
// estava escutando — a mensagem dizia "o servidor não subiu".
func (s *Server) SondaDeSaude() http.Handler {
	return http.HandlerFunc(s.handleHealth)
}
