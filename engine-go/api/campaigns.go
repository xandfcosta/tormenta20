package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"t20engine/campaign"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
	"t20engine/sheet"
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
	ID      int64            `json:"id"`
	Name    string           `json:"name"`
	Level   int64            `json:"level"`
	Classes []sheet.ClassDTO `json:"classes"`
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
	out, err := s.campaignRules().campaignList(r.Context(), currentUser(r))
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
func (rules campaignRules) campaignList(ctx context.Context, user AuthUser) ([]campaignListDTO, error) {
	rows, err := rules.visibleCampaigns(ctx, user)
	if err != nil {
		return nil, err
	}
	owners := rules.ownerNames(ctx, rows, user.ID)
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
		char, err := rules.queries.CallerCharacterInCampaign(ctx, sqlcgen.CallerCharacterInCampaignParams{Campaignid: c.ID, Ownerid: user.ID})
		if err == nil {
			classes, _ := rules.queries.ListClassesByCharacter(ctx, char.ID)
			cc := &campaignCharacterDTO{ID: char.ID, Name: char.Name, Level: char.Level, Classes: []sheet.ClassDTO{}}
			for _, cl := range classes {
				cc.Classes = append(cc.Classes, sheet.ClassDTO{ClassName: cl.Classname, Level: cl.Level})
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
func (rules campaignRules) visibleCampaigns(ctx context.Context, user AuthUser) ([]sqlcgen.Campaign, error) {
	if user.IsAdmin {
		return rules.queries.ListAllCampaigns(ctx)
	}
	return rules.queries.ListCampaignsForUser(ctx, user.ID)
}

// ownerNames labels the mesas the caller does not own, in ONE query — the list
// is short but an N+1 here would grow with the table.
func (rules campaignRules) ownerNames(ctx context.Context, rows []sqlcgen.Campaign, callerID int64) map[int64]string {
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
	users, err := rules.queries.ListUsersByIDs(ctx, ids)
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

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	// As DUAS recusas de uma vez, e em pt-BR: até a ALE-278 esta rota respondia
	// `err.Error()`, que era a frase inglesa herdada do NestJS, enquanto a cena
	// escrevia a portuguesa dela. Duas frases para uma regra é o que o `account`
	// desfez nesta mesma épica.
	name, descricaoTexto, erros := campaign.ValidateText(body.Name, body.Description)
	if len(erros) > 0 {
		plataforma.WriteValidationError(w, erros)
		return
	}
	descricao := trimOrNull(&descricaoTexto)
	now := plataforma.NowISO()
	c, err := s.campaignRules().createCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: name, Description: descricao,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create campaign")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, campaignScalars(c))
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.campaignRules().ownedCampaign(w, r, id); !ok {
		return
	}
	// ANTES de apagar (ALE-270): a campanha leva as sessões por cascata, e depois
	// disso não há mais como perguntar quais eram — o estado em memória delas
	// ficaria batendo na chave estrangeira até o processo reiniciar.
	s.CampaignDeleted(r.Context(), id)
	if err := s.queries.DeleteCampaign(r.Context(), id); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not delete campaign")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// Aqui morava o `handleResolveInvite`, que resolvia o token compartilhado de
// uma mesa em {campaignId, campaignName} para a tela de entrar da SPA. Ele e o
// `invites_test.go` saíram na ALE-277, e as duas garantias dele estão presas
// onde a pessoa hoje as vive, em `campaigns_join_test.go`: a carta de convite
// resolve o token NO SERVIDOR e traz o nome da mesa na primeira pintura, e o
// convite morto avisa que morreu em vez de oferecer um botão que não abre nada.
//
// O caso do token ROTACIONADO não veio junto, e não por descuido: nada no app
// rotaciona convite de campanha desde que a SPA saiu. O `SetInviteToken` fica —
// a coluna e a consulta são o lugar onde a capacidade volta a morar quando
// alguma cena oferecer o gesto —, mas hoje o único token que existe é o da
// seed, e um teste sobre trocar o que não se troca mede a si mesmo.

// resolveRole is the campaign-access domain rule,
// transport-agnostic so both the HTTP handlers and the WS gateway can gate on it: the
// owner is the "gm"; a user who owns a member character is a "player"; anyone else is
// forbidden. Returns the role + an HTTP-ish status the caller maps to its transport.
// The admin enters ANY mesa as "gm" (ALE-120): the role already exists, carries
// the tools they came to use, and nothing in the engine assumes a single GM:
// esta função devolve um PAPEL, e quem barra barra por papel, não por
// identidade. Two GMs can therefore drive initiative at once; that is the
// accepted cost of letting the table's owner fix a player's mesa mid-session.
//
// A frase original citava o `requireGm` do gateway de socket e a deduplicação
// por usuário da presença. O gateway morreu na ALE-253 e a presença deixou de
// ter quem a alimentasse quando a SPA saiu (ALE-272) — o argumento é o mesmo
// sem eles, e citá-los apontava para dois lugares que não decidem mais nada.
func (rules campaignRules) resolveRole(ctx context.Context, user AuthUser, campaignID int64) (string, int, error) {
	c, err := rules.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", http.StatusNotFound, fmt.Errorf("Campaign %d not found", campaignID)
	}
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	return rules.roleIn(ctx, user, c)
}

// roleIn is the same rule over a campaign the caller ALREADY loaded, so a
// handler that needs both the row and the role does not read it twice.
func (rules campaignRules) roleIn(ctx context.Context, user AuthUser, c sqlcgen.Campaign) (string, int, error) {
	if c.Ownerid == user.ID || user.IsAdmin {
		return "gm", http.StatusOK, nil
	}
	isMember, _ := rules.queries.IsCampaignMember(ctx, sqlcgen.IsCampaignMemberParams{Campaignid: c.ID, Ownerid: user.ID})
	if !isMember {
		return "", http.StatusForbidden, fmt.Errorf("Campaign %d is not accessible", c.ID)
	}
	return "player", http.StatusOK, nil
}

// loadOwnedCampaign is the owner-only campaign rule, transport-agnostic. The GM (owner)
// alone passes; everyone else gets Forbidden. This ONE function is the gate for six
// call sites (rename/delete, invite, members, sessions), which is why the admin
// bypass costs a single condition here (ALE-120).
func (rules campaignRules) loadOwnedCampaign(ctx context.Context, user AuthUser, id int64) (sqlcgen.Campaign, int, error) {
	c, err := rules.queries.GetCampaign(ctx, id)
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

func (rules campaignRules) ownedCampaign(w http.ResponseWriter, r *http.Request, id int64) (sqlcgen.Campaign, bool) {
	c, status, err := rules.loadOwnedCampaign(r.Context(), currentUser(r), id)
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
