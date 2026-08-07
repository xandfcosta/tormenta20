package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"t20engine/db/sqlcgen"
)

// Server holds the API dependencies (config, DB handle, typed queries) and builds
// the router.
type Server struct {
	cfg     Config
	db      *sql.DB
	queries *sqlcgen.Queries
}

// NewServer wires the API server. The DB is already opened + migrated (db.Open).
func NewServer(cfg Config, database *sql.DB) *Server {
	return &Server{cfg: cfg, db: database, queries: sqlcgen.New(database)}
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
	})
	return r
}
