package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"t20engine/catalog"
)

// As regras da iniciativa e do descanso, fora de qualquer transporte.
//
// Este arquivo é o que SOBROU de `realtime_initiative.go` quando o socket.io foi
// apagado (ALE-253). O corte foi pelo receptor: o que era `(g *realtimeGateway)`
// era transporte e morreu junto; o que está aqui é aplicação, e não mudou uma
// linha ao mudar de vizinho. As rotas HTTP em `session_commands.go` e
// `board_commands.go` chamam exatamente as mesmas funções que os eventos
// chamavam.

// selfInitiativeEntry monta a linha de quem registra a PRÓPRIA iniciativa:
// confere o d20, pergunta o bônus ao motor e soma.
//
// Transport-agnostic de propósito, como o `assertVitalsEditable`: é aqui que a
// regra mora e é aqui que ela se prova, com o handler em volta traduzindo erro
// em `exception`. Testar pelo socket exigiria um socket, e o que importa não é
// o transporte.
//
// O RECEPTOR virou `*Server` na ALE-219, e essa é a frase que faz a de cima
// deixar de ser aspiração: enquanto a regra pendia do gateway do socket,
// "transport-agnostic" era uma intenção escrita em comentário e desmentida pela
// assinatura — o segundo transporte (a página da Mesa em Datastar) não
// conseguia alcançá-la sem um socket que ele não tem.
//
// A ALE-253 provou o argumento por outro caminho: o socket saiu do projeto
// inteiro e esta função não mudou uma linha. (Reposto aqui na oitava puxada da
// main para a base da migração — o comentário se perdeu quando a função mudou
// de arquivo, e sem ele o `*Server` parece escolha de estilo.)
func (s *Server) selfInitiativeEntry(callerID, campaignID, charID, d20 int64) (InitiativeEntry, error) {
	if d20 < 1 || d20 > 20 {
		return InitiativeEntry{}, fmt.Errorf("d20 must be an integer from 1 to 20, got %d", d20)
	}
	bonus, err := s.initiativeBonus(context.Background(), charID)
	if err != nil {
		return InitiativeEntry{}, err
	}
	// Um payload NOVO e não o corpo recebido: escrever no mapa do cliente faria a
	// mensagem se reescrever a si mesma, e um `initiative` que ele tenha mandado
	// junto venceria a conta do servidor.
	return s.materializeEntry(context.Background(), callerID, campaignID, map[string]any{
		"characterId": charID, "initiative": d20 + bonus,
	})
}

// endSceneForTable é o gesto "Encerrar cena" INTEIRO, sem socket: a duração
// "cena" acaba para o grupo E a fila volta ao começo.
//
// A ordem importa e a recusa também. A expiração vem ANTES porque o estado
// desligado é o que a mesa vê: desligar a cena e só então falhar deixaria o
// mestre com a fila zerada e as bênçãos vivas — o defeito da ALE-220 outra vez,
// agora com o botão parecendo ter funcionado. Falha aqui é falha do gesto
// inteiro, e o mestre clica de novo.
func (s *Server) endSceneForTable(user AuthUser, campaignID, sessionID int64) (*SessionRuntimeState, error) {
	if _, _, err := s.expirePartyScene(user, campaignID, sessionID); err != nil {
		return nil, errors.New("Could not load campaign members")
	}
	return s.sessions.endScene(sessionID)
}

// populateParty adds each not-yet-present player combatant at initiative 0 with live vitals,
// returning the latest state and the first add error (with the partial state so the caller
// can still broadcast what landed).
func (s *Server) populateParty(sessionID int64, combatants []combatant) (*SessionRuntimeState, error) {
	existing := map[int64]bool{}
	for _, e := range s.sessions.getState(sessionID).Initiative {
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
		st, err := s.sessions.addInitiativeEntry(sessionID, InitiativeEntry{
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
func (s *Server) materializeEntry(ctx context.Context, callerID, campaignID int64, input map[string]any) (InitiativeEntry, error) {
	if _, hasChar := intField(input, "characterId"); !hasChar {
		return materializeNpcEntry(input)
	}
	return s.materializeCharacterEntry(ctx, callerID, campaignID, input)
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

func (s *Server) materializeCharacterEntry(ctx context.Context, callerID, campaignID int64, input map[string]any) (InitiativeEntry, error) {
	charID, _ := intField(input, "characterId")
	initiative, hasInit := intField(input, "initiative")
	if !hasInit {
		return InitiativeEntry{}, errors.New("entry.initiative is required")
	}
	stats, _, err := s.resolveCombatant(ctx, callerID, campaignID, charID)
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
