package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O ESPECIALISTA (ladino, p73; ALE-501): "Escolha um número de perícias
// treinadas igual a sua Inteligência, exceto bônus temporários (mínimo 1). Ao
// fazer um teste de uma dessas perícias, você pode gastar 1 PM para dobrar seu
// bônus de treinamento."
//
// A escolha não gasta vaga, e a ficha que sairia dela é RECUSADA em todo gesto
// que mexe no N ou no treino — decisão do dono.

const specialistPath = "escolha/class.ladino.especialista/"

// specialist é uma ladina de nível 7 (treino +4) com Inteligência `intelligence`
// e três perícias treinadas, e uma sem treino.
func specialist(t *testing.T, intelligence int64) (sceneFixture, int64) {
	t.Helper()
	f := umaMesaAoVivo(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Ladina", Origin: "soldado", Level: 7, Intelligence: intelligence,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a ladina: %v", err)
	}
	seedClasse(t, f.s, id, "Ladino", 7)
	seedMember(t, f.s, f.campaignID, id)
	seedPericia(t, f.s, id, "Ladinagem", "dexterity", true)
	seedPericia(t, f.s, id, "Furtividade", "dexterity", true)
	seedPericia(t, f.s, id, "Acrobacia", "dexterity", true)
	seedPericia(t, f.s, id, "Diplomacia", "charisma", false)
	f.charID = id
	return f, id
}

func picksSpecialist(t *testing.T, f sceneFixture, id int64, skills ...string) {
	t.Helper()
	for _, skill := range skills {
		if refusal := powerCommand(t, f, id, specialistPath+url.PathEscape(skill), ""); refusal != "" {
			t.Fatalf("escolher %s no Especialista foi recusado: %q", skill, refusal)
		}
	}
}

// sheetPost faz um gesto da ficha e devolve a recusa, ou "".
func sheetPost(t *testing.T, f sceneFixture, id int64, path, body string) string {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d%s", id, path), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s deu %d: %s", path, rec.Code, rec.Body.String())
	}
	return sceneRefusal(rec.Body.String())
}

func storedPowerChoices(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	_, choices := storedChoices(t, f.s, id)
	return choices
}

// Int 2: duas perícias, só treinadas, sem vaga de poder.
func TestTheSpecialistPicksTrainedSkillsUpToIntelligence(t *testing.T) {
	f, id := specialist(t, 2)

	if refusal := powerCommand(t, f, id, specialistPath+"Diplomacia", ""); !strings.Contains(refusal, "treinada") {
		t.Errorf("Diplomacia não é treinada e o Especialista a aceitou: %q", refusal)
	}
	picksSpecialist(t, f, id, "Ladinagem", "Furtividade")
	if refusal := powerCommand(t, f, id, specialistPath+"Acrobacia", ""); !strings.Contains(refusal, "Inteligência") {
		t.Errorf("a terceira perícia com Int 2 foi aceita, ou a recusa não diz o porquê: %q", refusal)
	}
	if got := storedPowerChoices(t, f, id); got != `{"class.ladino.especialista":["Ladinagem","Furtividade"]}` {
		t.Errorf("powerChoices gravado = %s", got)
	}
	// A ESCOLHA NÃO OCUPA VAGA: o ladino 7 tem as dele inteiras para poder.
	if refusal := powerCommand(t, f, id, "escolhe/class.ladino.assassinar", ""); refusal != "" {
		t.Errorf("com o Especialista escolhido, um poder de ladino foi recusado: %q", refusal)
	}
}

// "mínimo 1": Inteligência 0 ainda escolhe uma.
func TestTheSpecialistPicksAtLeastOne(t *testing.T) {
	f, id := specialist(t, 0)

	picksSpecialist(t, f, id, "Ladinagem")
	if refusal := powerCommand(t, f, id, specialistPath+"Furtividade", ""); refusal == "" {
		t.Error("Inteligência 0 aceitou a segunda perícia")
	}
}

