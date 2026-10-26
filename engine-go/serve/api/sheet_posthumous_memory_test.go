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

// ESCOLHER A PERÍCIA A TREINA, e destreiná-la é recusado (decisão do dono).
func TestThePosthumousMemoryTrainsTheChosenSkill(t *testing.T) {
	f, id := osteon(t)

	if !strings.Contains(powerScreen(t, f, id), "Osteon: Memória Póstuma: 1 escolha por fazer") {
		t.Error("o osteon sem a Memória escolhida não tem a pendência")
	}
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
	if !strings.Contains(powerPanel(screen), "Vontade de Ferro") {
		t.Error("o poder geral da Memória Póstuma não aparece na aba Poderes")
	}
	for _, want := range []string{"1 de 1", `aria-label="Tirar Vontade de Ferro de Memória Póstuma"`, `aria-label="Memória Póstuma: Acrobacia"`} {
		if !strings.Contains(screen, want) {
			t.Errorf("a tela não tem %q", want)
		}
	}
}
