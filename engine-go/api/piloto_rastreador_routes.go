package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/aovivo"
)

// OS COMANDOS DO MESTRE na Mesa em Datastar (ALE-265).
//
// Eles NÃO reusam as rotas da API JSON, e a escolha é a mesma das catorze
// fatias: a cena tem rota própria que chama a MESMA regra. O que impede as duas
// telas de divergirem não é compartilhar a rota — é compartilhar a regra e o
// store. A ALE-122 aconteceu com dois transportes escrevendo em dois LUGARES,
// não em dois caminhos.
//
// A autorização é a que já existe: o `sessionForCaller` resolve o papel pelo
// mesmo caminho que a API usa, e papel desconhecido cai em jogador. Esconder o
// botão é UX; a trava é aqui.

func (s *Server) rotasDoRastreador(r chi.Router) {
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/next-turn", s.comandoDoMestre(
		func(st *Server, sessionID int64) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.NextTurn(sessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/previous-turn", s.comandoDoMestre(
		func(st *Server, sessionID int64) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.PreviousTurn(sessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/scene/start", s.comandoDoMestre(
		func(st *Server, sessionID int64) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.StartScene(sessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/scene/end", s.comandoDoMestre(
		func(st *Server, sessionID int64) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.EndScene(sessionID)
		}))
}

// comandoDoMestre é o caminho único dos quatro comandos.
//
// Eles só diferem na MUTAÇÃO, e o resto — resolver a mesa, exigir o papel,
// publicar para a SPA, redesenhar a cena — é idêntico. Sem o parâmetro seriam
// quatro cópias, e é numa delas que alguém esquece de publicar e a mesa fica
// vendo o turno velho.
func (s *Server) comandoDoMestre(
	mutar func(*Server, int64) (*aovivo.SessionRuntimeState, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, sessionID, ok := mesaParams(w, r)
		if !ok {
			return
		}
		user := currentUser(r)
		_, papel, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		// A trava é aqui e não na tela: quem postar na mão leva 403, e o botão
		// escondido é só cortesia para quem não pode.
		if papel != "gm" {
			http.Error(w, "só o mestre comanda a mesa", http.StatusForbidden)
			return
		}

		estado, err := mutar(s, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// A SPA continua ouvindo o hub: enquanto as duas telas existirem, uma
		// escrita por aqui tem de chegar lá.
		s.publishSessionState(sessionID, estado)

		// E a cena de quem clicou é remendada NA HORA, em vez de esperar o
		// próximo tique do stream. O stream avisa-e-relê, então ele veria a
		// mesma coisa daqui a até 200ms e o hash o faria calar — o remendo aqui
		// é o que torna o botão mais clicado da sessão instantâneo.
		s.remendaAMesa(w, r, user, campaignID, sessionID)
	}
}

// remendaAMesa redesenha a cena para quem acabou de comandar.
func (s *Server) remendaAMesa(
	w http.ResponseWriter, r *http.Request,
	user AuthUser, campaignID, sessionID int64,
) {
	view, _, err := s.loadMesaView(r.Context(), user, campaignID, sessionID)
	if err != nil {
		// A mutação já aconteceu e já foi publicada; falhar ao redesenhar não a
		// desfaz. O stream corrige no próximo tique, e devolver erro aqui faria
		// a tela dizer que o comando falhou quando ele funcionou.
		return
	}
	sse := datastar.NewSSE(w, r)
	fragmento, err := renderFragmento(r.Context(), mesa(view))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragmento)
}
