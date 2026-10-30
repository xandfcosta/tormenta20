package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/serve/web/ds"
)

// A ABA COMBATE DA FICHA (ALE-473). O dado é o do painel
// (`panelCombat`); daqui sai só o que a linha da conta lê diferente da caixa.
// As resistências não entram — moram no topo das Perícias —, e não há
// "Atacar": na ficha solta não há alvo nem turno (decisões do dono).

// combatSumHead é a linha fechada de uma caixa do Combate. A caixa abrevia
// ("Atq CaC") porque não cabia mais; a linha tem espaço para o nome inteiro, e
// a perícia que o ataque usa vira a nota.
func combatSumHead(tile statTile) ds.SumHead {
	head := ds.SumHead{Title: tile.Title, Hint: tile.Sub, Total: tile.Value}
	switch tile.Key {
	case "attack-melee":
		head.Title, head.Hint = "Ataque corpo a corpo", "Luta"
	case "attack-ranged":
		head.Title, head.Hint = "Ataque à distância", "Pontaria"
	case "spell-dc":
		head.Title, head.Hint = "CD das magias", "dos testes de resistência"
	}
	return head
}

// weaponSumHead é a linha da arma: o ataque à direita, e o dano e o crítico na
// nota, que é a fórmula que a mesa pergunta junto.
//
//	weaponSumHead(espada) // {Espada longa, "Luta · 1d8+4 · 19-20/x2", +7}
func weaponSumHead(w weaponTile) ds.SumHead {
	return ds.SumHead{Title: w.Name, Hint: strings.Join([]string{w.Skill, w.Damage, w.Crit}, " · "), Total: w.Attack}
}

// sumTermsOf traduz as linhas da decomposição de hoje para as parcelas da
// conta. A penalidade é a parcela que TIRA, e é a única que ganha cor.
func sumTermsOf(rows []breakdownRow) []ds.SumTerm {
	terms := make([]ds.SumTerm, 0, len(rows))
	for _, row := range rows {
		terms = append(terms, ds.SumTerm{
			Label: row.Label, Note: row.Note, Value: row.Value,
			Penalty: strings.HasPrefix(row.Value, "-"), Muted: row.Muted,
		})
	}
	return terms
}

// weaponSumTerms são as duas contas de uma arma, na língua das Perícias: o
// ataque, com o porquê de a Destreza ter entrado quando entrou, e o dano, com
// o atributo que de fato soma nele. As duas fecham no número da arma: o que
// sobra sem nome vira "Outros", como em `expertiseSumTerms`.
func weaponSumTerms(card engine.WeaponCard) (attack, damage []ds.SumTerm) {
	attack = expertiseSumTerms(card.Expertise)
	if note := dexNote(card); note != "" {
		attack[0].Note = note
	}
	attack = withContributions(attack, card.Attack-card.Expertise.Total, card.AttackAll.Contributions, card.AttackOwn.Contributions)

	bonus := card.DamageBonus
	if card.Skill == "Luta" {
		strength := ds.SumTerm{Label: "Força", Value: book.WithSign(card.StrDamage)}
		if card.DexDamage {
			strength = ds.SumTerm{Label: "Destreza", Note: "Acuidade com Arma", Value: book.WithSign(card.StrDamage)}
		}
		damage = append(damage, strength)
		bonus -= card.StrDamage
	}
	damage = withContributions(damage, bonus, card.DamageAll.Contributions, card.DamageOwn.Contributions)
	return attack, damage
}

// withContributions acrescenta as parcelas com nome que somam `total`, e o
// resto, se houver, como "Outros".
func withContributions(terms []ds.SumTerm, total int, groups ...[]engine.BreakdownContribution) []ds.SumTerm {
	shown := 0
	for _, group := range groups {
		for _, c := range group {
			terms = append(terms, ds.SumTerm{Label: c.Source, Note: c.Note, Value: book.WithSign(c.Amount), Penalty: c.Amount < 0})
			shown += c.Amount
		}
	}
	if rest := total - shown; rest != 0 {
		terms = append(terms, ds.SumTerm{Label: "Outros", Value: book.WithSign(rest), Penalty: rest < 0})
	}
	return terms
}

// dexNote diz por que a Destreza entrou no ataque: a arma que o livro deixa
// usar assim (a adaga, com a página dela) ou a Acuidade com Arma.
func dexNote(card engine.WeaponCard) string {
	switch card.DexBy {
	case engine.DexByWeapon:
		note := card.Name + ": Destreza no lugar de Força"
		if item := book.ItemByID(card.CatalogID); item != nil && item.BookPage != 0 {
			note += " (p" + strconv.Itoa(item.BookPage) + ")"
		}
		return note
	case engine.DexByFinesse:
		return "Acuidade com Arma: Destreza no lugar de Força"
	}
	return ""
}
