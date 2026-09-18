package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"t20engine/domain/catalog"
	"t20engine/domain/live"
	"t20engine/infra/wire"
)

// As regras da iniciativa e do descanso, fora de qualquer transporte: quem as
// chama é a cena da Mesa, e elas não sabem por onde o pedido entrou.

// selfInitiativeEntry monta a linha de quem registra a PRÓPRIA iniciativa:
// confere o d20, pergunta o bônus ao motor e soma.
//
// O RECEPTOR é `*Server` e não o gateway de um transporte, e isso não é estilo:
// pendurada num transporte, a regra fica inalcançável para o segundo — e é aqui
// que ela mora e se prova, com quem chama traduzindo o erro para o formato da
// porta dele.
func (tr tableRules) selfInitiativeEntry(callerID, campaignID, charID, d20 int64) (live.InitiativeEntry, error) {
	if d20 < 1 || d20 > 20 {
		return live.InitiativeEntry{}, fmt.Errorf("d20 must be an integer from 1 to 20, got %d", d20)
	}
	bonus, err := tr.initiativeBonus(context.Background(), charID)
	if err != nil {
		return live.InitiativeEntry{}, err
	}
	// Um payload NOVO e não o corpo recebido: escrever no mapa do cliente faria a
	// mensagem se reescrever a si mesma, e um `initiative` que ele tenha mandado
	// junto venceria a conta do servidor.
	return tr.materializeEntry(context.Background(), callerID, campaignID, map[string]any{
		"characterId": charID, "initiative": d20 + bonus,
	})
}

// populateParty põe na fila, com iniciativa 0 e vitais vivos, cada combatente de
// jogador que ainda não está lá. Devolve o estado mais recente e o PRIMEIRO erro
// de `Add` junto com o estado parcial, para quem chama poder transmitir o que
// pousou.
func (tr tableRules) populateParty(sessionID int64, combatants []combatant) (*live.SessionRuntimeState, error) {
	existing := map[int64]bool{}
	for _, e := range tr.sessions.GetState(sessionID).Initiative {
		if e.CharacterID != nil {
			existing[*e.CharacterID] = true
		}
	}
	var state *live.SessionRuntimeState
	for _, c := range combatants {
		if existing[c.characterID] {
			continue
		}
		cid, hpc, hpm, mpc, mpm := c.characterID, c.hpCurrent, c.hpMax, c.mpCurrent, c.mpMax
		st, err := tr.sessions.AddInitiativeEntry(sessionID, live.InitiativeEntry{
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

// materializeEntry resolve um pedido de iniciativa numa linha concreta — um NPC
// (rótulo mais iniciativa) ou um personagem (nome e vitais buscados pelo
// `resolveCombatant`, com sobreposições opcionais do cliente).
func (tr tableRules) materializeEntry(ctx context.Context, callerID, campaignID int64, input map[string]any) (live.InitiativeEntry, error) {
	if _, hasChar := wire.IntField(input, "characterId"); !hasChar {
		return materializeNpcEntry(input)
	}
	return tr.materializeCharacterEntry(ctx, callerID, campaignID, input)
}

func materializeNpcEntry(input map[string]any) (live.InitiativeEntry, error) {
	label := strings.TrimSpace(wire.StringField(input, "label"))
	if label == "" {
		return live.InitiativeEntry{}, errors.New("entry.label is required for NPC entries")
	}
	initiative, hasInit := wire.IntField(input, "initiative")
	if !hasInit {
		return live.InitiativeEntry{}, errors.New("entry.initiative is required")
	}
	typ := "npc"
	if t := wire.StringField(input, "type"); t != "" {
		typ = t
	}
	// O PV vai junto quando o cliente o semeia (um monstro vindo do bestiário
	// sabe o próprio pool). Ausente continua ausente: um NPC pelado não tem vida
	// a acompanhar, e uma barra zerada diria algo que não é o caso.
	entry := live.InitiativeEntry{Label: label, Initiative: int(initiative), Type: typ}
	if hp, ok := wire.IntField(input, "hpCurrent"); ok {
		entry.HpCurrent = &hp
	}
	if hp, ok := wire.IntField(input, "hpMax"); ok {
		entry.HpMax = &hp
	}
	// O id do bestiário vem do cliente porque é ele que escolheu o verbete; o
	// servidor não valida contra o catálogo de propósito — um id desconhecido
	// vira "sem bloco" na tela, não um erro que derruba a adição no meio do
	// combate.
	if monsterID := strings.TrimSpace(wire.StringField(input, "monsterId")); monsterID != "" {
		entry.MonsterID = &monsterID
	}
	// O bloco de criatura do mestre. Mesma escolha do `monsterId`: o servidor não
	// confere se a criatura existe, porque um id órfão vira "sem bloco" na tela e
	// não um erro no meio do combate. Quem confere o dono é a rota HTTP que serve
	// o bloco, e ela só responde ao mestre.
	if creatureID, ok := wire.IntField(input, "creatureId"); ok && creatureID > 0 {
		entry.CreatureID = &creatureID
	}
	return entry, nil
}

func (tr tableRules) materializeCharacterEntry(ctx context.Context, callerID, campaignID int64, input map[string]any) (live.InitiativeEntry, error) {
	charID, _ := wire.IntField(input, "characterId")
	initiative, hasInit := wire.IntField(input, "initiative")
	if !hasInit {
		return live.InitiativeEntry{}, errors.New("entry.initiative is required")
	}
	stats, _, err := tr.resolveCombatant(ctx, callerID, campaignID, charID)
	if err != nil {
		return live.InitiativeEntry{}, err
	}
	label := stats.name
	if l := strings.TrimSpace(wire.StringField(input, "label")); l != "" {
		label = l
	}
	cid := charID
	return live.InitiativeEntry{
		Label: label, Initiative: int(initiative), Type: "character", CharacterID: &cid,
		HpCurrent: overrideInt(input, "hpCurrent", stats.hpCurrent),
		HpMax:     overrideInt(input, "hpMax", stats.hpMax),
		MpCurrent: overrideInt(input, "mpCurrent", stats.mpCurrent),
		MpMax:     overrideInt(input, "mpMax", stats.mpMax),
	}, nil
}

// parseEntryPatch lê um remendo de atualização do corpo cru: só campo PRESENTE
// vira não-nulo, para "deixe como está" ser distinto de "zere".
func parseEntryPatch(v any) live.EntryPatch {
	m, _ := v.(map[string]any)
	p := live.EntryPatch{}
	if m == nil {
		return p
	}
	if s, ok := m["label"].(string); ok {
		p.Label = &s
	}
	if s, ok := m["type"].(string); ok {
		p.Type = &s
	}
	if i, ok := wire.IntField(m, "initiative"); ok {
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
		// numa lista escrita à mão.
		{"creatureId", &p.CreatureID},
	} {
		if i, ok := wire.IntField(m, f.key); ok {
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

// overrideInt devolve o valor do corpo para a chave quando ele existe, senão o
// padrão — como ponteiro.
func overrideInt(m map[string]any, key string, def int64) *int64 {
	if v, ok := wire.IntField(m, key); ok {
		return live.PtrInt64(v)
	}
	return live.PtrInt64(def)
}

// parseConditions filtra pelo CATÁLOGO, que é onde as condições são autoradas:
// uma lista escrita à mão aqui seria a segunda cópia da tabela do livro, e ela
// desvia — uma condição faltando dá 400 na hora de aplicá-la. Id desconhecido é
// descartado em silêncio de propósito: a alternativa seria derrubar a aplicação
// inteira no meio do combate por causa de um item.
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
