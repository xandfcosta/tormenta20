package api

import (
	"context"
	"errors"
	"log"

	socket "github.com/zishang520/socket.io/servers/socket/v3"
)

// Vitals, session-rest and apply-effect socket handlers. Split out of realtime_gateway.go
// for the 500-line file cap.

// onVitalsPatch sets absolute hp/mp on an entry. The GM edits anyone; a player only their
// own character; NPC vitals are GM-only (assertVitalsEditable). Mirrors vitalsPatch.
func (g *realtimeGateway) onVitalsPatch(sock *socket.Socket, args []any) {
	ctx, entryID, ok := g.vitalsAccess(sock, args)
	if !ok {
		return
	}
	patch, _ := ctx.body["patch"].(map[string]any)
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.patchVitals(ctx.sessionID, entryID, optInt(patch, "hpCurrent"), optInt(patch, "mpCurrent"))
	})
}

// onVitalsDelta applies an hp/mp delta to an entry (same authorization as patch).
func (g *realtimeGateway) onVitalsDelta(sock *socket.Socket, args []any) {
	ctx, entryID, ok := g.vitalsAccess(sock, args)
	if !ok {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.deltaVitals(ctx.sessionID, entryID, optInt(ctx.body, "hpDelta"), optInt(ctx.body, "mpDelta"))
	})
}

// vitalsAccess resolves session access, reads the entryId, and authorizes the vitals edit —
// the shared preamble of vitals-patch / vitals-delta. Returns ok=false after emitting.
func (g *realtimeGateway) vitalsAccess(sock *socket.Socket, args []any) (msgCtx, string, bool) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return msgCtx{}, "", false
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return msgCtx{}, "", false
	}
	if err := g.assertVitalsEditable(ctx, entryID); err != nil {
		g.wsError(sock, err.Error())
		return msgCtx{}, "", false
	}
	return ctx, entryID, true
}

// assertVitalsEditable authorizes a vitals mutation: the GM edits any combatant; a player
// only their own character; NPC entries are GM-only. Mirrors assertVitalsEditable.
func (g *realtimeGateway) assertVitalsEditable(ctx msgCtx, entryID string) error {
	if ctx.role == "gm" {
		return nil
	}
	state := g.s.sessions.getState(ctx.sessionID)
	idx := findEntryIndex(state, entryID)
	if idx < 0 {
		return errors.New("Entry " + entryID + " not found")
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		return errors.New("Only the GM can edit NPC vitals")
	}
	_, err := g.s.assertCharacterOwner(context.Background(), ctx.userID, *entry.CharacterID)
	return err
}

// onSessionRest (GM) rests every member character: end-scene expires scene effects; end-day
// also expires day effects AND restores PV/PM, mirrored onto the live tracker. Broadcasts
// `session-rest` + (if anyone healed) `session-state`. Mirrors sessionRest.
func (g *realtimeGateway) onSessionRest(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	scope := stringField(ctx.body, "scope")
	condition := stringField(ctx.body, "condition")
	if condition == "" {
		condition = "normal"
	}
	done, total, err := g.restParty(sockData(sock).user, ctx.campaignID, ctx.sessionID, scope, condition)
	if err != nil {
		g.wsError(sock, "Could not load campaign members")
		return
	}
	if done > 0 {
		g.emitSessionState(ctx.sessionID, g.s.sessions.getState(ctx.sessionID))
	}
	g.io.To(socket.Room(sessionRoomName(ctx.sessionID))).Emit("session-rest", map[string]any{
		"sessionId": ctx.sessionID, "scope": scope, "condition": condition,
	})
	// `healed` continua no ack pelo nome antigo (o cliente o lê), mas agora ele
	// conta os dois escopos: encerrar cena que falha deixa de somar, e o mestre
	// consegue ver "3 de 5" em vez de um "descansou" que não é verdade inteira.
	ackOK(ctx.ack, map[string]any{"rested": scope, "characters": total, "healed": done})
}

