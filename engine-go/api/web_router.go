package api

import (
	"io/fs"
	"net/http"
	"t20engine/web/admin"
	"t20engine/web/campaigns"
	"t20engine/web/characters"
	"t20engine/web/door"
	"t20engine/web/finder"
	"t20engine/web/forge"
	"t20engine/web/grimoire"
	"t20engine/web/hub"
	"t20engine/web/master"
	"t20engine/web/reader"
	"t20engine/web/routes"
	"t20engine/web/sheetui"
	"t20engine/web/table"

	"github.com/go-chi/chi/v5"
)

// O ROTEADOR WEB do app (ALE-219; extraído da Mesa na ALE-278).
//
// Ele morava dentro do arquivo de rotas da MESA, e isso era acidente de
// história e não desenho: a Mesa foi a primeira cena do piloto, então o mux
// nasceu no arquivo dela e as onze cenas seguintes foram sendo penduradas ali.
// Quando a Mesa virou pacote, o roteador teria ido junto — e o `api`, que é
// quem monta o app, ficaria sem saber montar nada.
//
// É a mesma forma do guarda que media o próprio diretório: um símbolo que mora
// no arquivo errado só aparece quando o arquivo se move.

func (s *Server) WebRouter() http.Handler {
	r := chi.NewRouter()
	// QUATRO cenas já não recebem o `*Server` (ALE-278, fatia 6): o grimório, a
	// Mesa do Mestre, o leitor e os personagens são cumpridos INTEIRAMENTE pelo
	// `sceneCore`. Elas não pedem nada que só o servidor saiba — desenham o
	// livro, a folha de especificação e o elenco —, e por isso são as primeiras
	// a trocar. As outras sete ainda recebem o `s`, que atende pela ponte.
	// Os estáticos são anônimos: são o bundle do Datastar e a folha de estilo,
	// e exigir sessão para eles só quebraria o cache.
	r.Handle("/static/*", http.StripPrefix("/static/", pilotoStaticHandler()))
	// A PORTA (ALE-229) é anônima por necessidade: é ela que cria a sessão. Ela
	// tem de vir ANTES dos grupos com `requireAuth` — não por ordem de casamento
	// (o chi casa por rota, não por ordem), mas porque ficar dentro do grupo a
	// tornaria inalcançável para exatamente quem precisa dela.
	// `door.New(s)` passa o `Scene` como a porta que a cena declarou — a
	// interface está em `web/door`, e é nesta linha que o compilador cobra
	// quando ela deixa de ser cumprida.
	door.Routes(r, door.New(s))
	// O HUB (ALE-231): o menu principal, atrás de sessão como todo o resto.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		hub.Routes(r, hub.New(s))
		campaigns.Routes(r, campaigns.New(s))
		// PERSONAGENS (ALE-278) e a FORJA, irmãs no mesmo endereço: o elenco é de
		// onde se abre a folha em branco. Elas eram montadas UMA DENTRO DA OUTRA
		// e isso era organização, não dependência — o `chi` não liga para quem
		// registra o quê, e a linha subiu para cá quando a lista virou pacote.
		characters.Routes(r, characters.New(s.sceneCore()))
		forge.Routes(r, forge.New(s))
		// A FICHA (ALE-272) é filha do endereço do elenco: `/personagens/{id}`.
		sheetui.Routes(r, sheetui.New(s))
		grimoire.Routes(r, grimoire.New(s.sceneCore()))
		// A MESA DO MESTRE (ALE-278): o trilho, os nove catálogos, o bestiário,
		// os encontros e o improviso — mais o VERBETE, que sai junto porque ele
		// é uma rota fina sobre o desenho do acervo e não uma cena própria.
		master.Routes(r, master.New(s.sceneCore()))
		// O BUSCADOR (ALE-264) fica no grupo do Hub e não no do mestre: a caixa
		// abre em QUALQUER cena, inclusive na Mesa, e a rota tem de existir onde
		// quer que o ⌃K seja apertado.
		// O buscador não recebe nada: ele é a primeira cena sem porta (ALE-278).
		finder.Routes(r)
		// O LIVRO (ALE-264) é servido para quem ENTROU e não anonimamente como
		// os estáticos: os estáticos são o bundle do Datastar, e isto é um
		// arquivo do dono da mesa. Sem `LIVRO_PDF` a rota devolve 404 — o botão
		// que a levaria também não é desenhado.
		r.Handle(routes.Book, s.LivroDoPiloto())
		// O LEITOR é uma PÁGINA e o `/livro` é o arquivo. Rotas irmãs de
		// propósito: quem quiser o PDF cru (imprimir, buscar no visualizador do
		// navegador) tem o endereço de sempre. Desde a ALE-278 a página é um
		// pacote e o arquivo continua aqui — a divisão é por dependência: quem
		// serve o arquivo lê a configuração e o disco.
		reader.Routes(r, reader.New(s.sceneCore()))
	})
	// A MESA (ALE-278) é montada pelo pacote dela, DENTRO do grupo que exige
	// sessão: quem decide que a cena está atrás do login é o hospedeiro, e não
	// ela. Uma cena que se autoprotegesse daria a impressão de que a fronteira é
	// dela — e este `Group` é a linha que faz a página anônima ir para a porta
	// lembrando o caminho inteiro.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		table.Routes(r, s.tableScene)
	})
	// A SEGUNDA superfície (ALE-219): a administração. Mesmo `requireAdmin` da
	// API — a tela não decide quem pode ver, ela só deixa de oferecer o que o
	// servidor recusaria.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePage)
		r.Use(s.requireAdmin)
		admin.Routes(r, admin.New(s))
	})
	return r
}

// pilotoStaticHandler serve o bundle e a folha embutidos.
func pilotoStaticHandler() http.Handler {
	sub, err := fs.Sub(pilotoFS, "piloto/static")
	if err != nil {
		panic("piloto: static embutido ausente: " + err.Error())
	}
	return comCacheVersionado(versaoDosEstaticos, "public", http.FileServer(http.FS(sub)))
}
