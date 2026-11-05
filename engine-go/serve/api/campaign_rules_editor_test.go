package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/sheet"
	"t20engine/infra/events"
)

// O EDITOR DAS REGRAS DA MESA diante do que não é o caminho feliz: o texto com aspas, a regra que
// sumiu em outra aba, o valor que saiu do catálogo e a escolha que ainda não foi feita.

func theOnlyRule(t *testing.T, f sceneFixture) (id, name, conditions string) {
	t.Helper()
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT id, name, conditions FROM campaign_rules WHERE campaignId = ?", f.campaignID).Scan(&id, &name, &conditions); err != nil {
		t.Fatalf("ler a regra: %v", err)
	}
	return id, name, conditions
}

func marginByWeapon(t *testing.T, f sceneFixture, id int64) map[string]int {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	out := map[string]int{}
	for _, card := range dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)) {
		out[card.Name] = card.CritRange
	}
	return out
}

// A margem presa a uma arma não alarga a outra mão, e a regra sem item alarga as duas.
func TestAMarginOnAWeaponStaysOnIt(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded2")
	before := marginByWeapon(t, f, id)

	rule := seedsRule(t, f, "Margem da espada", `[{"kind":"item","value":"espada-longa"}]`, `{"kind":"adds","target":{"k":"critRange"},"amount":2}`)
	after := marginByWeapon(t, f, id)
	if after["Espada longa"] != before["Espada longa"]-2 {
		t.Errorf("a espada foi de %d a %d, e a regra alarga em 2", before["Espada longa"], after["Espada longa"])
	}
	if after["Adaga"] != before["Adaga"] {
		t.Errorf("a adaga foi de %d a %d, e a regra nomeia a espada", before["Adaga"], after["Adaga"])
	}

	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_rules SET conditions = '[]' WHERE id = ?", rule); err != nil {
		t.Fatal(err)
	}
	everyone := marginByWeapon(t, f, id)
	if everyone["Espada longa"] != before["Espada longa"]-2 || everyone["Adaga"] != before["Adaga"]-2 {
		t.Errorf("sem item, a regra alarga as duas: de %v a %v", before, everyone)
	}
}

// O sinal chega como JSON: aspas e barras têm de voltar como foram escritas.
func TestANameWithQuotesIsSavedAsWritten(t *testing.T) {
	f := newSceneFixture(t)
	written := `Olho "mau" \ de Hyninn`
	encoded, err := json.Marshal(written)
	if err != nil {
		t.Fatal(err)
	}
	savesRule(t, f, strings.Replace(espadaDaCasa, `"Espada da casa"`, string(encoded), 1))
	if _, name, _ := theOnlyRule(t, f); name != written {
		t.Errorf("o nome gravado é %q, e foi escrito %q", name, written)
	}
}

// O rascunho só devolve o que o servidor mudou. O nome e o número ficam com quem os digita.
func TestTheDraftGivesBackOnlyWhatTheServerMoved(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"),
		strings.Replace(espadaDaCasa, `"table_rule_kind_1":"item","table_rule_value_1":"espada-longa"`, `"table_rule_kind_1":"race","table_rule_value_1":"espada-longa"`, 1)).Body.String()
	for _, typed := range []string{`"table_rule_name"`, `"table_rule_amount"`} {
		if strings.Contains(body, typed) {
			t.Errorf("o rascunho devolveu %s, e desfaria o que se digitou depois", typed)
		}
	}
	if !strings.Contains(body, `"table_rule_value_1"`) {
		t.Error("a espécie mudou e o valor era da outra: o rascunho devia devolver o valor novo")
	}
}

