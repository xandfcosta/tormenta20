package api

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/events"
	"t20engine/plataforma"
	"t20engine/sheet"
	"t20engine/tabuleiro"
	"t20engine/web/bookui"
	"t20engine/web/hub"
	"t20engine/web/routes"
	"t20engine/web/table"
	"t20engine/web/ui"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server holds the API dependencies (config, DB handle, typed queries, primed
// rules catalogs) and builds the router.
type Server struct {
	cfg      plataforma.Config
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs         // nil if the catalog snapshot failed to Load
	sessions *aovivo.SessionStore     // in-memory realtime tracker state (B.6)
	boards   *tabuleiro.BoardStore    // tabuleiros táticos vivos por sessão (ALE-124, vários na ALE-205)
	presence *aovivo.PresenceRegistry // who's-online per session room (B.6)
	sse      *aovivo.SSEHub           // leitores SSE por sessão e papel (ALE-253)
	// bus é o barramento de eventos da casa (ALE-279): o que acontece numa mesa
	// vira notícia tipada aqui, e quem desenha cena escuta.
	//
	// Aqui morava `fichas aovivo.CharacterWatch`, o terceiro dos avisos que este
	// barramento substituiu.
	bus   *events.Bus
	livro livroServido // o PDF do livro, quando LIVRO_PDF aponta para um (ALE-264)
	// tableScene é a cena da Mesa, montada uma vez (ALE-278). Ver o construtor.
	tableScene table.Scene
	// Aqui moravam a LENTE (ALE-193, ALE-269) e as ABAS ESCOLHIDAS (ALE-205),
	// e elas foram para a `table.Scene` na ALE-278.
	//
	// O argumento delas não mudou: as duas moram no SERVIDOR e não num sinal do
	// navegador porque o stream não pergunta nada a ninguém — um modo em
	// `data-show` seria desfeito pelo primeiro quadro do SSE. O que mudou foi o
	// DONO: a pergunta "quem está vendo como a mesa vê" e "que tabuleiro cada um
	// está olhando" só existe numa tela, e um campo do `*Server` dizia o
	// contrário. Por isso a cena é montada UMA vez, no registro das rotas.
	// charMu serializes mutating HTTP requests per character (characterID → *sync.Mutex)
	// so concurrent read-modify-write mutations (rapid damage/vitals clicks) can't lose
	// updates. Mirrors the per-session lock used by the realtime store.
	charMu sync.Map
	// emSegundoPlano conta o trabalho que continua DEPOIS da resposta, e hoje é
	// um só: a persistência do estado da sessão, disparada em goroutine para o
	// mestre não esperar o disco no meio do turno.
	//
	// Ele existe porque uma goroutine que ninguém espera escreve num banco que
	// já fechou. Em PRODUÇÃO isso é o `Shutdown` cortando a gravação do estado
	// da mesa — justamente o que este store existe para guardar. No TESTE é
	// pior de ler: o `t.TempDir()` falha ao limpar com "directory not empty",
	// porque o SQLite recria `-wal`/`-shm` depois do `RemoveAll` — e a mensagem
	// que sobra fala da LIMPEZA, não do defeito (ALE-245, a mesma família).
	emSegundoPlano sync.WaitGroup
}

// WaitForBackground bloqueia até o trabalho disparado por resposta terminar.
//
// Quem chama é o encerramento — o `Shutdown` de produção e o `Cleanup` do teste
// —, sempre ANTES de fechar o banco. Sem isto o último estado de sessão da noite
// pode não chegar ao disco, e o log da falha aparece depois de o processo já
// estar indo embora.
func (s *Server) WaitForBackground() {
	s.emSegundoPlano.Wait()
}

// characterChanged avisa as mesas AO VIVO que a ficha de um personagem mudou
// por HTTP (ALE-245).
//
// O mestre aplica "Caído" num PC pela ficha do combatente, e sem isto a tela do
// jogador não fica sabendo. É pior que o chip faltando: o motor deriva Defesa e
// perícias da condição (ALE-28), então os dois passam a ver números diferentes
// do mesmo personagem, sem nada na tela dizendo que discordam.
//
// Era um GANCHO (`notifyCharacterChanged`) preenchido pelo `SocketHandler()`, e
// tinha de ser: o gateway do socket guardava `s *Server`, e o ponteiro nunca ia
// na direção contrária — nenhum handler HTTP conseguia falar com a sala. Com
// SSE o hub é campo do próprio `Server`, então a indireção sumiu junto com o
// socket (ALE-253).
//
// E ela era um risco real, não só uma volta a mais: apagar o gateway deixou o
// gancho SEM QUEM O LIGASSE, o Go inteiro seguiu verde — porque nulo era
// caminho normal e havia teste afirmando isso — e quem acusou foi o e2e de dois
// clientes. Um campo que precisa ser preenchido por outro arquivo para o
// recurso existir é um recurso que nasce desligado.
//
// A busca é por sessão VIVA e só as que têm o personagem na fila: avisar mesa
// que não tem aquele combatente mandaria todo cliente da casa refazer busca a
// cada ficha salva.
func (s *Server) characterChanged(characterID int64) {
	// O AVISO PARA AS CENAS DO SERVIDOR (ALE-275, no barramento desde a ALE-279).
	// Ele é por PERSONAGEM e não por sessão: quem escuta é o stream da Mesa de
	// quem tem essa ficha aberta, e a busca por sessão viva abaixo responde outra
	// pergunta — a do hub SSE, que fala com a sala inteira.
	s.bus.Publish(events.CharacterChanged{CharacterID: characterID})
	for _, sessionID := range s.sessions.LiveSessionsWithCharacter(characterID) {
		s.sse.Emit(sessionID, "", "character-changed", map[string]any{"characterId": characterID})
	}
}

// lockCharacter acquires the per-character write lock, returning the unlock func.
func (s *Server) lockCharacter(id int64) func() {
	m, _ := s.charMu.LoadOrStore(id, &sync.Mutex{})
	Mu := m.(*sync.Mutex)
	Mu.Lock()
	return Mu.Unlock
}

// serializeCharacterWrites serializes mutating requests (POST/PATCH/DELETE) per character,
// so concurrent read-modify-write handlers (damage, vitals, items, effects…) don't race on
// Load→compute→save and drop updates. Reads pass straight through.
func (s *Server) serializeCharacterWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		if id, ok := characterIDFromPath(r.URL.Path); ok {
			unlock := s.lockCharacter(id)
			defer unlock()
		}
		next.ServeHTTP(w, r)
	})
}

