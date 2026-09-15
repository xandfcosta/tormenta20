package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
	"t20engine/infra/platform"
	"t20engine/serve/web/table"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server é a RAIZ DE COMPOSIÇÃO: ele guarda o que o app inteiro precisa e
// cumpre a porta de cada cena.
type Server struct {
	cfg      platform.Config
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs       // nulo se o despejo do catálogo não carregou
	sessions *live.SessionStore     // a fila e a cena de cada sessão, em memória
	boards   *board.BoardStore      // os tabuleiros táticos vivos por sessão
	presence *live.PresenceRegistry // quem está online em cada sala
	sse      *live.SSEHub           // os leitores SSE por sessão e papel
	// bus é o barramento: o que acontece numa mesa vira notícia tipada, e quem
	// desenha cena escuta.
	bus   *events.Bus
	livro livroServido // o PDF do livro, quando `LIVRO_PDF` aponta para um
	// tableScene é a cena da Mesa, montada UMA vez — ver o construtor.
	tableScene table.Scene
	// charMu serializa as escritas por personagem (id → *sync.Mutex), para
	// cliques rápidos de dano e vitais não se perderem no ler-computar-gravar.
	charMu sync.Map
	// emSegundoPlano conta o trabalho que continua DEPOIS da resposta: a
	// persistência do estado da sessão, disparada em goroutine para o mestre não
	// esperar o disco no meio do turno.
	//
	// Ele existe porque uma goroutine que ninguém espera escreve num banco que
	// já fechou. Em PRODUÇÃO isso é o `Shutdown` cortando a gravação do estado
	// da mesa. No TESTE é pior de ler: o `t.TempDir()` falha ao limpar com
	// "directory not empty", porque o SQLite recria `-wal`/`-shm` depois do
	// `RemoveAll` — e a mensagem que sobra fala da LIMPEZA, não do defeito.
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

// characterChanged avisa as mesas AO VIVO que uma ficha mudou.
//
// Sem ele, o mestre aplica "Caído" pela ficha do combatente e a tela do jogador
// não fica sabendo — e o motor deriva Defesa e perícias da condição, então os
// dois passam a ver números diferentes do mesmo personagem sem nada dizer que
// discordam.
//
// A busca é por sessão VIVA e só as que têm o personagem na fila: avisar mesa
// que não o tem mandaria todo cliente da casa refazer busca a cada ficha salva.
//
// ELE É DO `sheetRules` E MORA NO ARQUIVO DO `Server`, e isso é decisão: ele é
// metade do mecanismo de serialização de escrita por personagem — a trava, o
// middleware e o id vêm logo abaixo e são do `Server`. Partir por RECEPTOR
// partiria em dois arquivos que ninguém lê junto o mecanismo, que é a coisa com
// uma razão para mudar.
func (sr sheetRules) characterChanged(characterID int64) {
	// O AVISO PARA AS CENAS DO SERVIDOR é por PERSONAGEM e não por sessão: quem
	// escuta é o stream da Mesa de quem tem essa ficha aberta, e a busca por
	// sessão viva abaixo responde outra pergunta — a do hub SSE, que fala com a
	// sala inteira.
	sr.bus.Publish(events.CharacterChanged{CharacterID: characterID})
	for _, sessionID := range sr.sessions.LiveSessionsWithCharacter(characterID) {
		sr.sse.Emit(sessionID, "", "character-changed", map[string]any{"characterId": characterID})
	}
}

// lockCharacter toma a trava de escrita deste personagem e devolve como soltar.
func (s *Server) lockCharacter(id int64) func() {
	m, _ := s.charMu.LoadOrStore(id, &sync.Mutex{})
	Mu := m.(*sync.Mutex)
	Mu.Lock()
	return Mu.Unlock
}

// serializeCharacterWrites serializa POST/PATCH/DELETE por personagem, para
// dois handlers de ler-computar-gravar não perderem uma atualização. Leitura
// passa direto.
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

// characterIDFromPath tira o {id} de `/personagens/{id}/…` para chavear a
// trava. Falso quando não há id numérico no caminho.
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

// NewServer monta o servidor. O banco já chega aberto e migrado pelo `db.Open`;
// `catalogs` pode ser nulo, e quem depende de regra confere.
func NewServer(cfg platform.Config, database *sql.DB, catalogs *engine.Catalogs) *Server {
	q := sqlcgen.New(database)
	// UM barramento para os dois stores e para o servidor: um por store faria
	// quem escuta juntar as peças de novo.
	bus := &events.Bus{}
	srv := &Server{
		cfg: cfg, db: database, queries: q, catalogs: catalogs,
		// Lido UMA vez, no boot: o dígito do endereço vem do `os.Stat`, e
		// refazê-lo por requisição seria ir ao disco para responder um cabeçalho.
		livro:    abreOLivro(cfg),
		sessions: live.NewSessionStore(q, live.NewUUID, sheetVitals{q: q}, bus),
		boards:   board.NewBoardStore(q, live.NewUUID, bus),
		bus:      bus,
		presence: live.NewPresenceRegistry(),
		sse:      live.NewSSEHub(),
	}
	// A CENA DA MESA é montada UMA vez e o servidor guarda a instância: ela tem
	// estado — a lente e a aba que cada pessoa escolheu —, e um `table.New` por
	// requisição daria um estado novo a cada pedido.
	srv.primeCatalogs(catalogs)
	return srv
}

// primeCatalogs troca o motor e RECONSTRÓI a cena da Mesa, e as duas coisas
// andam juntas: o adaptador da Mesa COPIA o `*engine.Catalogs` quando é
// montado, então trocar o campo sem reconstruir deixa a Mesa com o motor de
// antes.
func (s *Server) primeCatalogs(catalogs *engine.Catalogs) {
	s.catalogs = catalogs
	s.tableScene = table.New(s.tableHost())
}

// sceneCore é montado por chamada e não guardado num campo: são três ponteiros
// copiados, e um campo daria ao `*Server` mais uma coisa para manter
// consistente com ele mesmo.
func (s *Server) sceneCore() sceneCore {
	return sceneCore{queries: s.queries, catalogs: s.catalogs, livro: s.livro.endereco}
}

// Router é o que sobrou da API JSON, e nenhuma cena a chama — as cenas leem o
// banco pelo `Queries` da porta delas e desenham HTML. As rotas daqui NÃO
// carregam o prefixo `/api`: quem o põe é o `cmd/api`.
//
// `/health` é INFRAESTRUTURA — quem bate nele é o `healthcheck` do compose e o
// `-health` do próprio binário. É o contra-exemplo da faxina de rotas: "rota sem
// consumidor" se decide perguntando quem pergunta DE FORA.
//
// As demais são a bancada do e2e, e é o que faz a suíte ser REPETÍVEL — montar
// tudo pela tela troca segundos de setup por minutos.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// Sem origem configurada, NENHUM middleware de CORS — que é produção: o
	// binário serve as cenas, então toda chamada é da mesma origem. Montá-lo com
	// `[]string{""}` negaria as mesmas requisições, mas por acidente; e este
	// guarda mantém um `CORS_ORIGIN` vazio longe do padrão do go-chi para lista
	// vazia, que é liberar TODA origem — com credenciais ligadas, todo site.
	if len(s.cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
		}))
	}

	r.Get("/health", s.handleHealth)

	r.Route("/campanhas", func(r chi.Router) {
		r.Use(s.requireAuth)
		// A varredura do `auth.setup.ts`: lista, filtra pelo prefixo "E2E
		// Descartável" e apaga. Nomeia pelo PREFIXO e nunca por id — apagar por
		// id seria apagar seed.
		r.Get("/", s.handleListCampaigns)
		r.Delete("/{id}", s.handleDeleteCampaign)
		// A fixture do `board.spec.ts`: uma mesa descartável por corrida, montada
		// em duas chamadas em vez de seis telas.
		r.Post("/", s.handleCreateCampaign)
		r.Route("/{campaignId}/sessoes", func(r chi.Router) {
			r.Post("/", s.handleCreateSession)
		})
	})

	r.Route("/personagens", func(r chi.Router) {
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
