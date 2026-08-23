package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A tela de administração do piloto (ALE-219, segunda superfície).
//
// A diferença que ela existe para medir: NÃO há stream. Na Mesa, quem
// redesenhava era o SSE aberto; aqui quem redesenha é a RESPOSTA do próprio
// POST, que volta com o remendo do painel afetado. Nenhuma conexão longa,
// nenhum tique, nenhum comparador de hash — e, de quebra, a granularidade que
// eu tinha adiado na Mesa aparece aqui de graça, porque sem stream não faz
// sentido remendar a tela inteira.

// handleAdminPiloto desenha a tela inteira.
func (s *Server) handleAdminPiloto(w http.ResponseWriter, r *http.Request) {
	view, err := s.carregaAdmin(r.Context(), currentUser(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.escrevePagina(w, r, paginaPiloto{
		Titulo:        "Administração",
		TituloVisivel: "Administração",
		Voltar:        "/",
		// Sem `Init`: esta tela não abre stream nenhum. Os sinais existem só
		// para o diálogo e para os avisos — estado de INTERAÇÃO, não da
		// aplicação.
		Sinais: "{alvoId: 0, alvoNome: '', alvoCusto: '', copiado: '', erro: ''}",
	}, admin(view))
}

// handleAdminPilotoApagar apaga a conta e devolve os DOIS painéis que a conta
// tocava.
//
// Dois e não a tela inteira: apagar mexe na lista de jogadores e nas contagens
// do servidor, e não mexe nos convites. Enumerar os afetados é possível aqui
// porque a ação é conhecida — foi a mesma decisão que a SPA NÃO pôde tomar, e
// por isso ela invalida o prefixo `['admin']` inteiro e refaz as quatro
// leituras.
func (s *Server) handleAdminPilotoApagar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(w, r)
	// A REGRA é a mesma do handler HTTP — extraída para `deleteAccount` quando
	// esta tela precisou dela. O piloto não ganha uma segunda versão de "não se
	// apaga a própria conta"; se ganhasse, mediria a cópia.
	if _, _, err := s.deleteAccount(r, id, currentUser(r).ID); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": err.Error()})
		return
	}
	s.remendaPaineis(sse, r, painelJogadores, painelServidor)
}

// handleAdminPilotoBackup grava o backup e devolve só o painel do servidor.
func (s *Server) handleAdminPilotoBackup(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	if _, err := s.backupDatabase(r.Context(), time.Now()); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Não consegui fazer o backup: " + err.Error()})
		return
	}
	s.remendaPaineis(sse, r, painelServidor)
}

// painelAdmin é um painel da tela como FUNÇÃO, e não como nome.
//
// Era `renderFragmento("admin-jogadores", view)` até a ALE-227: uma string que
// só erra em runtime, e que exigia dois testes existindo apenas para afirmar
// que os nomes ainda casavam. Agora um painel que sumisse não compila.
type painelAdmin func(adminView) templ.Component

// remendaPaineis manda um `datastar-patch-elements` por painel.
//
// Cada fragmento carrega o próprio `id`, então o Datastar casa pelo id e o
// `selector` fica desnecessário — é o mesmo mecanismo do `#mesa`, só que
// apontado a pedaços em vez da tela toda.
func (s *Server) remendaPaineis(sse *datastar.ServerSentEventGenerator, r *http.Request, paineis ...painelAdmin) {
	view, err := s.carregaAdmin(r.Context(), currentUser(r))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erro": "Não consegui reler a tela."})
		return
	}
	for _, painel := range paineis {
		fragmento, err := renderFragmento(r.Context(), painel(view))
		if err != nil {
			continue
		}
		_ = sse.PatchElements(fragmento)
	}
	// Limpa o aviso anterior: sem isto, um erro de uma ação passada fica na
	// tela depois de a seguinte dar certo.
	_ = sse.MarshalAndPatchSignals(map[string]string{"erro": ""})
}
