package api

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// A FORJA EM NÍVEL ALTO. O estado continua sendo o formulário: `classes` leva uma linha por classe
// na ordem em que entraram, e a primeira é a que dá os PV iniciais e as perícias (p35).

// forgedHero forja com as respostas dadas e devolve o id do herói.
func forgedHero(t *testing.T, f sceneFixture, change func(url.Values)) int64 {
	t.Helper()
	answers := aFolhaPreenchida()
	answers.Set("race", "humano")
	change(answers)
	rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("a forja recusou (%d):\n%s", rec.Code, rec.Body.String())
	}
	found := regexp.MustCompile(`/personagens/(\d+)/atributos`).FindStringSubmatch(rec.Header().Get("Location"))
	if found == nil {
		t.Fatalf("a forja não levou aos atributos: %q", rec.Header().Get("Location"))
	}
	id, _ := strconv.ParseInt(found[1], 10, 64)
	return id
}

func forgedSheet(t *testing.T, f sceneFixture, id int64) sheet.CharacterDTO {
	t.Helper()
	ctx := context.Background()
	row, err := f.s.queries.GetCharacter(ctx, id)
	if err != nil {
		t.Fatalf("ler a ficha %d: %v", id, err)
	}
	dto, err := sheet.Load(ctx, f.s.queries, f.s.catalogs, row)
	if err != nil {
		t.Fatalf("carregar a ficha %d: %v", id, err)
	}
	return dto
}

func threeAndOne(answers url.Values) {
	answers.Del("class")
	answers["classes"] = []string{"Guerreiro:3", "Arcanista:1"}
}

// O herói forjado em 3/1 é o mesmo de quem subiu pelo `+` da ficha: é a promessa de "a mesma conta".
func TestAHeroForgedAtHighLevelEqualsOneWhoClimbed(t *testing.T) {
	f := newSceneFixture(t)
	forged := forgedHero(t, f, threeAndOne)

	climbed := forgedHero(t, f, func(url.Values) {})
	for range 2 {
		if refusal := stepClass(t, f, climbed, "Guerreiro", 1); refusal != "" {
			t.Fatalf("subir o guerreiro: %q", refusal)
		}
	}
	if refusal := enterClass(t, f, climbed, "Arcanista"); refusal != "" {
		t.Fatalf("entrar no arcanista: %q", refusal)
	}

	got, want := forgedSheet(t, f, forged), forgedSheet(t, f, climbed)
	if got.Level != 4 {
		t.Errorf("o nível do herói é %d, e 3 + 1 são 4", got.Level)
	}
	// Humano sem Constituição: 20 + 5 + 5 de guerreiro, +2 de arcanista; 3 x 3 + 6 de PM (p35).
	if got.HpMax != 32 || got.MpMax != 15 {
		t.Errorf("o 3/1 tem PV %d e PM %d, e a conta dá 32 e 15", got.HpMax, got.MpMax)
	}
	if got.HpMax != want.HpMax || got.MpMax != want.MpMax {
		t.Errorf("forjado PV %d PM %d, subido PV %d PM %d", got.HpMax, got.MpMax, want.HpMax, want.MpMax)
	}
	if got.Proficiencies != want.Proficiencies {
		t.Errorf("proficiências: forjado %v, subido %v", got.Proficiencies, want.Proficiencies)
	}
	if len(got.LevelChoices) != len(want.LevelChoices) {
		t.Errorf("escolhas de nível: forjado %d, subido %d", len(got.LevelChoices), len(want.LevelChoices))
	}
	if screen := powerScreen(t, f, forged); !strings.Contains(screen, "Por nível") {
		t.Error("as escolhas de cada nível não ficaram na ficha")
	}
}