// 1 PM dobra o treino: +4 no nível 7, só na escolhida.
func TestTheSpecialistDoublesTheTrainingForOnePm(t *testing.T) {
	f, id := specialist(t, 1)
	picksSpecialist(t, f, id, "Ladinagem")

	if refusal := rollsWith(t, f, id, "Ladinagem", `{"expertise_d20":10}`); refusal != "" {
		t.Fatalf("Ladinagem sem gasto: %q", refusal)
	}
	plain := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	before := pm(t, f, id)
	if refusal := rollsWith(t, f, id, "Ladinagem", `{"expertise_d20":10,"expertise_spends":{"especialista":1}}`); refusal != "" {
		t.Fatalf("Ladinagem com Especialista: %q", refusal)
	}
	paid := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if paid.Total-plain.Total != 4 {
		t.Errorf("com o Especialista o total foi %d, sem ele %d — want +4 (treino do nível 7)", paid.Total, plain.Total)
	}
	if spent := before - pm(t, f, id); spent != 1 {
		t.Errorf("o Especialista cobrou %d PM, e a p73 diz 1", spent)
	}
	if refusal := rollsWith(t, f, id, "Furtividade", `{"expertise_d20":10,"expertise_spends":{"especialista":1}}`); refusal == "" {
		t.Error("Furtividade não foi escolhida, e o Especialista valeu nela")
	}
}

// A FICHA FORA DA ESCOLHA NÃO SALVA: cada gesto que baixaria o N ou tiraria o
// treino de uma escolhida é recusado, e nada muda.
func TestTheSheetDoesNotLeaveTheSpecialistChoice(t *testing.T) {
	f, id := specialist(t, 1)
	earrings := seedConsumable(t, f.s, id, "brincos-da-sagacidade", "Brincos da sagacidade", 1)
	if _, err := f.s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: "Ofício (alquimista)", Attribute: "intelligence", Trained: 1, Custom: 1,
	}); err != nil {
		t.Fatalf("semear o ofício: %v", err)
	}
	if refusal := sheetPost(t, f, id, fmt.Sprintf("/itens/%d/equipa/vested", earrings), ""); refusal != "" {
		t.Fatalf("vestir os brincos: %q", refusal)
	}
	// Int 1 + brincos = 2: Ladinagem e o ofício.
	picksSpecialist(t, f, id, "Ladinagem", "Ofício (alquimista)")
	before := storedPowerChoices(t, f, id)

	for _, gesture := range []struct{ name, path string }{
		{"destreinar uma escolhida", "/pericias/treino/Ladinagem"},
		{"apagar o ofício escolhido", "/pericias/remover/Ofício%20(alquimista)"},
		{"guardar os brincos", fmt.Sprintf("/itens/%d/guarda", earrings)},
		{"remover os brincos", fmt.Sprintf("/itens/%d/remover", earrings)},
	} {
		if refusal := sheetPost(t, f, id, gesture.path, ""); !strings.Contains(refusal, "Especialista") {
			t.Errorf("%s: a ficha saiu da escolha do Especialista sem recusa (%q)", gesture.name, refusal)
		}
	}
	if forged := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/atributos/intelligence/-1", id), ""); !strings.Contains(forged.Body.String(), "Especialista") {
		t.Error("baixar a Inteligência na forja não foi recusado pelo Especialista")
	}

	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if row.Intelligence != 1 || storedPowerChoices(t, f, id) != before {
		t.Errorf("a ficha mudou com as recusas: Int %d, escolhas %s", row.Intelligence, storedPowerChoices(t, f, id))
	}
	// O CONTROLE: o gesto que não toca a escolha passa.
	if refusal := sheetPost(t, f, id, "/pericias/treino/Acrobacia", ""); refusal != "" {
		t.Errorf("destreinar Acrobacia, que não foi escolhida, foi recusado: %q", refusal)
	}
}

// A DISTRIBUIÇÃO DA RAÇA também conta: o humano que tira o +1 da Inteligência
// baixa o N, e a nova distribuição é recusada.
func TestTheRaceSpreadDoesNotLeaveTheSpecialistChoice(t *testing.T) {
	f, id := specialist(t, 1)
	seedRaca(t, f.s, id, "humano")
	if refusal := powerCommand(t, f, id, "atributos", `{"race_attributes":["intelligence","strength","dexterity"]}`); refusal != "" {
		t.Fatalf("distribuir o humano com Inteligência: %q", refusal)
	}
	picksSpecialist(t, f, id, "Ladinagem", "Furtividade")

	refusal := powerCommand(t, f, id, "atributos", `{"race_attributes":["wisdom","strength","dexterity"]}`)
	if !strings.Contains(refusal, "Especialista") {
		t.Errorf("tirar o +1 da Inteligência com duas perícias escolhidas não foi recusado: %q", refusal)
	}
}