// A prévia não conta quem não tem a postura: "Alcança hoje" é sobre esta mesa.
func TestThePreviewAsksWhoHasTheStance(t *testing.T) {
	furia := strings.Replace(espadaDaCasa, `"table_rule_kind_1":"item","table_rule_value_1":"espada-longa"`, `"table_rule_kind_1":"stance","table_rule_value_1":"furia"`, 1)

	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), furia).Body.String(); strings.Contains(body, "Combatente") && strings.Contains(body, "Alcança hoje") {
		t.Error("o guerreiro não tem Fúria, e a prévia diz que a regra o alcança")
	}

	f, id = barbaro(t, 3)
	inCampaign(t, f, id)
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), furia).Body.String(); !strings.Contains(body, "Alcança hoje") {
		t.Error("o bárbaro tem Fúria, e a prévia diz que a regra não alcança ninguém")
	}
}

// Encanto não vira linha de item na ficha: oferecê-lo seria uma condição que nunca vale.
func TestTheItemOptionsAreThingsThatCanBeInUse(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho?mais=1"),
		`{"table_rule_name":"","table_rule_scope":"mesa","table_rule_count":0,"table_rule_target":"attack","table_rule_amount":1}`).Body.String()
	if strings.Contains(body, "Abascanto") {
		t.Error("a condição de item oferece um encanto")
	}
	if !strings.Contains(body, "Espada longa") {
		t.Error("a condição de item não oferece a espada")
	}
}

// "Um personagem" sem escolha pede a escolha, e não acusa um personagem 0.
func TestOneCharacterAsksWhichOne(t *testing.T) {
	f := newSceneFixture(t)
	signals := strings.Replace(espadaDaCasa, `"table_rule_scope":"mesa"`, `"table_rule_scope":"um","table_rule_character":""`, 1)
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), signals).Body.String(); !strings.Contains(body, "Escolha o personagem") {
		t.Error("o select não pede a escolha")
	}
	body := savesRule(t, f, signals)
	if !strings.Contains(body, "Escolha o personagem") || !strings.Contains(body, `role="alert"`) {
		t.Errorf("a recusa devia pedir a escolha, e veio %q", body[:min(300, len(body))])
	}
	if total, _ := ruleCount(t, f); total != 0 {
		t.Errorf("gravou %d regras sem personagem", total)
	}
}

// Abrir só para ler não troca a condição: o valor que saiu do catálogo fica, e gravar o recusa.
func TestOpeningARuleKeepsAValueGoneFromTheBook(t *testing.T) {
	f := newSceneFixture(t)
	rule := seedsRule(t, f, "Velha", `[{"kind":"item","value":"espada-que-saiu"}]`, plusTwoAttack)
	body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/editor?regra="+rule), "").Body.String()
	if !strings.Contains(body, "espada-que-saiu") {
		t.Fatalf("abrir trocou o valor: %q", body[:min(400, len(body))])
	}
	signals := fmt.Sprintf(`{"table_rule_id":%q,"table_rule_name":"Velha","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"item","table_rule_value_1":"espada-que-saiu","table_rule_target":"attack","table_rule_amount":2}`, rule)
	if body := savesRule(t, f, signals); !strings.Contains(body, `role="alert"`) {
		t.Error("gravar um valor que não existe devia ser recusado")
	}
	if _, _, conditions := theOnlyRule(t, f); !strings.Contains(conditions, "espada-que-saiu") {
		t.Errorf("as condições mudaram: %s", conditions)
	}
}

// A regra apagada em outra aba: a recusa é conteúdo, porque o Datastar descarta o remendo de um 404.
func TestARuleGoneFromAnotherTabIsSaid(t *testing.T) {
	f := newSceneFixture(t)
	gone := strings.Replace(espadaDaCasa, `"table_rule_amount":2`, `"table_rule_amount":2,"table_rule_id":"regra-que-sumiu"`, 1)
	if body := savesRule(t, f, gone); !strings.Contains(body, "não existe mais") || !strings.Contains(body, `role="alert"`) {
		t.Errorf("gravar devia dizer que a regra não existe mais, e veio %q", body[:min(300, len(body))])
	}
	if total, _ := ruleCount(t, f); total != 0 {
		t.Errorf("gravar a regra sumida criou %d", total)
	}
	opened := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/editor?regra=regra-que-sumiu"), "")
	if !strings.Contains(opened.Body.String(), "não existe mais") {
		t.Error("abrir a regra sumida devia dizer que ela não existe mais")
	}
	for _, path := range []string{"/regra-que-sumiu/ligar", "/regra-que-sumiu/excluir"} {
		if rec := f.requests(t, f.gm, http.MethodPost, rulesURL(f, path), ""); rec.Code != http.StatusOK {
			t.Errorf("%s respondeu %d, e o Datastar descartaria o redesenho", path, rec.Code)
		}
	}
}

