package engine

// UntilHitPowerID é o Até Acertar do lutador (p76).
const UntilHitPowerID = "class.lutador.ate-acertar"

// UntilHitCard é a carta do golpe desarmado com o bônus do Até Acertar: +2 por erro contra o
// mesmo oponente, no teste de ataque e no dano (p76). A parcela entra nas duas decomposições, para
// a conta da carta dizer de onde veio o número.
//
//	UntilHitCard(card, 2) // +4 no ataque e no dano
func UntilHitCard(card WeaponCard, misses int) WeaponCard {
	if misses <= 0 {
		return card
	}
	bonus := 2 * misses
	card.Attack += bonus
	card.AttackAll = withContribution(card.AttackAll, "Até Acertar", bonus)
	card.DamageBonus += bonus
	card.DamageAll = withContribution(card.DamageAll, "Até Acertar", bonus)
	return card
}

func withContribution(t TotalContribs, source string, amount int) TotalContribs {
	t.Total += amount
	t.Contributions = append(append([]BreakdownContribution{}, t.Contributions...), BreakdownContribution{Source: source, Amount: amount})
	return t
}
