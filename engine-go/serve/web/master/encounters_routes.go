package master

import (
	"net/http"
	"strconv"
	"strings"

	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// As rotas e os handlers dos ENCONTROS — separados do `routes.go` na ALE-278.
//
// O REGISTRO das trinta rotas continua num lugar só, no `routes.go`: é ele
// que o `api` chama, e espalhá-lo faria a cena ter quatro portas de entrada.
// O que mora aqui são os handlers desta ferramenta. O arquivo único tinha
// 600 linhas e QUATRO famílias que não se chamam — arquivo é unidade de
// RESPONSABILIDADE e de conflito de merge, não de leitura.

// handleEncounters serve os dois casos numa rota. Sem autorização própria: o
// bestiário é o LIVRO, e o rascunho vive no navegador de quem monta.
func (s Scene) handleEncounters(w http.ResponseWriter, r *http.Request) {
	s.respondEncounter(w, r, nil)
}

// handleEncounterAdd serve o "acrescentar" do painel de busca E o `[+]` da
// linha, porque a álgebra é a MESMA: `acrescenta` sobe a contagem quando a
// criatura já está no encontro. Duas rotas para uma função é o que evita a
// tela ter dois caminhos que podem divergir.
func (s Scene) handleEncounterAdd(w http.ResponseWriter, r *http.Request) {
	s.respondEncounter(w, r, addRow)
}

func (s Scene) handleEncounterLess(w http.ResponseWriter, r *http.Request) {
	s.respondEncounter(w, r, lessRow)
}

func (s Scene) handleEncounterRemove(w http.ResponseWriter, r *http.Request) {
	s.respondEncounter(w, r, removeRow)
}

// respondEncounter lê o rascunho, aplica UM gesto e devolve a cena.
//
// A álgebra chega como função porque as quatro rotas só diferem nisso, e o
// resto — ler sinais, recalcular, remendar — é idêntico. Sem o parâmetro,
// seriam quatro cópias do mesmo handler, que é onde uma delas esquece de
// recalcular.
func (s Scene) respondEncounter(
	w http.ResponseWriter, r *http.Request,
	gesture func([]encounterRow, string) []encounterRow,
) {
	level, group, rows, search := draftFromRequest(r)
	if gesture != nil {
		rows = gesture(rows, chi.URLParam(r, "id"))
	}
	v := loadEncounters(level, group, rows, search)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragment, err := ui.RenderFragment(r.Context(), encountersScene(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragment)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:         "Encontros · Mesa do Mestre · Tormenta 20",
		Forma:         ui.ShellDense,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, masterBody("encontros", encountersScene(v)))
}

// draftFromRequest lê o encontro da URL na carga fria e dos SINAIS no remendo.
//
// A URL é o caminho do LINK COPIADO: `?nivel=3&grupo=4&c=goblin:4,ogro:1`. Os
// sinais são o caminho de todo o resto, e vencem quando existem — eles são o
// que o mestre acabou de clicar.
func draftFromRequest(r *http.Request) (int, int, []encounterRow, string) {
	q := r.URL.Query()
	level := numberFromURL(q.Get("nivel"), nivelPadrao)
	group := numberFromURL(q.Get("grupo"), grupoPadrao)
	rows := rowsFromURL(q.Get("c"))
	search := q.Get("busca")

	signals := struct {
		Level          *int            `json:"nivel"`
		Group          *int            `json:"grupo"`
		Encounter      *[]encounterRow `json:"encontro"`
		CreatureSearch *string         `json:"creature_search"`
	}{}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return level, group, rows, search
	}
	if signals.Level != nil {
		level = *signals.Level
	}
	if signals.Group != nil {
		group = *signals.Group
	}
	if signals.Encounter != nil {
		rows = *signals.Encounter
	}
	if signals.CreatureSearch != nil {
		search = *signals.CreatureSearch
	}
	return level, group, rows, search
}

func numberFromURL(raw string, standard int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return standard
	}
	return n
}
