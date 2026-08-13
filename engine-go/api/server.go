package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// Server holds the API dependencies (config, DB handle, typed queries, primed
// rules catalogs) and builds the router.
type Server struct {
	cfg      Config
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs  // nil if the catalog snapshot failed to load
	sessions *sessionStore     // in-memory realtime tracker state (B.6)
	presence *presenceRegistry // who's-online per session room (B.6)
	// charMu serializes mutating HTTP requests per character (characterID → *sync.Mutex)
	// so concurrent read-modify-write mutations (rapid damage/vitals clicks) can't lose
	// updates. Mirrors the per-session lock used by the realtime store.
	charMu sync.Map
}

// lockCharacter acquires the per-character write lock, returning the unlock func.
func (s *Server) lockCharacter(id int64) func() {
	m, _ := s.charMu.LoadOrStore(id, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// serializeCharacterWrites serializes mutating requests (POST/PATCH/DELETE) per character,
// so concurrent read-modify-write handlers (damage, vitals, items, effects…) don't race on
// load→compute→save and drop updates. Reads pass straight through.
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
func NewServer(cfg Config, database *sql.DB, catalogs *engine.Catalogs) *Server {
	q := sqlcgen.New(database)
	return &Server{
		cfg: cfg, db: database, queries: q, catalogs: catalogs,
		sessions: newSessionStore(q, newUUID, cfg.WSVitalsWriteThroughLive),
		presence: newPresenceRegistry(),
	}
}

// Router builds the HTTP handler: shared middleware + domain routes. Routes carry
// NO /api prefix — in dev the Vite proxy strips it, and in production cmd/api
// mounts this under http.StripPrefix("/api") while serving the SPA itself.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", s.handleHealth)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", s.handleRegister)
		r.Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)
		r.With(s.requireAuth).Get("/me", s.handleMe)
	})

	r.Route("/catalog", func(r chi.Router) {
		r.Get("/", s.handleCatalogIndex)
		r.Get("/{resource}", s.handleCatalogResource)
	})

	// Invite landing is anonymous (pre-login preview).
	r.Get("/invites/{token}", s.handleResolveInvite)

	r.With(s.requireAuth).Get("/users", s.handleListUsers)

	r.Route("/campaigns", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/", s.handleListCampaigns)
		r.Post("/", s.handleCreateCampaign)
		r.Get("/{id}", s.handleGetCampaign)
		r.Patch("/{id}", s.handleUpdateCampaign)
		r.Delete("/{id}", s.handleDeleteCampaign)
		r.Post("/{id}/invite", s.handleRotateInvite)
		r.Route("/{campaignId}/members", func(r chi.Router) {
			r.Get("/", s.handleListMembers)
			r.Post("/", s.handleAddMember)
			r.Patch("/{id}", s.handleUpdateMemberRole)
			r.Delete("/{id}", s.handleRemoveMember)
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