// characterIDFromPath extracts the {id} from /characters/{id}/... — used to key the write
// lock. Returns false for paths without a numeric id (e.g. POST /characters create).
func characterIDFromPath(path string) (int64, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range parts {
		if seg == "characters" && i+1 < len(parts) {
			if id, err := strconv.ParseInt(parts[i+1], 10, 64); err == nil {
				return id, true
			}
		}
	}
	return 0, false
}

// NewServer wires the API server. The DB is already opened + migrated (db.Open);
// catalogs may be nil (best-effort) — rule-heavy handlers guard on it.
func NewServer(cfg plataforma.Config, database *sql.DB, catalogs *engine.Catalogs) *Server {
	q := sqlcgen.New(database)
	// UM barramento para os dois stores e para o servidor (ALE-279). Compartilhar
	// é o ponto: um por store devolveria o problema que a issue veio resolver,
	// que é quem escuta ter de juntar as peças de novo.
	bus := &events.Bus{}
	srv := &Server{
		cfg: cfg, db: database, queries: q, catalogs: catalogs,
		// Lido UMA vez, no boot: o dígito do endereço vem do `os.Stat`, e
		// refazê-lo por requisição seria ir ao disco para responder um cabeçalho.
		livro:    abreOLivro(cfg),
		sessions: aovivo.NewSessionStore(q, aovivo.NewUUID, sheetVitals{q: q}, bus),
		boards:   tabuleiro.NewBoardStore(q, aovivo.NewUUID, bus),
		bus:      bus,
		presence: aovivo.NewPresenceRegistry(),
		sse:      aovivo.NewSSEHub(),
	}
	// A CENA DA MESA é montada UMA vez, e o servidor a guarda.
	//
	// Ela tem estado — a LENTE (quem está vendo como a mesa vê) e a ABA que cada
	// pessoa escolheu —, e um `table.New` por requisição daria um estado novo a
	// cada pedido: metade da mesa não veria a lente da outra metade, e o gesto de
	// mostrar à mesa nunca chegaria. Isso não é hipótese — foi o que oito testes
	// acusaram quando o estado saiu daqui e virou campo da cena (ALE-278).
	//
	// O DONO continua sendo a cena; o servidor só guarda a instância, como
	// guarda um store.
	srv.tableScene = table.New(srv)
	return srv
}

