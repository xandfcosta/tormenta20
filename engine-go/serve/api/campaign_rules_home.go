package api

import (
	"context"
	"database/sql"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/platform"
)

// AS REGRAS DE CAMPANHA E DE MESA, com casa própria: elas respondem às mesmas
// duas perguntas — **de quem é esta mesa** e **quem pode entrar nela** — para
// duas cenas, e nenhuma delas é um servidor.
//
// Carrega as consultas E o `*sql.DB` porque entrar numa mesa CLONA o personagem
// e escreve o membro na MESMA transação: o clone sem o membro é um herói
// duplicado que não está em mesa nenhuma.
//
// O receptor é `rules` e não uma letra: `campaigns.go` já usa `c` para a
// `sqlcgen.Campaign`, e um receptor `c` aqui faz o compilador reclamar de
// `c.Ownerid` num tipo que não tem dono.
type campaignRules struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func (s *Server) campaignRules() campaignRules {
	return campaignRules{db: s.db, queries: s.queries}
}

// createCampaign abre uma mesa, e ela nasce COM link de convite.
//
// O `CreateCampaign` gerado pelo sqlc NÃO escreve o `inviteToken`, e uma mesa
// com a coluna nula não aceita ninguém: o `joinTable` recusa já no
// `!c.Invitetoken.Valid`, antes de olhar o que a pessoa digitou.
//
// Cunhar AQUI e não no `INSERT` é o que garante que os dois caminhos passem por
// isto — a cena de campanhas e a rota JSON que o e2e usa como fixture. É um
// `UPDATE` logo depois do `INSERT` e não uma coluna com `DEFAULT` porque o token
// é aleatório de verdade (`crypto/rand`), e o SQLite não tem de onde tirar isso.
//
// Não é transação: o pior caso é uma mesa sem link, e ele tem conserto pela tela
// (`RotateInvite`).
func (rules campaignRules) createCampaign(
	ctx context.Context, p sqlcgen.CreateCampaignParams,
) (sqlcgen.Campaign, error) {
	c, err := rules.queries.CreateCampaign(ctx, p)
	if err != nil {
		return sqlcgen.Campaign{}, err
	}
	if _, err := rules.rotateInvite(ctx, c.ID); err != nil {
		return sqlcgen.Campaign{}, err
	}
	c.Invitetoken = sql.NullString{String: rules.inviteOf(ctx, c.ID), Valid: true}
	return c, nil
}

// rotateInvite cunha um link novo e INVALIDA o anterior.
//
// É o mesmo gesto para três coisas, e é por isso que ele tem um nome só: a mesa
// que nasce, a mesa antiga que nunca teve link, e o mestre que quer cortar quem
// já tem o link na mão.
func (rules campaignRules) rotateInvite(ctx context.Context, campanhaID int64) (string, error) {
	token := generateInviteToken()
	if _, err := rules.queries.SetInviteToken(ctx, sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: token, Valid: true},
		UpdatedAt:   platform.NowISO(),
		ID:          campanhaID,
	}); err != nil {
		return "", err
	}
	return token, nil
}

// inviteOf é o link de uma mesa, ou "" quando ela não tem um.
//
// Vazio é estado NORMAL e não erro — campanhas antigas nasceram sem link, e o
// que a tela faz com isso é oferecer o botão de gerar.
func (rules campaignRules) inviteOf(ctx context.Context, campanhaID int64) string {
	c, err := rules.queries.GetCampaign(ctx, campanhaID)
	if err != nil || !c.Invitetoken.Valid {
		return ""
	}
	return c.Invitetoken.String
}
