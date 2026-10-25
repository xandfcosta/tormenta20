package engine

import (
	"strings"
	"testing"
)

// O ITEM DO LIVRO PODE VIR COMPOSTO (p337, p340, ALE-445).
//
// Um item específico não é regra nova: ele é um item BASE com sobreposições já
// aplicadas, e o livro escreve quais em itálico no verbete —
//
//	"Este machado de guerra DE GELO ETERNO CONGELANTE FORMIDÁVEL fornece
//	 redução de fogo 10" (Avalanche, p337)
//
// — mais uma regra própria, que é o que sobra da frase.
//
// # A ficha não declara nada disso
//
// É o que estes casos prendem: o `CharacterItem` entra com `Enchants` vazio e
// `Improvements` em "[]", e mesmo assim a arma chega encantada. A composição
// mora no VERBETE, e a linha do personagem só aponta para ele — materializá-la
// na ficha copiaria o que o livro diz, e cópia não segue correção (a migração
// 00016 já nomeou esta armadilha).
func TestTheComposedWeaponBringsTheOverlaysTheBookWroteInItalics(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }

	ch := Character{
		Level: 3, Strength: 3,
		// PROFICIENTE, senão o −5 da p142 entra na mesma pilha de ataque e o
		// caso mediria a falta de treino junto com o encanto.
		Proficiencies: `["armas-marciais"]`,
		Items: []CharacterItem{{
			CatalogID: ptr("avalanche"), Name: "Avalanche",
			Equipped: ptr("wielded"), Improvements: "[]",
		}},
	}
	cards := wieldedCards(BookRuleset(catalogs).ComputeWeaponCards(ch, map[string]bool{}))
	if len(cards) != 1 {
		t.Fatalf("a Avalanche empunhada virou %d cartas de arma", len(cards))
	}
	card := cards[0]

	// O BLOCO DA BASE é herdado: machado de guerra, 1d12, margem 20, crítico x3
	// (p149). Sem ele a carta sairia sem dado, que é o sintoma de o verbete
	// específico não saber o que ele É.
	if card.Damage != "1d12" || card.CritRange != 20 || card.CritMult != 3 {
		t.Errorf("a carta saiu %s, margem %d, x%d — o machado de guerra da p149 é "+
			"1d12, margem 20, x3", card.Damage, card.CritRange, card.CritMult)
	}
	// O ENCANTO EM DADO vem da composição: a Congelante soma +1d6 de frio (p335).
	if len(card.ExtraDamage) != 1 || card.ExtraDamage[0].Dice != "1d6" ||
		card.ExtraDamage[0].Type != "frio" {
		t.Errorf("as parcelas extras são %+v, e a Congelante do verbete dá +1d6 de frio",
			card.ExtraDamage)
	}

	effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(ch))
	// O NÚMERO E A PROCEDÊNCIA. O +2 de ataque é da Formidável (p336), e quem
	// lê a decomposição tem de ver o nome DELA — "Avalanche +2" seria o número
	// certo com a origem errada, e é a diferença entre informar e só somar.
	attack := StatFor(effects, ModifierTarget{K: "attack", Scope: "this"})
	if attack.Total != 2 {
		t.Errorf("o ataque desta arma somou %d, e a Formidável dá +2 (p336)", attack.Total)
	}
	// A PROCEDÊNCIA viaja na NOTA, que é onde a ficha a mostra: a fonte é o
	// item que a pessoa empunha ("Avalanche") e a nota diz qual peça dele deu
	// o número. Sem ela a linha seria "Avalanche +2", que é o número certo com
	// a origem errada — e é a diferença entre informar e só somar.
	if len(attack.Contributions) != 1 ||
		!strings.Contains(attack.Contributions[0].Note, "Formidável") {
		t.Errorf("o +2 de ataque veio de %+v, e a nota dele tem de dizer Formidável",
			attack.Contributions)
	}
	// O dano soma DOIS encantamentos de baldes diferentes: +2 da Formidável
	// (enchantment) e +2 do Gelo eterno (enhancement), que a p333 manda somar.
	damage := StatFor(effects, ModifierTarget{K: "damage", Scope: "this"})
	if damage.Total != 4 {
		t.Errorf("o dano desta arma somou %d: +2 da Formidável e +2 do Gelo eterno, "+
			"que são baldes diferentes e a p333 manda somar", damage.Total)
	}
	// E A REGRA PRÓPRIA DO VERBETE, que não vem de sobreposição nenhuma: o
	// gelo eterno só dá redução de fogo a ARMADURA e ESCUDO (p166), e aqui
	// quem a dá é a Avalanche (p337).
	fire := StatFor(effects, ModifierTarget{K: "damageReduction", DamageType: "fogo"})
	if fire.Total != 10 {
		t.Errorf("a redução de fogo somou %d, e a Avalanche fornece 10 (p337)", fire.Total)
	}
}

// E DO LADO DA ARMADURA, com as quatro espécies de sobreposição.
//
// O Baluarte anão (p340) é "armadura completa REFORÇADA DEFENSORA DE ADAMANTE":
// uma melhoria, um encanto e um material sobre a base — e a regra própria é
// condicional, que é a terceira natureza que um verbete específico tem.
func TestTheComposedArmorStacksTheBaseTheImprovementAndTheEnchant(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }

	ch := Character{
		Level: 3, Dexterity: 2,
		Proficiencies: `["armaduras-pesadas"]`,
		Items: []CharacterItem{{
			CatalogID: ptr("baluarte-anao"), Name: "Baluarte anão",
			Equipped: ptr("vested"), Improvements: "[]",
		}},
	}
	sheet := BookRuleset(catalogs).ComputeSheet(ch, map[string]bool{})

	// 10 + 10 da armadura completa (p154) + 1 da Reforçada (p166) + 2 da
	// Defensora (p338). A Destreza não entra: a base é pesada, e o verbete
	// específico não muda isso.
	if sheet.Defense.Total != 23 {
		t.Errorf("a Defesa deu %d, e é 10 + 10 da armadura + 1 da Reforçada + 2 do "+
			"Defensor", sheet.Defense.Total)
	}
	if sheet.Defense.Attribute != "" {
		t.Errorf("a Defesa aplicou %q: a armadura completa que serve de base bloqueia "+
			"a Destreza (p152), e a composição não desfaz isso", sheet.Defense.Attribute)
	}
	// A PENALIDADE também soma as duas: −5 da armadura e −1 da Reforçada, que
	// é o preço que a melhoria cobra pelo +1 de Defesa (p166).
	effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(ch))
	if penalty := StatFor(effects, ModifierTarget{K: "armorPenalty"}).Total; penalty != -6 {
		t.Errorf("a penalidade de armadura deu %d, e é −5 da completa −1 da Reforçada",
			penalty)
	}
	// E O MATERIAL ESCOLHE A METADE CERTA: o adamante dá RD 5 em armadura
	// PESADA e RD 2 em leve (p166), e quem decide é a categoria da BASE.
	if rd := StatFor(effects, ModifierTarget{K: "damageReduction"}).Total; rd != 5 {
		t.Errorf("a RD deu %d, e o adamante em armadura pesada é 5 (p166)", rd)
	}
}
