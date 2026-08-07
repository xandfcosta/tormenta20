package api

import (
	"fmt"
	"sort"
)

// INITIATIVE_MAX_ENTRIES — hard ceiling on combatants in one tracker (runaway-add guard;
// the UI paginates poorly past ~20 anyway). Mirrors session-state.service.ts.
const initiativeMaxEntries = 50

// InitiativeEntry is one combatant row in a session's initiative tracker. The optional
// numeric fields use pointers + omitempty so the JSON matches the socket.io-client shape
// (absent ⇒ undefined) that the frontend InitiativeEntry expects.
type InitiativeEntry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Initiative  int    `json:"initiative"`
	Type        string `json:"type"` // "character" | "npc"
	CharacterID *int64 `json:"characterId,omitempty"`
	HpCurrent   *int64 `json:"hpCurrent,omitempty"`
	HpMax       *int64 `json:"hpMax,omitempty"`
	MpCurrent   *int64 `json:"mpCurrent,omitempty"`
	MpMax       *int64 `json:"mpMax,omitempty"`
}

// SessionRuntimeState is the live per-session tracker: a DESC-sorted initiative list, the
// current round, and the index of the combatant on turn (-1 before combat / after reset).
// Mirrors the frontend/backend SessionRuntimeState — the shape persisted in
// Session.runtimeState and broadcast on `session-state`.
type SessionRuntimeState struct {
	Initiative []InitiativeEntry `json:"initiative"`
	Round      int               `json:"round"`
	TurnIndex  int               `json:"turnIndex"`
}

// emptyRuntimeState is a fresh mutable tracker. Each call returns a new slice so different
// sessions never share the initiative array.
func emptyRuntimeState() *SessionRuntimeState {
	return &SessionRuntimeState{Initiative: []InitiativeEntry{}, Round: 0, TurnIndex: -1}
}

// entryPatch is a partial update of an entry (Partial<Omit<InitiativeEntry,'id'>>): only
// the non-nil fields are applied. Kept separate from InitiativeEntry so "leave unchanged"
// (nil) is distinct from "set to zero".
type entryPatch struct {
	Label       *string `json:"label"`
	Initiative  *int    `json:"initiative"`
	Type        *string `json:"type"`
	CharacterID *int64  `json:"characterId"`
	HpCurrent   *int64  `json:"hpCurrent"`
	HpMax       *int64  `json:"hpMax"`
	MpCurrent   *int64  `json:"mpCurrent"`
	MpMax       *int64  `json:"mpMax"`
}

// sortInitiative keeps the list DESC by initiative, ties broken by label ascending.
// NOTE: Nest uses String.localeCompare (pt-BR collation); Go compares byte-wise here.
// The two differ only for equal-initiative entries whose labels differ by accent — a
// cosmetic tie-break — and the server stays the single source of truth for order.
func sortInitiative(st *SessionRuntimeState) {
	sort.SliceStable(st.Initiative, func(i, j int) bool {
		a, b := st.Initiative[i], st.Initiative[j]
		if a.Initiative != b.Initiative {
			return a.Initiative > b.Initiative
		}
		return a.Label < b.Label
	})
}

func findEntryIndex(st *SessionRuntimeState, entryID string) int {
	for i := range st.Initiative {
		if st.Initiative[i].ID == entryID {
			return i
		}
	}
	return -1
}

// turnEntryID returns the id of the entry currently on turn, or "" when none — so a
// re-sort can restore turnIndex to the same combatant regardless of index shuffles.
func turnEntryID(st *SessionRuntimeState) string {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return ""
	}
	return st.Initiative[st.TurnIndex].ID
}

// restoreTurn points turnIndex back at the entry that was on turn before a re-sort.
func restoreTurn(st *SessionRuntimeState, id string) {
	if id == "" {
		return
	}
	if idx := findEntryIndex(st, id); idx >= 0 {
		st.TurnIndex = idx
	}
}

// addEntry appends a combatant (a fresh id is assigned via newID), re-sorts, and preserves
// who is on turn. Errors when the tracker is full. Mirrors SessionStateService.addEntry.
func addEntry(st *SessionRuntimeState, input InitiativeEntry, newID func() string) error {
	if len(st.Initiative) >= initiativeMaxEntries {
		return fmt.Errorf("Initiative tracker is full (max %d entries)", initiativeMaxEntries)
	}
	onTurn := turnEntryID(st)
	input.ID = newID()
	st.Initiative = append(st.Initiative, input)
	sortInitiative(st)
	restoreTurn(st, onTurn)
	return nil
}

