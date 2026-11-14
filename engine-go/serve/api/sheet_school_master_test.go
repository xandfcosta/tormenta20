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

// O ESPECIALISTA EM ESCOLA SE ESCOLHE UMA VEZ SÓ: a p38 diz "Escolha uma escola de magia", sem o
// "você pode escolher este poder outras vezes" dos repetíveis, e a p33 diz que, a menos que
// especificado, não se escolhe o mesmo poder mais de uma vez. Escolher outra escola TROCA a de antes.
const schoolSpecialistPath = "escolha/class.arcanista.especialista-em-escola/"

func TestTheSchoolSpecialistKeepsOneSchool(t *testing.T) {
	f := newSceneFixture(t)
	schoolSpecialistIn(t, f, "evocacao")

	if refusal := powerCommand(t, f, f.charID, schoolSpecialistPath+"ilusao", ""); refusal != "" {
		t.Fatalf("trocar a escola do Especialista foi recusado: %q", refusal)
	}
	got := storedPowerChoices(t, f, f.charID)
	if !strings.Contains(got, `"class.arcanista.especialista-em-escola":["ilusao"]`) {
		t.Errorf("o Especialista devia ter só a Ilusão, que trocou a Evocação: %s", got)
	}
}

// A FICHA QUE JÁ TEM DUAS ESCOLAS (de quando o catálogo deixava) é recusada no próximo gesto de
// poderes, como toda escolha fora da conta, e a frase diz qual poder tirar (decisão do dono).
func TestASheetWithTwoSpecialistSchoolsIsRefusedOnTheNextGesture(t *testing.T) {
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID,
		`["class.arcanista.caminho-mago","class.arcanista.especialista-em-escola","class.arcanista.mestre-em-escola"]`,
		`{"class.arcanista.especialista-em-escola":["evocacao","ilusao"]}`)

	refusal := powerCommand(t, f, f.charID, schoolMasterPath+"evocacao", "")
	if !strings.Contains(refusal, "Especialista em Escola") || !strings.Contains(refusal, "uma vez só") {
		t.Errorf("a ficha com duas escolas no Especialista deu %q; want a recusa que diz que ele se escolhe uma vez só", refusal)
	}
}
