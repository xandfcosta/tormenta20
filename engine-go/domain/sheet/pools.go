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
	c sqlcgen.Character, regra PoolRule,
) (Pools, error) {
	dto, err := Load(ctx, q, cat, c)
	if err != nil {
		return Pools{}, fmt.Errorf("carregar a ficha %d para mexer nos poços: %w", c.ID, err)
	}
	return ApplyToLoadedPools(ctx, q, &dto, regra)
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
	ctx context.Context, q *sqlcgen.Queries, dto *CharacterDTO, regra PoolRule,
) (Pools, error) {
	antes := Pools{
		HpMax: dto.HpMax, HpCurrent: dto.HpCurrent,
		MpMax: dto.MpMax, MpCurrent: dto.MpCurrent,
	}
	depois, err := regra(antes)
	if err != nil {
		return antes, err
	}
	// O máximo da regra é descartado sem aviso de propósito: a alternativa seria
	// recusar quem devolvesse um máximo diferente, e isso convidaria o gesto a
	// tentar. O poço é do catálogo, ponto.
	depois.HpMax, depois.MpMax = antes.HpMax, antes.MpMax
	depois.HpCurrent = WithinPool(depois.HpCurrent, depois.HpMax)
	depois.MpCurrent = WithinPool(depois.MpCurrent, depois.MpMax)
	dto.HpMax, dto.HpCurrent = depois.HpMax, depois.HpCurrent
	dto.MpMax, dto.MpCurrent = depois.MpMax, depois.MpCurrent
	return depois, savePools(ctx, q, dto.ID, depois)
}

// RefreshPools regrava os poços SEM mexer no que foi gasto.
//
// É o que um passo de atributo e um degrau de nível fazem. O `ShiftedByNewMax`
// vivia aqui: ele somava ao atual o mesmo delta do máximo, e precisava do máximo
// ANTIGO para calcular o delta. Guardando o DANO a conta some — quem apanhou 16
// continua devendo 16, e o atual acompanha o teto sozinho, nos dois sentidos.
func RefreshPools(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, c sqlcgen.Character,
) (Pools, error) {
	return ApplyToPools(ctx, q, cat, c, func(p Pools) (Pools, error) { return p, nil })
}

// FillPools apaga o que foi gasto: o personagem volta aos poços cheios.
//
// Tem um chamador, o NASCIMENTO, e é o encher à força que a prende ali — chamá-la
// de um gesto de ficha viva é uma bomba de cura.
func FillPools(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, c sqlcgen.Character,
) (Pools, error) {
	return ApplyToPools(ctx, q, cat, c, func(p Pools) (Pools, error) {
		p.HpCurrent, p.MpCurrent = p.HpMax, p.MpMax
		return p, nil
	})
}

// WithinPool prende um vital entre zero e o teto.
//
// Eram cinco grafias: duas funções com nome próprio, que saíram com o
// `app/character/vitals.go`, e três `min(max(…))` escritos à mão nos gestos. O
// `live.ClampVital` NÃO entrou nesta conta e continua onde
// está: ele prende a entrada do RASTREADOR, cujo máximo é opcional porque um NPC
// pode não ter nenhum. Poço de ficha sempre tem teto.
func WithinPool(valor, teto int64) int64 { return min(max(int64(0), valor), teto) }

// savePools grava o que de fato é ESTADO: o quanto se apanhou.
//
// A ausência de linha quer dizer INTACTO (migração 00014), então um personagem
// que volta ao cheio perde o registro em vez de ganhar um par de zeros — é o que
// mantém a tabela do tamanho de quem está machucado.
//
// # Por que ela relê a ficha antes de gravar
//
// Para não escrever quando nada mudou. O `updatedAt` de `characters` É a versão
// da ficha (`sheetui/view.go` e `table/stream.go` o servem como tal), então um
// `UPDATE` que não muda número nenhum manda a tela inteira se repedir. A
// comparação tem de ser contra o que está GRAVADO e não contra o que o funil
// derivou: derivado contra derivado dá sempre igual, e foi exatamente assim que
// a sincronização de vitais parou de regravar o máximo e congelou a coluna de um
// elfo em 19 quando o poço dele já era 20 (ALE-355).
func savePools(ctx context.Context, q *sqlcgen.Queries, id int64, p Pools) error {
	hpDano := WithinPool(p.HpMax-p.HpCurrent, p.HpMax)
	mpGasto := WithinPool(p.MpMax-p.MpCurrent, p.MpMax)
	gravado, err := storedPools(ctx, q, id)
	if err != nil {
		return err
	}
	if gravado == (storedState{Pools: p, HpDamage: hpDano, MpSpent: mpGasto}) {
		return nil
	}
	if hpDano == 0 && mpGasto == 0 {
		if err := q.ClearCharacterDamage(ctx, id); err != nil {
			return fmt.Errorf("apagar o dano da ficha %d: %w", id, err)
		}
	} else if err := q.SaveCharacterDamage(ctx, sqlcgen.SaveCharacterDamageParams{
		Characterid: id, Hpdamage: hpDano, Mpspent: mpGasto,
	}); err != nil {
		return fmt.Errorf("gravar o dano da ficha %d: %w", id, err)
	}
	// A PONTE, e ela é temporária: as quatro colunas de `characters` deixaram de
	// ser a verdade e ainda são lidas por quem não montou o agregado — os cartões
	// do grupo e a entrada da iniciativa, que pegam a linha por JOIN. Enquanto
	// elas existirem, o funil as mantém iguais ao derivado; elas saem por
	// migração quando o último leitor sair (ALE-355).
	if err := q.SetCharacterVitals(ctx, sqlcgen.SetCharacterVitalsParams{
		HpMax: p.HpMax, HpCurrent: p.HpCurrent,
		MpMax: p.MpMax, MpCurrent: p.MpCurrent,
		UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		return fmt.Errorf("espelhar os poços da ficha %d: %w", id, err)
	}
	return nil
}

// storedState é o que o banco tem hoje: as quatro colunas-ponte e o dano.
type storedState struct {
	Pools
	HpDamage int64
	MpSpent  int64
}

func storedPools(ctx context.Context, q *sqlcgen.Queries, id int64) (storedState, error) {
	c, err := q.GetCharacter(ctx, id)
	if err != nil {
		return storedState{}, fmt.Errorf("reler a ficha %d antes de gravar os poços: %w", id, err)
	}
	estado := storedState{Pools: Pools{
		HpMax: c.Hpmax, HpCurrent: c.Hpcurrent, MpMax: c.Mpmax, MpCurrent: c.Mpcurrent,
	}}
	dano, err := q.GetCharacterDamage(ctx, id)
	if err == nil {
		estado.HpDamage, estado.MpSpent = dano.Hpdamage, dano.Mpspent
	} else if !errors.Is(err, sql.ErrNoRows) {
		return storedState{}, fmt.Errorf("reler o dano da ficha %d: %w", id, err)
	}
	return estado, nil
}