// Nem o personagem nem a regra de outra mesa se alcançam pelo editor desta.
func TestTheEditorStaysInsideItsCampaign(t *testing.T) {
	f := newSceneFixture(t)
	savesRule(t, f, espadaDaCasa)
	rule, _, _ := theOnlyRule(t, f)
	other := seedCampaign(t, f.s, f.gm)
	gravar := fmt.Sprintf("/campanhas/%d/regras-da-mesa/gravar", other)

	theirs := strings.Replace(espadaDaCasa, `"table_rule_scope":"mesa"`, fmt.Sprintf(`"table_rule_scope":"um","table_rule_character":"%d"`, f.charID), 1)
	if body := f.requests(t, f.gm, http.MethodPost, gravar, theirs).Body.String(); !strings.Contains(body, `role="alert"`) {
		t.Error("o personagem de outra mesa devia ser recusado")
	}
	stolen := strings.Replace(espadaDaCasa, `"table_rule_amount":2`, fmt.Sprintf(`"table_rule_amount":9,"table_rule_id":%q`, rule), 1)
	f.requests(t, f.gm, http.MethodPost, gravar, stolen)
	if strings.Contains(rulesTab(t, f, f.gm), "+9 no ataque") {
		t.Error("o editor de outra mesa reescreveu esta regra")
	}
}

// Os sinais do editor são do mestre: a tela do jogador não tem leitor para eles.
func TestThePlayerPageCarriesNoEditorSignals(t *testing.T) {
	f := newSceneFixture(t)
	if strings.Contains(rulesTab(t, f, f.player), "table_rule_name") {
		t.Error("o jogador recebe os sinais do editor")
	}
}