// Acima do 1º nível, a bolsa é a da Tabela 3-1 e não os 4d6 (p140). O Acólito não concede dinheiro.
func TestAHeroAboveLevelOneStartsWithTheMoneyOfTheTable(t *testing.T) {
	f := newSceneFixture(t)
	if tibar := forgedSheet(t, f, forgedHero(t, f, threeAndOne)).Tibar; tibar != 1000 {
		t.Errorf("o herói de nível 4 nasceu com T$ %v, e a tabela dá T$ 1.000", tibar)
	}
	screen := draftStep(t, f, "Guerreiro", 1, url.Values{"classes": {"Guerreiro:3"}})
	if !strings.Contains(screen, "T$ 1.000") {
		t.Errorf("a folha não diz a bolsa do nível 4:\n%s", screen)
	}
	first := postaAForja(t, f, f.player, "/personagens/nova/esboco", url.Values{"classes": {"Guerreiro:1"}}).Body.String()
	if !strings.Contains(first, "4d6") {
		t.Error("no 1º nível a folha deixou de dizer os 4d6")
	}
}

// A outra classe não dá perícias nem proficiências (p35): quem as dá é a primeira.
func TestOnlyTheFirstClassGivesExpertisesAndProficiencies(t *testing.T) {
	f := newSceneFixture(t)
	first := forgedSheet(t, f, forgedHero(t, f, threeAndOne))
	for _, expertise := range first.Expertises {
		if expertise.Name == "Misticismo" && expertise.Trained {
			t.Error("o Misticismo do arcanista veio junto, e ele é a segunda classe")
		}
	}
	turned := forgedSheet(t, f, forgedHero(t, f, func(answers url.Values) {
		answers.Del("class")
		answers["classes"] = []string{"Arcanista:1", "Guerreiro:3"}
		answers.Del("weaponMartial")
		answers.Del("armor")
		answers.Del("shield")
	}))
	// Arcanista primeiro: 8 + 5 x 3 de PV, e a mesma ficha com a ordem trocada é outra (p35).
	if turned.HpMax != 23 {
		t.Errorf("o 1/3 com o arcanista primeiro tem PV %d, e a conta dá 23", turned.HpMax)
	}
	if strings.Contains(turned.Proficiencies, "armas-marciais") {
		t.Errorf("as proficiências do guerreiro vieram junto: %v", turned.Proficiencies)
	}
}

