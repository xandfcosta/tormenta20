package live

import "testing"

// O PERIGO NA MESA (ver GLOSSARY, **perigo**): o disparo fica no estado do mestre e não chega ao jogador, e o perigo
// que dura rodadas vira a marca que o mestre põe.

func firedGas() HazardRoll {
	return HazardRoll{
		Hazard: "Gás Venenoso", Kind: "armadilha", BookPage: 318,
		Parts: []SpellCastPart{{Dice: "1d12", Kind: "perda de vida por veneno", Rolls: []int{7}, Total: 7}},
		Save:  "Fortitude reduz à metade", DC: 20,
		Lingering: &QueueMark{Source: "Gás Venenoso", At: "inicio", Counted: true, TurnsLeft: 4,
			Parts: []QueueMarkPart{{Dice: "1d12", Kind: "perda de vida por veneno"}}},
	}
}

func TestTheFiredHazardReachesOnlyTheMaster(t *testing.T) {
	for _, inScene := range []bool{true, false} {
		st := EmptyRuntimeState()
		if inScene {
			st.Scene = &Scene{Kind: SceneAction}
		}
		if err := RecordRoll(st, firedGas()); err != nil {
			t.Fatal(err)
		}
		if st.LastHazard == nil || st.LastHazard.Hazard != "Gás Venenoso" {
			t.Fatalf("em cena %v, o disparo não ficou no estado do mestre: %+v", inScene, st.LastHazard)
		}
		if seen := RedactForPlayers(st).LastHazard; seen != nil {
			t.Errorf("em cena %v, o jogador recebeu o disparo do %s: a CD e o dano são da ficha do perigo, que ele não conhece",
				inScene, seen.Hazard)
		}
	}
}

func TestTheHazardThatLastsBecomesTheMarkTheMasterPuts(t *testing.T) {
	st := EmptyRuntimeState()
	st.Initiative = []InitiativeEntry{{ID: "ogro", Label: "Ogro", Type: "npc"}}
	if err := RecordRoll(st, firedGas()); err != nil {
		t.Fatal(err)
	}
	if st.Markable == nil || st.Markable.Source != "Gás Venenoso" || st.Markable.Who != "" || st.Markable.TurnsLeft != 4 {
		t.Fatalf("o gás disparado deixou a marca %+v; want a do Gás Venenoso, sem conjurador, por mais 4 turnos", st.Markable)
	}
	if err := MarkEntry(st, "ogro"); err != nil {
		t.Fatal(err)
	}
	if marks := st.Initiative[0].Marks; len(marks) != 1 || marks[0].Source != "Gás Venenoso" {
		t.Fatalf("o Ogro marcado com o gás tem as marcas %+v", marks)
	}

	// A ARMADILHA QUE NÃO DURA não toca a marca: a última magia que deixa marca continua a que o mestre pode pôr.
	st.Markable = &QueueMark{Source: "Flecha Ácida", Who: "Arwen"}
	pit := HazardRoll{Hazard: "Fosso Camuflado", Kind: "armadilha"}
	if err := RecordRoll(st, pit); err != nil {
		t.Fatal(err)
	}
	if st.Markable.Source != "Flecha Ácida" {
		t.Errorf("o fosso trocou a marca que o mestre podia pôr por %+v", st.Markable)
	}
}

// A MARCA SEM CONJURADOR escreve a cabeça da faixa sem o "quem": " · Gás Venenoso" seria um separador solto.
func TestTheMarkWithoutCasterHeadsTheBandWithTheSource(t *testing.T) {
	roll := SpellCastRoll{Kind: RollOfTheMark, Spell: "Gás Venenoso", Moment: "inicio", Target: "Ogro"}
	if got, want := roll.Head(), "Gás Venenoso · início do turno de Ogro"; got != want {
		t.Errorf("Head() = %q; want %q", got, want)
	}
	roll.Who = "Arwen"
	if got, want := roll.Head(), "Arwen · Gás Venenoso · início do turno de Ogro"; got != want {
		t.Errorf("com conjurador, Head() = %q; want %q", got, want)
	}
}
