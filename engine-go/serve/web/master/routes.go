package master

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/serve/web/routes"
)

// As rotas da MESA DO MESTRE.
//
// O prefixo é `/mestre/` e não o da sessão: as ferramentas do mestre valem fora
// de qualquer partida, e
// uma palavra com dois sentidos no mesmo espaço de endereço é o que o glossário
// existe para impedir.

// Routes registra as rotas do `/mestre/*` mais a do verbete.
//
// Ela recebe a cena e não o roteador sozinho porque esta cena TEM porta — o
// buscador, que declara zero dependências, é o caso oposto.
func Routes(r chi.Router, s Scene) {
	// `/mestre` sozinho não é uma tela: a trilha sempre tem uma ferramenta em
	// cena, e ele leva à primeira.
	r.Get("/mestre", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, routes.MasterBestiary, http.StatusSeeOther)
	})
	r.Get(routes.MasterBestiary, s.handleBestiary)
	r.Post("/mestre/bestiario/tipo/{tipo}", s.handleBestiaryType)
	// Cada catálogo é uma parada do trilho, e parada de trilho é uma cena: a aba
	// vem do CAMINHO. O laço, e não dez linhas escritas, para o catálogo que
	// entrar amanhã ganhar rota sozinho.
	for _, aba := range collectionTabs {
		r.Get("/mestre/"+aba.ID, s.handleCollection)
	}
	r.Get("/mestre/encontros", s.handleEncounters)
	r.Post("/mestre/encontros/adicionar/{id}", s.handleEncounterAdd)
	r.Post("/mestre/encontros/mais/{id}", s.handleEncounterAdd)
	r.Post("/mestre/encontros/menos/{id}", s.handleEncounterLess)
	r.Post("/mestre/encontros/remover/{id}", s.handleEncounterRemove)
	r.Get("/mestre/improviso", s.handleImprov)
	// A ferramenta DESCONHECIDA cai na primeira, e não em 404: ninguém valida o
	// slug antes de encaminhar, e um 404 transformaria uma URL velha ou digitada
	// à mão em página de erro em vez de abrir a Mesa.
	//
	// No chi o segmento ESTÁTICO ganha do parâmetro, então as rotas escritas
	// acima continuam sendo as que atendem; esta só recolhe o resto.
	r.Get("/mestre/{ferramenta}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, routes.MasterBestiary, http.StatusSeeOther)
	})
	r.Post("/mestre/improviso/{tabela}", s.handleImprovRoll)
	r.Post("/mestre/improviso/{tabela}/limpar", s.handleImprovClear)
	s.entryRoutes(r)
}
