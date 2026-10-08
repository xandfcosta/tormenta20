package character

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
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
	if can, reason := book.StanceDecision(*spec, steps, max, int(dto.ManaAvailable())); !can {
		return fmt.Errorf("%s: %s", spec.Name, reason)
	}
	cost := book.StanceCost(*spec, steps)

	// UMA POSTURA POR VEZ, no grupo dela — *"você só pode manter uma postura
	// por vez"* (p54).
	//
	// A anterior cai ANTES e FORA da transação de entrar, e as duas coisas são
	// decisão. Fora porque o `EndStance` abre a transação dele, e aninhar duas
	// no mesmo gesto é o abraço mortal que o `boards.Store` descreve. Antes
	// porque a ordem inversa deixaria as duas em pé no intervalo, e é
	// exatamente esse intervalo que a regra proíbe.
	//
	// SEM COBRAR NADA pela saída: o livro cobra para ASSUMIR, e trocar de
	// postura é um gesto só.
	if err := p.dropsTheOtherStances(ctx, row, dto, *spec); err != nil {
		return err
	}

	if err := p.inTx(ctx, "entrar na postura "+flag, func(q *sqlcgen.Queries) error {
		if err := chargeMp(ctx, q, p.catalogs, row, cost); err != nil {
			return err
		}
		if err := q.UpsertCharacterStance(ctx, sqlcgen.UpsertCharacterStanceParams{
			Characterid: row.ID, Flag: flag, Steps: int64(steps), Pmpaid: int64(cost),
		}); err != nil {
			return fmt.Errorf("registrar o pagamento da postura: %w", err)
		}
		// UMA CHAVE, a do GRUPO. A Fúria mexe em ataque, dano, Defesa e testes de
		// Vontade, e metade ligada é uma ficha que soma metade de uma regra do
		// livro — aqui isso deixou de ser possível de representar.
		//
		// Antes eram N linhas, uma por condicional, calculadas pela coleta no
		// instante de entrar. Isso tinha um segundo defeito além do tamanho: sair
		// recalculava o conjunto, então uma ficha que subisse de nível DENTRO da
		// postura deixava condicional ligado para trás.
		if err := q.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: engine.FlagGroupID(flag),
		}); err != nil {
			return fmt.Errorf("ligar a postura %q: %w", flag, err)
		}
		return writeStanceDegrees(ctx, q, row.ID, *spec, flag, steps)
	}); err != nil {
		return err
	}
	return p.applyStanceGrants(ctx, row, flag)
}

// writeStanceDegrees grava o que os degraus PAGOS valem, num efeito ativo com a
// chave do poder da postura.
//
// # POR QUE UM EFEITO ATIVO, e não uma coluna
//
// O `character_stances` já guarda `steps`, e durante três fatias ele foi o
// único lugar onde o degrau existia: cobrado, gravado, e lido por NINGUÉM. O
// número vinha de poderes que o catálogo concedia por nível, então pagar o
// degrau e não pagar davam o mesmo bônus (ALE-423).
//
// Fazer o motor ler a coluna seria dar a ele um segundo caminho de entrada só
// para isto. O efeito ativo é o caminho que JÁ existe para "estado de jogo que
// vira modificador" — é por ele que passam o bônus cumulativo e a reserva de PV
// temporários —, e ele vem com o escopo de cena, a aba Efeitos e a expiração de
// graça.
//
// DENTRO DA TRANSAÇÃO, ao contrário das concessões: o degrau é exatamente o que
// o PM extra comprou. Um commit que cobrasse o PM e perdesse o bônus é o defeito
// que esta função existe para consertar.
//
// ESCOPO CENA porque a postura é de cena: o `EndScene` apaga os efeitos de cena
// e baixa as posturas no mesmo gesto, então os dois morrem juntos sem ninguém
// precisar lembrar.
func writeStanceDegrees(
	ctx context.Context, q *sqlcgen.Queries, characterID int64,
	spec book.Activation, flag string, steps int,
) error {
	mods := book.StanceDegreeModifiers(flag, steps)
	if len(mods) == 0 {
		return nil
	}
	blob, err := json.Marshal(mods)
	if err != nil {
		return fmt.Errorf("escrever os %d degraus de %q: %w", steps, spec.ID, err)
	}
	if _, err := q.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: "power", Catalogid: spec.ID, Scope: "scene",
		Modifiers: string(blob), Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar os %d degraus de %q: %w", steps, spec.ID, err)
	}
	return nil
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
		if err := q.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: engine.FlagGroupID(flag),
		}); err != nil {
			return fmt.Errorf("desligar a postura %q: %w", flag, err)
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
		CurrentPM: int(dto.ManaAvailable()), UsedThisScene: uses.Scene, UsedToday: uses.Day,
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
	// PELO FUNIL DE GASTO, que drena a poça temporária antes do poço (p106).
	// Com `MpCurrent -= n` direto, o bardo com a poça do Golpe Mágico cheia
	// pagaria a postura com o mana de verdade e ficaria com a poça intocada.
	_, err := sheet.SpendMana(ctx, q, cat, row, howMuch)
	return err
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
	// E O QUE A POSTURA ACUMULOU sai com ela (decisão do dono, p42): o bônus
	// cumulativo da Sangue dos Inimigos é escrito pelo mesmo tipo de efeito e
	// com o id do mesmo poder, então ele morre pelo mesmo caminho — o que falta
	// é esta lista saber que ele existe. Sem a linha, o bárbaro sairia da fúria
	// carregando o +N dela.
	for _, spec := range book.FlagCumulatives(flag) {
		fromFlag[spec.ID] = true
	}
	// E OS DEGRAUS PAGOS, que moram num efeito de chave igual à do PODER da
	// postura. É a mesma chave que o bônus cumulativo usa, e as duas nunca
	// coincidem no mesmo poder — quem mantém isso verdade é o
	// `TestEveryStanceDegreeRidesANonStackingBonus`.
	if stance, found := book.StancesFromCatalog()[flag]; found {
		fromFlag[stance.ID] = true
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

// dropsTheOtherStances encerra as posturas do MESMO GRUPO que estejam em pé.
//
// Ela lê o que está aceso na ficha e não a lista inteira do grupo: encerrar uma
// postura que não está em pé é escrita à toa, e a `RemoveCharacterStance` não
// reclamaria — o gesto ficaria caro e calado.
func (p Plays) dropsTheOtherStances(
	ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, entrando book.Activation,
) error {
	group := entrando.StanceGroup
	if group == "" {
		return nil
	}
	acesas := map[string]bool{}
	for _, s := range dto.Stances {
		acesas[s.Flag] = true
	}
	for _, sibling := range book.StanceSiblings(group, entrando.Flag) {
		if !acesas[sibling.Flag] {
			continue
		}
		if err := p.EndStance(ctx, row, dto, sibling.Flag); err != nil {
			return fmt.Errorf("trocar de postura: %w", err)
		}
	}
	return nil
}
