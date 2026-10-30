package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// ROLAR DE NOVO O TESTE DE PERÍCIA (p226, ALE-507): "Efeitos que permitem que
// você role novamente o dado devem ser usados antes de o mestre declarar se
// você passou ou não no teste (e você deve ficar com o segundo valor rolado,
// mesmo que seja pior que o primeiro)." A janela é da mesa; aqui se confere a
// oferta, cobra o PM e rola o dado novo.

// RerollOffersFor são os "rolar de novo" que esta ficha tem para a perícia
// rolada.
//
//	offers := character.RerollOffersFor(dto, "Diplomacia") // [Etiqueta 1 PM]
func RerollOffersFor(dto sheet.CharacterDTO, skill string) []book.SkillSpendOffer {
	return rerollOffersOf(dto, book.SkillSpendTarget{Skill: skill, Flags: sheetFlags(dto)})
}

// rerollOffersOf são as ofertas das ativações de rolar de novo que a ficha
// POSSUI, para o teste do alvo.
func rerollOffersOf(dto sheet.CharacterDTO, target book.SkillSpendTarget) []book.SkillSpendOffer {
	owned, index := []book.Activation{}, sheet.OwnedIndexOf(dto)
	for _, a := range book.Activations() {
		if a.SkillSpend != nil && a.SkillSpend.Kind == book.SpendReroll && index.Has(a) {
			owned = append(owned, a)
		}
	}
	if len(owned) == 0 {
		return nil
	}
	return book.RerollOffers(owned, target)
}

// RerollSkillTest rola o teste de novo com o gasto `key`, cobra o PM e devolve
// o teste com o dado NOVO valendo. O d20 da mesa vem em `d20`, ou o servidor
// rola.
func (p Plays) RerollSkillTest(
	ctx context.Context, row sqlcgen.Character, test live.SkillTestRoll, key string, d20 *int,
) (live.SkillTestRoll, error) {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return test, fmt.Errorf("carregar a ficha de %s: %w", row.Name, err)
	}
	skill := test.Skill
	if test.UsedSkill != "" {
		skill = test.UsedSkill
	}
	offers := RerollOffersFor(dto, skill)
	at := slices.IndexFunc(offers, func(o book.SkillSpendOffer) bool { return o.Key == key })
	if at < 0 {
		return test, fmt.Errorf("%s não tem %q para rolar de novo um teste de %s", row.Name, key, skill)
	}
	offer := offers[at]
	if offer.PM > int(dto.ManaAvailable()) {
		return test, fmt.Errorf("%s pede %d PM, e %s tem %d", offer.Name, offer.PM, row.Name, dto.ManaAvailable())
	}
	rolled, err := p.dieFor(d20)
	if err != nil {
		return test, err
	}
	resolved, err := engine.ResolveSkillTest(test.Modifier+offer.Bonus, rolled)
	if err != nil {
		return test, err
	}
	reroll := live.DieReroll{Name: offer.Name, PM: offer.PM, Discarded: test.Roll, Bonus: offer.Bonus}
	test.Roll, test.Modifier, test.Total = rolled, test.Modifier+offer.Bonus, resolved.Total
	test.Natural20, test.Natural1, test.Rolls = resolved.Natural20, resolved.Natural1, nil
	// "Qualquer resultado 11 ou mais na segunda rolagem será considerado um 20
	// natural" (Sorte de Nimb, p48).
	if offer.NaturalFrom > 0 && rolled >= offer.NaturalFrom && !test.Natural20 {
		test.Natural20, test.Natural1, reroll.Natural = true, false, true
	}
	test.ByHand = d20 != nil
	test.Rerolls = append(slices.Clone(test.Rerolls), reroll)
	if offer.PM > 0 {
		if err := p.inTx(ctx, "pagar o rolar de novo de "+skill, func(q *sqlcgen.Queries) error {
			return chargeMp(ctx, q, p.catalogs, row, offer.PM)
		}); err != nil {
			return test, err
		}
	}
	return test, nil
}
