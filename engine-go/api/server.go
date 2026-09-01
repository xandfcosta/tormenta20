package api

import "t20engine/tabuleiro"

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"t20engine/aovivo"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
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
	// fichas é quem escuta "esta ficha mudou" (ALE-275). VALOR e não ponteiro:
	// o zero dele já funciona, então não existe estado desligado para alguém
	// tolerar — que é exatamente o defeito que o `characterChanged` abaixo
	// conta ter tido, quando era um gancho que outro arquivo preenchia.
	fichas aovivo.CharacterWatch
	livro  livroServido // o PDF do livro, quando LIVRO_PDF aponta para um (ALE-264)
	// lentes é quem está vendo a cena COMO A MESA (ALE-193, ALE-269). Mora aqui
	// e não num sinal do navegador porque o stream não pergunta nada a ninguém:
	// um modo em `data-show` seria desfeito pelo primeiro quadro do SSE, com a
	// peça escondida voltando sozinha à tela do mestre no meio da conferência.
	lentes *asLentes
	// abas é qual tabuleiro cada pessoa está olhando (ALE-205). Mora aqui pelo
	// MESMO motivo da lente, e o arquivo dela explica o argumento inteiro: o
	// stream desenha, e o que ele desenha não pode depender de um sinal que ele
	// não enxerga.
	abas *asAbasEscolhidas
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

// EsperaOSegundoPlano bloqueia até o trabalho disparado por resposta terminar.
//
// Quem chama é o encerramento — o `Shutdown` de produção e o `Cleanup` do teste
// —, sempre ANTES de fechar o banco. Sem isto o último estado de sessão da noite
// pode não chegar ao disco, e o log da falha aparece depois de o processo já
// estar indo embora.
func (s *Server) EsperaOSegundoPlano() {
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
	// O AVISO PARA AS CENAS DO SERVIDOR (ALE-275). Ele é por PERSONAGEM e não
	// por sessão: quem escuta é o stream da Mesa de quem tem essa ficha aberta,
	// e a busca por sessão viva abaixo responde outra pergunta — a do hub SSE,
	// que fala com a sala inteira.
	s.fichas.Avisar(characterID)
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
	return &Server{
		cfg: cfg, db: database, queries: q, catalogs: catalogs,
		// Lido UMA vez, no boot: o dígito do endereço vem do `os.Stat`, e
		// refazê-lo por requisição seria ir ao disco para responder um cabeçalho.
		livro:    abreOLivro(cfg),
		sessions: aovivo.NewSessionStore(q, aovivo.NewUUID, vitaisDaFicha{q: q}),
		boards:   tabuleiro.NewBoardStore(q, aovivo.NewUUID),
		lentes:   novasLentes(),
		abas:     novasAbas(),
		presence: aovivo.NewPresenceRegistry(),
		sse:      aovivo.NewSSEHub(),
	}
}

