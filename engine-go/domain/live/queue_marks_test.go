package live

import "testing"

// acidArrow é a marca que a Flecha Ácida deixa: o muco, no início dos dois próximos turnos de quem falhou (p193).
func acidArrow() *QueueMark {
	return &QueueMark{
		Spell: "Flecha Ácida", Who: "Arcanista", At: "inicio", TurnsLeft: 2, Counted: true,
		Parts: []QueueMarkPart{{Dice: "3d6", Kind: "ácido", Note: "o muco corrosivo"}},
	}
}

// combatWithAnOgre é uma cena de ação com o arcanista na vez e o ogro depois dele.
func combatWithAnOgre() *SessionRuntimeState {
	st := &SessionRuntimeState{
		Initiative: []InitiativeEntry{
			{ID: "arcanista", Label: "Arcanista", Initiative: 20, Type: "character"},
			{ID: "ogro", Label: "Ogro", Initiative: 10, Type: "npc"},
		},
		TurnIndex: 0, Round: 1,
		Scene: &Scene{Kind: SceneAction, Number: 1, StandardLeft: true, MovementLeft: true},
	}
	return st
}

func dueOf(t *testing.T, st *SessionRuntimeState) []DueRoll {
	t.Helper()
	entry := st.Initiative[st.TurnIndex]
	return DueRolls(entry, entry.Conditions)
}

// A MARCA NA FILA (ver GLOSSARY, **marca na fila**): o mestre marca o ogro com a magia da última conjuração que
// deixa marca, ela rola na vez DELE com o dado já aumentado, e a Flecha Ácida conta os dois turnos e sai sozinha.
func TestTheMarkRollsOnTheMarkedCreaturesTurnAndCountsItsTurns(t *testing.T) {
	st := combatWithAnOgre()
	if err := MarkEntry(st, "ogro"); err == nil {
		t.Fatal("marcar sem magia que deixe marca passou")
	}
	st.Markable = acidArrow()
	if err := MarkEntry(st, "ogro"); err != nil {
		t.Fatalf("marcar o ogro: %v", err)
	}
	if due := dueOf(t, st); len(due) != 0 {
		t.Fatalf("na vez do arcanista, o muco do ogro já rola: %+v", due)
	}

	for turn, want := range []int{1, 1, 0} {
		AdvanceTurn(st) // a vez do ogro
		due := dueOf(t, st)
		if len(due) != want {
			t.Fatalf("no %dº turno do ogro, %d dado(s) na vez: %+v; want %d", turn+1, len(due), due, want)
		}
		if want == 1 && (due[0].Spell != "Flecha Ácida" || due[0].Who != "Arcanista" || due[0].Parts[0].Dice != "3d6") {
			t.Errorf("no %dº turno do ogro, a vez rola %+v; want o muco de 3d6 da Flecha do Arcanista", turn+1, due[0])
		}
		AdvanceTurn(st) // a vez do arcanista
	}
	if marks := st.Initiative[1].Marks; len(marks) != 0 {
		t.Errorf("depois dos dois turnos, o ogro ainda está marcado: %+v", marks)
	}

	t.Run("o mestre tira a marca", func(t *testing.T) {
		if err := MarkEntry(st, "ogro"); err != nil {
			t.Fatalf("marcar de novo: %v", err)
		}
		if err := UnmarkEntry(st, "ogro", "Flecha Ácida"); err != nil {
			t.Fatalf("tirar a marca: %v", err)
		}
		if marks := st.Initiative[1].Marks; len(marks) != 0 {
			t.Errorf("a marca tirada ficou: %+v", marks)
		}
	})
}

// AS CHAMAS (p394): quem está Em Chamas rola 1d6 de fogo no início da vez dele, e a marca da Lança Ígnea troca esse
// dado pelo dela (p196). O Círculo da Restauração rola no FIM do turno.
func TestTheBurningConditionRollsAndTheLanceReplacesItsDie(t *testing.T) {
	ogre := InitiativeEntry{ID: "ogro", Label: "Ogro", Conditions: []string{"em-chamas"}}
	due := DueRolls(ogre, ogre.Conditions)
	if len(due) != 1 || due[0].Condition != "Em Chamas" || due[0].Spell != "" || due[0].Parts[0].Dice != "1d6" || due[0].Parts[0].Kind != "fogo" {
		t.Fatalf("o ogro Em Chamas rola %+v; want 1d6 de fogo", due)
	}

	ogre.Marks = []QueueMark{{Spell: "Lança Ígnea de Aleph", Who: "Arcanista", At: "inicio", Burning: true,
		Parts: []QueueMarkPart{{Dice: "3d6", Kind: "fogo"}}}}
	due = DueRolls(ogre, ogre.Conditions)
	if len(due) != 1 || due[0].Spell != "Lança Ígnea de Aleph" || due[0].Parts[0].Dice != "3d6" {
		t.Errorf("com as chamas da Lança, o ogro rola %+v; want só os 3d6 da Lança", due)
	}

	healed := InitiativeEntry{ID: "anao", Label: "Anão", Marks: []QueueMark{{Spell: "Círculo da Restauração", At: "fim",
		Parts: []QueueMarkPart{{Dice: "3d8+3", Kind: "cura"}}}}}
	due = DueRolls(healed, nil)
	if len(due) != 1 || due[0].At != "fim" {
		t.Errorf("o Círculo da Restauração rola %+v; want no fim do turno", due)
	}
}

// VOLTAR A VEZ E PASSAR DE NOVO NÃO GASTA OUTRO TURNO da marca: o "Turno anterior" é conserto de mesa, e o mesmo
// turno do ogro, entrado duas vezes, é um turno só.
func TestRewindingTheTurnDoesNotSpendAnotherTurnOfTheMark(t *testing.T) {
	st := combatWithAnOgre()
	st.Markable = acidArrow()
	if err := MarkEntry(st, "ogro"); err != nil {
		t.Fatalf("marcar o ogro: %v", err)
	}
	AdvanceTurn(st) // a vez do ogro: 2 → 1
	RewindTurn(st)  // volta ao arcanista
	AdvanceTurn(st) // a vez do ogro de novo, o mesmo turno
	if left := st.Initiative[1].Marks[0].TurnsLeft; left != 1 {
		t.Fatalf("depois de voltar e passar, a Flecha tem %d turno(s); want 1, o mesmo turno do ogro", left)
	}
	AdvanceTurn(st)
	AdvanceTurn(st) // o segundo turno do ogro: o último do muco
	if due := dueOf(t, st); len(due) != 1 || !due[0].Last {
		t.Errorf("no segundo turno do ogro, a vez rola %+v; want o último muco", due)
	}
}
