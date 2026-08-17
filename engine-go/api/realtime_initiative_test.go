package api

import "testing"

// O patch que chega do socket é montado por uma LISTA de campos escrita à mão,
// e uma lista assim envelhece: ao entrar o `creatureId` (ALE-137) o cliente
// passou a mandá-lo e o servidor a descartá-lo em silêncio, com tudo
// compilando. Este teste percorre os campos em vez de conferir um.
func TestParseEntryPatchNaoPerdeCampo(t *testing.T) {
	patch := parseEntryPatch(map[string]any{
		"label":       "Chefe bandido",
		"initiative":  float64(17),
		"characterId": float64(3),
		"hpCurrent":   float64(20),
		"hpMax":       float64(30),
		"mpCurrent":   float64(5),
		"mpMax":       float64(10),
		"hpHidden":    true,
		"creatureId":  float64(7),
	})

	faltando := []string{}
	if patch.Label == nil {
		faltando = append(faltando, "label")
	}
	if patch.Initiative == nil {
		faltando = append(faltando, "initiative")
	}
	if patch.CharacterID == nil {
		faltando = append(faltando, "characterId")
	}
	if patch.HpCurrent == nil || patch.HpMax == nil {
		faltando = append(faltando, "hp")
	}
	if patch.MpCurrent == nil || patch.MpMax == nil {
		faltando = append(faltando, "mp")
	}
	if patch.HpHidden == nil {
		faltando = append(faltando, "hpHidden")
	}
	if patch.CreatureID == nil {
		faltando = append(faltando, "creatureId")
	}
	if len(faltando) > 0 {
		t.Fatalf("o parser descartou: %v", faltando)
	}
	if *patch.CreatureID != 7 {
		t.Errorf("creatureId chegou como %d, queria 7", *patch.CreatureID)
	}
}