// A PONTE das cenas ainda não convertidas (ALE-278, fatia 6).
//
// As nove assinaturas compartilhadas moram no `sceneCore` desde esta fatia, e
// quatro cenas — grimório, mestre, leitor e personagens — já o recebem direto.
// As outras sete continuam recebendo o `*Server`, porque a porta delas pede
// coisas que ainda não têm adaptador próprio, e um `*Server` sem `Queries` não
// cumpre porta nenhuma.
//
// Estes cinco delegam para o núcleo em vez de repetir o corpo, e é essa a
// diferença entre uma ponte e uma segunda implementação: quando a última cena
// tiver adaptador, apagar este bloco não muda comportamento de nada. **A ordem
// é a da casa** — escrever o substituto, vê-lo verde, DEPOIS apagar; começar
// pelo apagar abriria uma janela em que sete cenas não compilam.
func (s *Server) Queries() *sqlcgen.Queries           { return s.sceneCore().Queries() }
func (s *Server) Catalogs() *engine.Catalogs          { return s.sceneCore().Catalogs() }
func (s *Server) BookAddress() bookui.BookAddress     { return s.sceneCore().BookAddress() }
func (s *Server) Asset(arquivo string) string         { return s.sceneCore().Asset(arquivo) }
func (s *Server) CurrentUserID(r *http.Request) int64 { return s.sceneCore().CurrentUserID(r) }
func (s *Server) WritePage(
	w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component,
) {
	s.sceneCore().WritePage(w, r, status, p, corpo)
}
func (s *Server) CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error) {
	return s.sceneCore().CharacterList(ctx, ownerID)
}

// sceneCore monta o núcleo que as cenas compartilham (ALE-278, fatia 6).
//
// Ele é montado por chamada e não guardado num campo: são três ponteiros
// copiados, e um campo daria ao `*Server` mais uma coisa para manter
// consistente com ele mesmo.
func (s *Server) sceneCore() sceneCore {
	return sceneCore{queries: s.queries, catalogs: s.catalogs, livro: s.livro.endereco}
}

