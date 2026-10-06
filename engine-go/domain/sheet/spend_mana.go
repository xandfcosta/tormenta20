package sheet

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// O GASTO DE PM, com a poça temporária na frente.
//
//	"Pontos temporários são sempre os primeiros a serem gastos." (p106)
//
// ELE É UM FUNIL, e é essa a razão de existir em vez de cada gesto drenar por
// conta: são TRÊS que cobram PM — ativar um poder, conjurar e sustentar no giro
// da vez — e a regra escrita três vezes é a que a próxima correção acerta em
// duas. O ±PM dos botões NÃO passa por aqui de propósito: ele é CORREÇÃO e não
// gasto, e mexe no poço de verdade (decisão do dono).

// SpendMana cobra `amount` de PM da ficha, drenando a poça temporária primeiro.
//
// A ORDEM DAS DUAS ESCRITAS é a poça antes do poço. As duas acontecem dentro da
// transação do gesto, então o par é atômico para quem chama de lá; a ordem só
// decide qual metade sobrevive a um estrago fora dela, e perder pontos que iam
// expirar sozinhos é mais barato que perder PM do poço.
//
// AMOUNT NÃO POSITIVO não cobra nada e não é erro: o gesto de custo zero existe
// (o truque, a postura gratuita), e recusá-lo aqui obrigaria cada chamador a um
// `if` antes. Ele ainda ATRAVESSA o funil, porque quem chama espelha o poço de
// volta e precisa do número mesmo quando nada mudou.
func SpendMana(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs,
	row sqlcgen.Character, amount int,
) (Pools, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return Pools{}, fmt.Errorf("ler as poças de PM da ficha %d: %w", row.ID, err)
	}
	var plan SpendPlan
	pools, err := ApplyToPools(ctx, q, cat, row, func(pools Pools) (Pools, error) {
		if amount <= 0 {
			return pools, nil
		}
		plan = PlanSpend(ParseTempMpPools(effects), int(pools.MpCurrent), amount)
		pools.MpCurrent = int64(plan.MpCurrent)
		return pools, nil
	})
	if err != nil {
		return pools, fmt.Errorf("cobrar %d PM da ficha %d: %w", amount, row.ID, err)
	}
	pools.TempMp = int64(plan.TempMpRemaining)
	return pools, WriteDrain(ctx, q, plan.Updates, plan.DeleteIDs)
}

// SpendLoadedMana é o MESMO funil para quem já tem o agregado na mão.
//
// Existe pela razão do `ApplyToLoadedPools`, que ela usa: o gesto de conjurar
// precisou da ficha inteira para saber o custo, e recarregá-la para cobrar
// seria a terceira leitura da mesma ficha no mesmo pedido.
func SpendLoadedMana(
	ctx context.Context, q *sqlcgen.Queries, dto *CharacterDTO, amount int,
) error {
	if amount <= 0 {
		return nil
	}
	effects, err := q.ListActiveEffectsByCharacter(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("ler as poças de PM da ficha %d: %w", dto.ID, err)
	}
	var plan SpendPlan
	if _, err := ApplyToLoadedPools(ctx, q, dto, func(pools Pools) (Pools, error) {
		plan = PlanSpend(ParseTempMpPools(effects), int(pools.MpCurrent), amount)
		pools.MpCurrent = int64(plan.MpCurrent)
		return pools, nil
	}); err != nil {
		return fmt.Errorf("cobrar %d PM da ficha %d: %w", amount, dto.ID, err)
	}
	if err := WriteDrain(ctx, q, plan.Updates, plan.DeleteIDs); err != nil {
		return err
	}
	// O AGREGADO EM MÃOS FICA EM DIA: quem chamou por aqui continua usando o
	// `dto` depois, e uma poça drenada que o `dto` ainda mostra cheia é a
	// segunda verdade que este funil existe para não ter.
	dto.ActiveEffects = effectsAfterDrain(dto.ActiveEffects, plan.Updates, plan.DeleteIDs)
	return nil
}

// WriteDrain grava o que um plano fez com as poças — as reescritas e as que
// esvaziaram.
//
// UM lugar para os dois vitais: o dano de PV e o gasto de PM produzem o mesmo
// par de listas, e a segunda cópia do laço divergiria no dia em que uma delas
// ganhasse um passo.
func WriteDrain(
	ctx context.Context, q *sqlcgen.Queries, updates []EffectModifierWrite, deleteIDs []int64,
) error {
	for _, u := range updates {
		if err := q.UpdateEffectModifiers(ctx, sqlcgen.UpdateEffectModifiersParams{
			Modifiers: u.Modifiers, ID: u.EffectID,
		}); err != nil {
			return fmt.Errorf("reescrever a poça do efeito %d: %w", u.EffectID, err)
		}
	}
	for _, id := range deleteIDs {
		if err := q.DeleteEffectByID(ctx, id); err != nil {
			return fmt.Errorf("apagar a poça vazia do efeito %d: %w", id, err)
		}
	}
	return nil
}

// effectsAfterDrain devolve a lista de efeitos do agregado como ela ficou.
func effectsAfterDrain(
	effects []EffectDTO, updates []EffectModifierWrite, deleteIDs []int64,
) []EffectDTO {
	rewritten := make(map[int64]string, len(updates))
	for _, u := range updates {
		rewritten[u.EffectID] = u.Modifiers
	}
	gone := make(map[int64]bool, len(deleteIDs))
	for _, id := range deleteIDs {
		gone[id] = true
	}
	out := make([]EffectDTO, 0, len(effects))
	for _, e := range effects {
		if gone[e.ID] {
			continue
		}
		if blob, found := rewritten[e.ID]; found {
			e.Modifiers = blob
		}
		out = append(out, e)
	}
	return out
}
