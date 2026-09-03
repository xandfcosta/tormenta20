package master

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"

	"t20engine/web/ui"

	"github.com/starfederation/datastar-go/datastar"
)

// As rotas e os handlers dos CATÁLOGOS — separados do `routes.go` na ALE-278.
//
// O REGISTRO das trinta rotas continua num lugar só, no `routes.go`: é ele
// que o `api` chama, e espalhá-lo faria a cena ter quatro portas de entrada.
// O que mora aqui são os handlers desta ferramenta. O arquivo único tinha
// 600 linhas e QUATRO famílias que não se chamam — arquivo é unidade de
// RESPONSABILIDADE e de conflito de merge, não de leitura.

// handleCollection serve os dois casos numa rota, como as outras cenas.
//
// Sem autorização própria e pelo mesmo motivo do bestiário: o catálogo é o
// LIVRO, igual para todo mundo. O `requirePage` do grupo já exige sessão.
// handleOldCollection manda o endereço antigo para a cena da aba pedida,
// preservando busca e entrada — um redirecionamento que perde a consulta
// devolveria a pessoa a uma tela que não é a que ela pediu.
func (s Scene) handleOldCollection(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	destino := "/mestre/" + knownTab(q.Get("aba"))
	q.Del("aba")
	if resto := q.Encode(); resto != "" {
		destino += "?" + resto
	}
	http.Redirect(w, r, destino, http.StatusMovedPermanently)
}

func (s Scene) handleCollection(w http.ResponseWriter, r *http.Request) {
	v := loadCollection(collectionCriteriaFromRequest(r), s.deps.BookAddress())

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), collectionScene(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:        tabLabel(v.Aba) + " · Mesa do Mestre · Tormenta 20",
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
	fora := map[string][]string{}
	for _, f := range filtersForTab(knownTab(aba)) {
		if valores := q[f.Chave]; len(valores) > 0 {
			fora[f.Chave] = valores
		}
	}
	return fora
}

// collectionCriteriaFromRequest lê a busca, a aba e a ENTRADA da URL na carga fria e
// dos SINAIS quando o Datastar chama — mesma decisão das outras cenas, e é ela
// que faz `?busca=fogo`, `?aba=magias` e `?entrada=medo` serem endereços que se
// recarregam.
//
// `entrada` NÃO vem de sinal, e é deliberado: ela é um endereço para UM verbete,
// escrito por um elo ou colado por alguém. Vindo de sinal, ela sobreviveria à
// próxima tecla digitada na busca e a cena ficaria presa num verbete só.
func collectionCriteriaFromRequest(r *http.Request) collectionCriteria {
	q := r.URL.Query()
	// A ABA vem do CAMINHO desde que cada catálogo virou uma cena
	// (`/mestre/condicoes`). A consulta continua sendo lida para o
	// endereço velho e para quem digitar `?aba=` à mão.
	aba := path.Base(r.URL.Path)
	if knownTab(aba) != aba {
		aba = q.Get("aba")
	}
	c := collectionCriteria{
		Term: q.Get("busca"), Aba: aba, Entrada: q.Get("entrada"),
		Filtros: filtersFromURL(q, aba),
	}

	// Os FILTROS vêm num mapa cru e não numa struct: as chaves dependem da aba
	// (`circulo` só existe em magias), e uma struct com os seis campos faria
	// toda cena declarar os filtros das outras.
	var todos map[string]json.RawMessage
	if err := datastar.ReadSignals(r, &todos); err == nil {
		for _, f := range filtersForTab(knownTab(aba)) {
			var valores []string
			if bruto, tem := todos[f.Chave]; tem && json.Unmarshal(bruto, &valores) == nil {
				c.Filtros[f.Chave] = valores
			}
		}
	}

	sinais := struct {
		Term *string `json:"busca"`
		Aba  *string `json:"aba"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return c
	}
	// Ponteiro para separar "não veio" de "veio vazio": busca APAGADA é valor
	// legítimo, e tratá-la como ausente ressuscitaria o texto da URL.
	if sinais.Term != nil {
		c.Term = *sinais.Term
	}
	if sinais.Aba != nil {
		c.Aba = *sinais.Aba
	}
	return c
}
