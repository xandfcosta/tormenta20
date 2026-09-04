package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"t20engine/web/ui"
)

// A tela de administração do piloto (ALE-219, segunda superfície).
//
// A diferença que ela existe para medir: NÃO há stream. Na Mesa, quem
// redesenhava era o SSE aberto; aqui quem redesenha é a RESPOSTA do próprio
// POST, que volta com o remendo do painel afetado. Nenhuma conexão longa,
// nenhum tique, nenhum comparador de hash — e, de quebra, a granularidade que
// eu tinha adiado na Mesa aparece aqui de graça, porque sem stream não faz
// sentido remendar a tela inteira.

// handleAdmin desenha a tela inteira.
func (s Scene) handleAdmin(w http.ResponseWriter, r *http.Request) {
	view, err := s.loadAdmin(r.Context(), s.deps.CurrentUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:        "Administração",
		Forma:         ui.ShellDense,
		TituloVisivel: "Administração",
		Voltar:        "/",
		// Sem `Init`: esta tela não abre stream nenhum. Os sinais existem só
		// para o diálogo e para os avisos — estado de INTERAÇÃO, não da
		// aplicação.
		Sinais: "{alvoId: 0, alvoNome: '', alvoCusto: '', copiado: '', erro: ''}",
	}, adminScene(view))
}

// handleDeleteAccount apaga a conta e devolve os DOIS painéis que a conta
// tocava.
//
// Dois e não a tela inteira: apagar mexe na lista de jogadores e nas contagens
// do servidor, e não mexe nos convites. Enumerar os afetados é possível aqui
// porque a ação é conhecida — foi a mesma decisão que a SPA NÃO pôde tomar, e
// por isso ela invalida o prefixo `['admin']` inteiro e refaz as quatro
// leituras.
func (s Scene) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(w, r)
	// A REGRA é a mesma do handler HTTP — extraída para `deleteAccount` quando
	// esta tela precisou dela. O piloto não ganha uma segunda versão de "não se
	// apaga a própria conta"; se ganhasse, mediria a cópia.
	if err := s.deps.DeleteAccount(r, id, s.deps.CurrentUserID(r)); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": err.Error()})
		return
	}
	s.patchPanels(sse, r, playersPanel, serverPanel)
}

// handleBackup grava o backup e devolve só o painel do servidor.
func (s Scene) handleBackup(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	if err := s.deps.BackupNow(r.Context(), time.Now()); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Não consegui fazer o backup: " + err.Error()})
		return
	}
	s.patchPanels(sse, r, serverPanel)
}

// adminPanel é um painel da tela como FUNÇÃO, e não como nome.
//
// Era `renderFragmento("admin-jogadores", view)` até a ALE-227: uma string que
// só erra em runtime, e que exigia dois testes existindo apenas para afirmar
// que os nomes ainda casavam. Agora um painel que sumisse não compila.
type adminPanel func(adminView) templ.Component

// patchPanels manda um `datastar-patch-elements` por painel.
//
// Cada fragmento carrega o próprio `id`, então o Datastar casa pelo id e o
// `selector` fica desnecessário — é o mesmo mecanismo do `#mesa`, só que
// apontado a pedaços em vez da tela toda.
func (s Scene) patchPanels(sse *datastar.ServerSentEventGenerator, r *http.Request, paineis ...adminPanel) {
	view, err := s.loadAdmin(r.Context(), s.deps.CurrentUserID(r))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Não consegui reler a tela."})
		return
	}
	for _, painel := range paineis {
		fragmento, err := ui.RenderFragment(r.Context(), painel(view))
		if err != nil {
			continue
		}
		_ = sse.PatchElements(fragmento)
	}
	// Limpa o aviso anterior: sem isto, um erro de uma ação passada fica na
	// tela depois de a seguinte dar certo.
	_ = sse.MarshalAndPatchSignals(map[string]string{"erro": ""})
}

// handleMintReset cunha o link de redefinição e devolve o remendo
// com ele. Nada mais muda na tela: gerar um link não altera jogador, convite
// nem servidor, então não há painel a remendar.
//
// A REGRA vem do `mintPasswordReset`, extraída do manipulador JSON quando esta
// tela precisou dela — sétima vez que a migração encontra regra soldada ao
// transporte, e a mesma resposta das outras seis. O piloto não ganha uma
// segunda versão do prazo de 24h; se ganhasse, as duas telas poderiam divergir
// sem ninguém notar.
func (s Scene) handleMintReset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(w, r)
	reset, err := s.deps.MintPasswordReset(r.Context(), id, s.deps.CurrentUserID(r))
	if s.deps.IsUnknownUser(err) {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Essa conta não existe mais."})
		return
	}
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": ui.NoticeInternal})
		return
	}
	// Só o CAMINHO: quem prefixa a origem é o navegador. Ver `mintedReset`.
	fragmento, err := ui.RenderFragment(r.Context(), mintedReset("/redefinir-senha?token="+url.QueryEscape(reset.Token)))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": ui.NoticeInternal})
		return
	}
	_ = sse.PatchElements(fragmento)
}

// handleMintInvite cunha o convite e devolve DOIS remendos: o link e o
// painel de convites.
//
// O segundo é o que separa esta rota da do Hub, e ele não é enfeite: cunhar
// muda a LISTA que está a três centímetros do botão, e sem remendá-la a tela
// diz "Convites abertos (0)" logo depois de a pessoa abrir um. No Hub não há
// essa lista, e por isso lá basta o link — mesma regra, transportes diferentes.
func (s Scene) handleMintInvite(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	invite, err := s.deps.MintAccountInvite(r.Context(), s.deps.CurrentUserID(r))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": ui.NoticeInternal})
		return
	}
	fragmento, err := ui.RenderFragment(r.Context(), ui.MintedInvite("/register?convite="+url.QueryEscape(invite.Token)))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": ui.NoticeInternal})
		return
	}
	_ = sse.PatchElements(fragmento)
	s.patchPanels(sse, r, invitesPanel)
}

// Routes monta a administração no roteador de quem a hospeda.
//
// Os endereços moram AQUI e não em quem monta (ALE-278): a cena é a dona do que
// ela atende. Antes desta fatia as cinco linhas estavam soltas no
// `routes.go`, entre as rotas de outras três cenas — esta é a
// primeira que sai sem nem ter um `Routes` próprio para mover.
func Routes(r chi.Router, s Scene) {
	r.Get("/admin", s.handleAdmin)
	r.Post("/admin/usuarios/{id}/apagar", s.handleDeleteAccount)
	r.Post("/admin/backup", s.handleBackup)
	r.Post("/admin/usuarios/{id}/redefinir", s.handleMintReset)
	r.Post("/admin/convites", s.handleMintInvite)
}
