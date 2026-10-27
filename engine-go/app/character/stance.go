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
	// SÓ ASSUME A POSTURA QUEM A TEM. O teto de degraus já se lia do nível NA
	// CLASSE (p40) e dá ZERO para quem não tem o poder, então um bárbaro entrava
	// na Inspiração do bardo (p44) com `steps: 0`: 2 PM cobrados, linha gravada,
	// e a aba Efeitos oferecendo encerrar. A tela nunca ofereceu o gesto — e é
	// por isso que a recusa mora aqui, porque travar na UI é UX (ALE-423).
	if !sheet.OwnsClassPower(dto, spec.ID) {
		return fmt.Errorf("%s: o personagem não tem este poder", spec.Name)
	}
	cost, err := scaledPrice(dto, *spec, steps)
	if err != nil {
		return err
	}

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
	return p.applyStanceGrants(ctx, row, dto, flag)
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

// scaledPrice é o PM de pagar os degraus pedidos, ou a recusa que diz por que
// não dá: acima do teto que o nível NA CLASSE abre (p40), ou sem PM para o total.
// Poder que não vende degrau tem teto zero.
func scaledPrice(dto sheet.CharacterDTO, spec book.Activation, steps int) (int, error) {
	max := PowerMaxSteps(dto, spec)
	if can, reason := book.ScaledDecision(spec, steps, max, int(dto.ManaAvailable())); !can {
		return 0, fmt.Errorf("%s: %s", spec.Name, reason)
	}
	return book.ScaledCost(spec, steps), nil
}

// UsePower gasta um uso de um poder instantâneo: cobra o PM e soma o contador.
//
// Os `steps` são os degraus A MAIS que o jogador pagou, no poder que os vende —
// a Marca da Presa (p50) troca o dado pela linha seguinte da tabela a cada um.
// Zero no poder que não vende degrau.
//
// As duas escritas são de coisas diferentes — o PM é da ficha, o contador é do
// estado de jogo — e a ordem importa: o PM primeiro, porque é ele que pode
// faltar. Somar o uso antes deixaria um uso gasto por um poder que não saiu.
func (p Plays) UsePower(
	ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, powerID string, choice UseChoice,
) error {
	steps := choice.Steps
	spec := book.ActivationOf(powerID, "")
	if spec == nil {
		return fmt.Errorf("o poder %q não tem ativação no catálogo", powerID)
	}
	if spec.Kind != "instant" {
		return fmt.Errorf("%q não é um poder de usar", spec.Name)
	}
	// SÓ USA O PODER QUEM O TEM. A decisão abaixo pergunta o PM, os usos e as
	// flags — nenhuma delas é sobre o dono —, então um bárbaro gastava 1 PM no
	// En Garde do bucaneiro (p48) e queimava o "1/cena" dele. Irmão da recusa
	// de postura (ALE-423), e pela mesma razão: a tela não oferece o gesto, e a
	// fronteira é o servidor.
	if !sheet.OwnsActivation(dto, *spec) {
		return fmt.Errorf("%s: o personagem não tem este poder", spec.Name)
	}
	uses := PowerUses(dto)[spec.ID]
	can, reason := book.UseDecision(*spec, book.UseContext{
		CurrentPM: int(dto.ManaAvailable()), UsedThisScene: uses.Scene, UsedToday: uses.Day,
		Flags: p.activeFlags(dto),
	})
	if !can {
		return fmt.Errorf("%s: %s", spec.Name, reason)
	}
	cost, err := scaledPrice(dto, *spec, steps)
	if err != nil {
		return err
	}
	riders := RidersOf(dto, *spec)
	if choice.Doubled && !book.RidersDouble(*spec, riders) {
		return fmt.Errorf("%s: nenhum poder da ficha dobra este uso (o Inimigo de (Criatura), p51)", spec.Name)
	}
	mods := book.GrantModifiersWithRiders(*spec, steps, riders, choice.Doubled)
	scope := book.ChargedScope(*spec)
	if scope == "" && spec.Grant == nil {
		// Sem limite COBRADO e sem concessão não há contador nem efeito, e sobra
		// uma escrita só: a transação seria um bloqueio que ninguém pediu.
		return chargeMp(ctx, p.queries, p.catalogs, row, cost)
	}
	return p.inTx(ctx, "usar o poder "+spec.ID, func(q *sqlcgen.Queries) error {
		if err := chargeMp(ctx, q, p.catalogs, row, cost); err != nil {
			return err
		}
		if scope != "" {
			if err := q.BumpCharacterPowerUse(ctx, sqlcgen.BumpCharacterPowerUseParams{
				Characterid: row.ID, Powerid: spec.ID, Scope: scope,
			}); err != nil {
				return fmt.Errorf("somar o uso de %q: %w", spec.ID, err)
			}
		}
		return writeGrantedEffect(ctx, q, row.ID, *spec, mods)
	})
}

