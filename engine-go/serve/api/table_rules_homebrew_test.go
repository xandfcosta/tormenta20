package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"t20engine/domain/engine"
	"testing"

	"t20engine/domain/sheet"
)

// O HOMEBREW QUE SAIU DO CÓDIGO volta como duas regras da mesa, criadas DESLIGADAS pela migração nas
// campanhas que têm ficha com o medalhão de prata ou com a Equilibrada. O mestre as liga.

// medallionCutOf é quanto o medalhão corta do custo das magias pessoais (p160): o instrumento que diz se o
// bônus de empunhar dele vale.
func medallionCutOf(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	cut := 0
	for _, c := range dto.Ruleset.ComputeSheet(ec, sheet.ToStringSet(dto.Conditionals)).ForSpell("pmCost", engine.SpellAt{Range: "pessoal"}).OnlyHere {
		cut -= c.Amount
	}
	return cut
}

func runsTheHomebrewMigration(t *testing.T, f sceneFixture) {
	t.Helper()
	raw, err := os.ReadFile("../../infra/db/migrations/00031_homebrew_becomes_table_rules.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	if _, err := f.s.db.ExecContext(context.Background(), up); err != nil {
		t.Fatalf("rodar a migração: %v", err)
	}
}

func TestTheHomebrewComesBackAsTwoRulesThatStartOff(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais","armas-exoticas"]`)
	itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	axe := itemSemeia(t, f, id, "machado-taurico", "Machado táurico", "wielded")
	if _, err := f.s.db.ExecContext(context.Background(), `UPDATE character_items SET improvements = '["melhoria-equilibrada"]' WHERE id = ?`, axe); err != nil {
		t.Fatal(err)
	}
	other := seedCampaign(t, f.s, f.gm)

	plainCut, plainAttack := medallionCutOf(t, f, id), attackByWeapon(t, f, id)["Machado táurico"]
	runsTheHomebrewMigration(t, f)

	var total, enabled, elsewhere int
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*), COALESCE(SUM(enabled), 0) FROM campaign_rules WHERE campaignId = ?", f.campaignID).Scan(&total, &enabled); err != nil {
		t.Fatal(err)
	}
	if total != 2 || enabled != 0 {
		t.Fatalf("a campanha que usa os dois devia ganhar 2 regras desligadas, e ganhou %d, %d ligadas", total, enabled)
	}
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM campaign_rules WHERE campaignId = ?", other).Scan(&elsewhere); err != nil {
		t.Fatal(err)
	}
	if elsewhere != 0 {
		t.Errorf("a campanha sem medalhão nem Equilibrada ganhou %d regras", elsewhere)
	}
	if got := medallionCutOf(t, f, id); got != plainCut {
		t.Errorf("desligada, a regra do medalhão mudou o corte de custo: de %d a %d", plainCut, got)
	}

	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET enabled = 1"); err != nil {
		t.Fatal(err)
	}
	if got := medallionCutOf(t, f, id); got != plainCut+1 {
		t.Errorf("ligada, a regra devolve ao medalhão vestido o -1 PM de empunhar: de %d a %d", plainCut, got)
	}
	if got := attackByWeapon(t, f, id)["Machado táurico"]; got != plainAttack+2 {
		t.Errorf("ligada, a Equilibrada anula a desbalanceada: o machado foi de %+d a %+d", plainAttack, got)
	}
}

// Sem regra, o medalhão vestido não dá o bônus de empunhar, e o livro não deixa vesti-lo.
func TestWithoutTheRuleTheMedallionIsTheBooks(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	medal := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "")
	base := medallionCutOf(t, f, id)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET equipped = 'vested' WHERE id = ?", medal); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		"INSERT INTO character_conditionals (characterId, conditionalId) VALUES (?, 'flag:homebrew-vestido-medalhao-de-prata')", id); err != nil {
		t.Fatal(err)
	}
	if got := medallionCutOf(t, f, id); got != base {
		t.Errorf("sem regra, o medalhão vestido deu o corte de empunhar: de %d a %d", base, got)
	}
	vest := fmt.Sprintf("/personagens/%d/itens/%d/equipa/vested?tab=bag", id, medal)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, vest, "").Body.String()); refusal == "" {
		t.Error("sem regra, o medalhão se vestiu")
	}
}

// A campanha em que o mestre já escreveu a regra à mão não ganha uma segunda: as duas ligadas somariam.
func TestTheMigrationSkipsTheCampaignThatAlreadyHasTheRule(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	axe := itemSemeia(t, f, id, "machado-taurico", "Machado táurico", "wielded")
	if _, err := f.s.db.ExecContext(context.Background(), `UPDATE character_items SET improvements = '["melhoria-equilibrada"]' WHERE id = ?`, axe); err != nil {
		t.Fatal(err)
	}
	seedsRule(t, f, "Minha Equilibrada", `[{"kind":"trait","value":"desbalanceada"},{"kind":"improvement","value":"melhoria-equilibrada"}]`, `{"kind":"adds","target":{"k":"attack"},"amount":2}`)
	runsTheHomebrewMigration(t, f)
	var equilibrada, medallion int
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM campaign_rules WHERE campaignId = ? AND conditions LIKE '%melhoria-equilibrada%'", f.campaignID).Scan(&equilibrada); err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM campaign_rules WHERE campaignId = ? AND conditions LIKE '%medalhao-de-prata%'", f.campaignID).Scan(&medallion); err != nil {
		t.Fatal(err)
	}
	if equilibrada != 1 {
		t.Errorf("o mestre já tinha a regra da Equilibrada, e a campanha ficou com %d", equilibrada)
	}
	if medallion != 1 {
		t.Errorf("a regra do medalhão devia nascer, e a campanha tem %d", medallion)
	}
}
