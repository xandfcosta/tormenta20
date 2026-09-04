package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"t20engine/account"
	"t20engine/plataforma"
	"time"

	"t20engine/db"
	"t20engine/db/sqlcgen"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// AuthUser is the identity contract returned to the client — mirrors
// backend/src/auth/auth-user.type.ts (name is null when unset).
type AuthUser struct {
	ID    int64   `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
	// IsAdmin is derived from ADMIN_EMAILS at every request, never stored: the
	// role has no row to go stale against, and this is what the UI reads to show
	// the admin door (ALE-120).
	IsAdmin bool `json:"isAdmin"`
}

func (s *Server) authUser(u sqlcgen.User) AuthUser {
	out := AuthUser{ID: u.ID, Email: u.Email, IsAdmin: s.cfg.IsAdmin(u.Email)}
	if u.Name.Valid {
		out.Name = &u.Name.String
	}
	return out
}

// A FORMA dos dois pedidos mora no `account` desde a ALE-278, junto com as
// validações que a lê. Aqui ficou o handler.

// handleRegister creates a user (bcrypt), issues the session cookie, returns the
// AuthUser. 201 on success; 409 on a duplicate email; 403 without a usable
// invite. Since ALE-119 the app answers on the LAN, so registration is no longer
// open — see registrationInvite.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body account.RegisterBody
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	body.Email = plataforma.NormalizeEmail(body.Email)
	if fields := account.ValidateRegister(body); len(fields) > 0 {
		plataforma.WriteValidationError(w, fields)
		return
	}
	user, err := s.createAccount(r.Context(), body)
	if err != nil {
		writeRegisterError(w, err, body.Email)
		return
	}
	if !s.issueSession(w, user) {
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, s.authUser(user))
}

// createAccount is the RULE behind registration: resolve the invite this
// address has to spend, hash, and write the row.
//
// Transport-agnostic, and this is the THIRD time the pilot has had to do this
// (after `selfInitiativeEntry` welded to the socket and `deleteAccount` welded
// to the HTTP handler). The pattern is the same every time — a rule pinned to
// whichever transport reached it first — and it only shows up when a second
// one arrives (ALE-229). Worth naming: it is not a coincidence, it is what a
// codebase with exactly one transport looks like.
func (s *Server) createAccount(ctx context.Context, body account.RegisterBody) (sqlcgen.User, error) {
	invite, err := s.registrationInvite(ctx, body.Email, body.InviteToken)
	if err != nil {
		return sqlcgen.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		return sqlcgen.User{}, err
	}
	now := plataforma.NowISO()
	return s.createUser(ctx, sqlcgen.CreateUserParams{
		Email:        body.Email,
		Name:         plataforma.NullString(body.Name),
		Passwordhash: string(hash),
		Createdat:    now,
		Updatedat:    now,
	}, invite)
}

// errInviteRejected is unknown, spent and expired alike — see `inviteRejected`.
var errInviteRejected = errors.New(inviteRejected)

// registrationInvite resolves the invite this registration has to spend. The
// ADMIN_EMAILS addresses are the exception, and the only one: the owner must be
// able to create their own account on a fresh machine, and "first to register
// wins the crown" would hand that to whoever opens the page first (ALE-120).
func (s *Server) registrationInvite(
	ctx context.Context, email, token string,
) (*sqlcgen.AccountInvite, error) {
	if s.cfg.IsAdmin(email) {
		return nil, nil
	}
	invite, ok := s.usableInvite(ctx, token)
	if !ok {
		return nil, errInviteRejected
	}
	return &invite, nil
}

func writeRegisterError(w http.ResponseWriter, err error, email string) {
	switch {
	case db.IsUniqueViolation(err):
		plataforma.WriteError(w, http.StatusConflict, "Email already registered: "+email)
	case errors.Is(err, errInviteRejected), errors.Is(err, errInviteSpent):
		plataforma.WriteError(w, http.StatusForbidden, inviteRejected)
	default:
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create user")
	}
}

// handleLogin validates credentials, issues the cookie, returns AuthUser (200).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body account.LoginBody
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if fields := account.ValidateLogin(body); len(fields) > 0 {
		plataforma.WriteValidationError(w, fields)
		return
	}
	user, err := s.authenticate(r.Context(), body.Email, body.Password)
	if err != nil {
		plataforma.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if !s.issueSession(w, user) {
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, s.authUser(user))
}

// errBadCredentials is the ONE answer for "no such account" and "wrong
// password": telling them apart hands an anonymous caller a way to enumerate
// who has an account here.
var errBadCredentials = errors.New("invalid credentials")

// authenticate is the RULE behind the login — extracted alongside
// `createAccount` and for the same reason.
//
// The bcrypt comparison runs even when the e-mail is unknown would be the next
// hardening step (it does not today, and that is a timing oracle worth an issue
// of its own); what matters here is that BOTH paths answer the same error.
func (s *Server) authenticate(ctx context.Context, email, password string) (sqlcgen.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, plataforma.NormalizeEmail(email))
	if err != nil {
		return sqlcgen.User{}, errBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Passwordhash), []byte(password)) != nil {
		return sqlcgen.User{}, errBadCredentials
	}
	return user, nil
}

// handleLogout clears the session cookie (204).
func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, s.sessionCookie("", -1))
	w.WriteHeader(http.StatusNoContent)
}

// handleMe returns the authenticated user (behind requireAuth).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	plataforma.WriteJSON(w, http.StatusOK, currentUser(r))
}

// issueSession signs a JWT for the user and sets the session cookie. Returns
// false (after writing a 500) if signing fails.
func (s *Server) issueSession(w http.ResponseWriter, user sqlcgen.User) bool {
	token, err := s.signToken(user)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not sign session")
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
	n, err := plataforma.ParseInt(s[:len(s)-1])
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

func extractBearer(h string) string {
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
