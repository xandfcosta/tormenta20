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

// amendmentsOf lê o que UMA mesa mudou no livro: as duas espécies, numa função
// só, porque as duas respondem à mesma campanha e quem as usa as quer juntas.
func amendmentsOf(
	ctx context.Context, q *sqlcgen.Queries, campaignID int64,
) (engine.Amendments, error) {
	out := engine.Amendments{}

	// A EMENDA DE VERBETE: o que a mesa acrescenta a uma entrada do livro.
	entries, err := q.ListCampaignItemPatches(ctx, campaignID)
	if err != nil {
		return out, fmt.Errorf("ler a emenda de verbete da campanha %d: %w", campaignID, err)
	}
	if len(entries) > 0 {
		out.Entries = make(map[string][]engine.Modifier, len(entries))
		for _, row := range entries {
			mods, err := engine.ParseModifiers(row.Adds)
			if err != nil {
				return engine.Amendments{}, fmt.Errorf(
					"a emenda da campanha %d para o verbete %q: %w", campaignID, row.Itemid, err)
			}
			out.Entries[row.Itemid] = mods
		}
	}

	// A EMENDA DE POPULAÇÃO: o que a mesa concede, e a quem.
	grants, err := q.ListCampaignGrants(ctx, campaignID)
	if err != nil {
		return out, fmt.Errorf("ler as concessões da campanha %d: %w", campaignID, err)
	}
	for _, row := range grants {
		mods, err := engine.ParseModifiers(row.Modifiers)
		if err != nil {
			return engine.Amendments{}, fmt.Errorf(
				"a concessão %q da campanha %d: %w", row.ID, campaignID, err)
		}
		out.Grants = append(out.Grants, engine.CampaignGrant{
			ID:        row.ID,
			Applies:   selectorOf(row.Characterid),
			Label:     row.Label,
			Modifiers: mods,
		})
	}
	return out, nil
}

// selectorOf traduz a coluna do banco no escopo do motor: NULA é a campanha
// inteira, preenchida é aquela ficha.
//
// A tradução mora AQUI e não no motor de propósito — `NULL` é vocabulário de
// banco, e o motor é uma função pura que não sabe o que é uma coluna.
func selectorOf(characterID sql.NullInt64) engine.Selector {
	if !characterID.Valid {
		return engine.EveryoneIn()
	}
	return engine.OnlyCharacter(int(characterID.Int64))
}
