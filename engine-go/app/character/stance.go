package character

import (
	"context"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// AS POSTURAS, e por que elas são os únicos gestos da ficha com TRANSAÇÃO.
//
// Entrar em Fúria não é uma escrita: o PM sai, o pagamento é registrado, e os
// condicionais da flag sobem. O registro do pagamento existe para SAIR não
// devolver PM — é para isso que a tabela `character_stances` serve —, e sem o
// contorno as três podiam ficar pela metade: PM cobrado e postura sem os
// condicionais dela, ou postura em pé sem registro de quanto foi pago.
//
// # O que fica FORA da transação, e é decisão e não esquecimento
//
// As CONCESSÕES (a reserva de PV temporários da Alma de Bronze, p41) são
// aplicadas depois do commit, de propósito: uma concessão que falha NÃO derruba
// a postura que a pessoa acabou de pagar. O erro sobe como recusa, a postura
// fica em pé com o que já aplicou, e é esse o estado que a tela mostra. A regra
// já estava escrita nas concessões da cena antes de haver transação — trazê-la
// para dentro seria mudá-la de contrabando (ALE-351).

// EnterStance entra numa postura com os degraus escolhidos.
//
// A `dto` atravessa porque a decisão precisa da ficha COMPUTADA — o teto de
// degraus sai do nível NA CLASSE, e os condicionais da flag saem do motor. Quem
// já a montou para desenhar a tela não a computa de novo.
func (p Plays) EnterStance(
	ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, flag string, steps int,
) error {
	spec := stanceOfFlag(flag)
	if spec == nil {
		return fmt.Errorf("%q não é uma postura do livro", flag)
	}
	max := 0
	if spec.Scaling != nil {
		max = book.LevelSteps(*spec.Scaling, ClassPowerLevel(dto, spec.ID))
	}
	if can, reason := book.StanceDecision(*spec, steps, max, int(dto.MpCurrent)); !can {
		return fmt.Errorf("%s: %s", spec.Name, reason)
	}
	cost := book.StanceCost(*spec, steps)

	if err := p.inTx(ctx, "entrar na postura "+flag, func(q *sqlcgen.Queries) error {
		if err := chargeMp(ctx, q, p.catalogs, row, cost); err != nil {
			return err
		}
		if err := q.UpsertCharacterStance(ctx, sqlcgen.UpsertCharacterStanceParams{
			Characterid: row.ID, Flag: flag, Steps: int64(steps), Pmpaid: int64(cost),
		}); err != nil {
			return fmt.Errorf("registrar o pagamento da postura: %w", err)
		}
		// TODOS os condicionais da flag sobem juntos — a Fúria mexe em ataque,
		// dano, Defesa e testes de Vontade, e metade ligada é uma ficha que soma
		// metade de uma regra do livro.
		for _, id := range p.conditionalsOfFlag(dto, flag) {
			if err := q.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
				Characterid: row.ID, Conditionalid: id,
			}); err != nil {
				return fmt.Errorf("ligar o condicional %q: %w", id, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return p.applyStanceGrants(ctx, row, flag)
}

// EndStance encerra a postura e leva junto o que ela tinha ligado.
//
// A reserva de PV temporários dura "enquanto a Fúria durar" (p41), e deixá-la
// para trás daria PV que a postura encerrada continua pagando. Aqui as três
// escritas ficam DENTRO do contorno — ao contrário de entrar, onde a concessão
// que falha não pode derrubar o que foi pago, sair não tem o que preservar: a
// postura encerrada pela metade é a ficha somando o que não existe mais.
func (p Plays) EndStance(
	ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, flag string,
) error {
	conditionals := p.conditionalsOfFlag(dto, flag)
	granted, err := p.grantedEffectIDs(ctx, row, flag)
	if err != nil {
		return err
	}
	return p.inTx(ctx, "encerrar a postura "+flag, func(q *sqlcgen.Queries) error {
		if err := q.RemoveCharacterStance(ctx, sqlcgen.RemoveCharacterStanceParams{
			Characterid: row.ID, Flag: flag,
		}); err != nil {
			return fmt.Errorf("apagar a postura: %w", err)
		}
		for _, id := range granted {
			if err := q.DeleteEffectByID(ctx, id); err != nil {
				return fmt.Errorf("apagar o efeito concedido %d: %w", id, err)
			}
		}
		for _, id := range conditionals {
			if err := q.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
				Characterid: row.ID, Conditionalid: id,
			}); err != nil {
				return fmt.Errorf("desligar o condicional %q: %w", id, err)
			}
		}
		return nil
	})
}

// UsePower gasta um uso de um poder instantâneo: cobra o PM e soma o contador.
//
// As duas escritas são de coisas diferentes — o PM é da ficha, o contador é do
// estado de jogo — e a ordem importa: o PM primeiro, porque é ele que pode
// faltar. Somar o uso antes deixaria um uso gasto por um poder que não saiu.
func (p Plays) UsePower(
	ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, powerID string,
) error {
	spec := book.ActivationOf(powerID, "")
	if spec == nil {
		return fmt.Errorf("o poder %q não tem ativação no catálogo", powerID)
	}
	if spec.Kind != "instant" {
		return fmt.Errorf("%q não é um poder de usar", spec.Name)
	}
	uses := PowerUses(dto)[spec.ID]
	can, reason := book.UseDecision(*spec, book.UseContext{
		CurrentPM: int(dto.MpCurrent), UsedThisScene: uses.Scene, UsedToday: uses.Day,
		Flags: p.activeFlags(dto),
	})
	if !can {
		return fmt.Errorf("%s: %s", spec.Name, reason)
	}
	scope := book.ChargedScope(*spec)
	if scope == "" {
		// Sem limite COBRADO não há contador, e sobra uma escrita só: a
		// transação seria um bloqueio que ninguém pediu.
		return chargeMp(ctx, p.queries, p.catalogs, row, book.ActivationPm(*spec))
	}
	return p.inTx(ctx, "usar o poder "+spec.ID, func(q *sqlcgen.Queries) error {
		if err := chargeMp(ctx, q, p.catalogs, row, book.ActivationPm(*spec)); err != nil {
			return err
		}
		if err := q.BumpCharacterPowerUse(ctx, sqlcgen.BumpCharacterPowerUseParams{
			Characterid: row.ID, Powerid: spec.ID, Scope: scope,
		}); err != nil {
			return fmt.Errorf("somar o uso de %q: %w", spec.ID, err)
		}
		return nil
	})
}

// chargeMp tira o PM da ficha, sem deixar o saldo abaixo de zero.
//
// O piso é do funil e vale para todo gesto, mas a razão dele é desta cobrança: a
// decisão que autorizou o gasto foi tomada com o saldo LIDO antes, e cobrar até
// o fundo é melhor que gravar um PM negativo, que a tela desenharia como barra
// para trás. A cobrança FICA dentro da transação — o `_txlock=immediate`
// serializa a escrita, mas a leitura que autorizou o gasto aconteceu fora dela.
func chargeMp(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs,
	row sqlcgen.Character, howMuch int,
) error {
	if howMuch <= 0 {
		return nil
	}
	if _, err := sheet.ApplyToPools(ctx, q, cat, row,
		func(pools sheet.Pools) (sheet.Pools, error) {
			pools.MpCurrent -= int64(howMuch)
			return pools, nil
		}); err != nil {
		return fmt.Errorf("cobrar %d PM da ficha %d: %w", howMuch, row.ID, err)
	}
	return nil
}

// inTx roda o corpo numa transação, e o `which` entra na mensagem de falha: um
// "abrir a transação" sem o gesto não diz qual dos três estourou.
func (p Plays) inTx(ctx context.Context, which string, body func(*sqlcgen.Queries) error) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir a transação de %s: %w", which, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := body(p.queries.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fechar a transação de %s: %w", which, err)
	}
	return nil
}

