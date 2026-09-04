package api

import (
	"context"
	"database/sql"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// AS REGRAS DE CAMPANHA E DE MESA, com casa própria (ALE-278, fatia 6).
//
// Treze métodos que respondem às mesmas duas perguntas: **de quem é esta mesa**
// e **quem pode entrar nela**. Eram do `*Server` pelo motivo de sempre — os
// handlers JSON que os chamavam eram dele — e a ALE-277 apagou esses handlers.
// Sobraram duas chamadoras, e nenhuma é um servidor: a cena de campanhas e a
// cena da Mesa.
//
// O `resolveRole` é a mais compartilhada e a que mais justifica o tipo: ela
// decide o PAPEL de quem pede (dono é "gm", quem tem personagem membro é
// "player", o resto é recusa), e as duas cenas dependem dela para desenhar
// coisas diferentes. Deixá-la no servidor obrigaria as duas a receber um
// servidor.
//
// Ele carrega as consultas e o `*sql.DB` — entrar numa mesa CLONA o personagem
// e escreve o membro na MESMA transação, e o clone sem o membro é um herói
// duplicado que não está em mesa nenhuma.
//
// O receptor é `rules` e não uma letra, e isso não é estilo: `campaigns.go` já
// usa `c` para a `sqlcgen.Campaign` em nove lugares, e um receptor `c` fez o
// compilador reclamar de `c.Ownerid` num tipo que não tem dono. A primeira
// tentativa desta fatia morreu assim.
type campaignRules struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

func (s *Server) campaignRules() campaignRules {
	return campaignRules{db: s.db, queries: s.queries}
}

// createCampaign abre uma mesa, e ela nasce COM link de convite (ALE-287).
//
// O `CreateCampaign` gerado pelo sqlc não escreve o `inviteToken`, e por isso
// toda mesa aberta pela tela nascia com a coluna nula. Isso não era um recurso
// faltando: o `joinTable` recusa quem não é dono já no `!c.Invitetoken.Valid`,
// antes de olhar o que a pessoa digitou — então **a mesa não aceitava
// ninguém**. As únicas em que alguém entrava eram as seis da `seed.sql`, com
// `seedtoken-0N` escrito à mão.
//
// Cunhar aqui e não no `INSERT` é o que garante que os DOIS caminhos passem por
// isto: a cena de campanhas e a rota JSON que a suíte de e2e usa como fixture.
// Um `UPDATE` logo depois do `INSERT` e não uma coluna com `DEFAULT` porque o
// token é aleatório de verdade (`crypto/rand`), e SQLite não tem de onde tirar
// isso.
//
// Não é transação: o pior caso é uma mesa sem link, que é exatamente o estado
// de antes desta issue — e ele tem conserto pela tela (`RotateInvite`), que é o
// mesmo botão que serve às mesas nascidas antes daqui.
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
// que nasce, a mesa antiga que nunca teve link (as criadas antes da ALE-287), e
// o mestre que quer cortar quem já tem o link na mão. A tela oferece as duas
// últimas pelo mesmo botão.
func (rules campaignRules) rotateInvite(ctx context.Context, campanhaID int64) (string, error) {
	token := generateInviteToken()
	if _, err := rules.queries.SetInviteToken(ctx, sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: token, Valid: true},
		UpdatedAt:   plataforma.NowISO(),
		ID:          campanhaID,
	}); err != nil {
		return "", err
	}
	return token, nil
}

// inviteOf é o link de uma mesa, ou "" quando ela não tem um.
//
// Vazio é estado NORMAL e não erro: toda campanha aberta antes da ALE-287
// nasceu sem link, e o que a tela faz com isso é oferecer o botão de gerar.
func (rules campaignRules) inviteOf(ctx context.Context, campanhaID int64) string {
	c, err := rules.queries.GetCampaign(ctx, campanhaID)
	if err != nil || !c.Invitetoken.Valid {
		return ""
	}
	return c.Invitetoken.String
}
