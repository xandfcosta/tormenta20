package api

import (
	"t20engine/book"
	"t20engine/engine"
)

// As LINHAS dos diálogos de decomposição (ALE-272, fatia 3).
//
// O motor é dono dos NÚMEROS; o que mora aqui são os rótulos estruturais — "½
// nível", "FOR", "Treino", "Base" — e a decisão de qual linha aparece. Elas
// vinham do `stat-rows.ts` da SPA, e vieram para cá pela mesma razão que a tabela
// de proficiências veio na fatia 2: um rótulo longe do número que ele explica
// diverge sem ninguém ver.

// defenseRows é a Defesa linha a linha.
//
// A DESTREZA BLOQUEADA por armadura pesada aparece como linha ZERADA e apagada,
// em vez de sumir. Sumir seria a resposta errada para a pergunta que o diálogo
// existe para responder: "por que minha Defesa está baixa?".
func defenseRows(sheet engine.ComputedSheetV2) []breakdownRow {
	rows := []breakdownRow{
		{Label: "Base", Value: book.WithSign(sheet.Defense.Base - dexInDefense(sheet))},
		dexterityRow(sheet),
	}
	rows = append(rows, rowsFromContributions(sheet.Defense.Contributions)...)
	return append(rows, directionalDefenseRows(sheet)...)
}

// dexInDefense é a Destreza que o MOTOR já embutiu no `Defense.Base`.
//
// Ela é subtraída porque o `Base` do motor e a linha "Base" da tela são coisas
// DIFERENTES, e confundi-las conta a Destreza duas vezes: o motor soma 10 + a
// Destreza num campo só, enquanto o diálogo mostra o 10 numa linha e a Destreza
// na seguinte. Somar as linhas tem de dar o total, e é isso que
// `TestTheDefenseRowsAddUpToTheTotal` prende — a subtração é exata porque o
// `effectiveAttribute` da defesa e o `Total` do atributo são a mesma expressão.
func dexInDefense(sheet engine.ComputedSheetV2) int {
	if !sheet.Defense.DexApplied {
		return 0
	}
	return sheet.Attributes["dexterity"].Total
}

func dexterityRow(sheet engine.ComputedSheetV2) breakdownRow {
	if !sheet.Defense.DexApplied {
		return breakdownRow{Label: "Destreza (bloqueada por armadura pesada)", Value: "+0", Muted: true}
	}
	return breakdownRow{Label: "Destreza", Value: book.WithSign(sheet.Attributes["dexterity"].Total)}
}

// directionalDefenseRows são as duas Defesas DIRECIONAIS, e elas só aparecem
// quando divergem da geral — hoje só o Caído as separa (p394: −5 contra corpo a
// corpo, +5 contra à distância). Quem está em pé vê a linha de sempre.
func directionalDefenseRows(sheet engine.ComputedSheetV2) []breakdownRow {
	d := sheet.Defense
	if d.VsMelee == d.Total && d.VsRanged == d.Total {
		return nil
	}
	return []breakdownRow{
		{Label: "Contra corpo a corpo", Value: book.WithSign(d.VsMelee - d.Total)},
		{Label: "Contra ataques à distância", Value: book.WithSign(d.VsRanged - d.Total)},
	}
}

// expertiseRows é um ataque ou uma resistência, linha a linha.
//
// `all` são os modificadores de escopo GLOBAL (Fúria e parentes) e ele é nulo
// para resistência, que não os recebe. O que vale para UMA arma fica de fora de
// propósito: ele já chega pela perícia, e somá-lo de novo o contaria duas vezes.
func expertiseRows(ex engine.ExpertiseBreakdown, all *engine.TotalContribs) []breakdownRow {
	rows := []breakdownRow{
		{Label: "½ nível", Value: book.WithSign(ex.HalfLevel)},
		{Label: attributeAbbr[ex.Attribute], Value: book.WithSign(ex.AttrValue)},
	}
	if ex.Training != 0 {
		rows = append(rows, breakdownRow{Label: "Treino", Value: book.WithSign(ex.Training)})
	}
	rows = append(rows, rowsFromContributions(ex.ItemContributions)...)
	if all != nil {
		rows = append(rows, rowsFromContributions(all.Contributions)...)
	}
	return rows
}

// weaponRows são as duas contas de uma arma: o ataque e o dano.
//
// A linha de FOR sai no dano de corpo a corpo e de arremesso (as duas rolam
// Luta) e NUNCA no de arma de disparo, onde o motor já devolve `StrDamage` zero.
func weaponRows(card engine.WeaponCard) (attack, damage []breakdownRow) {
	attack = expertiseRows(card.Expertise, &card.AttackAll)
	if card.Skill == "Luta" {
		damage = append(damage, breakdownRow{Label: "FOR", Value: book.WithSign(card.StrDamage)})
	}
	return attack, append(damage, rowsFromContributions(card.DamageAll.Contributions)...)
}

// pmLimitRows é o teto de PM por magia.
func pmLimitRows(sheet engine.ComputedSheetV2) []breakdownRow {
	rows := []breakdownRow{{Label: "Nível de conjurador", Value: book.WithSign(sheet.PmLimit.Base)}}
	return append(rows, rowsFromContributions(sheet.PmLimit.Contributions)...)
}

// spellDcRows é a CD das magias.
//
// A base vem do MOTOR — nível mais atributo-chave já resolvido —, e não de uma
// releitura do atributo cru: lido cru, o conjurador Osteon saía 1 abaixo.
func spellDcRows(sheet engine.ComputedSheetV2) []breakdownRow {
	base := 0
	if sheet.BestBaseSpellCd != nil {
		base = *sheet.BestBaseSpellCd
	}
	rows := []breakdownRow{{Label: "CD base (nível + atributo-chave)", Value: book.WithSign(base)}}
	if sheet.SpellDCBonus.Total == 0 {
		return append(rows, breakdownRow{Label: "Sem bônus de itens", Value: "+0", Muted: true})
	}
	return append(rows, rowsFromContributions(sheet.SpellDCBonus.Contributions)...)
}

// pmCostRows é o que desconta (ou encarece) o custo de uma magia.
func pmCostRows(sheet engine.ComputedSheetV2) []breakdownRow {
	if sheet.PmCostMod.Total == 0 {
		return []breakdownRow{{Label: "Sem mod de itens", Value: "+0", Muted: true}}
	}
	return rowsFromContributions(sheet.PmCostMod.Contributions)
}

// rowsFromContributions traduz a contribuição do motor numa linha de tela.
func rowsFromContributions(contributions []engine.BreakdownContribution) []breakdownRow {
	rows := make([]breakdownRow, 0, len(contributions))
	for _, c := range contributions {
		rows = append(rows, breakdownRow{Label: c.Source, Value: book.WithSign(c.Amount), Note: c.Note})
	}
	return rows
}

// rowsFromSourceAmounts é a mesma tradução para a contribuição SEM nota, que é a
// forma que a redução de dano usa.
func rowsFromSourceAmounts(sources []engine.SourceAmount) []breakdownRow {
	rows := make([]breakdownRow, 0, len(sources))
	for _, s := range sources {
		rows = append(rows, breakdownRow{Label: s.Source, Value: book.WithSign(s.Amount)})
	}
	return rows
}
