package master

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/web/routes"
)

// As rotas da MESA DO MESTRE (ALE-257).
//
// O prefixo é `/mestre/` e não `/mesa/` — a razão está no
// cabeçalho do `bestiary.templ`: `mesa` já nomeia a sessão ao vivo
// desde a fatia 1, e uma palavra com dois sentidos no mesmo espaço de endereço
// é o que o glossário existe para impedir.

// Routes registra as trinta rotas do `/mestre/*` mais a do verbete.
//
// Ela recebe a cena e não o roteador sozinho porque esta cena TEM porta — o
// buscador, que declara zero dependências, é o caso oposto (ALE-278).
func Routes(r chi.Router, s Scene) {
	// `/mestre` sozinho não é uma tela: a trilha sempre tem uma ferramenta em
	// cena. Ele leva à primeira, que é a mesma que a SPA abre.
	r.Get("/mestre", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, routes.MasterBestiary, http.StatusSeeOther)
	})
	r.Get(routes.MasterBestiary, s.handleBestiary)
	r.Post("/mestre/bestiario/tipo/{tipo}", s.handleBestiaryType)
	// CADA CATÁLOGO tem endereço próprio desde a ALE-264: eles viraram paradas do
	// trilho, e parada de trilho é uma cena. `/mestre/condicoes` em vez de
	// `/mestre/catalogos?aba=condicoes` — o mesmo handler, com a aba vindo
	// do CAMINHO.
	//
	// O laço sobre `collectionTabs` e não nove linhas escritas: o catálogo que
	// entrar amanhã ganha rota sozinho, e uma lista de rotas à mão é a que fica
	// para trás em silêncio.
	for _, aba := range collectionTabs {
		r.Get("/mestre/"+aba.ID, s.handleCollection)
	}
	// O endereço VELHO continua respondendo, redirecionando: ele foi o único por
	// duas fatias desta issue, e pode estar colado no chat de alguma mesa.
	r.Get("/mestre/catalogos", s.handleOldCollection)
	r.Get("/mestre/encontros", s.handleEncounters)
	r.Post("/mestre/encontros/adicionar/{id}", s.handleEncounterAdd)
	r.Post("/mestre/encontros/mais/{id}", s.handleEncounterAdd)
	r.Post("/mestre/encontros/menos/{id}", s.handleEncounterLess)
	r.Post("/mestre/encontros/remover/{id}", s.handleEncounterRemove)
	r.Get("/mestre/improviso", s.handleImprov)
	// A ferramenta DESCONHECIDA cai na primeira, e não em 404.
	//
	// Porte de comportamento: a `/gm/$tool` da SPA validava o slug e redirigia,
	// com o comentário "uma URL digitada à mão ou velha aterrissa na primeira
	// ferramenta em vez de num palco em branco". Com a virada, quem encaminha
	// não valida mais — se o servidor devolvesse 404, um link velho de mestre
	// viraria página de erro em vez de abrir a Mesa.
	//
	// No chi o segmento ESTÁTICO ganha do parâmetro, então as quatro rotas
	// acima continuam sendo as que atendem; esta só recolhe o resto.
	r.Get("/mestre/{ferramenta}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, routes.MasterBestiary, http.StatusSeeOther)
	})
	r.Post("/mestre/improviso/{tabela}", s.handleImprovRoll)
	r.Post("/mestre/improviso/{tabela}/limpar", s.handleImprovClear)
	s.entryRoutes(r)
}
