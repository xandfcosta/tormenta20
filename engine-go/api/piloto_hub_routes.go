package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// As rotas do HUB (ALE-231). Atrás de `requireAuth`: o menu principal é de quem
// entrou, e quem não entrou é mandado à porta pelo próprio middleware.

func (s *Server) rotasDoHub(r chi.Router) {
	r.Get("/", s.handleHub)
	r.Post("/sair", s.handleHubSair)
	r.Post("/convites", s.handleHubConvite)
}

func (s *Server) handleHub(w http.ResponseWriter, r *http.Request) {
	view, err := s.carregaHub(r.Context(), currentUser(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: "Tormenta 20",
		Forma:  cascaTitulo,
		Kicker: "— Grimório de Arton —",
		// Sem `data-voltar`: o Hub é a cena raiz e o Esc não tem para onde ir.
	}, hub(view))
}

// handleHubSair apaga o cookie e devolve à porta.
//
// POST e não GET, e isso é a regra de CSRF desta migração inteira: o cookie é
// `SameSite=Lax`, e Lax NÃO manda cookie em POST cross-site — então formulário
// autenticado está protegido pela política do cookie, sem token. O que Lax
// permite é navegação de topo por GET, e é por isso que nenhuma ação com efeito
// pode viver num GET. Um `<a href="/sair">` seria dispensável por qualquer
// imagem de terceiro.
func (s *Server) handleHubSair(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, s.sessionCookie("", -1))
	http.Redirect(w, r, "/piloto/entrar", http.StatusSeeOther)
}

// handleHubConvite cunha o link de conta e devolve o remendo com ele.
func (s *Server) handleHubConvite(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	if !currentUser(r).IsAdmin {
		// A tela não oferece o botão para quem não é admin, mas a trava é aqui:
		// a UI decide o que MOSTRAR, o servidor decide o que ACONTECE.
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Só quem administra pode convidar."})
		return
	}
	invite, err := s.mintAccountInvite(r.Context(), currentUser(r).ID)
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": avisoInterno})
		return
	}
	// Só o CAMINHO: quem prefixa a origem é o navegador. Ver `conviteGerado`.
	fragmento, err := renderFragmento(r.Context(), conviteGerado("/register?convite="+invite.Token))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": avisoInterno})
		return
	}
	_ = sse.PatchElements(fragmento)
}
