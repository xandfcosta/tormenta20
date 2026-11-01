package api

import (
	"net/url"
	"slices"
	"strings"
	"testing"
)

// O TREINAMENTO EM PERÍCIA (p131, ALE-510):
//
//	"Você se torna treinado em uma perícia a sua escolha. Você pode escolher
//	este poder outras vezes para perícias diferentes."
//
// Escolher grava o treino na coluna, como a Memória Póstuma (p29), e a
// escolhida não se destreina enquanto for a escolhida.

const skillTrainingPath = "escolha/treinamento-em-pericia/"

func TestSkillTrainingTrainsTheChosenSkillAndTakesASlot(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Percepção"), ""); refusal != "" {
		t.Fatalf("escolher Percepção no Treinamento em Perícia foi recusado: %q", refusal)
	}
	if !trainedOn(t, f, id, "Percepção") {
		t.Error("Percepção escolhida no Treinamento em Perícia não ficou treinada")
	}
	if got := storedClassPowers(t, f, id); !slices.Contains(got, "treinamento-em-pericia") {
		t.Errorf("o Treinamento em Perícia não ocupou vaga: poderes escolhidos %v", got)
	}
}

// "para perícias diferentes": nem a mesma duas vezes, nem a que já é treinada.
func TestSkillTrainingRefusesARepeatedOrAlreadyTrainedSkill(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Enganação"), ""); refusal == "" {
		t.Error("o Treinamento em Enganação, que já é treinada, passou")
	}
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Percepção"), ""); refusal != "" {
		t.Fatalf("escolher Percepção foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Percepção"), ""); refusal == "" {
		t.Error("o Treinamento em Percepção passou duas vezes; a p131 diz perícias diferentes")
	}
}

// Destreinar a escolhida desfaria o poder sem tirá-lo da ficha.
func TestSkillTrainingKeepsTheChosenSkillTrained(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Percepção"), ""); refusal != "" {
		t.Fatalf("escolher Percepção foi recusado: %q", refusal)
	}
	if refusal := sheetPost(t, f, id, "/pericias/treino/"+url.PathEscape("Percepção"), ""); !strings.Contains(refusal, "Treinamento em Perícia") {
		t.Errorf("destreinar Percepção, escolhida no Treinamento em Perícia, deu %q; want a recusa que nomeia o poder", refusal)
	}
}

// A perícia treinada pelo poder vale como treinada para o Foco em Perícia.
func TestSkillFocusTakesTheSkillTheTrainingTrained(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Percepção"), ""); refusal != "" {
		t.Fatalf("escolher Percepção no Treinamento foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, skillFocusPath+url.PathEscape("Percepção"), ""); refusal != "" {
		t.Errorf("o Foco em Percepção, treinada pelo Treinamento, foi recusado: %q", refusal)
	}
}

func TestSkillTrainingOffersOnlyTheUntrainedSkills(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	page := powerScreen(t, f, id)
	if !strings.Contains(page, "/escolha/treinamento-em-pericia/Percep") {
		t.Error("a lista de poderes não oferece Percepção, sem treino, ao Treinamento em Perícia")
	}
	if strings.Contains(page, "/escolha/treinamento-em-pericia/Engana") {
		t.Error("a lista oferece Enganação, já treinada, ao Treinamento em Perícia")
	}
}

// Trocar o Treinamento em Perícia leva o treino junto, e a perícia já treinada não entra.
func TestSwappingSkillTrainingMovesTheTraining(t *testing.T) {
	f, id := focusedAtTheTable(t, "soldado", "[]")
	seedPericia(t, f.s, id, "Sobrevivência", "wisdom", false)
	if refusal := powerCommand(t, f, id, skillTrainingPath+url.PathEscape("Percepção"), ""); refusal != "" {
		t.Fatalf("escolher Percepção: %q", refusal)
	}
	row := levelPickRow(t, f, id, "Percepção")
	if refusal := swapRow(t, f, id, row.ID, "treinamento-em-pericia/"+url.PathEscape("Percepção")); refusal != "" {
		t.Errorf("trocar pela mesma perícia não tem o que recusar, e a recusa foi %q", refusal)
	}
	if !trainedOn(t, f, id, "Percepção") {
		t.Fatal("trocar pela mesma perícia tirou o treino dela")
	}
	if refusal := swapRow(t, f, id, levelPickRow(t, f, id, "Percepção").ID, "treinamento-em-pericia/"+url.PathEscape("Sobrevivência")); refusal != "" {
		t.Fatalf("trocar: %q", refusal)
	}
	if trainedOn(t, f, id, "Percepção") {
		t.Error("a perícia que saiu da escolha continuou treinada")
	}
	if !trainedOn(t, f, id, "Sobrevivência") {
		t.Error("a perícia que entrou na escolha não ficou treinada")
	}
}
