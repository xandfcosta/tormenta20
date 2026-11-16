package api

import (
	"context"
	"testing"
)

// A CÓPIA DA MESA LEVA O QUE O JOGADOR DECIDIU SOBRE CADA ITEM.
//
// O herói que senta numa mesa é uma cópia do rascunho, e as decisões sobre um item
// moram em tabelas filhas DELE: a escola do tomo hermético, a magia do Harmonizado e a
// marca "feita por mim" que o Ferreiro e o Cano Raiado leem (p69). O clone copiava o
// encanto, a escolha do encanto, a marca de categoria maior e as doses, e deixava estas
// três para trás: a clava do Ferreiro chegava à mesa rolando o dado de antes.
//
// A BANDANA VEM PRIMEIRO de propósito: a cópia liga cada item ao original pela POSIÇÃO
// na mochila, e uma linha presa no item errado passaria num caso de item único.
func TestTheCampaignCopyKeepsTheItemDecisions(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "bandana", "Bandana", "vested")
	item := itemSemeia(t, f, f.charID, "adaga", "Adaga", "wielded")
	ctx := context.Background()
	for _, seed := range []string{
		`INSERT INTO character_item_choices (itemId, choice) VALUES (?, 'evocacao')`,
		`INSERT INTO character_item_improvement_choices (itemId, improvementId, choice) VALUES (?, 'harmonizada', 'arma-magica')`,
		`INSERT INTO character_item_made_by_owner (itemId) VALUES (?)`,
	} {
		if _, err := f.s.db.ExecContext(ctx, seed, item); err != nil {
			t.Fatalf("semear %q: %v", seed, err)
		}
	}

	copied := cloneForCampaignTest(t, f)

	for _, c := range []struct {
		what, query, want string
	}{
		{"a escolha do item", `SELECT c.choice FROM character_item_choices c
			JOIN character_items i ON i.id = c.itemId WHERE i.characterId = ? AND i.catalogId = 'adaga'`, "evocacao"},
		{"a escolha da melhoria", `SELECT c.improvementId || ':' || c.choice FROM character_item_improvement_choices c
			JOIN character_items i ON i.id = c.itemId WHERE i.characterId = ? AND i.catalogId = 'adaga'`, "harmonizada:arma-magica"},
		{"a marca de feita por mim", `SELECT i.catalogId FROM character_item_made_by_owner m
			JOIN character_items i ON i.id = m.itemId WHERE i.characterId = ?`, "adaga"},
	} {
		var got string
		if err := f.s.db.QueryRowContext(ctx, c.query, copied).Scan(&got); err != nil {
			t.Errorf("a cópia não trouxe %s: %v", c.what, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s da cópia é %q, e o rascunho tinha %q", c.what, got, c.want)
		}
	}
}
