package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/infra/db/sqlcgen"
)

// AS RECUSAS, tipadas. Quem traduz para um número é o transporte.
var (
	// ErrNotFound é "não existe, ou não é desta campanha" — os dois casos
	// respondem a mesma coisa de propósito: distinguir contaria a quem não
	// pertence à campanha quais sessões existem nela.
	ErrNotFound = errors.New("não encontrada")
	// ErrForbidden é "existe, você a alcança, mas este gesto não é seu".
	ErrForbidden = errors.New("não é seu")
	// ErrRefused é a REGRA dizendo não — encerrar o que nunca começou.
	ErrRefused = errors.New("recusado pela regra")
)

// Caller é quem está pedindo, na forma que o caso de uso precisa: um id e se
// administra.
//
// Não é o usuário inteiro, e não é o `*http.Request`: os dois são do
// transporte, e um caso de uso que os recebesse não poderia ser chamado de
// outro lugar.
type Caller struct {
	ID      int64
	IsAdmin bool
}

// Os dois papéis de quem senta numa campanha.
const (
	RoleGM     = "gm"
	RolePlayer = "player"
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
func (a Access) RoleIn(ctx context.Context, quem Caller, c sqlcgen.Campaign) (string, error) {
	if c.Ownerid == quem.ID || quem.IsAdmin {
		return RoleGM, nil
	}
	membro, err := a.queries.IsCampaignMember(ctx, sqlcgen.IsCampaignMemberParams{
		Campaignid: c.ID, Ownerid: quem.ID,
	})
	if err != nil {
		return "", fmt.Errorf("conferir se %d é membro da campanha %d: %w", quem.ID, c.ID, err)
	}
	if !membro {
		return "", fmt.Errorf("a campanha %d não é acessível para %d: %w", c.ID, quem.ID, ErrForbidden)
	}
	return RolePlayer, nil
}

// RoleInCampaign carrega a campanha e devolve o papel.
func (a Access) RoleInCampaign(ctx context.Context, quem Caller, campaignID int64) (string, error) {
	c, err := a.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("a campanha %d não existe: %w", campaignID, ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("carregar a campanha %d: %w", campaignID, err)
	}
	return a.RoleIn(ctx, quem, c)
}

// Session resolve o papel E a linha da sessão, conferindo que ela é DESTA
// campanha.
//
// A ordem é papel primeiro: quem não alcança a campanha não pode descobrir,
// pela diferença entre 403 e 404, se a sessão existe.
func (a Access) Session(
	ctx context.Context, quem Caller, campaignID, sessionID int64,
) (sqlcgen.Session, string, error) {
	papel, err := a.RoleInCampaign(ctx, quem, campaignID)
	if err != nil {
		return sqlcgen.Session{}, "", err
	}
	sess, err := a.queries.GetSession(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && sess.Campaignid != campaignID) {
		return sqlcgen.Session{}, "", fmt.Errorf(
			"a sessão %d não é da campanha %d: %w", sessionID, campaignID, ErrNotFound)
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
	ctx context.Context, quem Caller, campaignID, sessionID int64,
) (sqlcgen.Session, error) {
	sess, papel, err := a.Session(ctx, quem, campaignID, sessionID)
	if err != nil {
		return sqlcgen.Session{}, err
	}
	if papel != RoleGM {
		return sqlcgen.Session{}, fmt.Errorf(
			"a sessão %d é mestrada por outra pessoa: %w", sessionID, ErrForbidden)
	}
	return sess, nil
}
