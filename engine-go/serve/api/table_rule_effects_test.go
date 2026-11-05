package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// OS DOIS EFEITOS QUE NÃO SOMAM: "vale como empunhado" e "escolhe poderes de".

const wieldedMedallion = `{"kind":"wielded","target":{"k":""},"amount":0}`

// pmLimitOf é o limite de PM com os interruptores que a ficha tem ligados.
func pmLimitOf(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	return dto.Ruleset.ComputeSheet(ec, sheet.ToStringSet(dto.Conditionals)).PmLimit.Total
}

// O medalhão vestido, com a regra, guarda o bônus de empunhar. Sem a regra, vestido não dá nada.
func TestAWieldedRuleKeepsTheBonusOfTheVestedItem(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	staff := itemSemeia(t, f, id, "cajado-arcano", "Cajado arcano", "")
	before := pmLimitOf(t, f, id)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET equipped = 'vested' WHERE id = ?", staff); err != nil {
		t.Fatal(err)
	}
	if got := pmLimitOf(t, f, id); got != before {
		t.Fatalf("o controle: vestido sem a regra, o cajado não dá o bônus de empunhar, e foi de %d a %d", before, got)
	}
	seedsRule(t, f, "Cajado na cintura", `[{"kind":"item","value":"cajado-arcano"}]`, wieldedMedallion)
	if got := pmLimitOf(t, f, id); got != before+1 {
		t.Errorf("com a regra, o cajado vestido dá +1 no limite de PM, e foi de %d a %d", before, got)
	}
}

// A regra deixa vestir o item que o livro só deixa empunhar, e a folha do item oferece o Vestir.
func TestAWieldedRuleLetsTheItemBeVested(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	staff := itemSemeia(t, f, id, "cajado-arcano", "Cajado arcano", "")
	medal := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "")
	vest := fmt.Sprintf("/personagens/%d/itens/%d/equipa/vested?tab=bag", id, staff)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, vest, "").Body.String()); refusal == "" {
		t.Fatal("o controle: sem a regra, o cajado não se veste")
	}
	seedsRule(t, f, "Cajado na cintura", `[{"kind":"item","value":"cajado-arcano"}]`, wieldedMedallion)
	if !strings.Contains(itemSheetOfBag(t, f, id, staff), "Vestir") {
		t.Error("com a regra, a folha do cajado não oferece o Vestir")
	}
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, vest, "").Body.String()); refusal != "" {
		t.Errorf("com a regra, vestir o cajado foi recusado: %q", refusal)
	}
	if strings.Contains(itemSheetOfBag(t, f, id, medal), "Vestir") {
		t.Error("a regra nomeia o cajado, e a folha do medalhão oferece o Vestir")
	}
}

// Com a regra e o interruptor antigo do medalhão ligados juntos, o bônus vale uma vez: com a regra, o
// item entra como empunhado, e o interruptor não tem o que acrescentar.
func TestTheWieldedRuleAndTheOldToggleCountOnce(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	before := pmLimitOf(t, f, id)
	if _, err := f.s.db.ExecContext(context.Background(),
		"INSERT INTO character_conditionals (characterId, conditionalId) VALUES (?, 'flag:homebrew-vestido-medalhao-de-prata')", id); err != nil {
		t.Fatal(err)
	}
	withToggle := pmLimitOf(t, f, id)
	if withToggle != before+1 {
		t.Fatalf("o controle: o interruptor antigo dá +1, e foi de %d a %d", before, withToggle)
	}
	seedsRule(t, f, "Medalhão na cintura", `[{"kind":"item","value":"medalhao-de-prata"}]`, wieldedMedallion)
	if got := pmLimitOf(t, f, id); got != before+1 {
		t.Errorf("a regra e o interruptor somaram juntos: de %d a %d", before, got)
	}
}

const borrowsArsenal = `{"kind":"powers-of","target":{"k":""},"amount":0,"god":"arsenal"}`

