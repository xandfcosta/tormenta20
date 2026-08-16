package api

import "testing"

// counter returns a deterministic id generator ("e1", "e2", …) so tests can assert on
// order/turn behavior without random UUIDs.
func counter() func() string {
	n := 0
	return func() string {
		n++
		return "e" + itoa(n)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func npc(label string, init int) InitiativeEntry {
	return InitiativeEntry{Label: label, Initiative: init, Type: "npc"}
}

func charEntry(label string, init int, charID int64) InitiativeEntry {
	c := charID
	return InitiativeEntry{Label: label, Initiative: init, Type: "character", CharacterID: &c}
}

func labels(st *SessionRuntimeState) []string {
	out := make([]string, len(st.Initiative))
	for i, e := range st.Initiative {
		out[i] = e.Label
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAddEntrySortsAndAssignsID(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	for _, e := range []InitiativeEntry{npc("A", 10), npc("B", 20), npc("C", 5)} {
		if err := addEntry(st, e, id); err != nil {
			t.Fatalf("addEntry: %v", err)
		}
	}
	if got := labels(st); !eq(got, []string{"B", "A", "C"}) {
		t.Errorf("order=%v, want [B A C] (DESC by initiative)", got)
	}
	if st.Initiative[0].ID == "" {
		t.Error("id not assigned")
	}
}

func TestAddEntryTieBreakByLabel(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("Zé", 10), id)
	_ = addEntry(st, npc("Ana", 10), id)
	if got := labels(st); !eq(got, []string{"Ana", "Zé"}) {
		t.Errorf("order=%v, want [Ana Zé] (tie broken by label asc)", got)
	}
}

func TestAddEntryTieBreakCollation(t *testing.T) {
	// Accent-aware pt-BR collation: "Ávila" sorts with A (before "Bravo"). A byte
	// compare would do the opposite (Á's first byte 0xC3 > 'B'), so this proves the
	// collator is in effect. Same initiative → the tie-break decides.
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("Bravo", 10), id)
	_ = addEntry(st, npc("Ávila", 10), id)
	if got := labels(st); !eq(got, []string{"Ávila", "Bravo"}) {
		t.Errorf("order=%v, want [Ávila Bravo] (pt-BR collation, not byte order)", got)
	}
}

func TestAddEntryFull(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	for i := 0; i < initiativeMaxEntries; i++ {
		if err := addEntry(st, npc("x", i), id); err != nil {
			t.Fatalf("unexpected full at %d: %v", i, err)
		}
	}
	if err := addEntry(st, npc("overflow", 1), id); err == nil {
		t.Error("expected full-tracker error at 51st entry")
	}
}

func TestAddEntryPreservesTurnAcrossSort(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("A", 10), id) // e1
	_ = addEntry(st, npc("B", 20), id) // e2
	_ = addEntry(st, npc("C", 5), id)  // e3 → order B,A,C
	st.TurnIndex = 1                   // A is on turn
	onTurnID := st.Initiative[1].ID
	_ = addEntry(st, npc("D", 25), id) // order D,B,A,C
	if st.Initiative[st.TurnIndex].ID != onTurnID || labels(st)[st.TurnIndex] != "A" {
		t.Errorf("turn should stay on A, got index=%d labels=%v", st.TurnIndex, labels(st))
	}
}

func TestUpsertCharacterEntry(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	hp := int64(30)
	entry := charEntry("Herói", 12, 7)
	entry.HpCurrent = &hp
	_ = upsertCharacterEntry(st, entry, id)

	// Re-roll: same characterId, new initiative — keeps hp, updates initiative only.
	if err := upsertCharacterEntry(st, charEntry("Herói", 3, 7), id); err != nil {
		t.Fatalf("re-roll: %v", err)
	}
	if len(st.Initiative) != 1 {
		t.Fatalf("re-roll should upsert not add, len=%d", len(st.Initiative))
	}
	e := st.Initiative[0]
	if e.Initiative != 3 {
		t.Errorf("initiative=%d, want 3 (updated)", e.Initiative)
	}
	if e.HpCurrent == nil || *e.HpCurrent != 30 {
		t.Errorf("hp should be preserved at 30, got %v", e.HpCurrent)
	}

	// A different character is added, not merged.
	_ = upsertCharacterEntry(st, charEntry("Outro", 8, 9), id)
	if len(st.Initiative) != 2 {
		t.Errorf("different character should add, len=%d", len(st.Initiative))
	}
}

func TestUpdateEntry(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("A", 10), id)
	_ = addEntry(st, npc("B", 20), id) // order B,A
	aID := st.Initiative[1].ID

	// Patch initiative → A jumps above B, re-sorted.
	newInit := 30
	if err := updateEntry(st, aID, entryPatch{Initiative: &newInit}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !eq(labels(st), []string{"A", "B"}) {
		t.Errorf("after bump, order=%v want [A B]", labels(st))
	}

	// Patch label only → no re-sort.
	lbl := "Alpha"
	_ = updateEntry(st, aID, entryPatch{Label: &lbl})
	if st.Initiative[0].Label != "Alpha" {
		t.Errorf("label not patched: %v", st.Initiative[0].Label)
	}

	if err := updateEntry(st, "missing", entryPatch{Label: &lbl}); err == nil {
		t.Error("expected not-found error")
	}
}

func TestRemoveEntryTurnBookkeeping(t *testing.T) {
	build := func() (*SessionRuntimeState, []string) {
		st := emptyRuntimeState()
		id := counter()
		_ = addEntry(st, npc("C", 30), id)
		_ = addEntry(st, npc("B", 20), id)
		_ = addEntry(st, npc("A", 10), id) // order C,B,A
		ids := []string{st.Initiative[0].ID, st.Initiative[1].ID, st.Initiative[2].ID}
		return st, ids
	}

	t.Run("remove before turn shifts turnIndex left", func(t *testing.T) {
		st, ids := build()
		st.TurnIndex = 2 // A on turn
		_ = removeEntry(st, ids[0])
		if st.TurnIndex != 1 || st.Initiative[st.TurnIndex].Label != "A" {
			t.Errorf("turnIndex=%d labels=%v, want A at 1", st.TurnIndex, labels(st))
		}
	})
	t.Run("remove the tail on turn wraps round", func(t *testing.T) {
		st, ids := build()
		st.TurnIndex, st.Round = 2, 1 // A (tail) on turn
		_ = removeEntry(st, ids[2])
		if st.TurnIndex != 0 || st.Round != 2 {
			t.Errorf("turnIndex=%d round=%d, want 0/2", st.TurnIndex, st.Round)
		}
	})
	t.Run("remove after turn leaves turnIndex", func(t *testing.T) {
		st, ids := build()
		st.TurnIndex = 0 // C on turn
		_ = removeEntry(st, ids[2])
		if st.TurnIndex != 0 {
			t.Errorf("turnIndex=%d, want 0", st.TurnIndex)
		}
	})
	t.Run("remove last entry resets turn to -1", func(t *testing.T) {
		st := emptyRuntimeState()
		_ = addEntry(st, npc("solo", 1), counter())
		st.TurnIndex = 0
		_ = removeEntry(st, st.Initiative[0].ID)
		if st.TurnIndex != -1 {
			t.Errorf("turnIndex=%d, want -1", st.TurnIndex)
		}
	})
}

func TestAdvanceTurn(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("A", 30), id)
	_ = addEntry(st, npc("B", 20), id) // order A,B

	advanceTurn(st) // from -1 → first, round becomes 1
	if st.TurnIndex != 0 || st.Round != 1 {
		t.Fatalf("first advance: turnIndex=%d round=%d, want 0/1", st.TurnIndex, st.Round)
	}
	advanceTurn(st) // → index 1
	if st.TurnIndex != 1 || st.Round != 1 {
		t.Fatalf("second: turnIndex=%d round=%d, want 1/1", st.TurnIndex, st.Round)
	}
	advanceTurn(st) // wrap → index 0, round 2
	if st.TurnIndex != 0 || st.Round != 2 {
		t.Fatalf("wrap: turnIndex=%d round=%d, want 0/2", st.TurnIndex, st.Round)
	}

	empty := emptyRuntimeState()
	advanceTurn(empty)
	if empty.TurnIndex != -1 {
		t.Errorf("advance on empty must be a no-op, got %d", empty.TurnIndex)
	}
}

