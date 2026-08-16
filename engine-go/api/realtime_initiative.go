package api

import (
	"context"
	"errors"
	"strings"

	socket "github.com/zishang520/socket.io/servers/socket/v3"
)

// Initiative-tracker socket handlers. All GM-gated except initiative-self (a player rolls
// their own, authorized by resolveCombatant). Split out of realtime_gateway.go for the
// 500-line file cap.

// onInitiativeAdd (GM) materializes an entry (NPC or character) and appends it.
func (g *realtimeGateway) onInitiativeAdd(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryBody, _ := ctx.body["entry"].(map[string]any)
	if entryBody == nil {
		g.wsError(sock, "entry is required")
		return
	}
	entry, err := g.materializeEntry(ctx.userID, ctx.campaignID, entryBody)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.addInitiativeEntry(ctx.sessionID, entry)
	})
}

// onInitiativeSelf lets a player roll their OWN initiative (NOT GM-gated — resolveCombatant
// enforces they own the character). Upserts by characterId so a re-roll updates in place.
func (g *realtimeGateway) onInitiativeSelf(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	if _, has := intField(ctx.body, "characterId"); !has {
		g.wsError(sock, "characterId is required")
		return
	}
	entry, err := g.materializeEntry(ctx.userID, ctx.campaignID, ctx.body)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.upsertInitiativeEntry(ctx.sessionID, entry)
	})
}

// onInitiativeUpdate (GM) patches an entry's fields.
func (g *realtimeGateway) onInitiativeUpdate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	patch := parseEntryPatch(ctx.body["patch"])
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.updateInitiativeEntry(ctx.sessionID, entryID, patch)
	})
}

// onInitiativeRemove (GM) drops an entry.
func (g *realtimeGateway) onInitiativeRemove(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.removeInitiativeEntry(ctx.sessionID, entryID)
	})
}

// onNextTurn (GM) advances to the next combatant.
func (g *realtimeGateway) onNextTurn(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.nextTurn(ctx.sessionID)
	})
}

// onResetInitiative (GM) clears the tracker.
func (g *realtimeGateway) onResetInitiative(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.reset(ctx.sessionID)
	})
}

// onPopulate (GM) pulls the campaign's player characters into the tracker, skipping any
// already present. Idempotent. Broadcasts whatever landed; after an add error it doesn't
// also ack success (the client already got an `exception`). Mirrors initiativePopulate.
func (g *realtimeGateway) onPopulate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	combatants, err := g.s.listPlayerCombatants(context.Background(), ctx.campaignID)
	if err != nil {
		g.wsError(sock, "Could not load party")
		return
	}
	state, addErr := g.populateParty(ctx.sessionID, combatants)
	if state == nil { // nothing added (all already present) → still broadcast current state
		state = g.s.sessions.getState(ctx.sessionID)
	}
	// Always rebroadcast so a client with a drifted tracker is resynced, even on a no-op.
	g.emitSessionState(ctx.sessionID, state)
	if addErr != nil {
		g.wsError(sock, addErr.Error())
		return
	}
	ackOK(ctx.ack, state)
}

// populateParty adds each not-yet-present player combatant at initiative 0 with live vitals,
// returning the latest state and the first add error (with the partial state so the caller
// can still broadcast what landed).
func (g *realtimeGateway) populateParty(sessionID int64, combatants []combatant) (*SessionRuntimeState, error) {
	existing := map[int64]bool{}
	for _, e := range g.s.sessions.getState(sessionID).Initiative {
		if e.CharacterID != nil {
			existing[*e.CharacterID] = true
		}
	}
	var state *SessionRuntimeState
	for _, c := range combatants {
		if existing[c.characterID] {
			continue
		}
		cid, hpc, hpm, mpc, mpm := c.characterID, c.hpCurrent, c.hpMax, c.mpCurrent, c.mpMax
		st, err := g.s.sessions.addInitiativeEntry(sessionID, InitiativeEntry{
			Label: c.name, Initiative: 0, Type: "character", CharacterID: &cid,
			HpCurrent: &hpc, HpMax: &hpm, MpCurrent: &mpc, MpMax: &mpm,
		})
		if err != nil {
			return state, err
		}
		state = st
	}
	return state, nil
}

