package character

import (
	"context"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// OS DOIS GESTOS QUE SÓ REGRAVAM OS POÇOS, sem cobrar nem curar nada.
//
// Eles existem porque um passo de atributo e um degrau de nível mudam o MÁXIMO
// sem que ninguém tenha apanhado: o que precisa acontecer é a ponte de colunas
// acompanhar o derivado. Quem decide o atual é o `sheet.ApplyToPools`.

// FillPools deixa o personagem com os poços cheios — o gesto do NASCIMENTO.
func (b Births) FillPools(ctx context.Context, c sqlcgen.Character) error {
	_, err := sheet.FillPools(ctx, b.queries, b.catalogs, c)
	return err
}

// RefreshPools regrava os poços preservando o que já foi gasto.
//
// É o passo de atributo: a Constituição mexe no PV máximo (p34) e quem apanhou
// não se cura por mexer na ficha. Encher aqui seria uma bomba de cura de dois
// cliques — o `−` num atributo é sempre aceito dentro da faixa e o `+` devolve o
// ponto, então uma Lenda com 3 de 180 PV sairia com 180.
func (b Births) RefreshPools(ctx context.Context, c sqlcgen.Character) error {
	_, err := sheet.RefreshPools(ctx, b.queries, b.catalogs, c)
	return err
}

// SetHpTo crava o PV atual num valor exato.
//
// Tem UM chamador de verdade — o gerador do elenco de teste, que precisa de
// ficha machucada para a tela ter o que mostrar. É o único gesto que declara um
// TOTAL em vez de um passo, e por isso não drena PV temporários: drenar é regra
// de pancada, e isto aqui não é uma pancada.
func (p Plays) SetHpTo(ctx context.Context, c sqlcgen.Character, atual int64) error {
	_, err := sheet.ApplyToPools(ctx, p.queries, p.catalogs, c,
		func(pocos sheet.Pools) (sheet.Pools, error) {
			pocos.HpCurrent = atual
			return pocos, nil
		})
	return err
}
