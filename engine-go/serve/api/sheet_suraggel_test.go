package api

import (
	"context"
	"strings"
	"testing"
)

// O SURAGGEL TEM A HABILIDADE DA ASCENDÊNCIA DELE (p31, ALE-505): a Luz
// Sagrada é do aggelus, e as Sombras Profanas, do sulfure. A ascendência é a
// escolhida na distribuição de atributo — um lugar só.
func TestTheSuraggelSeesTheAbilityOfItsAncestry(t *testing.T) {
	f, id := barbaro(t, 3)
	seedRaca(t, f.s, id, "suraggel")
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET raceAttributeChoices = '{"ascendencia":"aggelus"}' WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}

	panel := powerPanel(powerScreen(t, f, id))
	if !strings.Contains(panel, "Luz Sagrada") {
		t.Error("o suraggel aggelus não tem a Luz Sagrada")
	}
	if strings.Contains(panel, "Sombras Profanas") {
		t.Error("o suraggel aggelus tem as Sombras Profanas, que são do sulfure")
	}
	if refusal := powerCommand(t, f, id, "usa/race.suraggel.luz-sagrada", ""); strings.Contains(refusal, "não é") {
		t.Errorf("o gesto da Luz Sagrada não é do suraggel aggelus: %q", refusal)
	}
}