// writeGrantedEffect grava o que o poder CONCEDE, num efeito ativo com a chave
// dele.
//
// DENTRO DA TRANSAÇÃO, ao contrário da concessão da postura: lá a reserva que
// falha não pode derrubar uma postura já paga; aqui o efeito É o que o PM
// comprou. Um commit que cobrasse o 1 PM da Armadura de Allihanna (p21) e
// perdesse os +2 na Defesa é o defeito que esta função existe para consertar.
//
// Chave igual à do PODER, que é o que faz usar de novo REESCREVER a mesma linha
// em vez de abrir uma segunda — o `active_effects` é único em (personagem,
// catálogo, escopo), e o livro não empilha a mesma fonte consigo.
func writeGrantedEffect(
	ctx context.Context, q *sqlcgen.Queries, characterID int64, spec book.Activation, mods []engine.Modifier,
) error {
	if spec.Grant == nil || spec.Grant.Kind != "active-effect" || len(spec.Grant.Modifiers) == 0 {
		return nil
	}
	blob, err := json.Marshal(mods)
	if err != nil {
		return fmt.Errorf("escrever a concessão de %q: %w", spec.ID, err)
	}
	if _, err := q.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: "power", Catalogid: spec.ID,
		Scope: spec.Grant.Scope, Modifiers: string(blob), Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar a concessão de %q: %w", spec.ID, err)
	}
	return nil
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
	// SEM PERGUNTAR A POSSE, ao contrário da aplicação: quem desescolheu o poder
	// com a postura em pé ainda tem a poça gravada, e filtrar aqui a deixaria
	// para trás — PV temporários de uma fúria encerrada.
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

// applyStanceGrants liga o que a flag concede A QUEM TEM O PODER, DEPOIS do
// commit — ver o cabeçalho do arquivo.
//
// A POSSE É PERGUNTADA AQUI, e não na `FlagGrants`: aquela lista o que a FLAG
// concede, e a concessão é um poder à parte da postura. A Alma de Bronze (p41)
// não tem `grantedAtLevel` — é poder de bárbaro ELETIVO, de vaga —, e o bárbaro
// que não a escolheu entrava em fúria com a reserva de PV dela (ALE-423).
//
// Mesma pergunta e mesma forma do bônus cumulativo, três arquivos ao lado
// (`cumulative.go`): sem a posse, todo bárbaro acumularia.
func (p Plays) applyStanceGrants(
	ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, flag string,
) error {
	for _, spec := range book.FlagGrants(flag) {
		if spec.Grant.Kind != "temp-hp" {
			continue
		}
		if !sheet.OwnsClassPower(dto, spec.ID) {
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

// PayAttackBoost cobra o PM do impulso que um golpe CONFIRMADO levou — o Golpe
// Divino, o Ataque Especial (ALE-487).
//
// Ele mora com os outros gastos de PM da ficha e passa pelo mesmo funil, que
// drena a poça temporária antes do poço (p106). Quem o chama é a confirmação do
// mestre, depois de o golpe pousar: a proposta só confere se a ficha PODE
// pagar, e o provisório cancelado não chega aqui.
func (p Plays) PayAttackBoost(ctx context.Context, characterID int64, pm int) error {
	if pm <= 0 {
		return nil
	}
	row, err := p.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("carregar o personagem %d para cobrar o impulso: %w", characterID, err)
	}
	return p.inTx(ctx, "cobrar o impulso do golpe", func(q *sqlcgen.Queries) error {
		return chargeMp(ctx, q, p.catalogs, row, pm)
	})
}

// OpenTheGuard grava a penalidade na Defesa que a investida CONFIRMADA deixa em
// quem investiu: –2, ou –5 na Imprudente, "até o seu próximo turno" (p235,
// p42, ALE-499). Quem a expira é a virada da vez dele, ou o fim da cena.
//
// Chave única por ficha (`engine.ChargeEffectID`): investir de novo reescreve
// a linha, e duas guardas abertas não somam — é a mesma guarda.
func (p Plays) OpenTheGuard(ctx context.Context, characterID int64, charge string) error {
	mods := engine.ChargeGuardModifiers(charge)
	if len(mods) == 0 {
		return nil
	}
	blob, err := json.Marshal(mods)
	if err != nil {
		return fmt.Errorf("escrever a guarda aberta da investida %q: %w", charge, err)
	}
	if _, err := p.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: "rule", Catalogid: engine.ChargeEffectID,
		Scope: engine.OwnTurnScope(), Modifiers: string(blob), Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar a guarda aberta do personagem %d: %w", characterID, err)
	}
	return nil
}
