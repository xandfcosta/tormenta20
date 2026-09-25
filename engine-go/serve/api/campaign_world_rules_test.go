package api

import (
	"context"
	"database/sql"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A EMENDA DE CATÁLOGO É DA MESA, E O MOLDE NÃO O VÊ (ALE-387).
//
// Cada campanha carrega a própria versão de Arton. Numa delas o medalhão de
// prata concede +1 em Luta além do +1 de limite de PM da p160; fora dela, o item
// é só o do livro.
//
// O que se prende é a metade que NENHUM teste do motor alcança: a ligação é
// `characters.campaignId`, e o MOLDE tem essa coluna nula. Uma emenda que
// casasse por dono, por membro ou pelo id do molde vazaria a regra de uma mesa
// para o elenco inteiro — e para as outras mesas do mesmo herói, que é a coisa
// que o modelo de clone existe para impedir.
//
// O caminho é o de PRODUÇÃO: o clone nasce do `campaign.Seating`, e não de um
// INSERT que a bancada escreve. Um arranjo curto mediria o arranjo.
func TestTheCampaignWorldReachesTheClonedSheetAndNotTheTemplate(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()

	owner := seedUser(t, s, "mestre@t20.local")
	campaignID := seedCampaign(t, s, owner)
	templateID := seedCharacter(t, s, owner, "Herói")
	seedFighterWithMedallion(t, s, templateID)

	if err := s.campaignSeating().Seat(ctx, owner, campaignID, templateID, ""); err != nil {
		t.Fatalf("sentar o herói à mesa: %v", err)
	}
	var clonedID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT id FROM characters WHERE sourceCharacterId = ? AND campaignId = ?`,
		templateID, campaignID).Scan(&clonedID); err != nil {
		t.Fatalf("achar a cópia do herói na mesa: %v", err)
	}

	// Antes da emenda os dois têm de ser IGUAIS. É o controle que separa "o
	// emenda chegou" de "o clone já era diferente do molde".
	before, cloneBefore := lutaOf(t, s, templateID), lutaOf(t, s, clonedID)
	if before != cloneBefore {
		t.Fatalf("a cópia já nasceu com Luta diferente do molde (%d contra %d) — o caso mediria o clone, não a emenda",
			cloneBefore, before)
	}

	seedCampaignAmendment(t, s, campaignID, "medalhao-de-prata",
		`[{"target":{"k":"expertise","name":"Luta"},"amount":1,"bonusType":"untyped","note":"Regra da mesa"}]`)

	if got := lutaOf(t, s, clonedID); got != before+1 {
		t.Errorf("a Luta na mesa deu %d e esperava %d (+1 da emenda) — o mundo da campanha não alcançou a ficha jogada", got, before+1)
	}
	if got := lutaOf(t, s, templateID); got != before {
		t.Errorf("a Luta do MOLDE virou %d e era %d — a regra de uma mesa vazou para o elenco.\n"+
			"A ligação é `characters.campaignId`, e a do molde é NULA.", got, before)
	}
}

// seedFighterWithMedallion dá ao herói a perícia e o item que o caso mede.
func seedFighterWithMedallion(t *testing.T, s *Server, characterID int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.queries.CreateExpertise(ctx, sqlcgen.CreateExpertiseParams{
		Characterid: characterID, Name: "Luta", Attribute: "strength", Trained: 1,
	}); err != nil {
		t.Fatalf("semear a perícia Luta: %v", err)
	}
	if _, err := s.queries.CreateItem(ctx, sqlcgen.CreateItemParams{
		Characterid: characterID,
		Catalogid:   sql.NullString{String: "medalhao-de-prata", Valid: true},
		Name:        "Medalhão de prata", Quantity: 1, Slots: 0,
		Equipped:     sql.NullString{String: "wielded", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("semear o medalhão: %v", err)
	}
}

// seedCampaignAmendment escreve a emenda direto na tabela: a tela do mestre que
// o AUTORA ainda não existe, e inventar uma porta para o teste seria a bancada
// escrevendo o que a produção não escreve.
func seedCampaignAmendment(t *testing.T, s *Server, campaignID int64, itemID, adds string) {
	t.Helper()
	if _, err := s.db.ExecContext(context.Background(),
		`INSERT INTO campaign_items (campaignId, itemId, adds, updatedAt) VALUES (?, ?, ?, ?)`,
		campaignID, itemID, adds, dbvalue.NowISO()); err != nil {
		t.Fatalf("gravar a emenda de %q na campanha %d: %v", itemID, campaignID, err)
	}
}

// lutaOf computa a ficha INTEIRA pelo caminho de produção e devolve o total da
// perícia — `LoadAndCompute` é o que as sete telas chamam.
func lutaOf(t *testing.T, s *Server, characterID int64) int {
	t.Helper()
	ctx := context.Background()
	row, err := s.queries.GetCharacter(ctx, characterID)
	if err != nil {
		t.Fatalf("ler a linha da ficha %d: %v", characterID, err)
	}
	computed, err := sheet.LoadAndCompute(ctx, s.queries, s.catalogs, row)
	if err != nil {
		t.Fatalf("computar a ficha %d: %v", characterID, err)
	}
	return lutaIn(t, computed)
}

func lutaIn(t *testing.T, computed engine.ComputedSheet) int {
	t.Helper()
	for _, ex := range computed.Expertises {
		if ex.Name == "Luta" {
			return ex.Total
		}
	}
	t.Fatal("a perícia Luta não apareceu na ficha computada")
	return 0
}

// O ESCOPO DE UMA CONCESSÃO SOBREVIVE AO BANCO (ALE-387).
//
// O motor já prova que o seletor escolhe quem é alcançado; o que só o banco
// prova é a TRADUÇÃO: `campaign_grants.characterId` nulo é a mesa inteira, e
// preenchido é aquela ficha.
//
// DOIS heróis na MESMA mesa, de propósito. Com um só, a concessão específica e a
// da mesa dão o mesmo número, e o caso mediria a metade em que o defeito é
// invisível por construção.
func TestACampaignGrantScopedToOneHeroDoesNotReachTheOther(t *testing.T) {
	s := newTestServer(t)

	owner := seedUser(t, s, "mestre@t20.local")
	guest := seedUser(t, s, "jogadora@t20.local")
	campaignID := seedCampaign(t, s, owner)
	openTheTable(t, s, campaignID, "convite-do-caso")

	mine := seatOwnHero(t, s, owner, campaignID, "Herói do mestre", "")
	theirs := seatOwnHero(t, s, guest, campaignID, "Heroína", "convite-do-caso")

	before, otherBefore := lutaOf(t, s, mine), lutaOf(t, s, theirs)
	if before != otherBefore {
		t.Fatalf("os dois heróis já nasceram com Luta diferente (%d e %d) — o caso mediria a diferença deles",
			before, otherBefore)
	}

	umaLuta := `[{"target":{"k":"expertise","name":"Luta"},"amount":1,"bonusType":"untyped"}]`
	seedCampaignGrant(t, s, "g-mesa", campaignID, sql.NullInt64{}, "Bênção da mesa", umaLuta)
	seedCampaignGrant(t, s, "g-so-dela", campaignID,
		sql.NullInt64{Int64: theirs, Valid: true}, "Pacto da Heroína", umaLuta)

	if got := lutaOf(t, s, theirs); got != before+2 {
		t.Errorf("a heroína tirou %d e esperava %d: a concessão da MESA mais a DELA", got, before+2)
	}
	if got := lutaOf(t, s, mine); got != before+1 {
		t.Errorf("o herói do mestre tirou %d e esperava %d.\n"+
			"Só a concessão da mesa o alcança — a outra tem `characterId` preenchido com a ficha DELA.",
			got, before+1)
	}
}

// openTheTable abre a mesa a quem tem o token: é o que o caminho de produção
// exige de quem não é o dono.
func openTheTable(t *testing.T, s *Server, campaignID int64, invite string) {
	t.Helper()
	if _, err := s.db.ExecContext(context.Background(),
		`UPDATE campaigns SET inviteToken = ? WHERE id = ?`, invite, campaignID); err != nil {
		t.Fatalf("abrir a campanha %d com convite: %v", campaignID, err)
	}
}

// seatOwnHero cria o molde e o SENTA à mesa pelo caminho de produção, devolvendo
// o id da CÓPIA — que é a ficha jogada, e a única que uma concessão alcança.
func seatOwnHero(t *testing.T, s *Server, ownerID, campaignID int64, name, invite string) int64 {
	t.Helper()
	ctx := context.Background()
	templateID := seedCharacter(t, s, ownerID, name)
	seedFighterWithMedallion(t, s, templateID)
	if err := s.campaignSeating().Seat(ctx, ownerID, campaignID, templateID, invite); err != nil {
		t.Fatalf("sentar %q à mesa: %v", name, err)
	}
	var clonedID int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT id FROM characters WHERE sourceCharacterId = ? AND campaignId = ?`,
		templateID, campaignID).Scan(&clonedID); err != nil {
		t.Fatalf("achar a cópia de %q na mesa: %v", name, err)
	}
	return clonedID
}

// seedCampaignGrant escreve a concessão direto na tabela: a tela do mestre que a
// AUTORA ainda não existe.
func seedCampaignGrant(
	t *testing.T, s *Server, id string, campaignID int64,
	characterID sql.NullInt64, label, modifiers string,
) {
	t.Helper()
	if _, err := s.db.ExecContext(context.Background(),
		`INSERT INTO campaign_grants (id, campaignId, characterId, label, modifiers, updatedAt)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, campaignID, characterID, label, modifiers, dbvalue.NowISO()); err != nil {
		t.Fatalf("gravar a concessão %q: %v", id, err)
	}
}