// Router builds the HTTP handler: shared middleware + domain routes. Routes carry
// NO /api prefix — in dev the Vite proxy strips it, and in production cmd/api
// mounts this under http.StripPrefix("/api") while serving the SPA itself.
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

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", s.handleRegister)
		r.Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)
		// Anonymous: see /password-resets above.
		r.Post("/reset-password", s.handleResetPassword)
		r.With(s.requireAuth).Get("/me", s.handleMe)
	})

	r.Route("/catalog", func(r chi.Router) {
		r.Get("/", s.handleCatalogIndex)
		r.Get("/{resource}", s.handleCatalogResource)
	})

	// Invite landing is anonymous (pre-login preview).
	r.Get("/invites/{token}", s.handleResolveInvite)
	// Account invite, a different thing from the campaign one above: this is the
	// link that lets someone CREATE an account, so it is read before any session
	// exists (ALE-120).
	r.Get("/account-invites/{token}", s.handleResolveAccountInvite)

	// Anonymous by necessity: whoever forgot their password cannot authenticate
	// to change it. What guards it is the single-use token (ALE-120).
	r.Get("/password-resets/{token}", s.handleResolvePasswordReset)

	r.Route("/admin", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Use(s.requireAdmin)
		r.Get("/users", s.handleAdminListUsers)
		r.Delete("/users/{id}", s.handleAdminDeleteUser)
		r.Post("/users/{id}/password-reset", s.handleAdminCreatePasswordReset)
		r.Get("/invites", s.handleAdminListInvites)
		r.Post("/invites", s.handleCreateAccountInvite)
		r.Get("/status", s.handleAdminStatus)
		r.Get("/backups", s.handleAdminListBackups)
		r.Post("/backups", s.handleAdminCreateBackup)
	})

	r.Route("/campaigns", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/", s.handleListCampaigns)
		r.Post("/", s.handleCreateCampaign)
		r.Get("/{id}", s.handleGetCampaign)
		r.Patch("/{id}", s.handleUpdateCampaign)
		r.Delete("/{id}", s.handleDeleteCampaign)
		r.Post("/{id}/invite", s.handleRotateInvite)
		// Regras opcionais (ALE-221). PUT porque a tela manda o conjunto INTEIRO
		// das regras desligadas, nunca um delta. Só o dono da campanha escreve.
		r.Put("/{id}/rules", s.handleReplaceCampaignRules)
		r.Route("/{campaignId}/members", func(r chi.Router) {
			r.Get("/", s.handleListMembers)
			r.Post("/", s.handleAddMember)
			r.Patch("/{id}", s.handleUpdateMemberRole)
			r.Delete("/{id}", s.handleRemoveMember)
		})
		// Bloco de criatura do mestre (ALE-137). Só o mestre lê e escreve — o
		// jogador continua vendo nome e barra de PV pela iniciativa.
		r.Route("/{campaignId}/creatures", func(r chi.Router) {
			r.Get("/", s.handleListCreatures)
			r.Post("/", s.handleCreateCreature)
			r.Patch("/{id}", s.handleUpdateCreature)
			r.Delete("/{id}", s.handleDeleteCreature)
		})
		r.Route("/{campaignId}/sessions", func(r chi.Router) {
			r.Get("/", s.handleListSessions)
			r.Post("/", s.handleCreateSession)
			r.Get("/{id}", s.handleGetSession)
			r.Patch("/{id}", s.handleUpdateSession)
			r.Delete("/{id}", s.handleDeleteSession)
			r.Post("/{id}/start", s.handleStartSession)
			r.Post("/{id}/end", s.handleEndSession)
			r.Post("/{id}/clear-tracker", s.handleClearTracker)
			// O fluxo de eventos ao vivo (ALE-253). Debaixo do `requireAuth`
			// como qualquer rota — o `EventSource` manda o cookie sozinho.
			r.Get("/{id}/events", s.handleSessionEvents)
			s.mountLiveRoutes(r)
		})
	})

	r.Route("/characters", func(r chi.Router) {
		// Public creation lists: the Forge reads them before anyone logs in.
		r.Get("/options", s.handleCharacterOptions)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Use(s.serializeCharacterWrites)
			r.Get("/", s.handleListCharacters)
			r.Post("/", s.handleCreateCharacter)
			r.Get("/{id}", s.handleGetCharacter)
			r.Get("/{id}/sheet", s.handleGetSheet)
			r.Get("/{id}/campaigns", s.handleListCharacterCampaigns)
			r.Post("/{id}/active-effects", s.handleApplyEffect)
			r.Patch("/{id}/active-effects/{effectId}", s.handleAdjustEffect)
			r.Delete("/{id}/active-effects/{effectId}", s.handleDeleteEffect)
			r.Post("/{id}/end-scene", s.handleEndScene)
			r.Post("/{id}/end-day", s.handleEndDay)
			r.Patch("/{id}/vitals", s.handleUpdateVitals)
			r.Patch("/{id}/tibar", s.handleUpdateTibar)
			r.Post("/{id}/damage", s.handleApplyDamage)
			r.Patch("/{id}/level", s.handleUpdateLevel)
			r.Patch("/{id}/classes/level", s.handleUpdateClassLevel)
			r.Patch("/{id}/abilities", s.handleUpdateAbilities)
			r.Patch("/{id}/proficiencies", s.handleUpdateProficiencies)
			r.Post("/{id}/items", s.handleAddItem)
			r.Patch("/{id}/items/{itemId}", s.handleUpdateItem)
			r.Delete("/{id}/items/{itemId}", s.handleDeleteItem)
			r.Post("/{id}/items/{itemId}/consume", s.handleConsumeItem)
			r.Patch("/{id}/conditions", s.handleUpdateConditions)
			// O estado de JOGO da ficha (ALE-222). `conditionals` e vizinho de
			// `conditions` uma linha acima e e OUTRA COISA: aquele e o opt-in do
			// jogador, este sao as condicoes do livro. Ver C6 no GLOSSARIO.md.
			r.Patch("/{id}/conditionals", s.handleUpdateConditionals)
			r.Post("/{id}/power-uses", s.handleBumpPowerUse)
			r.Put("/{id}/stances/{flag}", s.handleSetStance)
			r.Delete("/{id}/stances/{flag}", s.handleDeleteStance)
			r.Post("/{id}/expertises", s.handleAddExpertise)
			r.Patch("/{id}/expertises", s.handleUpdateExpertise)
			r.Delete("/{id}/expertises/{name}", s.handleDeleteExpertise)
			r.Post("/{id}/spells", s.handleLearnSpell)
			r.Delete("/{id}/spells/{catalogSpellId}", s.handleUnlearnSpell)
			r.Patch("/{id}/spells/{catalogSpellId}/prepared", s.handleSetSpellPrepared)
			r.Post("/{id}/spells/{catalogSpellId}/cast", s.handleCastSpell)
		})
	})
	return r
}
