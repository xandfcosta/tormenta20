package sheetui

import (
	"strings"

	"t20engine/serve/web/ds"
)

// A ABA COMBATE DA FICHA PARALELA (ALE-473). O dado é o da ficha de hoje
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
			Penalty: strings.HasPrefix(row.Value, "-"),
		})
	}
	return terms
}
