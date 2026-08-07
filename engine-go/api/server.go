package api

import (
	"database/sql"
	"net/http"

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
	catalogs *engine.Catalogs // nil if the catalog snapshot failed to load
}

// NewServer wires the API server. The DB is already opened + migrated (db.Open);
// catalogs may be nil (best-effort) — rule-heavy handlers guard on it.
func NewServer(cfg Config, database *sql.DB, catalogs *engine.Catalogs) *Server {
	return &Server{cfg: cfg, db: database, queries: sqlcgen.New(database), catalogs: catalogs}
}

// Router builds the HTTP handler: shared middleware + domain routes. Routes carry
// NO /api prefix — the Vite dev proxy strips it (rewrite ^/api → ""), matching the
// Nest controllers so the frontend contract is unchanged at cutover.
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

	r.Route("/characters", func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Get("/", s.handleListCharacters)
		r.Get("/{id}", s.handleGetCharacter)
		r.Patch("/{id}/vitals", s.handleUpdateVitals)
		r.Post("/{id}/damage", s.handleApplyDamage)
		r.Patch("/{id}/level", s.handleUpdateLevel)
		r.Patch("/{id}/classes/level", s.handleUpdateClassLevel)
		r.Patch("/{id}/abilities", s.handleUpdateAbilities)
		r.Patch("/{id}/proficiencies", s.handleUpdateProficiencies)
		r.Post("/{id}/items", s.handleAddItem)
		r.Patch("/{id}/items/{itemId}", s.handleUpdateItem)
		r.Delete("/{id}/items/{itemId}", s.handleDeleteItem)
		r.Patch("/{id}/conditions", s.handleUpdateConditions)
		r.Post("/{id}/expertises", s.handleAddExpertise)
		r.Patch("/{id}/expertises", s.handleUpdateExpertise)
		r.Delete("/{id}/expertises/{name}", s.handleDeleteExpertise)
		r.Post("/{id}/spells", s.handleLearnSpell)
		r.Delete("/{id}/spells/{catalogSpellId}", s.handleUnlearnSpell)
		r.Patch("/{id}/spells/{catalogSpellId}/prepared", s.handleSetSpellPrepared)
	})
	return r
}
