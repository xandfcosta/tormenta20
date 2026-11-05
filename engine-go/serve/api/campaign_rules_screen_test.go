package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// A ABA REGRAS DA MESA: só o mestre escreve, e o jogador lê as regras ligadas.

func rulesURL(f sceneFixture, path string) string {
	return fmt.Sprintf("/campanhas/%d/regras-da-mesa%s", f.campaignID, path)
}

func rulesTab(t *testing.T, f sceneFixture, who int64) string {
	t.Helper()
	return f.requests(t, who, http.MethodGet, fmt.Sprintf("/campanhas/%d?tab=regras", f.campaignID), "").Body.String()
}

// espadaDaCasa são os sinais do editor: +2 no ataque com a espada longa, para a mesa inteira.
const espadaDaCasa = `{"table_rule_name":"Espada da casa","table_rule_scope":"mesa","table_rule_count":1,
	"table_rule_kind_1":"item","table_rule_value_1":"espada-longa",
	"table_rule_target":"attack","table_rule_amount":2}`

func savesRule(t *testing.T, f sceneFixture, signals string) string {
	t.Helper()
	rec := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/gravar"), signals)
	if rec.Code != http.StatusOK {
		t.Fatalf("gravar deu %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func ruleCount(t *testing.T, f sceneFixture) (total, enabled int) {
	t.Helper()
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*), COALESCE(SUM(enabled), 0) FROM campaign_rules WHERE campaignId = ?", f.campaignID).Scan(&total, &enabled); err != nil {
		t.Fatalf("contar as regras: %v", err)
	}
	return total, enabled
}

func TestTheGMWritesARuleThatStartsOff(t *testing.T) {
	f := newSceneFixture(t)
	savesRule(t, f, espadaDaCasa)
	if total, enabled := ruleCount(t, f); total != 1 || enabled != 0 {
		t.Fatalf("a regra nasce desligada: %d regras, %d ligadas", total, enabled)
	}
	page := rulesTab(t, f, f.gm)
	for _, want := range []string{"Espada da casa", "Espada longa", "+2 no ataque"} {
		if !strings.Contains(page, want) {
			t.Errorf("a lista não diz %q", want)
		}
	}
}

func TestTheSwitchTurnsTheRuleOnAndTheSheetFollows(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	before := attackByWeapon(t, f, id)["Espada longa"]
	savesRule(t, f, espadaDaCasa)
	if got := attackByWeapon(t, f, id)["Espada longa"]; got != before {
		t.Fatalf("desligada, a regra mudou o ataque: %+d, e era %+d", got, before)
	}
	var rule string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT id FROM campaign_rules WHERE campaignId = ?", f.campaignID).Scan(&rule); err != nil {
		t.Fatal(err)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/"+rule+"/ligar"), ""); rec.Code != http.StatusOK {
		t.Fatalf("ligar deu %d", rec.Code)
	}
	if got := attackByWeapon(t, f, id)["Espada longa"]; got != before+2 {
		t.Errorf("ligada, a regra dá +2: %+d, e era %+d", got, before)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/"+rule+"/ligar"), ""); rec.Code != http.StatusOK {
		t.Fatalf("desligar deu %d", rec.Code)
	}
	if got := attackByWeapon(t, f, id)["Espada longa"]; got != before {
		t.Errorf("desligada de novo, o ataque ficou em %+d, e era %+d", got, before)
	}
}

func TestThePlayerOnlyReadsTheRulesThatAreOn(t *testing.T) {
	f := newSceneFixture(t)
	savesRule(t, f, espadaDaCasa)
	if page := rulesTab(t, f, f.player); strings.Contains(page, "Espada da casa") {
		t.Error("a regra desligada do mestre aparece para o jogador")
	}
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET enabled = 1"); err != nil {
		t.Fatal(err)
	}
	page := rulesTab(t, f, f.player)
	if !strings.Contains(page, "Espada da casa") {
		t.Error("o jogador não lê a regra ligada")
	}
	if strings.Contains(page, "Nova regra") || strings.Contains(page, "/gravar") {
		t.Error("o jogador recebe o editor")
	}
}

func TestOnlyTheGMWrites(t *testing.T) {
	f := newSceneFixture(t)
	for _, path := range []string{"/gravar", "/rascunho", "/editor", "/regra-qualquer/ligar", "/regra-qualquer/excluir"} {
		if rec := f.requests(t, f.player, http.MethodPost, rulesURL(f, path), espadaDaCasa); rec.Code != http.StatusForbidden {
			t.Errorf("%s respondeu %d ao jogador", path, rec.Code)
		}
	}
	if total, _ := ruleCount(t, f); total != 0 {
		t.Errorf("o jogador gravou %d regras", total)
	}
}

