package master

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"

	"t20engine/serve/web/ui"

	"github.com/starfederation/datastar-go/datastar"
)

// As rotas e os handlers dos CATÁLOGOS, separados do `routes.go`.
//
// O REGISTRO das trinta rotas continua num lugar só, no `routes.go`: é ele que o
// `api` chama, e espalhá-lo faria a cena ter quatro portas de entrada. O que
// mora aqui são os handlers desta ferramenta — arquivo é unidade de
// RESPONSABILIDADE e de conflito de merge, não de leitura.

// handleCollection serve os dois casos numa rota, como as outras cenas.
//
// Sem autorização própria e pelo mesmo motivo do bestiário: o catálogo é o
// LIVRO, igual para todo mundo. O `requirePage` do grupo já exige sessão.
func (s Scene) handleCollection(w http.ResponseWriter, r *http.Request) {
	v := loadCollection(collectionCriteriaFromRequest(r), s.deps.BookAddress())

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragment, err := ui.RenderFragment(r.Context(), collectionScene(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragment)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:         tabLabel(v.Aba) + " · Mesa do Mestre · Tormenta 20",
		Forma:         ui.ShellDense,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, masterBody(v.Aba, collectionScene(v)))
}

// filtersFromURL lê os crachás que a consulta pede, só os DA ABA: `?circulo=2&circulo=3`
// na cena das condições não filtra nada, e aceitá-lo faria a cena carregar um
// estado que ela não sabe desenhar.
func filtersFromURL(q url.Values, aba string) map[string][]string {
	outside := map[string][]string{}
	for _, f := range filtersForTab(knownTab(aba)) {
		if values := q[f.Key]; len(values) > 0 {
			outside[f.Key] = values
		}
	}
	return outside
}

// collectionCriteriaFromRequest lê a busca e a ENTRADA da URL na carga fria e
// dos SINAIS quando o Datastar chama — mesma decisão das outras cenas, e é ela
// que faz `?busca=fogo` e `?entrada=medo` serem endereços que se recarregam.
//
// `entrada` NÃO vem de sinal, e é deliberado: ela é um endereço para UM verbete,
// escrito por um elo ou colado por alguém. Vindo de sinal, ela sobreviveria à
// próxima tecla digitada na busca e a cena ficaria presa num verbete só.
func collectionCriteriaFromRequest(r *http.Request) collectionCriteria {
	q := r.URL.Query()
	// A ABA vem DO CAMINHO E SÓ DELE, porque o `Routes` registra uma rota por
	// slug conhecido: `path.Base` aqui é sempre uma aba da fileira.
	//
	// Havia um `?aba=` de reserva para o `/mestre/catalogos`, que saiu na
	// ALE-331, e ele não era só código morto: um caso de e2e pediu
	// `/mestre/condicoes?aba=poderes` por confiar nele e passou a medir o
	// catálogo mais magro do livro dizendo medir o mais gordo (ALE-332). Reserva
	// que nunca é alcançada não avisa que não funciona.
	aba := path.Base(r.URL.Path)
	c := collectionCriteria{
		Term: q.Get("busca"), Aba: aba, Entry: q.Get("entrada"),
		Filters: filtersFromURL(q, aba),
	}

	// Os FILTROS vêm num mapa cru e não numa struct: as chaves dependem da aba
	// (`circulo` só existe em magias), e uma struct com os seis campos faria
	// toda cena declarar os filtros das outras.
	var all map[string]json.RawMessage
	if err := datastar.ReadSignals(r, &all); err == nil {
		for _, f := range filtersForTab(knownTab(aba)) {
			var values []string
			if raw, found := all[f.Key]; found && json.Unmarshal(raw, &values) == nil {
				c.Filters[f.Key] = values
			}
		}
	}

	signals := struct {
		Term *string `json:"search"`
		Aba  *string `json:"aba"`
	}{}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return c
	}
	// Ponteiro para separar "não veio" de "veio vazio": busca APAGADA é valor
	// legítimo, e tratá-la como ausente ressuscitaria o texto da URL.
	if signals.Term != nil {
		c.Term = *signals.Term
	}
	if signals.Aba != nil {
		c.Aba = *signals.Aba
	}
	return c
}