// O DIÁLOGO DE ESCOLHAS mostra o Especialista no cartão do Ladino: quantas de
// quantas, as perícias TREINADAS como opção, e a pendência do que falta.
func TestTheChoicesDialogShowsTheSpecialist(t *testing.T) {
	f, id := specialist(t, 2)
	picksSpecialist(t, f, id, "Ladinagem")

	screen := powerScreen(t, f, id)
	if step := choiceStepTitled(t, screen, "Ladino · Especialista"); step.Tag != "1 de 2" {
		t.Errorf("o passo do Especialista não diz que falta uma perícia: %q", step.Tag)
	}
	field, found := choiceFieldIn(t, screen, "Ladino · Especialista", "Especialista")
	if !found {
		t.Fatal("o passo do Especialista não tem o bloco")
	}
	if field.Written != "1 de 2 · Int 2" {
		t.Errorf("o bloco do Especialista diz %q, quer 1 de 2 · Int 2", field.Written)
	}
	if pressed, offered := field.Options["Especialista: Furtividade"]; !offered || pressed {
		t.Errorf("o Especialista não oferece Furtividade desmarcada: %v", field.Options)
	}
	// A ESCOLHIDA é a opção marcada, e marcar e tirar são o mesmo toque.
	if !field.Options["Especialista: Ladinagem"] {
		t.Errorf("Ladinagem escolhida não está marcada no Especialista: %v", field.Options)
	}
	if _, offered := field.Options["Especialista: Diplomacia"]; offered {
		t.Error("Diplomacia não é treinada e aparece como opção do Especialista")
	}
}

// A saída da recusa de baixar a Inteligência: a escolha contada por atributo não ocupa vaga, então
// ela se libera. A ficha continua recusando enquanto a conta não fecha.
func TestFreeingASpecialistSkillLetsTheIntelligenceGoDown(t *testing.T) {
	f, id := specialist(t, 2)
	picksSpecialist(t, f, id, "Ladinagem", "Furtividade")
	lower := fmt.Sprintf("/personagens/%d/atributos/intelligence/-1", id)

	refused := f.requests(t, f.player, http.MethodPost, lower, "").Body.String()
	if !strings.Contains(refused, "libere 1") {
		t.Fatalf("a recusa de baixar a Inteligência não aponta a saída: %s", refused)
	}
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities&visao=nivel", id), "").Body.String()
	if !strings.Contains(page, "Liberar") {
		t.Error("com o Especialista cheio, a seção não oferece Liberar")
	}

	row := levelPickRow(t, f, id, "Furtividade")
	if refusal := sheetPost(t, f, id, fmt.Sprintf("/escolhas/%d/libera", row.ID), ""); refusal != "" {
		t.Fatalf("liberar foi recusado: %q", refusal)
	}
	if picks := storedPowerChoices(t, f, id); strings.Contains(picks, "Furtividade") || !strings.Contains(picks, "Ladinagem") {
		t.Errorf("devia sair só a liberada, e ficou %s", picks)
	}
	f.requests(t, f.player, http.MethodPost, lower, "")
	stored, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Intelligence != 1 {
		t.Errorf("com a perícia liberada, a Inteligência devia baixar para 1, e ficou %d", stored.Intelligence)
	}
}

// Liberar existe para quando a conta está cheia. Fora disso a escolha só se troca, e a vaga de um
// poder nunca fica vazia.
func TestOnlyAFullAttributeCountedChoiceCanBeFreed(t *testing.T) {
	f, id := specialist(t, 3)
	picksSpecialist(t, f, id, "Ladinagem", "Furtividade")
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities&visao=nivel", id), "").Body.String()
	if strings.Contains(page, "Liberar") {
		t.Error("com a conta sobrando, a seção ofereceu Liberar")
	}
	row := levelPickRow(t, f, id, "Furtividade")
	if refusal := sheetPost(t, f, id, fmt.Sprintf("/escolhas/%d/libera", row.ID), ""); !strings.Contains(refusal, "Trocar") {
		t.Errorf("liberar com a conta sobrando devia apontar o Trocar, e a recusa foi %q", refusal)
	}

	barbarian, barbarianID := barbaro(t, 3)
	if refusal := powerCommand(t, barbarian, barbarianID, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	slot := levelPickRow(t, barbarian, barbarianID, "strength")
	if refusal := sheetPost(t, barbarian, barbarianID, fmt.Sprintf("/escolhas/%d/libera", slot.ID), ""); !strings.Contains(refusal, "Trocar") {
		t.Errorf("a escolha que ocupa vaga não se libera, e a recusa foi %q", refusal)
	}
}
