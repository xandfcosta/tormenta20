package table

import (
	"testing"

	"t20engine/domain/live"
)

// A FAIXA SEM FIM DA SESSÃO PARALELA (ALE-473, decisão do dono): ela começa em
// quem joga agora, segue pela rodada seguinte e marca CADA volta com o número
// da rodada que começa — e não só a primeira, como a faixa de três.
func TestTheEndlessStripStartsNowAndMarksEveryNewRound(t *testing.T) {
	mine := int64(7)
	queue := []live.InitiativeEntry{
		{Label: "Arwen", CharacterID: &mine}, {Label: "Ogro"}, {Label: "Zumbi"},
	}
	strip := endlessStripOf(&live.SessionRuntimeState{
		Initiative: queue, TurnIndex: 1, Round: 2,
		Scene: &live.Scene{Kind: live.SceneAction, Number: 1}, ScenesSoFar: 1,
	}, map[int64]bool{7: true})

	want := []struct {
		label    string
		newRound int
		now      bool
	}{
		{"Ogro", 0, true}, {"Zumbi", 0, false}, {"Arwen", 3, false},
		{"Ogro", 0, false}, {"Zumbi", 0, false}, {"Arwen", 4, false},
	}
	if len(strip) != len(want) {
		t.Fatalf("a faixa veio com %d turnos, queria a fila inteira mais três (%d): %+v", len(strip), len(want), strip)
	}
	for i, w := range want {
		if strip[i].Label != w.label || strip[i].NewRound != w.newRound || strip[i].Now != w.now {
			t.Errorf("posição %d: %q, rodada nova %d, agora %v; queria %q, %d, %v",
				i, strip[i].Label, strip[i].NewRound, strip[i].Now, w.label, w.newRound, w.now)
		}
	}
	// "VOCÊ" vale em toda volta: é UM personagem só, repetido pela fila.
	for _, i := range []int{2, 5} {
		if !strip[i].SaysYou {
			t.Errorf("a posição %d é a minha e não diz \"você\"", i)
		}
	}
}

// O STREAM DA SESSÃO PARALELA manda só o que ela desenha: o cabeçalho. A lista
// e a página não podem discordar sobre quais regiões existem — um remendo
// para um id que não está na página é escrever no vazio.
func TestTheParallelSessionStreamsOnlyItsHeader(t *testing.T) {
	regions := TableRegions(View{Tempestade: true, Eu: &tableMe{}})
	if len(regions) != 1 || regions[0].ID != "table-header" {
		ids := []string{}
		for _, r := range regions {
			ids = append(ids, r.ID)
		}
		t.Errorf("a sessão paralela recebe as regiões %v; queria só o cabeçalho", ids)
	}
}
