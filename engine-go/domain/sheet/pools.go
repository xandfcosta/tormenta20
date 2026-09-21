package sheet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O FUNIL: todo PV e todo PM que mudam de valor passam por aqui.
//
// Com o máximo DERIVADO (ALE-355), nenhum gesto consegue mais decidir sozinho o
// que gravar. Sete sítios computavam o atual novo, e três deles só tinham a
// linha crua do banco em mãos: liam `row.Hpmax`, que deixou de ser verdade no
// instante em que o catálogo virou a autoridade. O funil dá a TODOS o mesmo
// material — o poço derivado — e guarda para si a única escrita.
//
// A divisão: o funil deriva e grava, a REGRA do gesto decide o atual novo. Uma
// dose cura até o teto, uma magia cobra PM, uma noite de descanso soma o que o
// livro manda — e nenhuma delas precisa saber que o que o banco guarda é DANO.

// Pools são os poços de um personagem num instante: o máximo derivado do
// catálogo e o atual, que é o que sobra dele depois do que já foi gasto.
type Pools struct {
	HpMax     int64
	HpCurrent int64
	MpMax     int64
	MpCurrent int64
}

// PoolRule decide o atual novo a partir do poço derivado. É a regra do gesto, e
// o máximo que ela devolver é IGNORADO — quem manda no máximo é o catálogo.
type PoolRule func(Pools) (Pools, error)

// ApplyToPools deriva os poços, deixa a regra do gesto decidir o atual, e grava.
//
// Devolve o poço como ficou, porque quase todo chamador precisa espelhá-lo: a
// cena redesenha a barra, o rastreador da mesa copia o número, a dose responde
// quanto curou.
//
// Ela carrega a ficha inteira porque o máximo depende do agregado — classes,
// raça, poderes e os atributos já somados pelos itens. Não há atalho a partir da
// linha crua: era justamente esse atalho que produzia a segunda verdade.
func ApplyToPools(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs,
	c sqlcgen.Character, rule PoolRule,
) (Pools, error) {
	dto, err := Load(ctx, q, cat, c)
	if err != nil {
		return Pools{}, fmt.Errorf("carregar a ficha %d para mexer nos poços: %w", c.ID, err)
	}
	return ApplyToLoadedPools(ctx, q, &dto, rule)
}

// ApplyToLoadedPools é o MESMO funil para quem já carregou o agregado, e ele
// remenda o DTO junto.
//
// A porta existe porque o gesto de conjurar chega com a ficha inteira em mãos —
// ele precisou dela para saber o custo — e recarregá-la para gravar seria a
// terceira leitura da mesma ficha no mesmo pedido. As duas portas gravam pelo
// mesmo `savePools`, que é o que o guarda `TestEveryVitalWriteGoesThroughTheFunnel`
// prende: o que não pode haver é uma SEGUNDA escrita, não uma segunda entrada.
func ApplyToLoadedPools(
	ctx context.Context, q *sqlcgen.Queries, dto *CharacterDTO, rule PoolRule,
) (Pools, error) {
	before := Pools{
		HpMax: dto.HpMax, HpCurrent: dto.HpCurrent,
		MpMax: dto.MpMax, MpCurrent: dto.MpCurrent,
	}
	after, err := rule(before)
	if err != nil {
		return before, err
	}
	// O máximo da regra é descartado sem aviso de propósito: a alternativa seria
	// recusar quem devolvesse um máximo diferente, e isso convidaria o gesto a
	// tentar. O poço é do catálogo, ponto.
	after.HpMax, after.MpMax = before.HpMax, before.MpMax
	after.HpCurrent = WithinHitPoints(after.HpCurrent, after.HpMax)
	after.MpCurrent = WithinPool(after.MpCurrent, after.MpMax)
	dto.HpMax, dto.HpCurrent = after.HpMax, after.HpCurrent
	dto.MpMax, dto.MpCurrent = after.MpMax, after.MpCurrent
	if err := savePools(ctx, q, dto.ID, after); err != nil {
		return after, err
	}
	return after, followTheHitPoints(ctx, q, dto, before.HpCurrent, after.HpCurrent)
}