// O turno ANTERIOR: um "Próximo turno" a mais é o erro mais comum da mesa, e
// hoje o único conserto é dar a volta na iniciativa inteira — passando por todo
// mundo de novo, o que também empurra a rodada. Voltar tem de desfazer, e isso
// inclui a RODADA quando o passo cruza a virada.
func TestRewindTurn(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("A", 30), id)
	_ = addEntry(st, npc("B", 20), id)

	advanceTurn(st) // A, rodada 1
	advanceTurn(st) // B
	rewindTurn(st)
	if st.TurnIndex != 0 || st.Round != 1 {
		t.Fatalf("voltar um: turnIndex=%d round=%d, queria 0/1", st.TurnIndex, st.Round)
	}

	advanceTurn(st) // B
	advanceTurn(st) // volta para A, rodada 2
	if st.TurnIndex != 0 || st.Round != 2 {
		t.Fatalf("preparo: turnIndex=%d round=%d, queria 0/2", st.TurnIndex, st.Round)
	}
	rewindTurn(st) // cruza a virada de volta: último da rodada 1
	if st.TurnIndex != 1 || st.Round != 1 {
		t.Errorf("voltar cruzando a virada: turnIndex=%d round=%d, queria 1/1", st.TurnIndex, st.Round)
	}

	// Antes do combate começar não há o que desfazer, e a rodada não pode ir a 0.
	inicio := emptyRuntimeState()
	_ = addEntry(inicio, npc("A", 30), counter())
	rewindTurn(inicio)
	if inicio.TurnIndex != -1 || inicio.Round != 0 {
		t.Errorf("voltar antes do primeiro turno: turnIndex=%d round=%d, queria -1/0", inicio.TurnIndex, inicio.Round)
	}

	advanceTurn(inicio) // primeiro turno, rodada 1
	rewindTurn(inicio)  // desfaz o primeiro: volta ao pré-combate
	if inicio.TurnIndex != -1 || inicio.Round != 1 {
		t.Errorf("desfazer o primeiro turno: turnIndex=%d round=%d, queria -1/1", inicio.TurnIndex, inicio.Round)
	}

	vazio := emptyRuntimeState()
	rewindTurn(vazio)
	if vazio.TurnIndex != -1 {
		t.Errorf("voltar sem combatente é no-op, got %d", vazio.TurnIndex)
	}
}

