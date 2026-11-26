package session

import (
	"testing"

	"t20engine/domain/live"
)

// O DADO DO EM CHAMAS é do motor, e quem o troca pelo da Lança Ígnea (p196) é o regime: os dois o conhecem pelo
// mesmo id, e é aqui, que junta os dois, que isso se prova.
func TestTheLanceReplacesTheFlamesTheEngineRolls(t *testing.T) {
	ogre := live.InitiativeEntry{ID: "ogro", Label: "Ogro"}
	due := DueRollsWith(ogre, []string{live.ConditionBurning})
	if len(due) != 1 || due[0].Condition != "Em Chamas" || due[0].Parts[0].Dice != "1d6" {
		t.Fatalf("o ogro Em Chamas rola %+v; want o 1d6 de fogo do motor", due)
	}
	ogre.Marks = []live.QueueMark{{Source: "Lança Ígnea de Aleph", At: "inicio", Burning: true,
		Parts: []live.QueueMarkPart{{Dice: "3d6", Kind: "fogo"}}}}
	if due := DueRollsWith(ogre, []string{live.ConditionBurning}); len(due) != 1 || due[0].Source != "Lança Ígnea de Aleph" {
		t.Errorf("com as chamas da Lança, o ogro rola %+v; want só a Lança", due)
	}
}
