package api

import (
	"io/fs"
	"net/http"
	"t20engine/serve/web/admin"
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

// O ROTEADOR WEB do app: cada cena registra as rotas dela, e aqui se decide
// quem fica atrás de qual porteiro.

func (s *Server) WebRouter() http.Handler {
	r := chi.NewRouter()
	// Os estáticos são ANÔNIMOS: são o bundle do Datastar e a folha de estilo, e
	// exigir sessão para eles só quebraria o cache.
	r.Handle("/static/*", http.StripPrefix("/static/", assetsHandler()))
	// A PORTA é anônima por necessidade: é ela que cria a sessão. Ela fica FORA
	// do grupo com `requirePage` — não por ordem de casamento, que o chi resolve
	// por rota, mas porque dentro dele ela seria inalcançável para exatamente
	// quem precisa dela.
	door.Routes(r, door.New(s.doorHost()))
	// O HUB (ALE-231): o menu principal, atrás de sessão como todo o resto.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		hub.Routes(r, hub.New(s.hubHost()))
		campaigns.Routes(r, campaigns.New(s.campaignsHost()))
		// PERSONAGENS e a FORJA são irmãs no mesmo endereço: o elenco é de onde
		// se abre a folha em branco.
		characters.Routes(r, characters.New(s.sceneCore()))
		forge.Routes(r, forge.New(s.forgeHost()))
		// A FICHA é filha do endereço do elenco: `/personagens/{id}`.
		sheetui.Routes(r, sheetui.New(s.sheetHost()))
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

// assetsHandler serve o bundle e a folha embutidos.
func assetsHandler() http.Handler {
	sub, err := fs.Sub(assetsFS, "assets/static")
	if err != nil {
		panic("piloto: static embutido ausente: " + err.Error())
	}
	return comCacheVersionado(versaoDosEstaticos, "public", http.FileServer(http.FS(sub)))
}
