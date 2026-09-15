package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"t20engine/domain/campaign"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/httpio"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
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
	// OwnerName só existe numa mesa que o chamador NÃO possui, o que hoje quer
	// dizer um admin vendo as de todo mundo. Ausente é o caso normal, então a
	// tela marca a exceção em vez de marcar toda linha.
	OwnerName *string `json:"ownerName,omitempty"`
}

type campaignDetailDTO struct {
	CampaignDTO
	Role string `json:"role"`
	// IgnoredRules acompanha o detalhe porque é nele que a campanha se configura
	// — pedir uma segunda rota para desenhar os interruptores faria a tela piscar
	// entre "tudo ligado" e o estado real.
	IgnoredRules []string `json:"ignoredRules"`
	// Mesma regra da lista: só numa mesa que o chamador não possui. Aqui ele
	// importa MAIS — esta é a tela onde se renomeia e se apaga.
	OwnerName *string `json:"ownerName,omitempty"`
}

func campaignScalars(c sqlcgen.Campaign) CampaignDTO {
	return CampaignDTO{
		ID: c.ID, OwnerID: c.Ownerid, Name: c.Name, Description: dbvalue.NullToPtr(c.Description),
		CreatedAt: c.Createdat, UpdatedAt: c.Updatedat,
	}
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	out, err := s.campaignRules().campaignList(r.Context(), currentUser(r))
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not list campaigns")
		return
	}
	httpio.WriteJSON(w, http.StatusOK, out)
}

// campaignList monta a lista COMO A TELA a mostra: o papel de quem olha, o nome
// do dono quando a mesa é de outra pessoa, e o personagem que o chamador tem
// nela.
//
// Independente de TRANSPORTE de propósito, para o handler HTTP e a cena em templ
// lerem a mesma regra.
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
			// Mesa de outra pessoa, na lista porque quem chamou administra o
			// servidor. A condição é `IsAdmin` e não "o mapa de donos tem um nome":
			// um jogador também não é dono aqui, e apoiar-se num mapa vazio faria
			// uma edição futura no `ownerNames` entregar "gm" a ele em silêncio.
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

// visibleCampaigns é o que o chamador pode ver listado: as dele mais as em que
// joga — e, para o admin, todas as mesas do servidor. Sem isto o admin só
// alcançaria a mesa de outra pessoa digitando a URL dela.
func (rules campaignRules) visibleCampaigns(ctx context.Context, user AuthUser) ([]sqlcgen.Campaign, error) {
	if user.IsAdmin {
		return rules.queries.ListAllCampaigns(ctx)
	}
	return rules.queries.ListCampaignsForUser(ctx, user.ID)
}

// ownerNames rotula as mesas que o chamador não possui, numa consulta SÓ — a
// lista é curta, mas um N+1 aqui cresceria com o servidor.
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

// displayName prefere o nome escolhido e cai no e-mail, que é como o jogador é
// chamado em todo o resto do app.
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
	if !httpio.DecodeJSON(w, r, &body) {
		return
	}
	// As DUAS recusas de uma vez, e em pt-BR: a mesma regra respondendo duas
	// frases diferentes conforme o transporte é o que faz uma delas envelhecer.
	name, descricaoTexto, erros := campaign.ValidateText(body.Name, body.Description)
	if len(erros) > 0 {
		httpio.WriteValidationError(w, erros)
		return
	}
	descricao := trimOrNull(&descricaoTexto)
	now := dbvalue.NowISO()
	c, err := s.campaignRules().createCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: name, Description: descricao,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not create campaign")
		return
	}
	httpio.WriteJSON(w, http.StatusCreated, campaignScalars(c))
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.campaignRules().ownedCampaign(w, r, id); !ok {
		return
	}
	// ANTES de apagar: a campanha leva as sessões por cascata, e depois disso não
	// há mais como perguntar quais eram — o estado em memória delas ficaria
	// batendo na chave estrangeira até o processo reiniciar.
	s.CampaignDeleted(r.Context(), id)
	if err := s.queries.DeleteCampaign(r.Context(), id); err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not delete campaign")
		return
	}
	httpio.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// resolveRole é a regra de ACESSO a uma campanha, independente de transporte: o
// dono é "gm"; quem tem personagem membro é "player"; o resto é barrado. Devolve
// o papel e um status à moda do HTTP, que o chamador traduz para o transporte
// dele.
//
// O admin entra em QUALQUER mesa como "gm": o papel já existe, carrega as
// ferramentas que ele veio usar, e nada no motor supõe um mestre só — esta
// função devolve um PAPEL, e quem barra barra por papel e não por identidade.
// Dois mestres podem então conduzir a iniciativa ao mesmo tempo, que é o custo
// aceito por deixar o dono do servidor consertar a mesa de um jogador no meio da
// sessão.
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

// roleIn é a mesma regra sobre uma campanha que o chamador JÁ carregou, para o
// handler que precisa da linha e do papel não a ler duas vezes.
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

// loadOwnedCampaign é a regra de campanha SÓ DO DONO, independente de
// transporte: passa o mestre (dono), e o resto recebe Forbidden. Esta função é o
// gargalo de meia dúzia de sítios (renomear/apagar, convite, membros, sessões),
// e é por isso que a exceção do admin custa uma condição só.
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
		httpio.WriteError(w, status, err.Error())
		return c, false
	}
	return c, true
}

func generateInviteToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
