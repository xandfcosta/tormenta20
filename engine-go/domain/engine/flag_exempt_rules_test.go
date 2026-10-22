package engine

import (
	"path/filepath"
	"testing"
)

// UMA FONTE PODE ISENTAR O PORTADOR DE UMA BANDEIRA (p340, ALE-449).
//
// A Cota élfica é uma *"cota de malha defensora de mitral"* — e a cota de malha
// é armadura PESADA, que acende a bandeira que zera a Destreza na Defesa
// (p152). O verbete desfaz exatamente isso, e só isso:
//
//	"Ela permite que você aplique sua Destreza na Defesa como se fosse uma
//	 armadura leve."
//
// Até aqui uma composição só ACRESCENTAVA: ela herda os modificadores da base
// inteiros, e não havia como dizer "este item não acende aquela bandeira". O
// alvo `flagExempt` é isso, e ele é da família do `armorPenaltyExempt` (p338),
// que já dizia "esta fonte isenta o portador de uma penalidade".
//
// # O que a isenção NÃO desfaz
//
// O resto da armadura pesada fica: a penalidade de armadura, o deslocamento
// reduzido, a fadiga ao dormir e a própria bandeira `armadura-pesada`. O livro
// diz "como se fosse leve" sobre a DESTREZA, e não sobre o item.
func TestTheExemptionErasesTheFlagTheBaseRaises(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	vestindo := func(item string) ComputedSheet {
		return BookRuleset(catalogs).ComputeSheet(Character{
			Level: 5, Dexterity: 3,
			Proficiencies: `["armaduras-pesadas"]`,
			Items: []CharacterItem{{
				CatalogID: ptr(item), Name: item, Equipped: ptr("vested"),
				Improvements: "[]",
			}},
		}, map[string]bool{})
	}

	// O CONTROLE: a cota de malha crua bloqueia a Destreza — 10 + 6 da armadura.
	crua := vestindo("cota-malha")
	if crua.Defense.Attribute != "" || crua.Defense.Total != 16 {
		t.Fatalf("o controle já estava errado: a cota de malha deu Defesa %d com o "+
			"atributo %q, e ela é pesada (10 + 6, sem Destreza)",
			crua.Defense.Total, crua.Defense.Attribute)
	}

	// A COTA ÉLFICA: a Destreza entra. 10 + DES 3 + 6 da cota + 2 do Defensor.
	elfica := vestindo("cota-elfica")
	if elfica.Defense.Attribute != "dexterity" {
		t.Errorf("a Defesa da cota élfica aplicou %q: o verbete isenta a bandeira "+
			"que a base acende, e a Destreza volta", elfica.Defense.Attribute)
	}
	if elfica.Defense.Total != 21 {
		t.Errorf("a Defesa da cota élfica deu %d, e é 10 + DES 3 + 6 da cota + 2 "+
			"do Defensor", elfica.Defense.Total)
	}

	// E O RESTO DA ARMADURA PESADA FICA: a bandeira de peso continua no ar — é
	// ela que o Encouraçado (p125) exige e a Pele de Ferro (p62) teme —, e a
	// penalidade de armadura continua, com os 2 que o mitral devolve (p167).
	effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(Character{
		Level: 5, Dexterity: 3, Proficiencies: `["armaduras-pesadas"]`,
		Items: []CharacterItem{{
			CatalogID: ptr("cota-elfica"), Name: "Cota élfica",
			Equipped: ptr("vested"), Improvements: "[]",
		}},
	}))
	if !effects.Flags["armadura-pesada"] {
		t.Error("a cota élfica deixou de contar como armadura pesada: o livro troca " +
			"a regra da DESTREZA, e não o peso do item")
	}
	if effects.Flags["cannot-apply-dex-to-defense"] {
		t.Error("a bandeira isenta continuou no ar")
	}
	if penalty := StatFor(effects, ModifierTarget{K: "armorPenalty"}).Total; penalty != 0 {
		t.Errorf("a penalidade de armadura deu %d, e é −2 da cota de malha +2 do "+
			"mitral (p167)", penalty)
	}
}
