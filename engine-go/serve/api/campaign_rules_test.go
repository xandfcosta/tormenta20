package api

import (
	"context"
	"strings"
	"testing"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
)

// AS REGRAS DA MESA: o mestre liga uma regra, e ela soma na ficha com o nome dela na conta.
// A regra desligada, a de outra campanha e o molde do elenco ficam como o livro.

// inCampaign põe a ficha numa campanha, que é o que dá a ela o mundo daquela mesa.
func inCampaign(t *testing.T, f sceneFixture, id int64) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE characters SET campaignId = ? WHERE id = ?", f.campaignID, id); err != nil {
		t.Fatalf("pôr a ficha na campanha: %v", err)
	}
}

// seedsRule grava a regra ligada na campanha da bancada. A tela que escreve é a fatia seguinte.
func seedsRule(t *testing.T, f sceneFixture, name, conditions, effect string) string {
	t.Helper()
	id := "regra-" + strings.ReplaceAll(strings.ToLower(name), " ", "-")
	if _, err := f.s.db.ExecContext(context.Background(),
		`INSERT INTO campaign_rules (id, campaignId, name, enabled, conditions, effect, updatedAt) VALUES (?, ?, ?, 1, ?, ?, ?)`,
		id, f.campaignID, name, conditions, effect, dbvalue.NowISO()); err != nil {
		t.Fatalf("gravar a regra %q: %v", name, err)
	}
	return id
}

const plusTwoAttack = `{"kind":"adds","target":{"k":"attack"},"amount":2}`

func TestARuleRaisesOnlyTheWeaponItNames(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded2")
	before := attackByWeapon(t, f, id)

	seedsRule(t, f, "Espada da casa", `[{"kind":"item","value":"espada-longa"}]`, plusTwoAttack)

	after := attackByWeapon(t, f, id)
	if after["Espada longa"] != before["Espada longa"]+2 {
		t.Errorf("a espada foi de %+d a %+d, e a regra dá +2", before["Espada longa"], after["Espada longa"])
	}
	if after["Adaga"] != before["Adaga"] {
		t.Errorf("a adaga foi de %+d a %+d, e a regra não a nomeia", before["Adaga"], after["Adaga"])
	}
}

func TestARuleThatIsOffChangesNothing(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	before := attackByWeapon(t, f, id)
	rule := seedsRule(t, f, "Desligada", `[{"kind":"item","value":"espada-longa"}]`, plusTwoAttack)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET enabled = 0 WHERE id = ?", rule); err != nil {
		t.Fatalf("desligar a regra: %v", err)
	}
	if after := attackByWeapon(t, f, id); after["Espada longa"] != before["Espada longa"] {
		t.Errorf("a regra desligada mudou o ataque de %+d para %+d", before["Espada longa"], after["Espada longa"])
	}
}

// Cada mesa tem o mundo dela, e fora de campanha vale o livro: é o molde do elenco.
func TestARuleStaysInItsCampaign(t *testing.T) {
	f, id := fighterFixture(t)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	inCampaign(t, f, id)
	before := attackByWeapon(t, f, id)
	rule := seedsRule(t, f, "Da outra mesa", `[{"kind":"item","value":"espada-longa"}]`, plusTwoAttack)
	if after := attackByWeapon(t, f, id); after["Espada longa"] != before["Espada longa"]+2 {
		t.Fatalf("o controle: na mesa dela, a regra dá +2, e veio %+d de %+d", after["Espada longa"], before["Espada longa"])
	}
	other := seedCampaign(t, f.s, f.gm)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET campaignId = ? WHERE id = ?", other, rule); err != nil {
		t.Fatalf("levar a regra: %v", err)
	}
	if after := attackByWeapon(t, f, id); after["Espada longa"] != before["Espada longa"] {
		t.Errorf("a regra de outra mesa mudou esta ficha: %+d virou %+d", before["Espada longa"], after["Espada longa"])
	}
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET campaignId = ? WHERE id = ?", f.campaignID, rule); err != nil {
		t.Fatalf("trazer a regra: %v", err)
	}
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE characters SET campaignId = NULL WHERE id = ?", id); err != nil {
		t.Fatalf("tirar a ficha da campanha: %v", err)
	}
	if after := attackByWeapon(t, f, id); after["Espada longa"] != before["Espada longa"] {
		t.Errorf("a regra vazou para a ficha sem campanha: %+d virou %+d", before["Espada longa"], after["Espada longa"])
	}
}

