package api

import (
	"strings"
	"testing"
)

// O VERSÁTIL (humano, p19): "Você se torna treinado em duas perícias a sua escolha (não
// precisam ser da sua classe). Você pode trocar uma dessas perícias por um poder geral a
// sua escolha."

const versatile = "humano-versatil"

func humanBarbarian(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedRaca(t, f.s, id, "humano")
	seedExpertiseRows(t, f, id)
	return f, id
}

func TestTheVersatileTrainsTwoSkills(t *testing.T) {
	f, id := humanBarbarian(t)
	assertRaceAbilityPending(t, parallelSheet(t, f, id, "abilities"), "Raça · Humano", "Versátil", "0 de 2")

	for _, skill := range []string{"Cavalgar", "Luta"} {
		if refusal := raceAbilityCommand(t, f, id, versatile, skill); refusal != "" {
			t.Fatalf("escolher %s: %q", skill, refusal)
		}
		if !trainedOn(t, f, id, skill) {
			t.Errorf("%s, escolhida no Versátil, não ficou treinada", skill)
		}
	}
	if refusal := raceAbilityCommand(t, f, id, versatile, "Atletismo"); !strings.Contains(refusal, "2") {
		t.Errorf("a terceira escolha devia ser recusada, e veio %q", refusal)
	}
	if refusal := sheetPost(t, f, id, "/pericias/treino/Luta", ""); !strings.Contains(refusal, "Versátil") {
		t.Errorf("destreinar a escolhida devia ser recusado, e veio %q", refusal)
	}
}

// As duas não gastam escolha da classe: é o defeito que fez a escolha existir.
func TestTheVersatileSkillsLeaveTheClassCount(t *testing.T) {
	f, id := humanBarbarian(t)
	for _, skill := range []string{"Adestramento", "Atletismo"} {
		if refusal := raceAbilityCommand(t, f, id, versatile, skill); refusal != "" {
			t.Fatalf("escolher %s: %q", skill, refusal)
		}
	}
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "0 de 4" {
		t.Errorf("as perícias do Versátil contaram na classe: o passo diz %q", tag)
	}
}

func TestTheVersatileSwapsOneSkillForAGeneralPower(t *testing.T) {
	f, id := humanBarbarian(t)
	if refusal := raceAbilityCommand(t, f, id, versatile, "Cavalgar"); refusal != "" {
		t.Fatalf("escolher a perícia: %q", refusal)
	}
	if refusal := raceAbilityCommand(t, f, id, versatile, "vontade-de-ferro"); refusal != "" {
		t.Fatalf("trocar uma perícia por um poder geral: %q", refusal)
	}
	if !strings.Contains(sheetTab(t, f, id, "abilities"), "Vontade de Ferro") {
		t.Error("o poder geral do Versátil não aparece na aba Poderes")
	}
}

// O livro deixa trocar UMA delas.
func TestTheVersatileRefusesTwoGeneralPowers(t *testing.T) {
	f, id := humanBarbarian(t)
	if refusal := raceAbilityCommand(t, f, id, versatile, "vontade-de-ferro"); refusal != "" {
		t.Fatalf("o primeiro poder: %q", refusal)
	}
	refusal := raceAbilityCommand(t, f, id, versatile, "sortudo")
	if !strings.Contains(refusal, "um poder geral") {
		t.Errorf("o segundo poder devia ser recusado pelo limite, e veio %q", refusal)
	}
}

// Tomar habilidade de outra raça é da Memória Póstuma (p29), e o Versátil não tem isso.
func TestTheVersatileTakesNoAbilityOfAnotherRace(t *testing.T) {
	f, id := humanBarbarian(t)
	if refusal := raceAbilityCommand(t, f, id, versatile, "goblin-rato-das-ruas"); refusal == "" {
		t.Error("o Versátil tomou a habilidade de outra raça")
	}
	if strings.Contains(parallelSheet(t, f, id, "abilities"), "habilidade de outra raça") {
		t.Error("o bloco do Versátil oferece habilidade de outra raça")
	}
}

// O humano antigo marcou as perícias à mão: não há como adivinhar quais vieram do Versátil,
// então ele as aponta uma vez, sem perder nenhuma.
func TestAnOldHumanPointsAtTheSkillsAlreadyTrained(t *testing.T) {
	f, id := humanBarbarian(t)
	for _, skill := range []string{"Adestramento", "Atletismo"} {
		trainExpertise(t, f, id, skill)
	}
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "2 de 4" {
		t.Fatalf("antes de apontar, as duas contam na classe, e o passo diz %q", tag)
	}
	assertRaceAbilityPending(t, parallelSheet(t, f, id, "abilities"), "Raça · Humano", "Versátil", "0 de 2")

	for _, skill := range []string{"Adestramento", "Atletismo"} {
		if refusal := raceAbilityCommand(t, f, id, versatile, skill); refusal != "" {
			t.Fatalf("apontar %s: %q", skill, refusal)
		}
		if !trainedOn(t, f, id, skill) {
			t.Errorf("apontar %s tirou o treino dela", skill)
		}
	}
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "0 de 4" {
		t.Errorf("depois de apontar, as duas saem da classe, e o passo diz %q", tag)
	}
}