// A regra de outra campanha não se alcança pelo endereço desta.
func TestARuleOfAnotherCampaignIsOutOfReach(t *testing.T) {
	f := newSceneFixture(t)
	savesRule(t, f, espadaDaCasa)
	var rule string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT id FROM campaign_rules").Scan(&rule); err != nil {
		t.Fatal(err)
	}
	other := seedCampaign(t, f.s, f.gm)
	f.requests(t, f.gm, http.MethodPost, fmt.Sprintf("/campanhas/%d/regras-da-mesa/%s/ligar", other, rule), "")
	f.requests(t, f.gm, http.MethodPost, fmt.Sprintf("/campanhas/%d/regras-da-mesa/%s/excluir", other, rule), "")
	if total, enabled := ruleCount(t, f); total != 1 || enabled != 0 {
		t.Errorf("o endereço de outra campanha mexeu na regra: %d regras, %d ligadas", total, enabled)
	}
}

func TestARuleSavedAgainReplacesIt(t *testing.T) {
	f := newSceneFixture(t)
	savesRule(t, f, espadaDaCasa)
	var rule string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT id FROM campaign_rules").Scan(&rule); err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(espadaDaCasa, `"table_rule_amount":2`, fmt.Sprintf(`"table_rule_amount":3,"table_rule_id":%q`, rule), 1)
	savesRule(t, f, edited)
	if total, _ := ruleCount(t, f); total != 1 {
		t.Fatalf("editar criou outra regra: %d", total)
	}
	if !strings.Contains(rulesTab(t, f, f.gm), "+3 no ataque") {
		t.Error("a edição não ficou")
	}
}

func TestARuleCanBeRemoved(t *testing.T) {
	f := newSceneFixture(t)
	savesRule(t, f, espadaDaCasa)
	var rule string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT id FROM campaign_rules").Scan(&rule); err != nil {
		t.Fatal(err)
	}
	f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/"+rule+"/excluir"), "")
	if total, _ := ruleCount(t, f); total != 0 {
		t.Errorf("a regra não saiu: %d", total)
	}
}

// A recusa diz o que está errado e o que se esperava, dentro do editor, e não grava nada.
func TestTheEditorRefusesWhatIsWrong(t *testing.T) {
	cases := []struct{ name, signals, says string }{
		{"sem nome", strings.Replace(espadaDaCasa, "Espada da casa", "  ", 1), "nome"},
		{"item que não existe", strings.Replace(espadaDaCasa, "espada-longa", "espada-de-brinquedo", 1), "espada-de-brinquedo"},
		{"espécie que não existe", strings.Replace(espadaDaCasa, `"item"`, `"lua-cheia"`, 1), "lua-cheia"},
		{"alvo que não existe", strings.Replace(espadaDaCasa, `"attack"`, `"sorte"`, 1), "sorte"},
		{"soma zero", strings.Replace(espadaDaCasa, `"table_rule_amount":2`, `"table_rule_amount":0`, 1), "zero"},
		{"perícia que não existe", strings.Replace(strings.Replace(espadaDaCasa, `"attack"`, `"expertise"`, 1), `"table_rule_amount"`, `"table_rule_skill":"Sorte","table_rule_amount"`, 1), "Sorte"},
		{"personagem de fora da mesa", strings.Replace(espadaDaCasa, `"table_rule_scope":"mesa"`, `"table_rule_scope":"um","table_rule_character":"99999"`, 1), "99999"},
		{"PV com postura", strings.Replace(strings.Replace(espadaDaCasa, `"attack"`, `"maxPv"`, 1), `"table_rule_kind_1":"item","table_rule_value_1":"espada-longa"`, `"table_rule_kind_1":"stance","table_rule_value_1":"furia"`, 1), "postura"},
		{"duas posturas", strings.Replace(espadaDaCasa, `"table_rule_count":1,`, `"table_rule_count":3,"table_rule_kind_2":"stance","table_rule_value_2":"furia","table_rule_kind_3":"stance","table_rule_value_3":"postura-muralha",`, 1), "postura"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			body := savesRule(t, f, c.signals)
			if !strings.Contains(body, c.says) || !strings.Contains(body, `role="alert"`) {
				t.Errorf("a recusa devia dizer %q num alerta, e veio %q", c.says, body[:min(300, len(body))])
			}
			if total, _ := ruleCount(t, f); total != 0 {
				t.Errorf("a recusa gravou %d regras", total)
			}
		})
	}
}