// As condições do mesmo item valem juntas, no mesmo item.
func TestItemConditionsLookAtTheSameItem(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	plain := attackByWeapon(t, f, id)
	seedsRule(t, f, "Espada certeira", `[{"kind":"item","value":"espada-longa"},{"kind":"improvement","value":"melhoria-certeira"}]`, plusTwoAttack)
	if after := attackByWeapon(t, f, id); after["Espada longa"] != plain["Espada longa"] {
		t.Fatalf("sem a melhoria, a regra não vale: %+d virou %+d", plain["Espada longa"], after["Espada longa"])
	}
	line := bagLinesOf(t, f, id, "Espada longa")[0].ID
	if _, err := f.s.db.ExecContext(context.Background(), `UPDATE character_items SET improvements = '["melhoria-certeira"]' WHERE id = ?`, line); err != nil {
		t.Fatalf("pôr a melhoria: %v", err)
	}
	// A Certeira também sobe o ataque, e o que se mede é o +2 da regra acima dela.
	withImprovement := attackByWeapon(t, f, id)["Espada longa"]
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET enabled = 0"); err != nil {
		t.Fatalf("desligar: %v", err)
	}
	if without := attackByWeapon(t, f, id)["Espada longa"]; withImprovement != without+2 {
		t.Errorf("com a melhoria, a regra dá +2: %+d com ela, %+d sem", withImprovement, without)
	}
}

// Um item guardado na mochila não vale.
func TestAnItemRuleNeedsTheItemInUse(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "")
	before := lutaOf(t, f.s, id)
	seedsRule(t, f, "Medalhão", `[{"kind":"item","value":"medalhao-de-prata"}]`, `{"kind":"adds","target":{"k":"expertise","name":"Luta"},"amount":1}`)
	if got := lutaOf(t, f.s, id); got != before {
		t.Fatalf("o medalhão guardado deu %+d em Luta, e era %+d", got, before)
	}
	line := bagLinesOf(t, f, id, "Medalhão de prata")[0].ID
	if _, err := f.s.db.ExecContext(context.Background(), `UPDATE character_items SET equipped = 'vested' WHERE id = ?`, line); err != nil {
		t.Fatalf("vestir: %v", err)
	}
	if got := lutaOf(t, f.s, id); got != before+1 {
		t.Errorf("o medalhão vestido devia dar +1 em Luta: %+d, e era %+d", got, before)
	}
}

// A regra pode alcançar uma ficha só.
func TestARuleCanReachOneCharacter(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	before := lutaOf(t, f.s, id)
	rule := seedsRule(t, f, "Só do outro", `[]`, `{"kind":"adds","target":{"k":"expertise","name":"Luta"},"amount":1}`)
	if got := lutaOf(t, f.s, id); got != before+1 {
		t.Fatalf("sem seletor, a regra vale para a mesa: %+d, e era %+d", got, before)
	}
	other := seedCharacter(t, f.s, f.player, "Outro")
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET characterId = ? WHERE id = ?", other, rule); err != nil {
		t.Fatalf("apontar a regra: %v", err)
	}
	if got := lutaOf(t, f.s, id); got != before {
		t.Errorf("a regra de outra ficha mudou esta: %+d, e era %+d", got, before)
	}
}

