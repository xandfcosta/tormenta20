package master

import (
	"net/http"
	"strings"

	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// Os handlers do IMPROVISO.
//
// O REGISTRO das rotas da cena continua num lugar só, no `routes.go`: é ele que
// o `api` chama, e espalhá-lo faria a cena ter quatro portas de entrada. O que
// mora aqui são os handlers desta ferramenta, porque arquivo é unidade de
// RESPONSABILIDADE e de conflito de merge.

// handleImprov desenha a cena com os históricos que vieram nos sinais.
func (s Scene) handleImprov(w http.ResponseWriter, r *http.Request) {
	s.respondImprov(w, r, "")
}

// handleImprovRoll rola UMA tabela e empilha o resultado no histórico dela.
func (s Scene) handleImprovRoll(w http.ResponseWriter, r *http.Request) {
	s.respondImprov(w, r, chi.URLParam(r, "tabela"))
}

// handleImprovClear zera o histórico de UMA tabela.
//
// Zera só a dela, e não as quatro: as tabelas são independentes, e limpar a
// ruína não pode levar junto o evento de perseguição que o mestre acabou de
// tirar.
func (s Scene) handleImprovClear(w http.ResponseWriter, r *http.Request) {
	s.respondImprov(w, r, "limpar:"+chi.URLParam(r, "tabela"))
}

// respondImprov é o caminho único das seis rotas.
//
// `table` vazio significa "só redesenhe" — é a carga fria e o campo de salas.
// Com tabela, rola e empilha ANTES de montar a cena, porque o histórico é o que
// a cena desenha.
func (s Scene) respondImprov(w http.ResponseWriter, r *http.Request, table string) {
	v := improvFromRequest(r)

	if target, found := strings.CutPrefix(table, "limpar:"); found {
		if _, known := improvRolls[target]; !known {
			http.Error(w, "tabela de improviso desconhecida: "+target, http.StatusBadRequest)
			return
		}
		v = clearTable(v, target)
		table = ""
	}
	if table != "" {
		roll, ok := improvRolls[table]
		if !ok {
			// Tabela inventada é 400 e não silêncio: a rota é montada a partir
			// da própria lista, então um nome errado aqui só chega por URL
			// digitada à mão — e devolver a cena intacta faria parecer que o
			// botão não funciona.
			http.Error(w, "tabela de improviso desconhecida: "+table, http.StatusBadRequest)
			return
		}
		drawn, err := roll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		v = pushOnto(v, table, drawn)
	}

	ready := loadImprov(v)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragment, err := ui.RenderFragment(r.Context(), improvScene(ready))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragment)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:         "Improviso · Mesa do Mestre · Tormenta 20",
		Forma:         ui.ShellDense,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, masterBody("improviso", improvScene(ready)))
}

// improvRolls liga o nome da rota à função que rola. A tela e a rota leem a
// MESMA tabela, então um nome novo aparece nos dois lugares ou em nenhum.
var improvRolls = map[string]func() (roll, error){
	"ruina":       rollRuin,
	"perseguicao": rollChase,
	"recompensa":  rollReward,
	"ideias":      rollIdea,
}

// clearTable apaga o histórico de uma tabela e deixa as outras três intactas.
func clearTable(v improvView, table string) improvView {
	switch table {
	case "ruina":
		v.Ruin = nil
	case "perseguicao":
		v.Chase = nil
	case "recompensa":
		v.Reward = nil
	case "ideias":
		v.Ideas = nil
	}
	return v
}

func pushOnto(v improvView, table string, s roll) improvView {
	switch table {
	case "ruina":
		v.Ruin = push(v.Ruin, s)
	case "perseguicao":
		v.Chase = push(v.Chase, s)
	case "recompensa":
		v.Reward = push(v.Reward, s)
	case "ideias":
		v.Ideas = push(v.Ideas, s)
	}
	return v
}

// improvFromRequest lê os quatro históricos e o número de salas dos SINAIS.
//
// Aqui não há caminho pela URL, e é diferente das outras cenas de propósito: um
// histórico de rolagens não é endereço — ninguém cola "os cinco dados que eu
// tirei" no chat da mesa, e pôr isso na URL só encheria o histórico do
// navegador a cada clique no botão de rolar.
func improvFromRequest(r *http.Request) improvView {
	signals := struct {
		Ruin   []roll `json:"ruina"`
		Chase  []roll `json:"perseguicao"`
		Reward []roll `json:"recompensa"`
		Ideas  []roll `json:"ideias"`
		Rooms  *int   `json:"rooms"`
	}{}
	v := improvView{Rooms: salasPadrao}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return v
	}
	v.Ruin, v.Chase = signals.Ruin, signals.Chase
	v.Reward, v.Ideas = signals.Reward, signals.Ideas
	if signals.Rooms != nil {
		v.Rooms = *signals.Rooms
	}
	return v
}
