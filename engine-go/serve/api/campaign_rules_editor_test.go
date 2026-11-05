package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/sheet"
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
