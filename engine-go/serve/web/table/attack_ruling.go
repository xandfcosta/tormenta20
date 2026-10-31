package table

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app"
	"t20engine/app/combat"
	"t20engine/domain/live"
)

// O SEGUNDO TEMPO DO GOLPE NA MESA: o mestre decide errou, acertou ou crítico,
// e quem atacou rola o dano que a decisão pede — no servidor ou na mão.

// rulingsByPath são as três decisões, como o caminho as escreve.
var rulingsByPath = map[string]string{
	"errou":   live.RulingMiss,
	"acertou": live.RulingHit,
	"critico": live.RulingCritical,
}

// rulesAttack guarda a decisão do mestre sobre o golpe. O ERRO encerra o golpe,
// e por isso deixa o que o ✓ deixaria (`afterTheStrike`), lido do provisório
// JÁ decidido: a sugestão do livro podia ser acerto.
func rulesAttack(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	ruling, known := rulingsByPath[chi.URLParam(c.R, "decisao")]
	if !known {
		return nil, fmt.Errorf("a decisão %q não existe: é errou, acertou ou critico", chi.URLParam(c.R, "decisao"))
	}
	_, _, req, err := st.pendingStrike(c)
	if err != nil {
		return nil, err
	}
	ruled, err := st.strike.Rule(c.R.Context(), app.Caller{ID: c.User}, c.Role, req, ruling)
	if err != nil || ruling != live.RulingMiss || ruled.PendingAttack == nil {
		return ruled, err
	}
	return ruled, st.afterTheStrike(c, ruled, ruled.PendingAttack, nil)
}

// rollsAttackDamage rola o dano no servidor.
func rollsAttackDamage(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.rollDamage(c, nil)
}

// rollsAttackDamageByHand usa os dados que a mesa rolou e digitou, um campo
// por dado, na ordem da fórmula.
func rollsAttackDamageByHand(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.PendingAttack == nil || state.PendingAttack.DamageToRoll == nil {
		return nil, errors.New("o dano vem depois de o mestre dizer se o ataque acertou")
	}
	hand, err := handDamageDice(c, diceCount(*state.PendingAttack.DamageToRoll))
	if err != nil {
		return nil, err
	}
	return st.rollDamage(c, hand)
}

func (st Scene) rollDamage(c commandCtx, hand []int) (*live.SessionRuntimeState, error) {
	_, _, req, err := st.pendingStrike(c)
	if err != nil {
		return nil, err
	}
	if _, err := st.strike.RollDamage(c.R.Context(), app.Caller{ID: c.User}, c.Role, req, hand); err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}

// pendingStrike é o provisório em jogo e o pedido que o refaz — com a posse
// resolvida CONTRA O BANCO e a peça atacada relida do tabuleiro.
func (st Scene) pendingStrike(c commandCtx) (*live.SessionRuntimeState, live.PendingAttack, combat.Request, error) {
	ctx := c.R.Context()
	state, err := st.deps.Sessions().State(ctx, c.SessionID)
	if err != nil {
		return nil, live.PendingAttack{}, combat.Request{}, err
	}
	if state == nil || state.PendingAttack == nil {
		return nil, live.PendingAttack{}, combat.Request{}, errors.New("não há ataque na mesa")
	}
	pending := *state.PendingAttack
	var object *combat.ObjectTarget
	if pending.TargetTokenID != "" {
		if object, err = st.objectTokenOf(c, pending.TargetTokenID); err != nil {
			return nil, live.PendingAttack{}, combat.Request{}, err
		}
	}
	owns := false
	if attacker := entryByID(state, pending.AttackerEntryID); attacker != nil && attacker.CharacterID != nil {
		_, mine, _ := st.tableRoster(ctx, c.User, c.CampaignID)
		owns = mine[*attacker.CharacterID]
	}
	return state, pending, combat.ReplayRequest(c.CampaignID, c.SessionID, pending, owns, object), nil
}

func diceCount(toRoll live.DamageToRoll) int {
	count := 0
	for _, group := range toRoll.Dice {
		count += group.Count
	}
	return count
}

// handDamageSignal é o sinal do i-ésimo dado digitado, contando de 1.
func handDamageSignal(i int) string { return "damage_die_" + strconv.Itoa(i) }

// handDamageDice lê os dados digitados. Todos ou nenhum: um campo vazio no meio
// é um dado que a pessoa esqueceu, e rolá-lo no servidor misturaria as mesas.
func handDamageDice(c commandCtx, count int) ([]int, error) {
	var signals map[string]json.RawMessage
	if err := datastar.ReadSignals(c.R, &signals); err != nil {
		return nil, fmt.Errorf("não entendi os dados digitados: %v", err)
	}
	dice := make([]int, 0, count)
	for i := 1; i <= count; i++ {
		raw := strings.Trim(string(signals[handDamageSignal(i)]), `" `)
		if raw == "" || raw == "null" {
			return nil, fmt.Errorf("faltou o %dº dado do dano: são %d campos, um por dado", i, count)
		}
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("o %dº dado do dano tem de ser um número, e veio %q", i, raw)
		}
		dice = append(dice, value)
	}
	return dice, nil
}