// stanceOfFlag acha a ativação da postura pela flag que ela acende.
func stanceOfFlag(flag string) *book.Activation {
	stance, found := book.StancesFromCatalog()[flag]
	if !found {
		return nil
	}
	return book.ActivationOf("", stance.Name)
}

// conditionalsOfFlag são os ids dos condicionais que aquela flag acende.
//
// Motor ausente devolve vazio, e não erro: sem catálogo primado não há o que
// ligar, e recusar o gesto inteiro deixaria a postura fora do alcance de quem
// roda sem o arquivo. É o ÚLTIMO recuo desse tipo — o dos poços virou recusa
// quando o arranque passou a exigir o catálogo (ALE-355).
func (p Plays) conditionalsOfFlag(dto sheet.CharacterDTO, flag string) []string {
	if dto.Ruleset == nil {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return nil
	}
	outside := []string{}
	for _, c := range engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional {
		if c.Flag == flag {
			outside = append(outside, engine.ConditionalID(c))
		}
	}
	return outside
}

// activeFlags são as flags acesas agora, para a decisão de usar um poder que
// depende de postura.
func (p Plays) activeFlags(dto sheet.CharacterDTO) map[string]bool {
	outside := map[string]bool{}
	for _, s := range dto.Stances {
		outside[s.Flag] = true
	}
	return outside
}

// grantedEffectIDs são os efeitos em curso que vieram das concessões da flag.
//
// A busca é pelo id do PODER na coluna `catalogId` do efeito — é assim que o
// efeito guarda de onde veio, e é o que permite encerrar sem lembrar de nada
// entre uma requisição e outra.
func (p Plays) grantedEffectIDs(
	ctx context.Context, row sqlcgen.Character, flag string,
) ([]int64, error) {
	fromFlag := map[string]bool{}
	for _, spec := range book.FlagGrants(flag) {
		fromFlag[spec.ID] = true
	}
	if len(fromFlag) == 0 {
		return nil, nil
	}
	effects, err := p.queries.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("ler os efeitos da ficha %d: %w", row.ID, err)
	}
	outside := []int64{}
	for _, e := range effects {
		if fromFlag[e.Catalogid] {
			outside = append(outside, e.ID)
		}
	}
	return outside, nil
}

// applyStanceGrants liga o que a flag concede, DEPOIS do commit — ver o
// cabeçalho do arquivo.
func (p Plays) applyStanceGrants(ctx context.Context, row sqlcgen.Character, flag string) error {
	for _, spec := range book.FlagGrants(flag) {
		if spec.Grant.Kind != "temp-hp" {
			continue
		}
		howMuch, ok := p.TempHpAmount(ctx, row, spec.Grant.Attribute)
		if !ok {
			continue
		}
		if _, err := p.ApplyTempHpPool(
			ctx, row.ID, "power", spec.ID, spec.Grant.Scope, howMuch, "PV temporários"); err != nil {
			return err
		}
	}
	return nil
}
