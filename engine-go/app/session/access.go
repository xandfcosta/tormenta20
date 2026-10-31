package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/app"
	"t20engine/infra/db/sqlcgen"
)

// Access é a TRAVA: a sessão existe, é desta campanha, e quem pede é o quê.
//
// Ela é o passo que todo gesto do ciclo faz primeiro, e é por isso que mora
// aqui e não na cena: autorização é metade do que faz um caso de uso valer.
type Access struct {
	queries *sqlcgen.Queries
}

func NewAccess(q *sqlcgen.Queries) Access { return Access{queries: q} }

// RoleIn diz o papel de quem pede numa campanha JÁ CARREGADA.
//
// O dono mestra; o membro joga; e o ASSENTO VENCE o admin: o admin com herói na
// mesa de outra pessoa é jogador nela. Sem isso a página da campanha lhe abria a
// configuração, o acervo e as sessões da mesa em que ele só joga, enquanto a
// própria mesa o tratava como jogador (ALE-526).
//
// Sem assento, o admin mestra: é com as ferramentas de mestre que ele vem
// consertar a mesa de um jogador. Quem não é membro nem admin recebe recusa,
// senão a cena de outra pessoa abriria vazia em vez de barrar.
func (a Access) RoleIn(ctx context.Context, who app.Caller, c sqlcgen.Campaign) (string, error) {
	if c.Ownerid == who.ID {
		return app.RoleGM, nil
	}
	member, err := a.isMember(ctx, who, c.ID)
	if err != nil {
		return "", err
	}
	switch {
	case member:
		return app.RolePlayer, nil
	case who.IsAdmin:
		return app.RoleGM, nil
	default:
		return "", fmt.Errorf("a campanha %d não é acessível para %d: %w", c.ID, who.ID, app.ErrForbidden)
	}
}

func (a Access) isMember(ctx context.Context, who app.Caller, campaignID int64) (bool, error) {
	member, err := a.queries.IsCampaignMember(ctx, sqlcgen.IsCampaignMemberParams{
		Campaignid: campaignID, Ownerid: who.ID,
	})
	if err != nil {
		return false, fmt.Errorf("conferir se %d é membro da campanha %d: %w", who.ID, campaignID, err)
	}
	return member, nil
}

// RoleInCampaign carrega a campanha e devolve o papel.
func (a Access) RoleInCampaign(ctx context.Context, who app.Caller, campaignID int64) (string, error) {
	c, err := a.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("a campanha %d não existe: %w", campaignID, app.ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("carregar a campanha %d: %w", campaignID, err)
	}
	return a.RoleIn(ctx, who, c)
}

// OwnedCampaign é a trava SÓ DO DONO: passa o mestre, e o resto recebe recusa.
//
// Mais estreita que o `RoleIn`, e a diferença é o gesto: renomear, apagar,
// convidar, abrir sessão e montar o acervo são do DONO da campanha; ver a mesa é
// de qualquer membro. O admin passa só onde não tem assento — ver o `RoleIn`.
func (a Access) OwnedCampaign(
	ctx context.Context, who app.Caller, campaignID int64,
) (sqlcgen.Campaign, error) {
	c, err := a.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return c, fmt.Errorf("a campanha %d não existe: %w", campaignID, app.ErrNotFound)
	}
	if err != nil {
		return c, fmt.Errorf("carregar a campanha %d: %w", campaignID, err)
	}
	if c.Ownerid == who.ID {
		return c, nil
	}
	if who.IsAdmin {
		seated, err := a.isMember(ctx, who, campaignID)
		if err != nil {
			return c, err
		}
		if !seated {
			return c, nil
		}
	}
	return c, fmt.Errorf("a campanha %d é de outra pessoa: %w", campaignID, app.ErrForbidden)
}

// Session resolve o papel E a linha da sessão, conferindo que ela é DESTA
// campanha.
//
// A ordem é papel primeiro: quem não alcança a campanha não pode descobrir,
// pela diferença entre 403 e 404, se a sessão existe.
func (a Access) Session(
	ctx context.Context, who app.Caller, campaignID, sessionID int64,
) (sqlcgen.Session, string, error) {
	role, err := a.RoleInCampaign(ctx, who, campaignID)
	if err != nil {
		return sqlcgen.Session{}, "", err
	}
	sess, err := a.queries.GetSession(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && sess.Campaignid != campaignID) {
		return sqlcgen.Session{}, "", fmt.Errorf(
			"a sessão %d não é da campanha %d: %w", sessionID, campaignID, app.ErrNotFound)
	}
	if err != nil {
		return sqlcgen.Session{}, "", fmt.Errorf("carregar a sessão %d: %w", sessionID, err)
	}
	return sess, role, nil
}

// GM resolve a sessão e EXIGE o mestre.
//
// Os quatro gestos do ciclo passam por aqui, e é a única trava deles: a tela
// esconder o botão é cortesia, e quem postar na mão bate nesta linha.
func (a Access) GM(
	ctx context.Context, who app.Caller, campaignID, sessionID int64,
) (sqlcgen.Session, error) {
	sess, role, err := a.Session(ctx, who, campaignID, sessionID)
	if err != nil {
		return sqlcgen.Session{}, err
	}
	if role != app.RoleGM {
		return sqlcgen.Session{}, fmt.Errorf(
			"a sessão %d é mestrada por outra pessoa: %w", sessionID, app.ErrForbidden)
	}
	return sess, nil
}
