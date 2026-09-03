package master

import (
	"net/http"
	"strconv"
	"strings"

	"t20engine/web/ui"

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
	gesto func([]encounterRow, string) []encounterRow,
) {
	nivel, grupo, linhas, busca := draftFromRequest(r)
	if gesto != nil {
		linhas = gesto(linhas, chi.URLParam(r, "id"))
	}
	v := loadEncounters(nivel, grupo, linhas, busca)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), encountersScene(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:        "Encontros · Mesa do Mestre · Tormenta 20",
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
	nivel := numberFromURL(q.Get("nivel"), nivelPadrao)
	grupo := numberFromURL(q.Get("grupo"), grupoPadrao)
	linhas := rowsFromURL(q.Get("c"))
	busca := q.Get("busca")

	sinais := struct {
		Nivel         *int            `json:"nivel"`
		Grupo         *int            `json:"grupo"`
		Encontro      *[]encounterRow `json:"encontro"`
		BuscaCriatura *string         `json:"buscaCriatura"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return nivel, grupo, linhas, busca
	}
	if sinais.Nivel != nil {
		nivel = *sinais.Nivel
	}
	if sinais.Grupo != nil {
		grupo = *sinais.Grupo
	}
	if sinais.Encontro != nil {
		linhas = *sinais.Encontro
	}
	if sinais.BuscaCriatura != nil {
		busca = *sinais.BuscaCriatura
	}
	return nivel, grupo, linhas, busca
}

func numberFromURL(bruto string, padrao int) int {
	n, err := strconv.Atoi(strings.TrimSpace(bruto))
	if err != nil {
		return padrao
	}
	return n
}
