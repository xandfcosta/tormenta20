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
		func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.NextTurn(c.SessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/initiative/previous-turn", s.comandoDoMestre(
		func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.PreviousTurn(c.SessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/scene/start", s.comandoDoMestre(
		func(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
			return st.sessions.StartScene(c.SessionID)
		}))
	r.Post("/mesa/{campaignId}/{sessionId}/scene/end", s.comandoDoMestre(encerraACena))
}

// mesaComando é o que a mutação de um comando do mestre recebe.
//
// Os quatro primeiros só precisavam do id da SESSÃO, e a assinatura era um
// `int64` — foi essa economia que deixou passar o defeito que este arquivo
// acabou de consertar: `encerrar cena` precisa da CAMPANHA, porque é de lá que
// vem o grupo cujas fichas expiram, e não tendo como recebê-la ela chamou o
// helper que não precisa dela e faz menos.
type mesaComando struct {
	R          *http.Request
	User       AuthUser
	CampaignID int64
	SessionID  int64
}

// encerraACena é o gesto INTEIRO, e a razão de ser função nomeada em vez de um
// literal na lista acima é que ela faz duas coisas que as outras três não fazem.
//
// A primeira é a REGRESSÃO da ALE-220, reaberta por este piloto: `EndScene` do
// store só mexe no rastreador, então a fila zerava na tela e a bênção de duração
// "cena" continuava viva na FICHA. O livro não deixa margem — "a habilidade dura
// uma cena inteira, encerrando-se quando esse momento da história acaba" (p227)
// —, e o `endSceneForTable` é o caminho único que expira as fichas do grupo
// ANTES de desligar a cena. Aqui é a mesma chamada e não a mesma sequência
// reescrita: gesto repetido é gesto que diverge.
//
// A segunda é o aviso: as fichas não estão no estado do rastreador, então sem o
// `session-rest` o efeito morto e o "usado 1/cena" ficariam na tela da SPA até
// alguém recarregar.
func encerraACena(st *Server, c mesaComando) (*aovivo.SessionRuntimeState, error) {
	estado, err := st.endSceneForTable(c.User, c.CampaignID, c.SessionID)
	if err != nil {
		return nil, err
	}
	st.sse.Emit(c.SessionID, "", "session-rest", map[string]any{
		"sessionId": c.SessionID, "scope": "scene",
	})
	return estado, nil
}

// comandoDoMestre é o caminho único dos quatro comandos.
//
// Eles só diferem na MUTAÇÃO, e o resto — resolver a mesa, exigir o papel,
// publicar para a SPA, redesenhar a cena — é idêntico. Sem o parâmetro seriam
// quatro cópias, e é numa delas que alguém esquece de publicar e a mesa fica
// vendo o turno velho.
func (s *Server) comandoDoMestre(
	mutar func(*Server, mesaComando) (*aovivo.SessionRuntimeState, error),
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

		estado, err := mutar(s, mesaComando{
			R: r, User: user, CampaignID: campaignID, SessionID: sessionID,
		})
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