// O devoto de Tauron escolhe os poderes de Arsenal, no limite do livro: um.
func TestTauronsDevoteeChoosesArsenalsPowers(t *testing.T) {
	f := newSceneFixture(t)
	id := devotedTo(t, f, "Guerreiro", "tauron")
	inCampaign(t, f, id)
	if refusal := powerCommand(t, f, id, grantedPath("Sangue de Ferro"), ""); refusal == "" {
		t.Fatal("o controle: sem a regra, o devoto de Tauron não escolhe Sangue de Ferro")
	}
	seedsRule(t, f, "Tauron copia Arsenal", `[{"kind":"devotion","value":"tauron"}]`, borrowsArsenal)
	if screen := powerScreen(t, f, id); !strings.Contains(screen, "Sangue de Ferro") || !strings.Contains(screen, "Conjurar Arma") {
		t.Error("com a regra, a ficha não oferece os poderes de Arsenal")
	}
	if refusal := powerCommand(t, f, id, grantedPath("Sangue de Ferro"), ""); refusal != "" {
		t.Fatalf("com a regra, escolher Sangue de Ferro foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Coragem Total"), ""); refusal != "" {
		t.Fatalf("numa vaga só, o segundo troca o primeiro, e foi recusado: %q", refusal)
	}
	if got := storedGrantedPowers(t, f, id); !slices.Equal(got, []string{"Coragem Total"}) {
		t.Errorf("o limite é um, e a ficha guardou %v", got)
	}
}

// A lista emprestada SOMA à do deus: o devoto de Khalmyr escolhe dos dois.
func TestABorrowedListAddsToTheGodsOwn(t *testing.T) {
	f := newSceneFixture(t)
	id := devotedTo(t, f, "Guerreiro", "khalmyr")
	inCampaign(t, f, id)
	seedsRule(t, f, "Khalmyr e Arsenal", `[{"kind":"devotion","value":"khalmyr"}]`, borrowsArsenal)
	screen := powerScreen(t, f, id)
	if !strings.Contains(screen, "Sangue de Ferro") {
		t.Error("a lista de Arsenal não entrou")
	}
	if !strings.Contains(screen, "Dom da Verdade") {
		t.Error("a lista de Khalmyr saiu, e a emprestada devia somar")
	}
	if refusal := powerCommand(t, f, id, grantedPath("Dom da Verdade"), ""); refusal != "" {
		t.Errorf("o poder do próprio deus foi recusado: %q", refusal)
	}
}

// Trocar para Tauron, com a regra, conta as vagas e mantém o poder de Arsenal que ele também dá.
func TestChangingToTauronKeepsWhatTheRuleLends(t *testing.T) {
	f := newSceneFixture(t)
	id := devotedTo(t, f, "Guerreiro", "arsenal")
	inCampaign(t, f, id)
	seedsRule(t, f, "Tauron copia Arsenal", `[{"kind":"devotion","value":"tauron"}]`, borrowsArsenal)
	if refusal := powerCommand(t, f, id, grantedPath("Sangue de Ferro"), ""); refusal != "" {
		t.Fatal(refusal)
	}
	if screen := powerScreen(t, f, id); !strings.Contains(screen, "Sangue de Ferro continua valendo") {
		t.Error("o aviso da troca para Tauron não diz que Sangue de Ferro continua")
	}
	if refusal := powerCommand(t, f, id, devotionPath("tauron"), ""); refusal != "" {
		t.Fatal(refusal)
	}
	if got := storedGrantedPowers(t, f, id); !slices.Equal(got, []string{"Sangue de Ferro"}) {
		t.Errorf("Tauron, pela regra, também concede Sangue de Ferro, e a ficha guardou %v", got)
	}
}

func TestTheEditorWritesTheNewEffects(t *testing.T) {
	f := newSceneFixture(t)
	wielded := `{"table_rule_name":"Cajado","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"item","table_rule_value_1":"cajado-arcano","table_rule_effect":"wielded"}`
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), wielded).Body.String(); !strings.Contains(body, "vestido, vale como empunhado") {
		t.Error("a prévia não escreve o vale como empunhado")
	}
	borrows := `{"table_rule_name":"Tauron","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"devotion","table_rule_value_1":"tauron","table_rule_effect":"powers-of","table_rule_god":"arsenal"}`
	if body := f.requests(t, f.gm, http.MethodPost, rulesURL(f, "/rascunho"), borrows).Body.String(); !strings.Contains(body, "escolhe poderes de Arsenal") || !strings.Contains(body, "Sangue de Ferro") {
		t.Error("a prévia não escreve de quem são os poderes e quais")
	}
	savesRule(t, f, borrows)
	if total, _ := ruleCount(t, f); total != 1 {
		t.Fatalf("a regra de poderes não gravou: %d", total)
	}
	var effect string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT effect FROM campaign_rules").Scan(&effect); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(effect, `"god":"arsenal"`) || !strings.Contains(effect, `"kind":"powers-of"`) {
		t.Errorf("o efeito gravado é %s", effect)
	}
}

