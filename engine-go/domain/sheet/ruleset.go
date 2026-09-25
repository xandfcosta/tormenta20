package sheet

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// O MUNDO EM QUE UMA FICHA É COMPUTADA (ALE-387).
//
// Cada campanha carrega a própria versão de Arton — o livro mais o que o mestre
// mudou —, e é isso que faz a ficha jogada ser um CLONE: dois mundos não se
// afetam. Fora de campanha vale o livro puro, e é o que o MOLDE do elenco vê.
//
// Quem resolve é o carregamento, UMA vez por ficha, e o mundo viaja no agregado
// até quem computa. O motor não aceita computar sem ele — o `Ruleset` é tipo, e
// quem só tem o livro na mão não compila.

// RulesetFor é o mundo de uma campanha, ou o livro puro quando não há campanha.
//
// @example sheet.RulesetFor(ctx, q, book, row.Campaignid)
func RulesetFor(
	ctx context.Context, q *sqlcgen.Queries, book *engine.Catalogs, campaignID sql.NullInt64,
) (*engine.Ruleset, error) {
	if !campaignID.Valid {
		return engine.BookRuleset(book), nil
	}
	mesa, err := amendmentsOf(ctx, q, campaignID.Int64)
	if err != nil {
		return nil, err
	}
	return engine.RulesetOf(book, mesa), nil
}

// RulesetsFor resolve o mundo de VÁRIAS fichas de uma vez, com uma consulta por
// campanha DISTINTA — a mesa inteira costuma dividir um mundo só, e resolver por
// ficha faria a mesma leitura N vezes.
//
// A chave é o id do personagem, e não o da campanha, porque é o personagem que o
// chamador tem na mão.
func RulesetsFor(
	ctx context.Context, q *sqlcgen.Queries, book *engine.Catalogs, campaignOf map[int64]sql.NullInt64,
) (map[int64]*engine.Ruleset, error) {
	puro := engine.BookRuleset(book)
	porCampanha := map[int64]*engine.Ruleset{}
	out := make(map[int64]*engine.Ruleset, len(campaignOf))
	for characterID, campaignID := range campaignOf {
		if !campaignID.Valid {
			out[characterID] = puro
			continue
		}
		if known, ok := porCampanha[campaignID.Int64]; ok {
			out[characterID] = known
			continue
		}
		mesa, err := amendmentsOf(ctx, q, campaignID.Int64)
		if err != nil {
			return nil, err
		}
		world := engine.RulesetOf(book, mesa)
		porCampanha[campaignID.Int64] = world
		out[characterID] = world
	}
	return out, nil
}

// amendmentsOf lê o que UMA mesa mudou no livro.
//
// Ela RECUSA alto o que não decodifica, e é o contrário do que o
// `ListIgnoredRulesForCharacter` faz ao lado — o lado para o qual cada um erra é
// diferente. Regra opcional que não carrega cai no padrão do livro, que é o
// severo; emenda que não carrega tira da ficha um bônus que o mestre escreveu,
// sem uma palavra na tela.
func amendmentsOf(
	ctx context.Context, q *sqlcgen.Queries, campaignID int64,
) (engine.Amendments, error) {
	rows, err := q.ListCampaignItemPatches(ctx, campaignID)
	if err != nil {
		return engine.Amendments{}, fmt.Errorf(
			"ler a emenda de catálogo da campanha %d: %w", campaignID, err)
	}
	if len(rows) == 0 {
		return engine.Amendments{}, nil
	}
	adds := make(map[string]string, len(rows))
	for _, row := range rows {
		adds[row.Itemid] = row.Adds
	}
	return engine.AmendmentsFrom(adds)
}
