package character

import (
	"context"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// ROLAR UM TESTE DE PERÍCIA (p220-221).
//
// O MODIFICADOR NÃO É CALCULADO AQUI: ele é o TOTAL que a ficha computada já
// traz, com treino, atributo, penalidade de armadura e tudo o mais que a
// decomposição mostra. Recalculá-lo seria a segunda definição que o oráculo das
// 18 fichas existe para impedir — e seria a que diverge, porque a da ficha é a
// que alguém olha.

// RollExpertise rola 1d20 + o valor da perícia, e devolve a conta inteira.
//
// O D20 É OPCIONAL, e os dois caminhos são de primeira classe (decisão do dono):
// nulo é o servidor rolar, e um valor é o que a mesa rolou no dado de verdade e
// informou. É o mesmo desenho do `combat.Request.D20`, e a razão é que as duas
// mesas existem.
//
//	teste, err := plays.RollExpertise(ctx, row, "Atletismo", nil, nil)
func (p Plays) RollExpertise(
	ctx context.Context, row sqlcgen.Character, name string, d20 *int, spends []book.SkillSpendChoice,
) (engine.SkillTest, error) {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return engine.SkillTest{}, fmt.Errorf("carregar a ficha de %s: %w", row.Name, err)
	}
	computed, err := sheet.Compute(dto)
	if err != nil {
		return engine.SkillTest{}, fmt.Errorf("computar a ficha de %s: %w", row.Name, err)
	}
	modifier, found := expertiseTotalOf(computed, name)
	if !found {
		return engine.SkillTest{}, fmt.Errorf(
			"%s não tem a perícia %q na ficha", row.Name, name)
	}
	// OS GASTOS são conferidos contra o que a ficha OFERECE para esta perícia
	// (ALE-489): a tela só desenha o que vale, e a fronteira é aqui.
	paid, err := book.PaySkillSpends(SkillSpendOffersFor(dto, computed, name), spends)
	if err != nil {
		return engine.SkillTest{}, err
	}
	// SÓ PARA TREINADOS, sem treino, não se rola (p115) — a não ser que um gasto
	// pago destrave (Eclético, Cultura Exótica). A tela já escondia o botão;
	// recusar aqui é a fronteira.
	if trainedOnly(name) && !isTrained(computed, name) && !paid.Unlocked {
		return engine.SkillTest{}, fmt.Errorf("%s só se rola com treinamento (p115), e %s não é treinado nela",
			name, row.Name)
	}
	if paid.PM > int(dto.ManaAvailable()) {
		return engine.SkillTest{}, fmt.Errorf("os gastos do teste pedem %d PM, e %s tem %d",
			paid.PM, row.Name, dto.ManaAvailable())
	}
	rolled, err := p.dieFor(d20)
	if err != nil {
		return engine.SkillTest{}, err
	}
	test, err := engine.ResolveSkillTest(modifier+paid.Bonus, rolled)
	if err != nil {
		return engine.SkillTest{}, err
	}
	for _, part := range paid.Parts {
		test.Spends = append(test.Spends, engine.SkillTestSpend{Name: part.Name, Bonus: part.Bonus, PM: part.PM})
	}
	// O PM SAI AO ROLAR (decisão do dono): o teste de perícia não tem
	// confirmação do mestre, e o número com o bônus já é o que aconteceu.
	if paid.PM > 0 {
		if err := p.inTx(ctx, "pagar o teste de "+name, func(q *sqlcgen.Queries) error {
			return chargeMp(ctx, q, p.catalogs, row, paid.PM)
		}); err != nil {
			return engine.SkillTest{}, err
		}
	}
	return test, nil
}

// isTrained diz se a perícia é treinada na ficha computada.
func isTrained(computed engine.ComputedSheet, name string) bool {
	for _, e := range computed.Expertises {
		if e.Name == name {
			return e.Trained
		}
	}
	return false
}

// dieFor devolve o d20: o que veio da mesa, ou um que o servidor rola.
//
// A CONFERÊNCIA do recebido é do `ResolveSkillTest`, e não se repete aqui — uma
// segunda validação do mesmo número é a que fica para trás quando a faixa muda.
func (p Plays) dieFor(given *int) (int, error) {
	if given != nil {
		return *given, nil
	}
	if p.rollDie == nil {
		return 0, fmt.Errorf("este caso de uso foi montado sem dado: não há como rolar")
	}
	return p.rollDie(20)
}

// expertiseTotalOf acha o valor da perícia na ficha COMPUTADA, pelo nome que a
// tela mostra.
//
// Pelo NOME e não por um id, porque é o nome que a perícia inventada tem: o
// Ofício que o jogador criou não está em catálogo nenhum, e um id o deixaria de
// fora da única parte do gesto que lhe interessa.
func expertiseTotalOf(computed engine.ComputedSheet, name string) (int, bool) {
	for _, pericia := range computed.Expertises {
		if pericia.Name == name {
			return pericia.Total, true
		}
	}
	return 0, false
}