func TestTheEditorRefusesTheNewEffectsWhenWrong(t *testing.T) {
	cases := []struct{ name, signals, says string }{
		{"empunhado sem item", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"race","table_rule_value_1":"humano","table_rule_effect":"wielded"}`, "item"},
		{"deus sem lista", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":0,"table_rule_effect":"powers-of","table_rule_god":"tauron"}`, "Tauron"},
		{"efeito que não existe", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":0,"table_rule_effect":"voar"}`, "voar"},
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

// itemSheetOfBag é a folha daquele item, na página da mochila.
func itemSheetOfBag(t *testing.T, f sceneFixture, id, item int64) string {
	t.Helper()
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=bag", id), "").Body.String()
	start := strings.Index(page, fmt.Sprintf(`id="item-%d"`, item))
	if start < 0 {
		t.Fatalf("a página não traz a folha do item %d", item)
	}
	end := strings.Index(page[start:], "</dialog>")
	return page[start : start+end]
}

// O silêncio do mestre segue valendo no item vestido pela regra: o endereço do bônus é o mesmo.
func TestTheWieldedRuleKeepsTheGMsSilence(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	itemSemeia(t, f, id, "cajado-arcano", "Cajado arcano", "vested")
	before := pmLimitOf(t, f, id)
	staff := book.ItemByID("cajado-arcano")
	at := slices.IndexFunc(staff.Modifiers, func(m engine.Modifier) bool { return m.Target.K == "pmLimit" })
	seedCampaignSilence(t, f.s, f.campaignID, sql.NullInt64{}, engine.TermID("cajado-arcano", staff.Modifiers[at]))
	seedsRule(t, f, "Cajado na cintura", `[{"kind":"item","value":"cajado-arcano"}]`, wieldedMedallion)
	if got := pmLimitOf(t, f, id); got != before {
		t.Errorf("o mestre calou o bônus do cajado, e vestido pela regra ele voltou: de %d a %d", before, got)
	}
}

// A regra só aceita as condições que o efeito dela cumpre.
func TestTheEditorRefusesConditionsTheEffectIgnores(t *testing.T) {
	cases := []struct{ name, signals, says string }{
		{"poderes com item", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"item","table_rule_value_1":"medalhao-de-prata","table_rule_effect":"powers-of","table_rule_god":"arsenal"}`, "Escolhe poderes de"},
		{"poderes vestido", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"vested","table_rule_value_1":"","table_rule_effect":"powers-of","table_rule_god":"arsenal"}`, "Escolhe poderes de"},
		{"poderes com postura", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":1,"table_rule_kind_1":"stance","table_rule_value_1":"furia","table_rule_effect":"powers-of","table_rule_god":"arsenal"}`, "Escolhe poderes de"},
		{"empunhado com postura", `{"table_rule_name":"A","table_rule_scope":"mesa","table_rule_count":2,"table_rule_kind_1":"item","table_rule_value_1":"cajado-arcano","table_rule_kind_2":"stance","table_rule_value_2":"furia","table_rule_effect":"wielded"}`, "Vale como empunhado"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			body := savesRule(t, f, c.signals)
			if !strings.Contains(body, c.says) || !strings.Contains(body, `role="alert"`) {
				t.Errorf("a recusa devia nomear %q num alerta, e veio %q", c.says, body[:min(300, len(body))])
			}
			if total, _ := ruleCount(t, f); total != 0 {
				t.Errorf("a recusa gravou %d regras", total)
			}
		})
	}
}
