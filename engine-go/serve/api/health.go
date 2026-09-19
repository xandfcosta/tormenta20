package api

import (
	"net/http"
	"t20engine/infra/httpio"

	"t20engine/domain/catalog"
)

// handleHealth diz se o servidor está VIVO e se está INTEIRO — duas coisas
// diferentes, e o app já subiu no estado do meio sem ninguém saber (ALE-155).
//
// O CATÁLOGO deixou de ser degradação possível em produção (ALE-355): o PV
// máximo passou a ser derivado dele, e o `primeCatalogs` do `cmd/api` agora
// DERRUBA o processo em vez de subir servindo ficha errada em silêncio. O boot
// era best-effort com a promessa de que "vitais continuam funcionando sem eles",
// e essa promessa morreu junto com a coluna.
//
// A linha do catálogo fica porque o `/health` também roda em BANCADA, onde
// montar um servidor sem catálogo é arranjo legítimo — e porque um relatório
// que só sabe dizer "ok" não serve para descobrir o estado do meio.
//
// O que continua degradação de verdade são as ATIVAÇÕES, que carregam preguiçosa
// e podem faltar sem impedir o arranque.
//
// Continua 200 mesmo degradado, e isso é decisão: nada aqui se conserta
// reiniciando o processo, então responder 503 só criaria um laço de reinício em
// quem monitora. Quem quer saber, lê o corpo.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	degraded := []string{}
	if s.catalogs == nil {
		degraded = append(degraded, "catalogs")
	}
	if !catalog.ActivationsLoaded() {
		degraded = append(degraded, "activations")
	}
	if len(degraded) == 0 {
		httpio.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		return
	}
	httpio.WriteJSON(w, http.StatusOK, map[string]any{"status": "degraded", "degraded": degraded})
}

// HealthProbe é o `/health` na RAIZ, ao lado do `/api/health`.
//
// São dois endereços para a mesma resposta, e isso é deliberado: quem pergunta
// da raiz não é o app, é a INFRAESTRUTURA — o `healthcheck` do compose, o
// `-health` do próprio binário, um monitor externo —, e nenhum deles sabe que a
// API mora sob um prefixo. Quando a API saiu da raiz (ALE-272, fatia 10c) o
// `/health` foi junto sem que ninguém pensasse nele, e o sintoma foi o CI
// esperando trinta segundos por uma sonda que respondia 404 num servidor que já
// estava escutando — a mensagem dizia "o servidor não subiu".
func (s *Server) HealthProbe() http.Handler {
	return http.HandlerFunc(s.handleHealth)
}