func draftStep(t *testing.T, f sceneFixture, class string, step int, answers url.Values) string {
	t.Helper()
	path := "/personagens/nova/esboco/nivel/" + url.PathEscape(class) + "/" + strconv.Itoa(step)
	rec := postaAForja(t, f, f.player, path, answers)
	if rec.Code != http.StatusOK {
		t.Fatalf("o degrau de %s respondeu %d: %s", class, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

// O `+` e o `−` só redesenham, e a soma aparece no alto da lista.
func TestTheStepperMovesTheLevelAndShowsTheSum(t *testing.T) {
	f := newSceneFixture(t)
	body := draftStep(t, f, "Guerreiro", 1, url.Values{"classes": {"Guerreiro:3", "Arcanista:1"}})
	if !strings.Contains(body, "Nível 5") {
		t.Errorf("a soma não diz o nível 5:\n%s", body)
	}
	if !strings.Contains(body, `value="Guerreiro:4"`) || !strings.Contains(body, `value="Arcanista:1"`) {
		t.Errorf("o formulário não guardou os níveis:\n%s", body)
	}
	if !strings.Contains(body, "1ª classe") {
		t.Error("a primeira classe não diz que é a primeira")
	}
	if !strings.Contains(body, "+2 PV e +6 PM por nível") {
		t.Error("a outra classe não diz o que soma por nível")
	}
}

// O `−` no nível 1 tira a classe, e a seguinte passa a ser a primeira. É assim que se troca a 1ª.
func TestLoweringFromLevelOneTakesTheClassOut(t *testing.T) {
	f := newSceneFixture(t)
	body := draftStep(t, f, "Guerreiro", -1, url.Values{"classes": {"Guerreiro:1", "Arcanista:2"}})
	if strings.Contains(body, `value="Guerreiro:`) {
		t.Errorf("o guerreiro continuou na folha:\n%s", body)
	}
	if !strings.Contains(body, `value="Arcanista:2"`) || !strings.Contains(body, "Nível 2") {
		t.Errorf("o arcanista não ficou sozinho no nível 2:\n%s", body)
	}
	if !strings.Contains(body, "Arcanistas começam sem armadura") {
		t.Error("o equipamento não passou a seguir a nova primeira classe")
	}
	empty := draftStep(t, f, "Arcanista", -1, url.Values{"classes": {"Arcanista:1"}})
	if strings.Contains(empty, `name="classes"`) || strings.Contains(empty, `name="weaponSimple"`) {
		t.Errorf("sem classe, a folha ainda tem classe ou equipamento:\n%s", empty)
	}
}

// A folha só oferece a classe que o herói ainda não tem, e o rádio novo entra no nível 1.
func TestAnotherClassEntersAtLevelOneAndLeavesTheList(t *testing.T) {
	f := newSceneFixture(t)
	body := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"classes": {"Guerreiro:3"}, "class": {"Arcanista"}}).Body.String()
	if !strings.Contains(body, `value="Guerreiro:3"`) || !strings.Contains(body, `value="Arcanista:1"`) {
		t.Errorf("as classes não ficaram em ordem no formulário:\n%s", body)
	}
	for _, class := range []string{"Guerreiro", "Arcanista"} {
		if strings.Contains(body, `name="class" form="forge-form" value="`+class+`"`) {
			t.Errorf("a folha ainda oferece %s, que o herói já tem", class)
		}
	}
	if !strings.Contains(body, `value="Bardo"`) {
		t.Error("a folha deixou de oferecer as outras classes")
	}
}

// O 20º nível é o último (p35), e a recusa diz o número.
func TestTheForgeRefusesPastLevelTwenty(t *testing.T) {
	f := newSceneFixture(t)
	answers := aFolhaPreenchida()
	threeAndOne(answers)
	answers["classes"] = []string{"Guerreiro:15", "Arcanista:6"}
	rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("a forja aceitou 21 níveis: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "21") {
		t.Errorf("a recusa não cita o 21:\n%s", rec.Body.String())
	}
	atTheEnd := draftStep(t, f, "Guerreiro", 1, url.Values{"classes": {"Guerreiro:15", "Arcanista:5"}})
	if !strings.Contains(atTheEnd, `value="Guerreiro:15"`) {
		t.Error("o `+` passou do 20º nível")
	}
	if strings.Contains(atTheEnd, `id="forge-pick-class"`) {
		t.Error("o nível 20 ainda oferece outra classe")
	}
}

// Linha feita à mão não vira ficha: classe repetida, classe que não existe e nível fora da faixa.
func TestTheForgeRefusesAClassLineThatMakesNoSense(t *testing.T) {
	f := newSceneFixture(t)
	for line, wants := range map[string]string{
		"Guerreiro:0": "Guerreiro:0",
		"Guerreiro:x": "Guerreiro:x",
		"Guerreiro":   "Guerreiro",
		"Xamã:2":      "Xamã",
	} {
		answers := aFolhaPreenchida()
		answers.Del("class")
		answers["classes"] = []string{line}
		rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
		if rec.Code == http.StatusSeeOther {
			t.Errorf("a forja aceitou a linha %q", line)
			continue
		}
		if !strings.Contains(rec.Body.String(), wants) {
			t.Errorf("a recusa da linha %q não a cita:\n%s", line, rec.Body.String())
		}
	}
	answers := aFolhaPreenchida()
	answers.Del("class")
	answers["classes"] = []string{"Guerreiro:1", "Guerreiro:2"}
	if rec := postaAForja(t, f, f.player, "/personagens/nova", answers); rec.Code == http.StatusSeeOther {
		t.Error("a forja aceitou a mesma classe duas vezes")
	}
}
