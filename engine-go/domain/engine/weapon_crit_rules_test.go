package engine

import (
	"path/filepath"
	"testing"
)

// A MARGEM DE AMEAÇA E O MULTIPLICADOR CHEGAM À CARTA DA ARMA (ALE-411).
//
// Eles não chegavam, e o defeito é o da ALE-406 repetido: o número APARECE na
// aba Efeitos — o `effects_labels.go` tem rótulo para `critRange` e `critMult` —
// e a carta da arma copiava o valor cru do catálogo. Quem comprava a melhoria
// Precisa por 300 PO via "+1 margem de ameaça" na ficha e atacava com a margem
// de sempre.
//
// Ninguém tinha notado porque até a ALE-364 não havia resolução de ataque: o
// `critRange` era um número impresso na carta e nada o consumia. Hoje o
// `ResolveAttack` o lê, e o número inerte virou regra errada.
func TestTheThreatRangeImprovementReachesTheWeaponCard(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	var oracle struct {
		Char Character `json:"char"`
	}
	readJSON(t, filepath.Join(dir, "bardo-versatil-nv7.json"), &oracle)

	card := func(improvements string) WeaponCard {
		ch := oracle.Char
		ch.Items = []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: improvements,
		}}
		cards := BookRuleset(catalogs).ComputeWeaponCards(ch, map[string]bool{})
		if len(cards) == 0 {
			t.Fatal("a espada longa empunhada não virou carta")
		}
		return cards[0]
	}

	// O CONTROLE: sem melhoria, a espada longa é a do catálogo (19/x2).
	if base := card("[]"); base.CritRange != 19 || base.CritMult != 2 {
		t.Fatalf("o controle já estava errado: a espada longa nua deu %d/x%d e o "+
			"catálogo diz 19/x2", base.CritRange, base.CritMult)
	}

	// "A margem de ameaça aumenta em 1 ponto" (p166). A margem AUMENTA quando o
	// número DESCE: 19-20 vira 18-20, que é um resultado natural a mais.
	if got := card(`["melhoria-precisa"]`).CritRange; got != 18 {
		t.Errorf("a espada longa Precisa ameaça em %d e a p166 diz 18 (19 menos 1 ponto "+
			"de margem).\n19 quer dizer que o modificador de `critRange` não chega à "+
			"carta — ele aparece na aba Efeitos e o ataque nunca o vê", got)
	}
	// "O multiplicador de crítico da arma aumenta em 1 ponto" (p165).
	if got := card(`["melhoria-macica"]`).CritMult; got != 3 {
		t.Errorf("a espada longa Maciça multiplica por %d e a p165 diz 3 (x2 mais 1)", got)
	}
}
