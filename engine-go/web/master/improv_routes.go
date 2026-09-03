package master

import (
	"net/http"
	"strings"

	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// As rotas e os handlers do IMPROVISO — separados do `routes.go` na ALE-278.
//
// O REGISTRO das trinta rotas continua num lugar só, no `routes.go`: é ele
// que o `api` chama, e espalhá-lo faria a cena ter quatro portas de entrada.
// O que mora aqui são os handlers desta ferramenta. O arquivo único tinha
// 600 linhas e QUATRO famílias que não se chamam — arquivo é unidade de
// RESPONSABILIDADE e de conflito de merge, não de leitura.

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
// `tabela` vazio significa "só redesenhe" — é a carga fria e o campo de salas.
// Com tabela, rola e empilha ANTES de montar a cena, porque o histórico é o que
// a cena desenha.
func (s Scene) respondImprov(w http.ResponseWriter, r *http.Request, tabela string) {
	v := improvFromRequest(r)

	if alvo, achou := strings.CutPrefix(tabela, "limpar:"); achou {
		if _, conhecida := improvRolls[alvo]; !conhecida {
			http.Error(w, "tabela de improviso desconhecida: "+alvo, http.StatusBadRequest)
			return
		}
		v = clearTable(v, alvo)
		tabela = ""
	}
	if tabela != "" {
		rolar, ok := improvRolls[tabela]
		if !ok {
			// Tabela inventada é 400 e não silêncio: a rota é montada a partir
			// da própria lista, então um nome errado aqui só chega por URL
			// digitada à mão — e devolver a cena intacta faria parecer que o
			// botão não funciona.
			http.Error(w, "tabela de improviso desconhecida: "+tabela, http.StatusBadRequest)
			return
		}
		sorteado, err := rolar()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		v = pushOnto(v, tabela, sorteado)
	}

	pronta := loadImprov(v)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), improvScene(pronta))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:        "Improviso · Mesa do Mestre · Tormenta 20",
		Forma:         ui.ShellDense,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, masterBody("improviso", improvScene(pronta)))
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
func clearTable(v improvView, tabela string) improvView {
	switch tabela {
	case "ruina":
		v.Ruina = nil
	case "perseguicao":
		v.Perseguicao = nil
	case "recompensa":
		v.Recompensa = nil
	case "ideias":
		v.Ideias = nil
	}
	return v
}

func pushOnto(v improvView, tabela string, s roll) improvView {
	switch tabela {
	case "ruina":
		v.Ruina = push(v.Ruina, s)
	case "perseguicao":
		v.Perseguicao = push(v.Perseguicao, s)
	case "recompensa":
		v.Recompensa = push(v.Recompensa, s)
	case "ideias":
		v.Ideias = push(v.Ideias, s)
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
	sinais := struct {
		Ruina       []roll `json:"ruina"`
		Perseguicao []roll `json:"perseguicao"`
		Recompensa  []roll `json:"recompensa"`
		Ideias      []roll `json:"ideias"`
		Salas       *int   `json:"salas"`
	}{}
	v := improvView{Salas: salasPadrao}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return v
	}
	v.Ruina, v.Perseguicao = sinais.Ruina, sinais.Perseguicao
	v.Recompensa, v.Ideias = sinais.Recompensa, sinais.Ideias
	if sinais.Salas != nil {
		v.Salas = *sinais.Salas
	}
	return v
}
