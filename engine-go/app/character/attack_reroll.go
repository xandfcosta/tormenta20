package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// ROLAR DE NOVO O ATAQUE (p226, ALE-508). O teste de ataque é de Luta no corpo
// a corpo e de Pontaria à distância (p230), e é por essa perícia que o
// Sortudo, a Sorte de Nimb e a Sorte dos Loucos o alcançam; o Mestre em Arma
// pede a arma que escolheu (p66). Quem rola de novo é o `combat.Strike`.

// AttackRerollOffersFor são os "rolar de novo" que esta ficha tem para o
// ataque com a arma `weaponID`.
//
//	offers := character.AttackRerollOffersFor(dto, true, "espada-longa") // [Mestre em Arma 2 PM]
func AttackRerollOffersFor(dto sheet.CharacterDTO, melee bool, weaponID string) []book.SkillSpendOffer {
	skill := "Pontaria"
	if melee {
		skill = "Luta"
	}
	return rerollOffersOf(dto, book.SkillSpendTarget{
		Skill: skill, Flags: sheetFlags(dto), PowerPicks: sheet.PowerPicks(dto.PowerChoices), WeaponID: weaponID,
	})
}

// PayAttackReroll confere que a ficha tem o gasto `key` e o PM dele, roda
// `reroll` com a oferta e só então cobra: o rolar de novo que o motor recusa
// não custa nada.
func (p Plays) PayAttackReroll(
	ctx context.Context, row sqlcgen.Character, melee bool, weaponID, key string,
	reroll func(book.SkillSpendOffer) error,
) error {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return fmt.Errorf("carregar a ficha de %s: %w", row.Name, err)
	}
	offers := AttackRerollOffersFor(dto, melee, weaponID)
	at := slices.IndexFunc(offers, func(o book.SkillSpendOffer) bool { return o.Key == key })
	if at < 0 {
		return fmt.Errorf("%s não tem %q para rolar de novo este ataque", row.Name, key)
	}
	offer := offers[at]
	if offer.PM > int(dto.ManaAvailable()) {
		return fmt.Errorf("%s pede %d PM, e %s tem %d", offer.Name, offer.PM, row.Name, dto.ManaAvailable())
	}
	if err := reroll(offer); err != nil {
		return err
	}
	return p.inTx(ctx, "pagar o rolar de novo do ataque", func(q *sqlcgen.Queries) error {
		return chargeMp(ctx, q, p.catalogs, row, offer.PM)
	})
}
