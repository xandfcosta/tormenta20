package admin

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/accounts"
	"t20engine/serve/web/ui"
)

// A tela de administração do app.
//
// NÃO há stream aqui. Quem redesenha é a RESPOSTA do próprio POST, que volta
// com o remendo do painel afetado: nenhuma conexão longa, nenhum tique, nenhum
// comparador de hash. É o que dá a granularidade de graça — sem stream não faz
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
		Sinais: "{target_id: 0, target_name: '', target_cost: '', copied: '', error: ''}",
	}, adminScene(view))
}

// handleDeleteAccount apaga a conta e devolve os DOIS painéis que a conta
// tocava.
//
// Dois e não a tela inteira: apagar mexe na lista de jogadores e nas contagens
// do servidor, e não mexe nos convites. Enumerar os afetados é possível porque
// a ação é conhecida.
func (s Scene) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(w, r)
	// A REGRA mora no `accounts.Roster`: o app não ganha uma segunda versão de
	// "não se apaga a própria conta".
	if _, err := s.roster.Delete(r.Context(), s.deps.CurrentUserID(r), id); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": deleteRefusal(err)})
		return
	}
	s.patchPanels(sse, r, playersPanel, serverPanel)
}

// deleteRefusal escolhe a frase que o dono lê.
//
// A recusa de apagar a PRÓPRIA conta é a única que vale ser dita com todas as
// letras — ela acontece por um gesto, não por um defeito. O resto é interno, e
// despejar o `err.Error()` na tela mostraria ao dono o texto de uma transação
// que falhou.
func deleteRefusal(err error) string {
	if errors.Is(err, accounts.ErrCannotDeleteSelf) {
		return "Você não pode apagar a própria conta."
	}
	return ui.NoticeInternal
}

// handleBackup grava o backup e devolve só o painel do servidor.
func (s Scene) handleBackup(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	if err := s.deps.BackupNow(r.Context(), time.Now()); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": "Não consegui fazer o backup: " + err.Error()})
		return
	}
	s.patchPanels(sse, r, serverPanel)
}

// adminPanel é um painel da tela como FUNÇÃO, e não como nome.
//
// Um nome de fragmento em string só erra em runtime, e exige um teste que
// exista apenas para afirmar que os nomes ainda casam. Assim, um painel que
// sumir não compila.
type adminPanel func(adminView) templ.Component

// patchPanels manda um `datastar-patch-elements` por painel.
//
// Cada fragmento carrega o próprio `id`, então o Datastar casa pelo id e o
// `selector` fica desnecessário — é o mesmo mecanismo do `#table`, só que
// apontado a pedaços em vez da tela toda.
func (s Scene) patchPanels(sse *datastar.ServerSentEventGenerator, r *http.Request, paineis ...adminPanel) {
	view, err := s.loadAdmin(r.Context(), s.deps.CurrentUserID(r))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": "Não consegui reler a tela."})
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
	_ = sse.MarshalAndPatchSignals(map[string]string{"error": ""})
}

// handleMintReset cunha o link de redefinição e devolve o remendo
// com ele. Nada mais muda na tela: gerar um link não altera jogador, convite
// nem servidor, então não há painel a remendar.
//
// A REGRA vem do `accounts.Resets`: o app não ganha uma segunda versão do
// prazo de 24h, que é como duas telas divergem sem ninguém notar.
func (s Scene) handleMintReset(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(w, r)
	reset, err := s.resets.Mint(r.Context(), id, s.deps.CurrentUserID(r))
	if errors.Is(err, accounts.ErrUnknownAccount) {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": "Essa conta não existe mais."})
		return
	}
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": ui.NoticeInternal})
		return
	}
	// Só o CAMINHO: quem prefixa a origem é o navegador. Ver `mintedReset`.
	fragmento, err := ui.RenderFragment(r.Context(), mintedReset("/redefinir-senha?token="+url.QueryEscape(reset.Token)))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": ui.NoticeInternal})
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
	invite, err := s.gate.MintInvite(r.Context(), s.deps.CurrentUserID(r))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": ui.NoticeInternal})
		return
	}
	fragmento, err := ui.RenderFragment(r.Context(), ui.MintedInvite("/register?convite="+url.QueryEscape(invite.Token),
		"Cada convite serve para UMA conta. Gere outro para o próximo jogador."))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"error": ui.NoticeInternal})
		return
	}
	_ = sse.PatchElements(fragmento)
	s.patchPanels(sse, r, invitesPanel)
}

// Routes monta a administração no roteador de quem a hospeda.
//
// Os endereços moram AQUI e não em quem monta: a cena é a dona do que ela
// atende. Soltas num `routes.go` comum, as linhas de uma cena ficam misturadas
// com as das outras.
func Routes(r chi.Router, s Scene) {
	r.Get("/admin", s.handleAdmin)
	r.Post("/admin/usuarios/{id}/apagar", s.handleDeleteAccount)
	r.Post("/admin/backup", s.handleBackup)
	r.Post("/admin/usuarios/{id}/redefinir", s.handleMintReset)
	r.Post("/admin/convites", s.handleMintInvite)
}
