package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"t20engine/db"
	"t20engine/db/sqlcgen"
)

const bcryptCost = 12

// AuthUser is the identity contract returned to the client — mirrors
// backend/src/auth/auth-user.type.ts (name is null when unset).
type AuthUser struct {
	ID    int64   `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
}

func authUserFrom(u sqlcgen.User) AuthUser {
	out := AuthUser{ID: u.ID, Email: u.Email}
	if u.Name.Valid {
		out.Name = &u.Name.String
	}
	return out
}

type registerBody struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     *string `json:"name"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleRegister creates a user (bcrypt), issues the session cookie, returns the
// AuthUser. 201 on success; 409 on a duplicate email.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body registerBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if fields := validateRegister(body); len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not hash password")
		return
	}
	now := nowISO()
	user, err := s.queries.CreateUser(r.Context(), sqlcgen.CreateUserParams{
		Email:        body.Email,
		Name:         nullString(body.Name),
		Passwordhash: string(hash),
		Createdat:    now,
		Updatedat:    now,
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			writeError(w, http.StatusConflict, "Email already registered: "+body.Email)
			return
		}
		writeError(w, http.StatusInternalServerError, "Could not create user")
		return
	}
	if !s.issueSession(w, user) {
		return
	}
	writeJSON(w, http.StatusCreated, authUserFrom(user))
}

// handleLogin validates credentials, issues the cookie, returns AuthUser (200).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if fields := validateLogin(body); len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}
	user, err := s.queries.GetUserByEmail(r.Context(), body.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Passwordhash), []byte(body.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if !s.issueSession(w, user) {
		return
	}
	writeJSON(w, http.StatusOK, authUserFrom(user))
}

// handleLogout clears the session cookie (204).
func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, s.sessionCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

// handleMe returns the authenticated user (behind requireAuth).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

// issueSession signs a JWT for the user and sets the session cookie. Returns
// false (after writing a 500) if signing fails.
func (s *Server) issueSession(w http.ResponseWriter, user sqlcgen.User) bool {
	token, err := s.signToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not sign session")
		return false
	}
	http.SetCookie(w, s.sessionCookie(token, int(sessionTTL.Seconds())))
	return true
}

const sessionTTL = 7 * 24 * time.Hour

// signToken: HS256 over {sub, email} with the
// configured expiry. `sub` is a NUMBER (not a string), which is the shape every
// across the cutover (and vice versa).
func (s *Server) signToken(user sqlcgen.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(parseExpiry(s.cfg.JWTExpiresIn)).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
}

// verifyToken checks the HS256 signature + expiry and returns the user id (sub).
func (s *Server) verifyToken(tokenStr string) (int64, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return 0, errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}
	sub, ok := claims["sub"].(float64) // JSON numbers decode as float64
	if !ok {
		return 0, errors.New("missing sub")
	}
	return int64(sub), nil
}

func (s *Server) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

// parseExpiry handles the JWT_EXPIRES_IN forms this config accepts ("7d", "12h",
// "30m"). Falls back to 7 days on anything unrecognized.
func parseExpiry(s string) time.Duration {
	if s == "" {
		return sessionTTL
	}
	unit := s[len(s)-1]
	n, err := parseInt(s[:len(s)-1])
	if err != nil {
		return sessionTTL
	}
	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour
	case 'h':
		return time.Duration(n) * time.Hour
	case 'm':
		return time.Duration(n) * time.Minute
	default:
		return sessionTTL
	}
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func extractBearer(h string) string {
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