// restParty aplica o descanso a cada personagem do grupo (encerrar cena, ou
// encerrar dia + curar + espelhar) e devolve quantos DERAM CERTO e o total.
//
// Best-effort por personagem de propósito — uma ficha que falha não pode
// impedir o descanso das outras quatro. Mas o resultado é CONTADO e volta no
// ack (ALE-155): antes, o encerrar-cena era `_, _ =` e nem entrava na conta, de
// modo que o mestre lia "descansou" enquanto duas de cinco fichas não tinham
// descansado. Best-effort é sobre continuar apesar da falha, não sobre esconder
// que ela houve.
func (g *realtimeGateway) restParty(user AuthUser, campaignID, sessionID int64, scope, condition string) (done, total int, err error) {
	charIDs, err := g.s.listMemberCharacterIds(context.Background(), campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, cid := range charIDs {
		if scope != "day" {
			if _, e := g.s.endScene(context.Background(), user, cid); e != nil {
				log.Printf("session %d: encerrar cena do personagem %d falhou (%v)", sessionID, cid, e)
				continue
			}
			done++
			continue
		}
		if _, e := g.s.endDay(context.Background(), user, cid); e != nil {
			log.Printf("session %d: encerrar dia do personagem %d falhou (%v)", sessionID, cid, e)
			continue
		}
		vitals, _, e := g.s.restVitals(context.Background(), user, cid, condition)
		if e != nil {
			log.Printf("session %d: descanso do personagem %d falhou (%v)", sessionID, cid, e)
			continue
		}
		g.mirrorVitalsToTracker(sessionID, cid, vitals)
		done++
	}
	return done, len(charIDs), nil
}

// mirrorVitalsToTracker copies freshly-persisted PV/PM onto the matching live tracker entry
// (if the character is in the current initiative) so bars update without a reload.
func (g *realtimeGateway) mirrorVitalsToTracker(sessionID, characterID int64, vitals restedVitals) {
	for _, e := range g.s.sessions.getState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			hp, mp := vitals.hpCurrent, vitals.mpCurrent
			_, _ = g.s.sessions.patchVitals(sessionID, e.ID, &hp, &mp)
			return
		}
	}
}

// onApplyEffect (GM) applies a spell buff to a combatant's character and notifies the room
// so clients holding that sheet refetch (activeEffects aren't in tracker state). Does NOT
// broadcast session-state. Mirrors applyEffect.
func (g *realtimeGateway) onApplyEffect(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	spellID := stringField(ctx.body, "spellId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	if spellID == "" {
		g.wsError(sock, "spellId is required")
		return
	}
	characterID, ok := g.characterEntryID(sock, ctx.sessionID, entryID)
	if !ok {
		return
	}
	var scope *string
	if s := stringField(ctx.body, "scope"); s != "" {
		scope = &s
	}
	if _, _, err := g.s.applySpellBuffEffect(context.Background(), characterID, spellID, scope); err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.io.To(socket.Room(sessionRoomName(ctx.sessionID))).Emit("effect-applied", map[string]any{
		"sessionId": ctx.sessionID, "characterId": characterID, "spellId": spellID,
	})
	ackOK(ctx.ack, map[string]any{"applied": spellID, "characterId": characterID})
}

// characterEntryID resolves an entry id to its characterId, rejecting a missing entry or an
// NPC entry (which can't receive spell effects).
func (g *realtimeGateway) characterEntryID(sock *socket.Socket, sessionID int64, entryID string) (int64, bool) {
	state := g.s.sessions.getState(sessionID)
	idx := findEntryIndex(state, entryID)
	if idx < 0 {
		g.wsError(sock, "Entry "+entryID+" not found")
		return 0, false
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		g.wsError(sock, "Only character entries can receive spell effects")
		return 0, false
	}
	return *entry.CharacterID, true
}
