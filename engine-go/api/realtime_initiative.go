package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	socket "github.com/zishang520/socket.io/servers/socket/v3"

	"t20engine/catalog"
)

// Initiative-tracker socket handlers. All GM-gated except initiative-self (a player
// registers their own, authorized by resolveCombatant). Split out of realtime_gateway.go
// for the 500-line file cap.

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

// onInitiativeSelf lets a player register their OWN initiative (NOT GM-gated —
// resolveCombatant enforces they own the character). Upserts by characterId so um
// re-registro atualiza no lugar.
//
// O cliente manda o D20 e o SERVIDOR soma (ALE-213). Antes ele mandava o total
// já somado, e isso punha uma regra do livro na tela: quem decidisse o bônus da
// perícia Iniciativa seria o navegador, livre para divergir do motor — que é
// exatamente a segunda implementação que a ALE-104 apagou. O d20 continua vindo
// de fora, e vem de propósito: a mesa que rola dado FÍSICO digita o número, e
// nesse caminho não existe dado para o servidor rolar.
func (g *realtimeGateway) onInitiativeSelf(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	charID, hasChar := intField(ctx.body, "characterId")
	if !hasChar {
		g.wsError(sock, "characterId is required")
		return
	}
	d20, _ := intField(ctx.body, "d20")
	entry, err := g.selfInitiativeEntry(ctx.userID, ctx.campaignID, charID, d20)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.upsertInitiativeEntry(ctx.sessionID, entry)
	})
}

// selfInitiativeEntry monta a linha de quem registra a PRÓPRIA iniciativa:
// confere o d20, pergunta o bônus ao motor e soma.
//
// Transport-agnostic de propósito, como o `assertVitalsEditable`: é aqui que a
// regra mora e é aqui que ela se prova, com o handler em volta traduzindo erro
// em `exception`. Testar pelo socket exigiria um socket, e o que importa não é
// o transporte.
func (g *realtimeGateway) selfInitiativeEntry(callerID, campaignID, charID, d20 int64) (InitiativeEntry, error) {
	if d20 < 1 || d20 > 20 {
		return InitiativeEntry{}, fmt.Errorf("d20 must be an integer from 1 to 20, got %d", d20)
	}
	bonus, err := g.s.initiativeBonus(context.Background(), charID)
	if err != nil {
		return InitiativeEntry{}, err
	}
	// Um payload NOVO e não o corpo recebido: escrever no mapa do cliente faria a
	// mensagem se reescrever a si mesma, e um `initiative` que ele tenha mandado
	// junto venceria a conta do servidor.
	return g.materializeEntry(callerID, campaignID, map[string]any{
		"characterId": charID, "initiative": d20 + bonus,
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

// onPreviousTurn (GM) desfaz um turno — inclusive a virada de rodada.
func (g *realtimeGateway) onPreviousTurn(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.previousTurn(ctx.sessionID)
	})
}

// onSceneStart (GM) liga a cena (ALE-210). É o gesto que abre a fila para a
// mesa: até aqui `redactForPlayers` entregava rastreador vazio aos jogadores.
func (g *realtimeGateway) onSceneStart(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.startScene(ctx.sessionID)
	})
}

// onSceneEnd (GM) desliga a cena, guardando a fila para a próxima — e expira a
// duração "cena" das fichas do grupo (ALE-220).
//
// Não usa o `mutateAndBroadcast`: além do estado, a mesa precisa do aviso de
// que as FICHAS mudaram. Elas não estão no estado do rastreador, então sem ele
// o efeito morto e o "usado 1/cena" ficariam na tela até alguém recarregar.
// O aviso é o `session-rest` que a Recuperação já emite: no fio esse evento
// significa "o servidor expirou o escopo X do grupo", que é exatamente o que
// aconteceu aqui.
func (g *realtimeGateway) onSceneEnd(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	state, err := g.endSceneForTable(sockData(sock).user, ctx.campaignID, ctx.sessionID)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.emitSessionState(ctx.sessionID, state)
	g.io.To(socket.Room(sessionRoomName(ctx.sessionID))).
		Emit("session-rest", map[string]any{"sessionId": ctx.sessionID, "scope": "scene"})
	ackOK(ctx.ack, stateForRole(ctx.role, state))
}

// endSceneForTable é o gesto "Encerrar cena" INTEIRO, sem socket: a duração
// "cena" acaba para o grupo E a fila volta ao começo.
//
// A ordem importa e a recusa também. A expiração vem ANTES porque o estado
// desligado é o que a mesa vê: desligar a cena e só então falhar deixaria o
// mestre com a fila zerada e as bênçãos vivas — o defeito da ALE-220 outra vez,
// agora com o botão parecendo ter funcionado. Falha aqui é falha do gesto
// inteiro, e o mestre clica de novo.
func (g *realtimeGateway) endSceneForTable(user AuthUser, campaignID, sessionID int64) (*SessionRuntimeState, error) {
	if _, _, err := g.expirePartyScene(user, campaignID, sessionID); err != nil {
		return nil, errors.New("Could not load campaign members")
	}
	return g.s.sessions.endScene(sessionID)
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
	ackOK(ctx.ack, stateForRole(ctx.role, state))
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
	// O bloco de criatura do mestre (ALE-137). Mesma escolha do `monsterId`: o
	// servidor não confere se a criatura existe, porque um id órfão vira "sem
	// bloco" na tela e não um erro no meio do combate. Quem confere o dono é a
	// rota HTTP que serve o bloco, e ela só responde ao mestre.
	if creatureID, ok := intField(input, "creatureId"); ok && creatureID > 0 {
		entry.CreatureID = &creatureID
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
	if b, ok := m["hpHidden"].(bool); ok {
		p.HpHidden = &b
	}
	for _, f := range []struct {
		key string
		dst **int64
	}{
		{"characterId", &p.CharacterID}, {"hpCurrent", &p.HpCurrent},
		{"hpMax", &p.HpMax}, {"mpCurrent", &p.MpCurrent}, {"mpMax", &p.MpMax},
		// Sem esta linha o cliente manda `creatureId` e o servidor DESCARTA em
		// silêncio, com tudo compilando: campo novo na struct não entra sozinho
		// numa lista escrita à mão. Foi o segundo caso do mesmo mecanismo no
		// mesmo dia — o outro era o `cloneState` zerando o contador de turnos.
		{"creatureId", &p.CreatureID},
	} {
		if i, ok := intField(m, f.key); ok {
			v := i
			*f.dst = &v
		}
	}
	if raw, ok := m["conditions"]; ok {
		list := parseConditions(raw)
		p.Conditions = &list
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

// parseConditions filtra pelo CATÁLOGO, que é onde as condições são autoradas.
// Uma lista escrita à mão aqui seria a segunda cópia da tabela do livro, e a
// primeira já desviou uma vez: faltava `enfeitiçado`, e aplicá-la dava 400
// (ALE-122). Id desconhecido é descartado em silêncio de propósito — a
// alternativa seria derrubar a aplicação inteira no meio do combate por causa
// de um item.
func parseConditions(raw any) []string {
	items, _ := raw.([]any)
	out := []string{}
	seen := map[string]bool{}
	for _, item := range items {
		id, _ := item.(string)
		if id == "" || seen[id] || !catalog.IsCondition(id) {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
