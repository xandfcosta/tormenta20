package character

import (
	"context"
	"fmt"

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
//	teste, err := plays.RollExpertise(ctx, row, "Atletismo", nil)
func (p Plays) RollExpertise(
	ctx context.Context, row sqlcgen.Character, name string, d20 *int,
) (engine.SkillTest, error) {
	computed, err := sheet.LoadAndCompute(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return engine.SkillTest{}, fmt.Errorf("computar a ficha de %s: %w", row.Name, err)
	}
	modifier, found := expertiseTotalOf(computed, name)
	if !found {
		return engine.SkillTest{}, fmt.Errorf(
			"%s não tem a perícia %q na ficha", row.Name, name)
	}
	rolled, err := p.dieFor(d20)
	if err != nil {
		return engine.SkillTest{}, err
	}
	return engine.ResolveSkillTest(modifier, rolled)
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