// Mudar uma regra acorda a ficha aberta: o carimbo É a versão da ficha, e o aviso a faz ser relida.
func TestAChangedRuleWakesTheSheetsAtTheTable(t *testing.T) {
	f := newSceneFixture(t)
	id := f.charID
	updatedAt := func() string {
		var stamp string
		if err := f.s.db.QueryRowContext(context.Background(), "SELECT updatedAt FROM characters WHERE id = ?", id).Scan(&stamp); err != nil {
			t.Fatal(err)
		}
		return stamp
	}
	subscription, stop := f.s.bus.Subscribe(events.OfCharacter(id))
	defer stop()
	woke := func() bool {
		select {
		case <-subscription.C:
			return true
		default:
			return false
		}
	}
	forget := func(stamp string) {
		if _, err := f.s.db.ExecContext(context.Background(), "UPDATE characters SET updatedAt = ? WHERE id = ?", stamp, id); err != nil {
			t.Fatal(err)
		}
	}
	const old = "2020-01-01T00:00:00.000Z"
	forget(old)

	savesRule(t, f, espadaDaCasa)
	if woke() || updatedAt() != old {
		t.Error("a regra nova nasce desligada e não muda ficha nenhuma: ela não devia acordar ninguém")
	}
	rule, _, _ := theOnlyRule(t, f)

	gestures := []struct {
		name string
		does func()
	}{
		{"ligar", func() { f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/"+rule+"/ligar"), "") }},
		{"editar", func() {
			savesRule(t, f, strings.Replace(espadaDaCasa, `"table_rule_amount":2`, fmt.Sprintf(`"table_rule_amount":3,"table_rule_id":%q`, rule), 1))
		}},
		{"excluir", func() { f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/"+rule+"/excluir"), "") }},
	}
	for _, gesture := range gestures {
		forget(old)
		gesture.does()
		if !woke() {
			t.Errorf("%s não avisou a ficha aberta", gesture.name)
		}
		if updatedAt() == old {
			t.Errorf("%s não carimbou a ficha, e a mesa compara o carimbo para reler", gesture.name)
		}
	}
}

// O jogador lê as regras da mesa inteira e as do personagem dele. As de outra pessoa são da outra pessoa.
func TestThePlayerReadsTheTableRulesAndTheirOwn(t *testing.T) {
	f := newSceneFixture(t)
	insert := func(name string, character any) {
		if _, err := f.s.db.ExecContext(context.Background(),
			`INSERT INTO campaign_rules (id, campaignId, characterId, name, enabled, conditions, effect, updatedAt) VALUES (?, ?, ?, ?, 1, '[]', ?, '2026-11-05T00:00:00.000Z')`,
			"regra-"+name, f.campaignID, character, name, plusTwoAttack); err != nil {
			t.Fatal(err)
		}
	}
	someoneElse := seedCharacterAtLevel(t, f.s, f.gm, "Heroína do mestre", "Guerreiro", 3, 0, 0)
	if _, err := f.s.db.ExecContext(context.Background(), "INSERT INTO campaign_members (campaignId, characterId, addedAt) VALUES (?, ?, '2026-11-05T00:00:00.000Z')", f.campaignID, someoneElse); err != nil {
		t.Fatal(err)
	}
	insert("DaMesa", nil)
	insert("Minha", f.charID)
	insert("Alheia", someoneElse)

	page := rulesTab(t, f, f.player)
	for _, want := range []string{"DaMesa", "Minha"} {
		if !strings.Contains(page, want) {
			t.Errorf("o jogador não lê a regra %q", want)
		}
	}
	if strings.Contains(page, "Alheia") || strings.Contains(page, "Heroína do mestre") {
		t.Error("o jogador lê a regra do personagem de outra pessoa")
	}
	gmPage := rulesTab(t, f, f.gm)
	if !strings.Contains(gmPage, "Alheia") {
		t.Error("o mestre lê todas as regras")
	}
}

// Com condição de item que não nomeia um só, a regra soma por item, até o limite quando há limite.
func TestARuleCountsPerItemUpToItsLimit(t *testing.T) {
	cases := []struct {
		name  string
		limit string
		adds  int
	}{
		{"sem limite", "", 3},
		{"até 2", `,"limit":2`, 2},
		{"até 1", `,"limit":1`, 1},
		{"o limite acima do que se veste", `,"limit":9`, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			inCampaign(t, f, id)
			for _, item := range []string{"algemas", "andrajos-aldeao", "anel-da-liberdade"} {
				itemSemeia(t, f, id, item, item, "vested")
			}
			beforeDefense, beforeHP := ruleSheet(t, f, id).Defense.Total, poolsOf(t, f.s, id).HpMax
			seedsRule(t, f, "Defesa", `[{"kind":"vested"}]`, fmt.Sprintf(`{"kind":"adds","target":{"k":"defense"},"amount":1%s}`, c.limit))
			seedsRule(t, f, "Vigor", `[{"kind":"vested"}]`, fmt.Sprintf(`{"kind":"adds","target":{"k":"maxPv"},"amount":1%s}`, c.limit))
			if got := ruleSheet(t, f, id).Defense.Total; got != beforeDefense+c.adds {
				t.Errorf("a Defesa foi de %d a %d, e devia somar %d", beforeDefense, got, c.adds)
			}
			if got := poolsOf(t, f.s, id).HpMax; int(got) != int(beforeHP)+c.adds {
				t.Errorf("o PV foi de %d a %d, e devia somar %d", beforeHP, got, c.adds)
			}
		})
	}
}

// O campo de limite só existe onde a regra soma por item, e a prévia diz o que ele faz.
func TestThePreviewSaysHowTheRuleCounts(t *testing.T) {
	vested := `"table_rule_kind_1":"vested","table_rule_value_1":""`
	named := `"table_rule_kind_1":"item","table_rule_value_1":"espada-longa"`
	signals := func(condition, target, limit string) string {
		return fmt.Sprintf(`{"table_rule_name":"R","table_rule_scope":"mesa","table_rule_count":1,%s,"table_rule_target":%q,"table_rule_amount":1,"table_rule_limit":%q}`, condition, target, limit)
	}
	cases := []struct {
		name, signals, says string
		field               bool
	}{
		{"sem limite", signals(vested, "defense", ""), "vestido: +1 na Defesa por item", true},
		{"com limite", signals(vested, "defense", "3"), "vestido: +1 na Defesa por item, até +3", true},
		{"até um", signals(vested, "defense", "1"), "vestido: +1 na Defesa<", true},
		{"o item nomeado é um só", signals(named, "defense", ""), "com Espada longa: +1 na Defesa<", false},
		{"o golpe é de cada arma", signals(vested, "attack", ""), "vestido: +1 no ataque<", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), c.signals).Body.String()
			if !strings.Contains(body, c.says) {
				t.Errorf("a prévia devia dizer %q", c.says)
			}
			if got := strings.Contains(body, `aria-label="Até quantos itens"`); got != c.field {
				t.Errorf("o campo de limite aparece = %v, e devia ser %v", got, c.field)
			}
		})
	}
}

