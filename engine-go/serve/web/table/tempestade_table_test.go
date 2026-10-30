package table

import (
	"strings"
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

// O STREAM DA SESSÃO PARALELA manda só o que ela desenha: o cabeçalho, o
// cartão do ataque, a barra de alvo das Ações e as três peças da Mesa. A lista e a página não podem
// discordar sobre quais regiões existem — um remendo para um id que não está
// na página é escrever no vazio.
func TestTheParallelSessionStreamsOnlyWhatItDraws(t *testing.T) {
	regions := TableRegions(View{Tempestade: true, Eu: &tableMe{}})
	ids := []string{}
	for _, r := range regions {
		ids = append(ids, r.ID)
	}
	want := []string{"table-header", "table-attack", "table-turn-target", "table-register", "table-party", "table-tracker"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("a sessão paralela recebe as regiões %v; queria %v", ids, want)
	}
}

// OS GESTOS DA MESA PARALELA levam a marca: a resposta deles redesenha a mesa,
// e sem ela devolveria o desenho de hoje (o mesmo defeito do stream).
func TestATableGestureOfTheParallelSessionCarriesTheMark(t *testing.T) {
	v := View{CampaignID: 6, SessionID: 9, Tempestade: true}
	if got := tableCommand(v, "POST", "sangramento/d20"); !strings.Contains(got, "/sangramento/d20?tempestade=1") {
		t.Errorf("o gesto da mesa paralela saiu sem a marca: %s", got)
	}
	v.Tempestade = false
	if got := tableCommand(v, "POST", "sangramento/d20"); strings.Contains(got, "tempestade") {
		t.Errorf("o gesto da mesa de hoje levou a marca: %s", got)
	}
}