// materializeEntry resolves an initiative payload into a concrete entry — an NPC (label +
// initiative) or a character (name/vitals pulled via resolveCombatant, with optional client
// overrides).
func (g *realtimeGateway) materializeEntry(callerID, campaignID int64, input map[string]any) (InitiativeEntry, error) {
	if _, hasChar := intField(input, "characterId"); !hasChar {
		return materializeNpcEntry(input)
	}
	return g.materializeCharacterEntry(callerID, campaignID, input)
}

func materializeNpcEntry(input map[string]any) (InitiativeEntry, error) {
	label := strings.TrimSpace(stringField(input, "label"))
	if label == "" {
		return InitiativeEntry{}, errors.New("entry.label is required for NPC entries")
	}
	initiative, hasInit := intField(input, "initiative")
	if !hasInit {
		return InitiativeEntry{}, errors.New("entry.initiative is required")
	}
	typ := "npc"
	if t := stringField(input, "type"); t != "" {
		typ = t
	}
	// PV rides along when the client seeds it (a monster dropped in from the
	// bestiary knows its own pool). Absent stays absent: a bare NPC has no
	// health to track, and a zeroed bar would mean something it does not.
	entry := InitiativeEntry{Label: label, Initiative: int(initiative), Type: typ}
	if hp, ok := intField(input, "hpCurrent"); ok {
		entry.HpCurrent = &hp
	}
	if hp, ok := intField(input, "hpMax"); ok {
		entry.HpMax = &hp
	}
	// O id do bestiário vem do cliente porque é ele que escolheu o verbete; o
	// servidor não valida contra o catálogo de propósito — um id desconhecido
	// vira "sem bloco" na tela, não um erro que derruba a adição no meio do
	// combate (ALE-122).
	if monsterID := strings.TrimSpace(stringField(input, "monsterId")); monsterID != "" {
		entry.MonsterID = &monsterID
	}
	return entry, nil
}

func (g *realtimeGateway) materializeCharacterEntry(callerID, campaignID int64, input map[string]any) (InitiativeEntry, error) {
	charID, _ := intField(input, "characterId")
	initiative, hasInit := intField(input, "initiative")
	if !hasInit {
		return InitiativeEntry{}, errors.New("entry.initiative is required")
	}
	stats, _, err := g.s.resolveCombatant(context.Background(), callerID, campaignID, charID)
	if err != nil {
		return InitiativeEntry{}, err
	}
	label := stats.name
	if l := strings.TrimSpace(stringField(input, "label")); l != "" {
		label = l
	}
	cid := charID
	return InitiativeEntry{
		Label: label, Initiative: int(initiative), Type: "character", CharacterID: &cid,
		HpCurrent: overrideInt(input, "hpCurrent", stats.hpCurrent),
		HpMax:     overrideInt(input, "hpMax", stats.hpMax),
		MpCurrent: overrideInt(input, "mpCurrent", stats.mpCurrent),
		MpMax:     overrideInt(input, "mpMax", stats.mpMax),
	}, nil
}

// parseEntryPatch reads an update patch from the raw body (only present fields become
// non-nil, so "leave unchanged" is distinct from "set to zero").
func parseEntryPatch(v any) entryPatch {
	m, _ := v.(map[string]any)
	p := entryPatch{}
	if m == nil {
		return p
	}
	if s, ok := m["label"].(string); ok {
		p.Label = &s
	}
	if s, ok := m["type"].(string); ok {
		p.Type = &s
	}
	if i, ok := intField(m, "initiative"); ok {
		n := int(i)
		p.Initiative = &n
	}
	for _, f := range []struct {
		key string
		dst **int64
	}{
		{"characterId", &p.CharacterID}, {"hpCurrent", &p.HpCurrent},
		{"hpMax", &p.HpMax}, {"mpCurrent", &p.MpCurrent}, {"mpMax", &p.MpMax},
	} {
		if i, ok := intField(m, f.key); ok {
			v := i
			*f.dst = &v
		}
	}
	return p
}

// overrideInt returns the body's value for key when present, else def — as a pointer.
func overrideInt(m map[string]any, key string, def int64) *int64 {
	if v, ok := intField(m, key); ok {
		return ptrInt64(v)
	}
	return ptrInt64(def)
}