// followTheHitPoints liga e desliga as condições que a mudança de PV pede
// (p236): cair liga Inconsciente e Sangrando, curar estabiliza, chegar a 1
// acorda. QUAIS é decisão do `engine.DyingConditionChange`; aqui só se grava.
//
// Ela mora NO FUNIL porque ele é o único caminho de escrita de vital (o
// `TestEveryVitalWriteGoesThroughTheFunnel` o prende): pancada do mestre, dose,
// descanso e conjuração passam todos aqui, e nenhum escapa das condições.
func followTheHitPoints(ctx context.Context, q *sqlcgen.Queries, dto *CharacterDTO, before, after int64) error {
	add, drop := engine.DyingConditionChange(before, after, dto.HpMax)
	if len(add) == 0 && len(drop) == 0 {
		return nil
	}
	current := UnmarshalStrings(dto.ActiveConditions)
	next := make([]string, 0, len(current)+len(add))
	dropped := map[string]bool{}
	for _, c := range drop {
		dropped[c] = true
	}
	present := map[string]bool{}
	for _, c := range current {
		if !dropped[c] {
			next = append(next, c)
			present[c] = true
		}
	}
	for _, c := range add {
		if !present[c] {
			next = append(next, c)
		}
	}
	marshaled := MarshalStrings(&next)
	if marshaled == dto.ActiveConditions {
		return nil
	}
	if err := q.UpdateConditions(ctx, sqlcgen.UpdateConditionsParams{
		ActiveConditions: marshaled, UpdatedAt: dbvalue.NowISO(), ID: dto.ID,
	}); err != nil {
		return fmt.Errorf("gravar as condições que o PV da ficha %d pede: %w", dto.ID, err)
	}
	dto.ActiveConditions = marshaled
	return nil
}

// Aqui moravam o `RefreshPools` e o `FillPools`, e os dois viraram NADA.
//
// O primeiro regravava os poços sem mexer no gasto — o gesto do passo de
// atributo e do degrau de nível. O segundo enchia os poços de quem nasce. Com o
// máximo derivado e o dano guardado, os dois não têm o que fazer: o teto se move
// sozinho quando o nível ou a Constituição mudam, e um personagem sem linha em
// `character_damage` já está cheio. Eles existiam para manter as quatro colunas
// de espelho, e elas saíram na 00015 (ALE-355).

// WithinPool prende o PM entre zero e o teto, e o `WithinHitPoints` logo abaixo
// prende o PV — o PV desce até o limiar da morte, o PM não (ALE-366). As duas
// são as ÚNICAS grafias da regra.
//
// Eram cinco: duas funções com nome próprio, que saíram com o
// `app/character/vitals.go`, e três `min(max(…))` escritos à mão nos gestos. A
// sexta nasceu na MESMA fatia que escreveu esta frase, num ajudante que não
// existe mais a um arquivo daqui — e é por isso que a contagem deixou de ser
// afirmada por extenso. Quem varre é o `TestNoSecondSpellingOfTheVitalClamp`,
// que não envelhece.
//
// TRÊS regras parecidas ficam de fora, e a semelhança é só de FORMA:
//
//   - o `live.ClampVital` prende a entrada do RASTREADOR, cujo máximo é opcional
//     porque um NPC pode não ter nenhum. Poço de ficha sempre tem teto;
//   - o `BalanceAfterMoneyGesture` do `tibar.go` **não prende: RECUSA**, com a
//     frase que diz quanto a pessoa tem. Um clamp ali gastaria o dinheiro até o
//     fundo em silêncio, em vez de explicar por que não dá;
//   - o `domain/engine` prende teto de BÔNUS DO LIVRO — a Insolência do
//     Bucaneiro é "+Carisma na Defesa, até o nível de Bucaneiro" (p47) —, e ele
//     não pode chamar isto aqui: a direção de import é `sheet → engine`.
func WithinPool(value, ceiling int64) int64 { return min(max(int64(0), value), ceiling) }

