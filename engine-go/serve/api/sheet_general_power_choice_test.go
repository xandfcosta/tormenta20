package api

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O PODER GERAL QUE PEDE ESCOLHA (p128, p130, ALE-510).
//
//	"Foco em Perícia. Escolha uma perícia. Quando faz um teste dessa perícia,
//	você pode gastar 1 PM para rolar dois dados e usar o melhor resultado. Você
//	pode escolher este poder outras vezes para perícias diferentes. Este poder
//	não pode ser aplicado em Luta e Pontaria (mas veja Foco em Arma).
//	Pré-requisito: treinado na perícia escolhida."
//
//	"Foco em Arma. Escolha uma arma. Você recebe +2 em testes de ataque com
//	essa arma. Você pode escolher este poder outras vezes para armas
//	diferentes."
//
// A escolha mora em `powerChoices`, como a dos poderes de classe, e cada uma
// ocupa uma vaga — menos a que veio da origem (Membro de Guilda).

const skillFocusPath = "escolha/foco-em-pericia/"

// focusedAtTheTable é uma bucaneira de nível 5 na mesa, treinada em Enganação,
// Intimidação e Luta e não em Percepção, com a origem pedida.
func focusedAtTheTable(t *testing.T, origin, originChoices string) (sceneFixture, int64) {
	t.Helper()
	f := umaMesaAoVivo(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Focada", Origin: origin, Level: 5,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: originChoices, ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a ficha: %v", err)
	}
	seedClasse(t, f.s, id, "Bucaneiro", 5)
	seedMember(t, f.s, f.campaignID, id)
	seedPericia(t, f.s, id, "Enganação", "charisma", true)
	seedPericia(t, f.s, id, "Intimidação", "charisma", true)
	seedPericia(t, f.s, id, "Percepção", "wisdom", false)
	// LUTA TREINADA, para a recusa do Foco em Luta ser a da p130 e não a do
	// pré-requisito.
	seedPericia(t, f.s, id, "Luta", "strength", true)
	f.charID = id
	return f, id
}

func storedClassPowers(t *testing.T, f sceneFixture, id int64) []string {
	t.Helper()
	powers, _ := storedChoices(t, f.s, id)
	return sheet.UnmarshalStrings(powers)
}

// rollsTwoDice diz se o teste com o Foco em Perícia rolou os dois d20.
func rollsTwoDice(t *testing.T, f sceneFixture, id int64, skill string) bool {
	t.Helper()
	if refusal := rollsWith(t, f, id, skill, `{"expertise_spends":{"foco_em_pericia":1}}`); refusal != "" {
		return false
	}
	return len(stateOf(t, f.s.sessions, f.sessionID).LastSkillTest.Rolls) == 2
}

func TestSkillFocusRollsTwoDiceOnlyInTheChosenSkill(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape("Enganação"), ""); refusal != "" {
		t.Fatalf("escolher Enganação no Foco em Perícia foi recusado: %q", refusal)
	}
	if !rollsTwoDice(t, f, id, "Enganação") {
		t.Error("o Foco em Enganação não rolou dois d20 em Enganação")
	}
	if rollsTwoDice(t, f, id, "Intimidação") {
		t.Error("o Foco em Enganação rolou dois d20 em Intimidação, que não foi escolhida")
	}
}

// "Pré-requisito: treinado na perícia escolhida", e nunca Luta nem Pontaria.
func TestSkillFocusRefusesAnUntrainedSkillAndTheAttackSkills(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape("Percepção"), ""); !strings.Contains(refusal, "treinad") {
		t.Errorf("o Foco em Percepção, sem treino, deu %q; want a recusa pelo pré-requisito", refusal)
	}
	if refusal := powerCommand(t, f, id, skillFocusPath+"Luta", ""); refusal == "" {
		t.Error("o Foco em Luta passou, e a p130 o manda para o Foco em Arma")
	}
}

// O pré-requisito vale enquanto o poder fica: destreinar a escolhida é
// recusado, e destreinar outra perícia não.
func TestSkillFocusKeepsTheChosenSkillTrained(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape("Intimidação"), ""); refusal != "" {
		t.Fatalf("escolher Intimidação no Foco foi recusado: %q", refusal)
	}
	if refusal := sheetPost(t, f, id, "/pericias/treino/"+url.PathEscape("Enganação"), ""); refusal != "" {
		t.Fatalf("destreinar Enganação, sem poder nenhum nela, foi recusado: %q", refusal)
	}
	if refusal := sheetPost(t, f, id, "/pericias/treino/"+url.PathEscape("Intimidação"), ""); !strings.Contains(refusal, "Foco em Perícia") {
		t.Errorf("destreinar Intimidação, escolhida no Foco em Perícia, deu %q; want a recusa que nomeia o poder", refusal)
	}
}

// ESCOLHER DE NOVO é outra perícia e outra vaga.
func TestSkillFocusTakenTwiceKeepsBothSkills(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	for _, skill := range []string{"Enganação", "Intimidação"} {
		if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape(skill), ""); refusal != "" {
			t.Fatalf("escolher %s foi recusado: %q", skill, refusal)
		}
	}
	if !rollsTwoDice(t, f, id, "Intimidação") || !rollsTwoDice(t, f, id, "Enganação") {
		t.Error("o Foco escolhido duas vezes não vale nas duas perícias")
	}
	if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape("Enganação"), ""); refusal == "" {
		t.Error("o Foco em Enganação passou duas vezes; a p130 diz perícias diferentes")
	}
}

// O FOCO DA ORIGEM (Membro de Guilda, p92) pede a perícia e não gasta vaga.
func TestTheOriginSkillFocusTakesItsSkillWithoutASlot(t *testing.T) {
	f, id := focusedAtTheTable(t, "membro-de-guilda", `["poder-foco-em-pericia"]`)
	if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape("Enganação"), ""); refusal != "" {
		t.Fatalf("escolher a perícia do Foco da origem foi recusado: %q", refusal)
	}
	if got := storedClassPowers(t, f, id); len(got) != 0 {
		t.Errorf("o Foco da origem entrou na lista de poderes escolhidos (%v) e gastou vaga", got)
	}
	if !rollsTwoDice(t, f, id, "Enganação") {
		t.Error("o Foco da origem em Enganação não rolou dois d20")
	}
}

// A FICHA DIZ o que falta escolher, e a lista oferece só as treinadas.
func TestTheOriginSkillFocusIsPendingAndOffersTheTrainedSkills(t *testing.T) {
	f, id := focusedAtTheTable(t, "membro-de-guilda", `["poder-foco-em-pericia"]`)
	page := powerScreen(t, f, id)
	if step := choiceStepTitled(t, page, "Foco em Perícia · perícia"); step.Tag != "falta 1" {
		t.Errorf("o passo do Foco da origem diz %q, e falta escolher a perícia", step.Tag)
	}
	if !strings.Contains(page, "/escolha/foco-em-pericia/Engana") {
		t.Error("a lista de poderes não oferece Enganação, treinada, ao Foco em Perícia")
	}
	if strings.Contains(page, "/escolha/foco-em-pericia/Percep") {
		t.Error("a lista oferece Percepção, sem treino, ao Foco em Perícia")
	}
}
