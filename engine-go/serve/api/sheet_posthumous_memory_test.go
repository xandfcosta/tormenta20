package api

import (
	"context"
	"strings"
	"testing"
)

// A MEMÓRIA PÓSTUMA (osteon, p29; ALE-502): "Você se torna treinado em uma
// perícia (não precisa ser da sua classe) ou recebe um poder geral a sua
// escolha." Uma escolha só.

func osteon(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := barbaro(t, 3)
	seedRaca(t, f.s, id, "osteon")
	seedPericia(t, f.s, id, "Luta", "strength", false)
	return f, id
}

func trainedOn(t *testing.T, f sceneFixture, id int64, skill string) bool {
	t.Helper()
	rows, err := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as perícias: %v", err)
	}
	for _, e := range rows {
		if e.Name == skill {
			return e.Trained == 1
		}
	}
	t.Fatalf("a ficha não tem %s", skill)
	return false
}

// posthumousMemoryField é o bloco da Memória Póstuma no passo do osteon.
func posthumousMemoryField(t *testing.T, screen string) choiceFieldOnScreen {
	t.Helper()
	field, found := choiceFieldIn(t, screen, "Raça · Osteon", "Memória Póstuma")
	if !found {
		t.Fatal("o passo do osteon não tem o bloco da Memória Póstuma")
	}
	return field
}

// ESCOLHER A PERÍCIA A TREINA, e destreiná-la é recusado (decisão do dono).
func TestThePosthumousMemoryTrainsTheChosenSkill(t *testing.T) {
	f, id := osteon(t)

	assertRaceAbilityPending(t, powerScreen(t, f, id), "Raça · Osteon", "Memória Póstuma", "0 de 1")
	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "Luta"); refusal != "" {
		t.Fatalf("escolher Luta na Memória Póstuma: %q", refusal)
	}
	if !trainedOn(t, f, id, "Luta") {
		t.Error("Luta escolhida na Memória Póstuma não ficou treinada")
	}
	if refusal := sheetPost(t, f, id, "/pericias/treino/Luta", ""); !strings.Contains(refusal, "Memória Póstuma") {
		t.Errorf("destreinar Luta, escolhida na Memória Póstuma, não foi recusado: %q", refusal)
	}
	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "vontade-de-ferro"); !strings.Contains(refusal, "1") {
		t.Errorf("a segunda escolha da Memória Póstuma foi aceita: %q", refusal)
	}
}

// O PODER GERAL é da ficha: aparece no acervo.
func TestThePosthumousMemoryGivesAGeneralPower(t *testing.T) {
	f, id := osteon(t)

	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "vontade-de-ferro"); refusal != "" {
		t.Fatalf("escolher Vontade de Ferro: %q", refusal)
	}
	screen := powerScreen(t, f, id)
	if _, found := rowTitled(powerRowsOnScreen(t, screen), "Vontade de Ferro"); !found {
		t.Error("o poder geral da Memória Póstuma não aparece na aba Poderes")
	}
	field := posthumousMemoryField(t, screen)
	if field.Written != "1 de 1" {
		t.Errorf("a Memória Póstuma diz %q, quer 1 de 1", field.Written)
	}
	// A ESCOLHIDA é a opção marcada, e marcar e tirar são o mesmo toque.
	if !field.Options["Memória Póstuma: Vontade de Ferro"] {
		t.Error("Vontade de Ferro escolhida não está marcada na Memória Póstuma")
	}
	if _, offered := field.Options["Memória Póstuma: Acrobacia"]; !offered {
		t.Error("a Memória Póstuma não oferece a perícia Acrobacia")
	}
}

// A TERCEIRA OPÇÃO (ALE-504): uma habilidade de outra raça HUMANOIDE que não
// humano — e o tamanho dela.
func TestThePosthumousMemoryBorrowsAHumanoidAbility(t *testing.T) {
	f, id := osteon(t)

	for _, foreign := range []string{"humano-versatil", "lefou-deformidade"} {
		if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", foreign); refusal == "" {
			t.Errorf("%s não é de raça humanoide que não humano, e a Memória Póstuma a aceitou", foreign)
		}
	}
	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "goblin-rato-das-ruas"); refusal != "" {
		t.Fatalf("emprestar Rato das Ruas do goblin: %q", refusal)
	}
	screen := powerScreen(t, f, id)
	if _, found := rowTitled(powerRowsOnScreen(t, screen), "Rato das Ruas"); !found {
		t.Error("a habilidade emprestada não aparece na aba Poderes")
	}
	field := posthumousMemoryField(t, screen)
	if !strings.Contains(field.HTML, "Goblin · Pequeno") {
		t.Error("o grupo do goblin não diz o tamanho que vem junto")
	}
	if !field.Options["Memória Póstuma: Rato das Ruas"] {
		t.Error("Rato das Ruas emprestada não está marcada na Memória Póstuma")
	}
}

// A EMPRESTADA QUE PEDE ESCOLHA a faz logo abaixo, e trocar a emprestada
// apaga a escolha dela.
func TestTheBorrowedAbilityMakesItsOwnChoice(t *testing.T) {
	f, id := osteon(t)

	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "qareen-resistencia-elemental"); refusal != "" {
		t.Fatalf("emprestar a Resistência Elemental: %q", refusal)
	}
	assertRaceAbilityPending(t, powerScreen(t, f, id), "Raça · Osteon", "Resistência Elemental", "0 de 1")
	if refusal := raceAbilityCommand(t, f, id, "qareen-resistencia-elemental", "qareen-res-fogo"); refusal != "" {
		t.Fatalf("escolher fogo na emprestada: %q", refusal)
	}
	if field, _ := choiceFieldIn(t, powerScreen(t, f, id), "Raça · Osteon", "Resistência Elemental"); !field.Options["Resistência Elemental: Fogo (do fogo)"] {
		t.Errorf("o elemento escolhido não aparece marcado no sub-bloco: %v", field.Options)
	}
	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "qareen-resistencia-elemental"); refusal != "" {
		t.Fatalf("devolver a emprestada: %q", refusal)
	}
	if got := storedRaceAttributeChoices(t, f, id); strings.Contains(got, "qareen") {
		t.Errorf("devolver a habilidade emprestada deixou a escolha dela para trás: %s", got)
	}
}
