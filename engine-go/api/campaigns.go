package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"t20engine/db/sqlcgen"
)

// CampaignDTO is the base campaign row (create/update responses).
type CampaignDTO struct {
	ID          int64   `json:"id"`
	OwnerID     int64   `json:"ownerId"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type campaignCharacterDTO struct {
	ID      int64      `json:"id"`
	Name    string     `json:"name"`
	Level   int64      `json:"level"`
	Classes []ClassDTO `json:"classes"`
}

// campaignListDTO adds the caller's role + own member character (GET /campaigns).
type campaignListDTO struct {
	CampaignDTO
	Role      string                `json:"role"`
	Character *campaignCharacterDTO `json:"character"`
	// OwnerName is present ONLY on a mesa the caller does not own, which today
	// means an admin seeing everyone's (ALE-120). Absent is the normal case, so
	// the UI marks the exception instead of every row.
	OwnerName *string `json:"ownerName,omitempty"`
}

type campaignDetailDTO struct {
	CampaignDTO
	Role string `json:"role"`
	// IgnoredRules acompanha o detalhe porque é nele que a campanha se configura
	// (ALE-221) — pedir uma segunda rota para desenhar os interruptores faria a
	// tela piscar entre "tudo ligado" e o estado real.
	IgnoredRules []string `json:"ignoredRules"`
	// Same rule as the list: present only on a mesa the caller does not own. It
	// matters MORE here — this is the screen where you rename and delete.
	OwnerName *string `json:"ownerName,omitempty"`
}

func campaignScalars(c sqlcgen.Campaign) CampaignDTO {
	return CampaignDTO{
		ID: c.ID, OwnerID: c.Ownerid, Name: c.Name, Description: plataforma.NullToPtr(c.Description),
		CreatedAt: c.Createdat, UpdatedAt: c.Updatedat,
	}
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	out, err := s.campaignList(r.Context(), currentUser(r))
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not list campaigns")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, out)
}

// campaignList monta a lista COMO A TELA a mostra: o papel de quem olha, o nome
// do dono quando a mesa é de outra pessoa, e o personagem que o chamador tem
// nela.
//
// Transport-agnostic, e esta é a QUINTA vez que a migração encontra a mesma
// forma — depois do `selfInitiativeEntry`, do `deleteAccount`, do trio da porta
// (ALE-229) e do `mintAccountInvite` (ALE-231). Não é descuido de ninguém: é o
// que uma base com exatamente um transporte parece por dentro, e o segundo
// transporte é o que torna isso visível (ALE-234).
func (s *Server) campaignList(ctx context.Context, user AuthUser) ([]campaignListDTO, error) {
	rows, err := s.visibleCampaigns(ctx, user)
	if err != nil {
		return nil, err
	}
	owners := s.ownerNames(ctx, rows, user.ID)
	out := make([]campaignListDTO, 0, len(rows))
	for _, c := range rows {
		item := campaignListDTO{CampaignDTO: campaignScalars(c), Role: "player"}
		switch {
		case c.Ownerid == user.ID:
			item.Role = "gm"
		case user.IsAdmin:
			// Someone else's mesa, in the list because the caller administers the
			// table. The condition is IsAdmin and not "the owner map has a name":
			// a player is also a non-owner here, and leaning on an empty map would
			// make a future edit to ownerNames hand them "gm" in silence.
			name := owners[c.Ownerid]
			item.Role, item.OwnerName = "gm", &name
		}
		char, err := s.queries.CallerCharacterInCampaign(ctx, sqlcgen.CallerCharacterInCampaignParams{Campaignid: c.ID, Ownerid: user.ID})
		if err == nil {
			classes, _ := s.queries.ListClassesByCharacter(ctx, char.ID)
			cc := &campaignCharacterDTO{ID: char.ID, Name: char.Name, Level: char.Level, Classes: []ClassDTO{}}
			for _, cl := range classes {
				cc.Classes = append(cc.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
			}
			item.Character = cc
		}
		out = append(out, item)
	}
	return out, nil
}

// visibleCampaigns is what the caller may see listed: their own plus the ones
// they play in — and, for the admin, every mesa in the table (ALE-120). Without
// this the admin could reach another's mesa only by typing its URL.
func (s *Server) visibleCampaigns(ctx context.Context, user AuthUser) ([]sqlcgen.Campaign, error) {
	if user.IsAdmin {
		return s.queries.ListAllCampaigns(ctx)
	}
	return s.queries.ListCampaignsForUser(ctx, user.ID)
}

// ownerNames labels the mesas the caller does not own, in ONE query — the list
// is short but an N+1 here would grow with the table.
func (s *Server) ownerNames(ctx context.Context, rows []sqlcgen.Campaign, callerID int64) map[int64]string {
	var ids []int64
	for _, c := range rows {
		if c.Ownerid != callerID {
			ids = append(ids, c.Ownerid)
		}
	}
	names := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return names
	}
	users, err := s.queries.ListUsersByIDs(ctx, ids)
	if err != nil {
		return names
	}
	for _, u := range users {
		names[u.ID] = displayName(u.Name, u.Email)
	}
	return names
}

// displayName prefers the chosen name and falls back to the e-mail, which is
// what the player is called everywhere else in the app.
func displayName(name sql.NullString, email string) string {
	if name.Valid && name.String != "" {
		return name.String
	}
	return email
}

// handleGetCampaign: owner → gm, member → player, else 403.
//
// It used to inline a COPY of resolveRole, which is how the same rule ended up
// with two implementations — and why granting the admin access looked like it
// needed a fourth edit. It calls the rule now (ALE-120).
func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	c, err := s.queries.GetCampaign(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Campaign %d not found", id))
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not load campaign")
		return
	}
	user := currentUser(r)
	role, status, err := s.roleIn(r.Context(), user, c)
	if err != nil {
		plataforma.WriteError(w, status, err.Error())
		return
	}
	out := campaignDetailDTO{
		CampaignDTO:  campaignScalars(c),
		Role:         role,
		IgnoredRules: s.ignoredRulesOf(r.Context(), c.ID),
	}
	// IsAdmin, not merely "not the owner": a PLAYER is also a non-owner here, and
	// marking their mesa would replace their "Jogando" with "Mesa de Fulano" —
	// which is exactly what the e2e caught when this condition was looser.
	if user.IsAdmin && c.Ownerid != user.ID {
		name := s.ownerNames(r.Context(), []sqlcgen.Campaign{c}, user.ID)[c.Ownerid]
		out.OwnerName = &name
	}
	plataforma.WriteJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	name, err := nomeDeCampanha(body.Name)
	if err != nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"name": {err.Error()}})
		return
	}
	descricao, err := descricaoDeCampanha(body.Description)
	if err != nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"description": {err.Error()}})
		return
	}
	now := plataforma.NowISO()
	c, err := s.queries.CreateCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: name, Description: descricao,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create campaign")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, campaignScalars(c))
}

func (s *Server) handleUpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	var set setBuilder
	if body.Name != nil {
		name, err := nomeDeCampanha(*body.Name)
		if err != nil {
			plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"name": {err.Error()}})
			return
		}
		set.Add("name = ?", name)
	}
	if body.Description != nil {
		// Mesma regra do criar, e agora literalmente a mesma FUNÇÃO: descrição
		// de puros espaços vira NULL nos dois caminhos, senão o cliente lê ""
		// de um e null do outro para a mesma entrada.
		descricao, err := descricaoDeCampanha(body.Description)
		if err != nil {
			plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"description": {err.Error()}})
			return
		}
		set.Add("description = ?", nullableArg(descricao))
	}
	if set.empty() {
		plataforma.WriteError(w, http.StatusBadRequest, "No fields to update")
		return
	}
	if err := set.execTouched(r.Context(), s.db, "UPDATE campaigns", id); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update campaign")
		return
	}
	c, _ := s.queries.GetCampaign(r.Context(), id)
	plataforma.WriteJSON(w, http.StatusOK, campaignScalars(c))
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	if err := s.queries.DeleteCampaign(r.Context(), id); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not delete campaign")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// handleRotateInvite ports rotateInviteToken (owner-only): {campaignId, token}.
func (s *Server) handleRotateInvite(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	row, err := s.queries.SetInviteToken(r.Context(), sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: generateInviteToken(), Valid: true}, UpdatedAt: plataforma.NowISO(), ID: id,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not rotate invite")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"campaignId": row.ID, "token": row.Invitetoken.String})
}

// handleResolveInvite resolves a shared token to {campaignId, campaignName}
// (public). The frontend's CampaignInvitePreview expects camelCase keys —
// returning {id, name} left the join form with an undefined campaignId, so the
// "Entrar" button stayed disabled forever (ALE-18). Mirrors handleRotateInvite,
// which already returns campaignId.
//
// An unknown or rotated token is a 404. It used to answer 200 with a `null`
// body, which made a dead invite arrive at the client as a SUCCESS carrying no
// campaign — indistinguishable from one still loading, so the join screen sat
// there with a disabled button and no explanation (ALE-80). A missing thing is
// a 404; only a genuine lookup failure is a 500.
func (s *Server) handleResolveInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		plataforma.WriteError(w, http.StatusNotFound, "Invite not found")
		return
	}
	c, err := s.queries.GetCampaignByToken(r.Context(), sql.NullString{String: token, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		plataforma.WriteError(w, http.StatusNotFound, "Invite not found")
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not resolve invite")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"campaignId": c.ID, "campaignName": c.Name})
}

// resolveRole is the campaign-access domain rule,
// transport-agnostic so both the HTTP handlers and the WS gateway can gate on it: the
// owner is the "gm"; a user who owns a member character is a "player"; anyone else is
// forbidden. Returns the role + an HTTP-ish status the caller maps to its transport.
// The admin enters ANY mesa as "gm" (ALE-120): the role already exists, carries
// the tools they came to use, and nothing in the engine assumes a single GM —
// presence de-duplicates per user and `requireGm` gates by role, not identity.
// Two GMs can therefore drive initiative at once; that is the accepted cost of
// letting the table's owner fix a player's mesa mid-session.
func (s *Server) resolveRole(ctx context.Context, user AuthUser, campaignID int64) (string, int, error) {
	c, err := s.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", http.StatusNotFound, fmt.Errorf("Campaign %d not found", campaignID)
	}
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	return s.roleIn(ctx, user, c)
}

// roleIn is the same rule over a campaign the caller ALREADY loaded, so a
// handler that needs both the row and the role does not read it twice.
func (s *Server) roleIn(ctx context.Context, user AuthUser, c sqlcgen.Campaign) (string, int, error) {
	if c.Ownerid == user.ID || user.IsAdmin {
		return "gm", http.StatusOK, nil
	}
	isMember, _ := s.queries.IsCampaignMember(ctx, sqlcgen.IsCampaignMemberParams{Campaignid: c.ID, Ownerid: user.ID})
	if !isMember {
		return "", http.StatusForbidden, fmt.Errorf("Campaign %d is not accessible", c.ID)
	}
	return "player", http.StatusOK, nil
}

// loadOwnedCampaign is the owner-only campaign rule, transport-agnostic. The GM (owner)
// alone passes; everyone else gets Forbidden. This ONE function is the gate for six
// call sites (rename/delete, invite, members, sessions), which is why the admin
// bypass costs a single condition here (ALE-120).
func (s *Server) loadOwnedCampaign(ctx context.Context, user AuthUser, id int64) (sqlcgen.Campaign, int, error) {
	c, err := s.queries.GetCampaign(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return c, http.StatusNotFound, fmt.Errorf("Campaign %d not found", id)
	}
	if err != nil {
		return c, http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	if c.Ownerid != user.ID && !user.IsAdmin {
		return c, http.StatusForbidden, fmt.Errorf("Campaign %d belongs to another user", id)
	}
	return c, http.StatusOK, nil
}

func (s *Server) ownedCampaign(w http.ResponseWriter, r *http.Request, id int64) (sqlcgen.Campaign, bool) {
	c, status, err := s.loadOwnedCampaign(r.Context(), currentUser(r), id)
	if err != nil {
		plataforma.WriteError(w, status, err.Error())
		return c, false
	}
	return c, true
}

func generateInviteToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