// upsertCharacterEntry adds a character's entry, or — if that character is already in the
// tracker — updates only its initiative (a re-roll), keeping mid-combat hp/mp. Preserves
// who is on turn. Mirrors SessionStateService.upsertCharacterEntry.
func upsertCharacterEntry(st *SessionRuntimeState, input InitiativeEntry, newID func() string) error {
	idx := -1
	if input.CharacterID != nil {
		for i := range st.Initiative {
			if st.Initiative[i].CharacterID != nil && *st.Initiative[i].CharacterID == *input.CharacterID {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return addEntry(st, input, newID)
	}
	onTurn := turnEntryID(st)
	st.Initiative[idx].Initiative = input.Initiative
	sortInitiative(st)
	restoreTurn(st, onTurn)
	return nil
}

// updateEntry applies a partial patch to an entry (re-sorting + preserving turn only when
// initiative changes). Errors if the entry is gone. Mirrors SessionStateService.updateEntry.
func updateEntry(st *SessionRuntimeState, entryID string, patch entryPatch) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if patch.Label != nil {
		e.Label = *patch.Label
	}
	if patch.Type != nil {
		e.Type = *patch.Type
	}
	if patch.CharacterID != nil {
		e.CharacterID = patch.CharacterID
	}
	if patch.HpCurrent != nil {
		e.HpCurrent = patch.HpCurrent
	}
	if patch.HpMax != nil {
		e.HpMax = patch.HpMax
	}
	if patch.MpCurrent != nil {
		e.MpCurrent = patch.MpCurrent
	}
	if patch.MpMax != nil {
		e.MpMax = patch.MpMax
	}
	if patch.Initiative != nil {
		e.Initiative = *patch.Initiative
		onTurn := turnEntryID(st)
		sortInitiative(st)
		restoreTurn(st, onTurn)
	}
	return nil
}

// removeEntry drops an entry and fixes turnIndex: shift left when a row before the current
// turn leaves; wrap to a new round when the row on turn was the tail. Mirrors
// SessionStateService.removeEntry.
func removeEntry(st *SessionRuntimeState, entryID string) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	st.Initiative = append(st.Initiative[:idx], st.Initiative[idx+1:]...)
	if len(st.Initiative) == 0 {
		st.TurnIndex = -1
		return nil
	}
	if idx < st.TurnIndex {
		st.TurnIndex--
	} else if idx == st.TurnIndex && st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
	return nil
}

// advanceTurn moves to the next combatant, wrapping to index 0 and bumping the round.
// From the pre-combat state (turnIndex -1) it puts the first combatant on turn without
// bumping the round. Mirrors SessionStateService.nextTurn.
func advanceTurn(st *SessionRuntimeState) {
	if len(st.Initiative) == 0 {
		return
	}
	if st.TurnIndex < 0 {
		st.TurnIndex = 0
		if st.Round < 1 {
			st.Round = 1
		}
		return
	}
	st.TurnIndex++
	if st.TurnIndex >= len(st.Initiative) {
		st.TurnIndex = 0
		st.Round++
	}
}

// resetInitiative clears the tracker but keeps the session tracked.
func resetInitiative(st *SessionRuntimeState) {
	st.Initiative = []InitiativeEntry{}
	st.Round = 0
	st.TurnIndex = -1
}

// patchEntryVitals sets absolute hp/mp on an entry, clamped to its max when present.
// Mirrors SessionStateService.patchVitals (the DB write-through lives in the store layer).
func patchEntryVitals(st *SessionRuntimeState, entryID string, hpCurrent, mpCurrent *int64) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if hpCurrent != nil {
		e.HpCurrent = ptrInt64(clampVital(*hpCurrent, e.HpMax))
	}
	if mpCurrent != nil {
		e.MpCurrent = ptrInt64(clampVital(*mpCurrent, e.MpMax))
	}
	return nil
}

// deltaEntryVitals applies an hp/mp delta ("sofreu 10 de dano" ⇒ hpDelta -10). Absent
// current counts as 0. Mirrors SessionStateService.deltaVitals.
func deltaEntryVitals(st *SessionRuntimeState, entryID string, hpDelta, mpDelta *int64) error {
	idx := findEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("Entry %s not found", entryID)
	}
	e := &st.Initiative[idx]
	if hpDelta != nil {
		e.HpCurrent = ptrInt64(clampVital(derefOr(e.HpCurrent, 0)+*hpDelta, e.HpMax))
	}
	if mpDelta != nil {
		e.MpCurrent = ptrInt64(clampVital(derefOr(e.MpCurrent, 0)+*mpDelta, e.MpMax))
	}
	return nil
}

// clampVital floors a vital at 0 and caps it at max when present. Negative floor is 0: the
// tracker never displays below 0 (a character at 0 is unconscious/dying, handled
// narratively). Mirrors the clampVital in session-state.service.ts.
func clampVital(value int64, max *int64) int64 {
	floored := value
	if floored < 0 {
		floored = 0
	}
	if max == nil {
		return floored
	}
	if floored > *max {
		return *max
	}
	return floored
}

func ptrInt64(v int64) *int64 { return &v }

func derefOr(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}
