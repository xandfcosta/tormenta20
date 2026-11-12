package api

import (
	"strings"
	"testing"
)

// O MESTRE EM ESCOLA (p38): "Escolha uma escola de magia. O custo para lançar magias dessa escola diminui
// em –1 PM. Pré-requisitos: Especialista em Escola com a escola escolhida". A escolha é dele, e só entre
// as escolas que o Especialista já escolheu.

const schoolMasterPath = "escolha/class.arcanista.mestre-em-escola/"

func schoolSpecialistIn(t *testing.T, f sceneFixture, school string) {
	t.Helper()
	seedPowerChoices(t, f.s, f.charID,
		`["class.arcanista.caminho-mago","class.arcanista.especialista-em-escola","class.arcanista.mestre-em-escola"]`,
		`{"class.arcanista.especialista-em-escola":["`+school+`"]}`)
}

func TestTheSchoolMasterChoosesAmongTheSpecialistSchools(t *testing.T) {
	f := newSceneFixture(t)
	schoolSpecialistIn(t, f, "evocacao")

	page := powerScreen(t, f, f.charID)
	if !strings.Contains(page, "/"+schoolMasterPath+"evocacao") {
		t.Error("o Mestre em Escola não oferece a Evocação, que o Especialista escolheu")
	}
	if strings.Contains(page, "/"+schoolMasterPath+"ilusao") {
		t.Error("o Mestre em Escola oferece a Ilusão, que o Especialista não escolheu")
	}

	if refusal := powerCommand(t, f, f.charID, schoolMasterPath+"ilusao", ""); !strings.Contains(refusal, "Especialista em Escola") {
		t.Errorf("o Mestre em Ilusão sem o Especialista em Ilusão deu %q; want a recusa que nomeia o Especialista", refusal)
	}
	if refusal := powerCommand(t, f, f.charID, schoolMasterPath+"evocacao", ""); refusal != "" {
		t.Fatalf("o Mestre em Evocação, com o Especialista em Evocação, foi recusado: %q", refusal)
	}
	if got := storedPowerChoices(t, f, f.charID); !strings.Contains(got, `"class.arcanista.mestre-em-escola":["evocacao"]`) {
		t.Errorf("a escolha do Mestre não foi gravada: %s", got)
	}
}