func TestCharacterConditions(t *testing.T) {
	adds := `{"kind":"adds","target":{"k":"expertise","name":"Luta"},"amount":1}`
	cases := []struct {
		name, condition string
		arrange         func(t *testing.T, f sceneFixture, id int64)
	}{
		{"raça", `{"kind":"race","value":"minotauro"}`, func(t *testing.T, f sceneFixture, id int64) { seedRaca(t, f.s, id, "minotauro") }},
		{"classe", `{"kind":"class","value":"Guerreiro"}`, func(t *testing.T, f sceneFixture, id int64) { seedClasse(t, f.s, id, "Guerreiro", 1) }},
		{"devoção", `{"kind":"devotion","value":"arsenal"}`, func(t *testing.T, f sceneFixture, id int64) {
			if _, err := f.s.db.ExecContext(context.Background(), `UPDATE characters SET god = 'arsenal' WHERE id = ?`, id); err != nil {
				t.Fatal(err)
			}
		}},
		{"poder", `{"kind":"power","value":"vontade-de-ferro"}`, func(t *testing.T, f sceneFixture, id int64) {
			seedPowerChoices(t, f.s, id, `["vontade-de-ferro"]`, "{}")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id := barbaro(t, 3)
			inCampaign(t, f, id)
			seedPericia(t, f.s, id, "Luta", "strength", false)
			before := lutaOf(t, f.s, id)
			seedsRule(t, f, "Condição", "["+c.condition+"]", adds)
			if got := lutaOf(t, f.s, id); got != before {
				t.Fatalf("sem a condição, a regra deu %+d, e era %+d", got, before)
			}
			c.arrange(t, f, id)
			now := lutaOf(t, f.s, id)
			if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET enabled = 0"); err != nil {
				t.Fatal(err)
			}
			if without := lutaOf(t, f.s, id); now != without+1 {
				t.Errorf("com a condição, a regra dá +1: %+d com ela, %+d sem", now, without)
			}
		})
	}
}

// A postura vale enquanto está ligada: é o mesmo interruptor das outras.
func TestAStanceConditionFollowsTheStance(t *testing.T) {
	f, id := barbaro(t, 3)
	inCampaign(t, f, id)
	seedPericia(t, f.s, id, "Luta", "strength", false)
	before := lutaOf(t, f.s, id)
	seedsRule(t, f, "Em fúria", `[{"kind":"stance","value":"furia"}]`, `{"kind":"adds","target":{"k":"expertise","name":"Luta"},"amount":1}`)
	if got := lutaOf(t, f.s, id); got != before {
		t.Fatalf("fora da fúria, a regra deu %+d, e era %+d", got, before)
	}
	if refusal := powerCommand(t, f, id, "postura/furia/entra", ""); refusal != "" {
		t.Fatalf("entrar em fúria: %s", refusal)
	}
	now := lutaOf(t, f.s, id)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET enabled = 0"); err != nil {
		t.Fatal(err)
	}
	if without := lutaOf(t, f.s, id); now != without+1 {
		t.Errorf("em fúria, a regra dá +1: %+d com ela, %+d sem", now, without)
	}
}

// Uma espécie de condição que este motor não conhece não alcança ninguém: é o lado seguro.
func TestAnUnknownConditionReachesNobody(t *testing.T) {
	f, id := barbaro(t, 3)
	inCampaign(t, f, id)
	seedPericia(t, f.s, id, "Luta", "strength", false)
	before := lutaOf(t, f.s, id)
	seedsRule(t, f, "Do futuro", `[{"kind":"lua-cheia","value":"sim"}]`, `{"kind":"adds","target":{"k":"expertise","name":"Luta"},"amount":1}`)
	if got := lutaOf(t, f.s, id); got != before {
		t.Errorf("a condição desconhecida passou: %+d, e era %+d", got, before)
	}
}

// Quem abre a conta sabe de onde veio o número.
func TestTheSheetNamesTheRuleInTheBreakdown(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	seedsRule(t, f, "Espada da casa", `[{"kind":"item","value":"espada-longa"}]`, plusTwoAttack)
	if screen := sheetTab(t, f, id, "combat"); !strings.Contains(screen, "Espada da casa") {
		t.Error("a aba Combate não diz o nome da regra na conta")
	}
	dto, ec := loadedCharacter(t, f, id)
	found := false
	for _, card := range dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)) {
		for _, c := range card.AttackOwn.Contributions {
			found = found || strings.Contains(c.Source, "Espada da casa")
		}
	}
	if !found {
		t.Error("o ataque da carta não leva a origem da regra")
	}
}