// Router builds the HTTP handler: shared middleware + domain routes. Routes carry
// NO /api prefix — in dev the Vite proxy strips it, and in production cmd/api
// mounts this under http.StripPrefix("/api") while serving the SPA itself.
// Router é o que sobrou da API JSON depois da ALE-277: SETE rotas.
//
// # Ela foi escrita para a SPA, e a SPA morreu
//
// Eram 76 rotas e 113 handlers. Medido antes do corte: **nenhuma cena chama
// `/api/*`** — as onze cenas em Datastar leem o banco pelo `Queries` da porta
// delas e desenham HTML. O único consumidor que sobrou é a SUÍTE DE E2E, e ela
// usa seis endereços; o sétimo é o `/health`.
//
// # O que ficou, e por quê cada um
//
//   - `/health` é INFRAESTRUTURA, e é o contra-exemplo que esta casa já pagou
//     uma vez: ele parece rota de API e quem bate nele é o `healthcheck` do
//     compose e o `-health` do próprio binário. Tirar "rota sem consumidor" sem
//     perguntar quem pergunta DE FORA foi o defeito que o CI pegou na ALE-272.
//   - As seis restantes são a bancada do e2e: listar e apagar campanha de teste,
//     listar ficha e limpar condição que uma execução anterior deixou, e montar
//     a mesa descartável do spec do tabuleiro. Elas não são produto — são o que
//     faz a suíte ser REPETÍVEL, e a alternativa (montar tudo pela tela) troca
//     segundos de setup por minutos.
//
// # O que saiu junto, e que não aparece nesta função
//
// O `mountLiveRoutes` e o `/events`: dezoito rotas do tempo real da SPA. A Mesa
// em Datastar tem stream próprio (`/mesa/{campanha}/{sessao}/stream`) e escreve
// pelos comandos dela — medido, ninguém abria o `EventSource` daqui.
//
// E os quinze `*_http_test.go`, com 71 casos. Teste verde sobre código que
// ninguém usa é a pior dívida: cobra manutenção e não protege nada.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// No configured origin → no CORS middleware at all, which is production: the
	// binary serves the SPA itself, so every call is same-origin and no other
	// site has business reaching it. Mounting it with []string{""} would deny
	// the same requests, but says it by accident; the guard also keeps a future
	// list-valued CORS_ORIGIN away from go-chi's empty-list default, which is
	// "allow ALL origins" — with credentials on, that is every website (ALE-119).
	if len(s.cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}))
	}

	r.Get("/health", s.handleHealth)

	r.Route("/campaigns", func(r chi.Router) {
		r.Use(s.requireAuth)
		// A varredura do `auth.setup.ts`: lista, filtra pelo prefixo "E2E
		// Descartável" e apaga. Nomeia pelo PREFIXO e nunca por id — apagar por
		// id seria apagar seed.
		r.Get("/", s.handleListCampaigns)
		r.Delete("/{id}", s.handleDeleteCampaign)
		// A fixture do `piloto-tabuleiro.spec.ts`: uma mesa descartável por
		// corrida, montada em duas chamadas em vez de seis telas.
		r.Post("/", s.handleCreateCampaign)
		r.Route("/{campaignId}/sessions", func(r chi.Router) {
			r.Post("/", s.handleCreateSession)
		})
	})

	r.Route("/characters", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Use(s.serializeCharacterWrites)
		// A varredura das CONDIÇÕES: o spec da sessão aplica Abalado, Agarrado e
		// Cego para medir a faixa cheia, e a limpeza dele mora no corpo do teste
		// — quando ele falha, a condição fica gravada e ele falha PARA SEMPRE.
		r.Get("/", s.handleListCharacters)
		r.Patch("/{id}/conditions", s.handleUpdateConditions)
	})
	return r
}

// CurrentViewer traduz quem está pedindo para a língua do HUB (ALE-278).
//
// A tradução é o preço da fronteira, e ela é barata: quatro campos. O que ela
// compra é o hub não conhecer o `AuthUser` — e portanto não importar este
// pacote, que o importa de volta para montar rota.
func (s *Server) CurrentViewer(r *http.Request) hub.Viewer {
	eu := currentUser(r)
	return hub.Viewer{ID: eu.ID, Email: eu.Email, Name: eu.Name, IsAdmin: eu.IsAdmin}
}

// MintAccountInvite e ExpiredSessionCookie são o que o hub pede da CASA: cunhar
// convite e apagar a sessão dependem de configuração e de política, e nenhuma
// das duas é da tela.
func (s *Server) MintAccountInvite(ctx context.Context, byUserID int64) (sqlcgen.AccountInvite, error) {
	return s.mintAccountInvite(ctx, byUserID)
}

func (s *Server) ExpiredSessionCookie() *http.Cookie { return s.sessionCookie("", -1) }

// TableRoute é o endereço de uma sessão ao vivo. Quem sabe onde cada cena está
// montada é quem monta.
func (s *Server) TableRoute(campaignID, sessionID int64) string {
	return routes.Table(campaignID, sessionID)
}

// Asset é o endereço versionado de um estático, para as cenas que carregam
// bundle próprio. Ele já era injetado na casca (`ui.Page.Asset`); aqui ele vira
// método porque uma cena inteira o pede.
