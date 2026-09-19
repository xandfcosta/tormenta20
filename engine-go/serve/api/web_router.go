package api

import (
	"net/http"
	"t20engine/serve/web/admin"
	"t20engine/serve/web/assets"
	"t20engine/serve/web/campaigns"
	"t20engine/serve/web/characters"
	"t20engine/serve/web/door"
	"t20engine/serve/web/finder"
	"t20engine/serve/web/forge"
	"t20engine/serve/web/grimoire"
	"t20engine/serve/web/hub"
	"t20engine/serve/web/master"
	"t20engine/serve/web/reader"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/sheetui"
	"t20engine/serve/web/table"

	"github.com/go-chi/chi/v5"
)

// O ROTEADOR DO PROCESSO: tudo o que o binário atende, num lugar só.
//
// Era em DOIS — este e um roteador da biblioteca padrão no `cmd/api`, que
// montava as fontes, o favicon, a saúde e o `/api` ao lado das cenas. A divisão
// não tinha razão escrita em lugar nenhum, e cobrava duas:
//
// Quem quisesse saber o que o processo serve lia dois arquivos, e o mais
// externo ficava no `cmd`, longe de tudo. E o roteador da biblioteca padrão NÃO
// SE PERCORRE: o `chi.Walk` é o que o `route_params_test.go` usa para perguntar
// ao roteador — em vez de a um regex sobre a fonte — quais rotas existem, e as
// quatro rotas de lá eram invisíveis para ele.
//
// A ordem abaixo é de fora para dentro: primeiro o que qualquer um alcança,
// depois o que exige sessão.
func (s *Server) WebRouter() http.Handler {
	r := chi.NewRouter()
	// Os estáticos são ANÔNIMOS: são o bundle do Datastar e a folha de estilo, e
	// exigir sessão para eles só quebraria o cache.
	r.Handle("/static/*", http.StripPrefix("/static/", assets.Handler()))
	// As FONTES, que a folha pede por caminho absoluto (`/fonts/…`), e o ÍCONE
	// que o layout pede. Anônimos pelo mesmo motivo dos estáticos.
	r.Handle("/fonts/*", assets.FontsHandler())
	r.Handle("/favicon.svg", assets.FaviconHandler())
	// A SAÚDE responde na RAIZ além de `/api/health`: quem pergunta é o
	// `healthcheck` do compose e o `-health` do próprio binário, e infraestrutura
	// não sabe de prefixo.
	r.Handle("/health", s.HealthProbe())
	// A API JSON, sob `/api`. O `Mount` do chi tira o prefixo sozinho — era um
	// `http.StripPrefix` escrito à mão no mux.
	r.Mount("/api", s.Router())
	// A PORTA é anônima por necessidade: é ela que cria a sessão. Ela fica FORA
	// do grupo com `requirePage` — não por ordem de casamento, que o chi resolve
	// por rota, mas porque dentro dele ela seria inalcançável para exatamente
	// quem precisa dela.
	door.Routes(r, door.New(s.doorHost()))
	// O HUB: o menu principal, atrás de sessão como todo o resto.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		hub.Routes(r, hub.New(s.hubHost()))
		campaigns.Routes(r, campaigns.New(s.campaignsHost()))
		// PERSONAGENS e a FORJA são irmãs no mesmo endereço: o elenco é de onde
		// se abre a folha em branco.
		characters.Routes(r, characters.New(s.sceneCore()))
		forge.Routes(r, forge.New(s.sceneCore(), s.characterBirths()))
		// A FICHA é filha do endereço do elenco: `/personagens/{id}`.
		sheetui.Routes(r, sheetui.New(s.sheetHost(), s.characterPlays()))
		grimoire.Routes(r, grimoire.New(s.sceneCore()))
		// A MESA DO MESTRE: o trilho, os catálogos, o bestiário, os encontros e o
		// improviso.
		master.Routes(r, master.New(s.sceneCore()))
		// O BUSCADOR fica no grupo do Hub e não no do mestre: a caixa abre em
		// QUALQUER cena, e a rota tem de existir onde quer que o ⌃K seja
		// apertado. Ele é a única cena sem porta — não pede nada.
		finder.Routes(r)
		// O LIVRO é servido a quem ENTROU, ao contrário dos estáticos: ele é um
		// arquivo do dono da mesa. Sem `LIVRO_PDF` a rota devolve 404.
		r.Handle(routes.Book, s.BookFileHandler())
		// O LEITOR é a PÁGINA; o `/livro` acima é o arquivo. A divisão é por
		// dependência: quem serve o arquivo lê a configuração e o disco.
		reader.Routes(r, reader.New(s.sceneCore()))
	})
	// Quem decide que a cena está atrás do login é o HOSPEDEIRO, e não ela: uma
	// cena que se autoprotegesse daria a impressão de que a fronteira é dela.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		table.Routes(r, s.tableScene)
	})
	// A administração passa pelo mesmo `requireAdmin` da API: a tela não decide
	// quem pode ver, ela só deixa de oferecer o que o servidor recusaria.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		r.Use(s.requireAdmin)
		admin.Routes(r, admin.New(s.adminHost()))
	})
	return r
}
