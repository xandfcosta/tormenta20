package character

import (
	"context"
	"fmt"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

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

// SpreadAttributes grava os seis atributos base de uma vez.
//
// Os SEIS juntos e nunca um: a compra de pontos (p17) é um espalhamento, e uma
// escrita parcial seria um espalhamento que ninguém escolheu — é a mesma razão
// escrita na própria query.
//
// Ela é do NASCIMENTO e não de um gesto de ficha viva: distribuir atributos é a
// segunda cena da forja, e quem chega aqui é um herói que ainda está sendo
// feito. A RECUSA da compra de pontos não é daqui — o motor já a decide antes,
// e repeti-la aqui daria duas respostas para a mesma pergunta (ALE-359).
func (b Births) SpreadAttributes(ctx context.Context, id int64, espalhamento map[string]int) error {
	if err := b.queries.SetCharacterAttributes(ctx, sqlcgen.SetCharacterAttributesParams{
		Strength: int64(espalhamento["strength"]), Dexterity: int64(espalhamento["dexterity"]),
		Constitution: int64(espalhamento["constitution"]), Intelligence: int64(espalhamento["intelligence"]),
		Wisdom: int64(espalhamento["wisdom"]), Charisma: int64(espalhamento["charisma"]),
		UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		return fmt.Errorf("gravar o espalhamento da ficha %d: %w", id, err)
	}
	return nil
}