func TestResetInitiative(t *testing.T) {
	st := emptyRuntimeState()
	_ = addEntry(st, npc("A", 1), counter())
	st.TurnIndex, st.Round = 0, 3
	resetInitiative(st)
	if len(st.Initiative) != 0 || st.Round != 0 || st.TurnIndex != -1 {
		t.Errorf("reset gave %+v, want empty/0/-1", st)
	}
}

func TestPatchAndDeltaVitals(t *testing.T) {
	mk := func() (*SessionRuntimeState, string) {
		st := emptyRuntimeState()
		e := npc("A", 10)
		hp, hpMax, mp, mpMax := int64(8), int64(10), int64(2), int64(6)
		e.HpCurrent, e.HpMax, e.MpCurrent, e.MpMax = &hp, &hpMax, &mp, &mpMax
		st.Initiative = []InitiativeEntry{e}
		st.Initiative[0].ID = "e1"
		return st, "e1"
	}

	t.Run("patch clamps to max and floor", func(t *testing.T) {
		st, id := mk()
		over := int64(99)
		_ = patchEntryVitals(st, id, &over, nil)
		if *st.Initiative[0].HpCurrent != 10 {
			t.Errorf("hp=%d, want 10 (clamped to max)", *st.Initiative[0].HpCurrent)
		}
		neg := int64(-5)
		_ = patchEntryVitals(st, id, &neg, nil)
		if *st.Initiative[0].HpCurrent != 0 {
			t.Errorf("hp=%d, want 0 (floored)", *st.Initiative[0].HpCurrent)
		}
	})
	t.Run("delta from current, clamped", func(t *testing.T) {
		st, id := mk()
		d := int64(-20)
		_ = deltaEntryVitals(st, id, &d, nil)
		if *st.Initiative[0].HpCurrent != 0 {
			t.Errorf("hp=%d, want 0", *st.Initiative[0].HpCurrent)
		}
		up := int64(100)
		_ = deltaEntryVitals(st, id, nil, &up)
		if *st.Initiative[0].MpCurrent != 6 {
			t.Errorf("mp=%d, want 6 (clamped)", *st.Initiative[0].MpCurrent)
		}
	})
	t.Run("delta treats absent current as 0", func(t *testing.T) {
		st := emptyRuntimeState()
		e := npc("A", 10)
		hpMax := int64(10)
		e.HpMax = &hpMax // HpCurrent nil
		st.Initiative = []InitiativeEntry{e}
		st.Initiative[0].ID = "e1"
		d := int64(3)
		_ = deltaEntryVitals(st, "e1", &d, nil)
		if st.Initiative[0].HpCurrent == nil || *st.Initiative[0].HpCurrent != 3 {
			t.Errorf("hp=%v, want 3 (0 + 3)", st.Initiative[0].HpCurrent)
		}
	})
	t.Run("not found", func(t *testing.T) {
		st, _ := mk()
		v := int64(1)
		if err := patchEntryVitals(st, "missing", &v, nil); err == nil {
			t.Error("expected not-found error")
		}
	})
}
