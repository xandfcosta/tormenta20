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
// O dono mestra; o admin também, e a exceção custa uma condição só porque esta
// é a única pergunta de papel da casa. Quem não é membro não recebe "player":
// recebe recusa, senão a cena de outra pessoa abriria vazia em vez de barrar.
func (a Access) RoleIn(ctx context.Context, quem app.Caller, c sqlcgen.Campaign) (string, error) {
	if c.Ownerid == quem.ID || quem.IsAdmin {
		return app.RoleGM, nil
	}
	membro, err := a.queries.IsCampaignMember(ctx, sqlcgen.IsCampaignMemberParams{
		Campaignid: c.ID, Ownerid: quem.ID,
	})
	if err != nil {
		return "", fmt.Errorf("conferir se %d é membro da campanha %d: %w", quem.ID, c.ID, err)
	}
	if !membro {
		return "", fmt.Errorf("a campanha %d não é acessível para %d: %w", c.ID, quem.ID, app.ErrForbidden)
	}
	return app.RolePlayer, nil
}

// RoleInCampaign carrega a campanha e devolve o papel.
func (a Access) RoleInCampaign(ctx context.Context, quem app.Caller, campaignID int64) (string, error) {
	c, err := a.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("a campanha %d não existe: %w", campaignID, app.ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("carregar a campanha %d: %w", campaignID, err)
	}
	return a.RoleIn(ctx, quem, c)
}

// OwnedCampaign é a trava SÓ DO DONO: passa o mestre, e o resto recebe recusa.
//
// Mais estreita que o `RoleIn`, e a diferença é o gesto: renomear, apagar,
// convidar, abrir sessão e montar o acervo são do DONO da campanha; ver a mesa é
// de qualquer membro. O admin passa pela mesma porta, e a exceção custa uma
// condição só.
func (a Access) OwnedCampaign(
	ctx context.Context, quem app.Caller, campaignID int64,
) (sqlcgen.Campaign, error) {
	c, err := a.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return c, fmt.Errorf("a campanha %d não existe: %w", campaignID, app.ErrNotFound)
	}
	if err != nil {
		return c, fmt.Errorf("carregar a campanha %d: %w", campaignID, err)
	}
	if c.Ownerid != quem.ID && !quem.IsAdmin {
		return c, fmt.Errorf("a campanha %d é de outra pessoa: %w", campaignID, app.ErrForbidden)
	}
	return c, nil
}

// Session resolve o papel E a linha da sessão, conferindo que ela é DESTA
// campanha.
//
// A ordem é papel primeiro: quem não alcança a campanha não pode descobrir,
// pela diferença entre 403 e 404, se a sessão existe.
func (a Access) Session(
	ctx context.Context, quem app.Caller, campaignID, sessionID int64,
) (sqlcgen.Session, string, error) {
	papel, err := a.RoleInCampaign(ctx, quem, campaignID)
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
	return sess, papel, nil
}

// GM resolve a sessão e EXIGE o mestre.
//
// Os quatro gestos do ciclo passam por aqui, e é a única trava deles: a tela
// esconder o botão é cortesia, e quem postar na mão bate nesta linha.
func (a Access) GM(
	ctx context.Context, quem app.Caller, campaignID, sessionID int64,
) (sqlcgen.Session, error) {
	sess, papel, err := a.Session(ctx, quem, campaignID, sessionID)
	if err != nil {
		return sqlcgen.Session{}, err
	}
	if papel != app.RoleGM {
		return sqlcgen.Session{}, fmt.Errorf(
			"a sessão %d é mestrada por outra pessoa: %w", sessionID, app.ErrForbidden)
	}
	return sess, nil
}
