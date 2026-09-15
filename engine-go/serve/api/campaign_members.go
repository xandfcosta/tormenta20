package api

import (
	"context"
	"database/sql"
	"errors"

	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/platform"
)

var campaignMemberRoles = map[string]bool{"player": true, "gm": true}

// Aqui moravam o `memberDTO` e o `memberScalars`, a forma de FIO de um membro
// para a rota `GET /campanhas/{id}/members`. A rota saiu na ALE-277 por não ter
// consumidor, e os dois ficaram sem chamador — método sem uso não quebra
// compilação, então eles atravessaram a issue inteira. O que os denunciou foi a
// coluna `role` sumindo debaixo deles (ALE-287).

// joinCampaign clona o personagem para a mesa e cria o membro NA MESMA
// transação (ALE-156).
//
// Eram duas antes, e a que falhava era a segunda: um `CreateMember` com erro
// deixava a cópia órfã no banco — e cópia órfã é pior que nada, porque o
// `campaignHasCopyOf` passa a responder "já está na mesa" e o herói fica
// impedido de entrar para sempre, sem membro nenhum que se possa remover para
// desfazer.
//
// A cópia existe porque a ficha da mesa é um instantâneo (ALE-33): editar
// durante a sessão não pode vazar para as outras campanhas.
func (rules campaignRules) joinCampaign(ctx context.Context, sourceID, campaignID, ownerID int64, role string) (sqlcgen.CampaignMember, error) {
	tx, err := rules.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// A checagem é REFEITA aqui dentro, e é isto que fecha a corrida.
	//
	// A de fora existe para a mensagem amigável e para o caminho rápido; ela
	// roda sem transação, então dois pedidos simultâneos passam os dois por
	// ela. Com o `_txlock=immediate` (ALE-156), a transação já nasce com a
	// trava de escrita, então o segundo pedido ESPERA o primeiro terminar — e
	// esta releitura enxerga o que ele gravou.
	//
	// É a mesma forma do commit de movimento no tabuleiro, que reconfere a vez:
	// entre decidir e escrever, a mesa pode ter mudado.
	if err := assertCanJoin(ctx, rules.queries.WithTx(tx), tx, sourceID, campaignID, ownerID, role); err != nil {
		return sqlcgen.CampaignMember{}, err
	}

	copyID, err := cloneCharacterTx(ctx, tx, sourceID, campaignID)
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	member, err := rules.queries.WithTx(tx).CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: copyID, Addedat: platform.NowISO(),
	})
	if err != nil {
		return sqlcgen.CampaignMember{}, err
	}
	return member, tx.Commit()
}

// errAlreadyInCampaign: a releitura dentro da transação achou o que a checagem
// de fora não tinha achado — alguém ganhou a corrida.
var errAlreadyInCampaign = errors.New("personagem já está na campanha")

// assertCanJoin repete, DENTRO da transação, as duas travas que o handler já
// tentou por fora. Repetição de propósito: a de fora é pela mensagem, esta é
// pela verdade (ALE-156).
func assertCanJoin(ctx context.Context, q *sqlcgen.Queries, tx *sql.Tx, sourceID, campaignID, ownerID int64, role string) error {
	if role == "player" {
		hasPc, err := q.HasPlayerPc(ctx, sqlcgen.HasPlayerPcParams{Campaignid: campaignID, Ownerid: ownerID})
		if err != nil {
			return err
		}
		if hasPc {
			return errAlreadyInCampaign
		}
	}
	var hasCopy bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM characters WHERE sourceCharacterId = ? AND campaignId = ?)`,
		sourceID, campaignID).Scan(&hasCopy)
	if err != nil {
		return err
	}
	if hasCopy {
		return errAlreadyInCampaign
	}
	return nil
}