// A prévia escreve a regra e diz quem ela alcança hoje.
func TestThePreviewSaysWhoTheRuleReaches(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	var name string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT name FROM characters WHERE id = ?", id).Scan(&name); err != nil {
		t.Fatal(err)
	}
	body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), espadaDaCasa).Body.String()
	if !strings.Contains(body, "+2 no ataque") {
		t.Error("a prévia não escreve o efeito")
	}
	if !strings.Contains(body, name) {
		t.Errorf("a prévia não diz que alcança %s", name)
	}
	other := strings.Replace(espadaDaCasa, "espada-longa", "adaga", 1)
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), other).Body.String(); !strings.Contains(body, "ninguém") {
		t.Error("com a adaga, que ninguém usa, a prévia devia dizer que não alcança ninguém")
	}
}

// Trocar a espécie de condição troca a lista de opções.
func TestTheDraftOffersTheOptionsOfTheKind(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"),
		strings.Replace(espadaDaCasa, `"table_rule_kind_1":"item","table_rule_value_1":"espada-longa"`, `"table_rule_kind_1":"race","table_rule_value_1":""`, 1)).Body.String()
	if !strings.Contains(body, "Minotauro") {
		t.Error("com a espécie raça, o rascunho não oferece as raças")
	}
	if strings.Contains(body, "Espada longa") {
		t.Error("com a espécie raça, o rascunho ainda oferece as armas")
	}
}

// Cada alvo chega ao número dele, e cada número é lido por um caminho.
func TestEveryTargetAddsToItsNumber(t *testing.T) {
	cases := []struct {
		name, target string
		read         func(t *testing.T, f sceneFixture, id int64) int
	}{
		{"dano", `{"k":"damage"}`, func(t *testing.T, f sceneFixture, id int64) int {
			return ruleCard(t, f, id).DamageAll.Total + ruleCard(t, f, id).DamageOwn.Total
		}},
		{"Defesa", `{"k":"defense"}`, func(t *testing.T, f sceneFixture, id int64) int { return ruleSheet(t, f, id).Defense.Total }},
		{"PV", `{"k":"maxPv"}`, func(t *testing.T, f sceneFixture, id int64) int { return int(poolsOf(t, f.s, id).HpMax) }},
		{"PM", `{"k":"maxPm"}`, func(t *testing.T, f sceneFixture, id int64) int { return int(poolsOf(t, f.s, id).MpMax) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			inCampaign(t, f, id)
			saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
			itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
			before := c.read(t, f, id)
			seedsRule(t, f, "Alvo", "[]", fmt.Sprintf(`{"kind":"adds","target":%s,"amount":3}`, c.target))
			if got := c.read(t, f, id); got != before+3 {
				t.Errorf("foi de %d a %d, e a regra dá +3", before, got)
			}
		})
	}
	// A condição da ficha chega ao PV, que é derivado por outro caminho.
	t.Run("PV de quem tem a devoção", func(t *testing.T) {
		f, id := fighterFixture(t)
		inCampaign(t, f, id)
		before := poolsOf(t, f.s, id).HpMax
		seedsRule(t, f, "Devotos", `[{"kind":"devotion","value":"arsenal"}]`, `{"kind":"adds","target":{"k":"maxPv"},"amount":3}`)
		if got := poolsOf(t, f.s, id).HpMax; got != before {
			t.Fatalf("sem a devoção, o PV foi de %d a %d", before, got)
		}
		if _, err := f.s.db.ExecContext(context.Background(), `UPDATE characters SET god = 'arsenal' WHERE id = ?`, id); err != nil {
			t.Fatal(err)
		}
		if got := poolsOf(t, f.s, id).HpMax; got != before+3 {
			t.Errorf("com a devoção, o PV foi de %d a %d, e a regra dá +3", before, got)
		}
	})
	t.Run("margem de ameaça", func(t *testing.T) {
		f, id := fighterFixture(t)
		inCampaign(t, f, id)
		saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
		itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
		before := ruleCard(t, f, id).CritRange
		seedsRule(t, f, "Margem", `[{"kind":"item","value":"espada-longa"}]`, `{"kind":"adds","target":{"k":"critRange"},"amount":2}`)
		if got := ruleCard(t, f, id).CritRange; got != before-2 {
			t.Errorf("a margem foi de %d a %d, e a regra alarga em 2", before, got)
		}
	})
}
