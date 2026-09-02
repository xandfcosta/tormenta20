package hub

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"t20engine/web/ui"
)

// As rotas do HUB (ALE-231). Atrás de `requireAuth`: o menu principal é de quem
// entrou, e quem não entrou é mandado à porta pelo próprio middleware.

// Routes monta o hub no roteador de quem o hospeda.
//
// Os endereços moram AQUI (ALE-278): a cena é a dona do que ela atende.
func Routes(r chi.Router, s Scene) {
	r.Get("/", s.handleHub)
	r.Post("/sair", s.handleHubSignOut)
	r.Post("/convites", s.handleHubInvite)
}

func (s Scene) handleHub(w http.ResponseWriter, r *http.Request) {
	view, err := s.loadHub(r.Context(), s.deps.CurrentViewer(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: "Tormenta 20",
		Forma:  ui.ShellTitled,
		Kicker: "— Grimório de Arton —",
		// Sem `data-voltar`: o Hub é a cena raiz e o Esc não tem para onde ir.
	}, hub(view))
}

// handleHubSignOut apaga o cookie e devolve à porta.
//
// POST e não GET, e isso é a regra de CSRF desta migração inteira: o cookie é
// `SameSite=Lax`, e Lax NÃO manda cookie em POST cross-site — então formulário
// autenticado está protegido pela política do cookie, sem token. O que Lax
// permite é navegação de topo por GET, e é por isso que nenhuma ação com efeito
// pode viver num GET. Um `<a href="/sair">` seria dispensável por qualquer
// imagem de terceiro.
func (s Scene) handleHubSignOut(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, s.deps.ExpiredSessionCookie())
	http.Redirect(w, r, "/entrar", http.StatusSeeOther)
}

// handleHubInvite cunha o link de conta e devolve o remendo com ele.
func (s Scene) handleHubInvite(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	if !s.deps.CurrentViewer(r).IsAdmin {
		// A tela não oferece o botão para quem não é admin, mas a trava é aqui:
		// a UI decide o que MOSTRAR, o servidor decide o que ACONTECE.
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Só quem administra pode convidar."})
		return
	}
	invite, err := s.deps.MintAccountInvite(r.Context(), s.deps.CurrentViewer(r).ID)
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": internalNotice})
		return
	}
	// Só o CAMINHO: quem prefixa a origem é o navegador. Ver `conviteGerado`.
	fragmento, err := ui.RenderFragment(r.Context(), ui.MintedInvite("/register?convite="+invite.Token))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": internalNotice})
		return
	}
	_ = sse.PatchElements(fragmento)
}

// internalNotice é a frase que a pessoa lê quando algo do servidor falhou.
//
// CÓPIA da porta de entrada, e consciente: é texto de tela, e cada cena é dona
// do que ela escreve. Uma constante compartilhada obrigaria as duas a mudarem
// juntas para sempre por causa de uma frase de dez palavras.
const internalNotice = "Não consegui completar agora. Tente de novo."
