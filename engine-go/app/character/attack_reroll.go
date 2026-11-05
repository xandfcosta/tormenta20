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

// AttackRerollOffer acha o gasto `key` na ficha e confere o PM dele, para recusar antes de
// rolar. Quem cobra é o `session.Store.RerollAttack`, na transação que troca o dado.
func (p Plays) AttackRerollOffer(
	ctx context.Context, row sqlcgen.Character, melee bool, weaponID, key string,
) (book.SkillSpendOffer, error) {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return book.SkillSpendOffer{}, fmt.Errorf("carregar a ficha de %s: %w", row.Name, err)
	}
	offers := AttackRerollOffersFor(dto, melee, weaponID)
	at := slices.IndexFunc(offers, func(o book.SkillSpendOffer) bool { return o.Key == key })
	if at < 0 {
		return book.SkillSpendOffer{}, fmt.Errorf("%s não tem %q para rolar de novo este ataque", row.Name, key)
	}
	offer := offers[at]
	if offer.PM > int(dto.ManaAvailable()) {
		return book.SkillSpendOffer{}, fmt.Errorf("%s pede %d PM, e %s tem %d", offer.Name, offer.PM, row.Name, dto.ManaAvailable())
	}
	return offer, nil
}