// O limite grava, volta ao abrir, sai quando a regra deixa de somar por item, e não aceita negativo.
func TestTheLimitIsSavedWithTheRule(t *testing.T) {
	vested := `{"table_rule_name":"Peças","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"vested","table_rule_value_1":"","table_rule_target":"defense","table_rule_amount":1,"table_rule_limit":"3"}`
	f := newSceneFixture(t)
	savesRule(t, f, vested)
	rule, _, _ := theOnlyRule(t, f)
	if page := rulesTab(t, f, f.gm); !strings.Contains(page, "por item, até +3") {
		t.Error("a lista não diz o limite")
	}
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/editor?regra="+rule), "").Body.String(); !strings.Contains(body, `"table_rule_limit":"3"`) {
		t.Error("abrir a regra não traz o limite")
	}
	byTheSword := strings.Replace(vested, `"table_rule_kind_1":"vested","table_rule_value_1":""`, `"table_rule_kind_1":"item","table_rule_value_1":"espada-longa"`, 1)
	savesRule(t, f, strings.Replace(byTheSword, `"table_rule_name"`, fmt.Sprintf(`"table_rule_id":%q,"table_rule_name"`, rule), 1))
	var effect string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT effect FROM campaign_rules WHERE id = ?", rule).Scan(&effect); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(effect, "limit") {
		t.Errorf("o item nomeado é um só, e o limite ficou guardado: %s", effect)
	}

	f = newSceneFixture(t)
	body := savesRule(t, f, strings.Replace(vested, `"table_rule_limit":"3"`, `"table_rule_limit":"-2"`, 1))
	if !strings.Contains(body, "-2") || !strings.Contains(body, `role="alert"`) {
		t.Errorf("o limite negativo devia ser recusado com o valor, e veio %q", body[:min(300, len(body))])
	}
}

// O limite é do que soma na ficha. O número preso a uma arma é de cada arma, e o limite não o corta.
func TestAWeaponRuleIgnoresTheLimit(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	itemSemeia(t, f, id, "florete", "Florete", "wielded")
	itemSemeia(t, f, id, "cimitarra", "Cimitarra", "wielded2")
	before := attackByWeapon(t, f, id)
	seedsRule(t, f, "Ágeis", `[{"kind":"trait","value":"agil"}]`, `{"kind":"adds","target":{"k":"attack"},"amount":1,"limit":1}`)
	after := attackByWeapon(t, f, id)
	for _, name := range []string{"Florete", "Cimitarra"} {
		if after[name] != before[name]+1 {
			t.Errorf("%s foi de %+d a %+d, e cada arma ágil ganha +1", name, before[name], after[name])
		}
	}
}