// WithinHitPoints é o mesmo, com o piso do PV: ele desce abaixo de zero até o
// limiar da morte (–10 ou menos a metade dos PV totais, p236), e o PM não
// (decisão do dono, ALE-366). O limiar é do motor; o PV no limiar é morto.
//
// @example WithinHitPoints(-40, 30) // -15
func WithinHitPoints(value, ceiling int64) int64 {
	return min(max(engine.DeathThreshold(ceiling), value), ceiling)
}

// savePools grava o que de fato é ESTADO: o quanto se apanhou.
//
// A ausência de linha quer dizer INTACTO (migração 00014), então um personagem
// que volta ao cheio perde o registro em vez de ganhar um par de zeros — é o que
// mantém a tabela do tamanho de quem está machucado.
//
// # Por que ela relê o dano antes de gravar
//
// Para não carimbar quando nada mudou. O `updatedAt` de `characters` É a versão
// da ficha — o `sheetui/view.go` e o `table/stream.go` o servem como tal —,
// então um carimbo sem mudança nenhuma manda a tela inteira se repedir.
//
// A comparação é contra o DANO GRAVADO, e não contra o poço que o funil acabou
// de derivar: derivado contra derivado dá sempre igual, e foi exatamente assim
// que a sincronização de vitais parou de regravar o máximo e congelou a coluna
// de um elfo em 19 quando o poço dele já era 20.
//
// # E o carimbo é explícito porque perdeu a carona
//
// Ele vinha de graça no `UPDATE` das quatro colunas de vitais. Elas saíram na
// 00015, e sem o `TouchCharacter` uma pancada do mestre deixaria de acender a
// versão: o jogador ficaria com o PV de antes na tela, sem erro em lugar nenhum
// (ALE-355).
func savePools(ctx context.Context, q *sqlcgen.Queries, id int64, p Pools) error {
	// O dano pode passar do máximo — é o PV negativo —, e o funil já prendeu o
	// atual ao limiar; preso de novo aqui, contra zero, ele voltaria a mentir.
	hpDamage := p.HpMax - WithinHitPoints(p.HpCurrent, p.HpMax)
	mpSpent := WithinPool(p.MpMax-p.MpCurrent, p.MpMax)
	saved, err := storedDamage(ctx, q, id)
	if err != nil {
		return err
	}
	if saved.Hpdamage == hpDamage && saved.Mpspent == mpSpent {
		return nil
	}
	if hpDamage == 0 && mpSpent == 0 {
		if err := q.ClearCharacterDamage(ctx, id); err != nil {
			return fmt.Errorf("apagar o dano da ficha %d: %w", id, err)
		}
	} else if err := q.SaveCharacterDamage(ctx, sqlcgen.SaveCharacterDamageParams{
		Characterid: id, Hpdamage: hpDamage, Mpspent: mpSpent,
	}); err != nil {
		return fmt.Errorf("gravar o dano da ficha %d: %w", id, err)
	}
	if err := q.TouchCharacter(ctx, sqlcgen.TouchCharacterParams{
		UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		return fmt.Errorf("carimbar a versão da ficha %d: %w", id, err)
	}
	return nil
}

// storedDamage lê o dano gravado. Linha ausente é um par de zeros, que é o que
// "intacto" quer dizer.
func storedDamage(
	ctx context.Context, q *sqlcgen.Queries, id int64,
) (sqlcgen.GetCharacterDamageRow, error) {
	damage, err := q.GetCharacterDamage(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return sqlcgen.GetCharacterDamageRow{}, nil
	}
	if err != nil {
		return damage, fmt.Errorf("reler o dano da ficha %d: %w", id, err)
	}
	return damage, nil
}
