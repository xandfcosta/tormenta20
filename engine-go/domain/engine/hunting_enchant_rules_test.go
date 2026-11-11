package engine

import (
	"path/filepath"
	"slices"
	"testing"
)

// A CAÇADORA PERSEGUE O ALVO (p335): "anulando penalidades por camuflagem leve e total e por cobertura leve".
//
// Três linhas da Tabela 5-3 deixam de valer para a arma dela, e a COBERTURA TOTAL não está entre elas: o livro
// nomeia as três, e a total não é penalidade — ela proíbe o ataque (p239).

// hunting é a arma da `underCircumstances`, com a Caçadora.
func hunting() WeaponCard {
	card := aWeapon("1d8", 3, 20, 2, 5)
	for _, row := range []SpecialSituation{TargetUnderLightConcealment, TargetUnderTotalConcealment, TargetUnderLightCover} {
		card.IgnoresSituations = append(card.IgnoresSituations, IgnoredSituation{Row: row, By: "Caçadora"})
	}
	return card
}

func TestTheHuntingWeaponIgnoresLightCover(t *testing.T) {
	out, err := suggestedAttackUnder(hunting(), AttackTarget{Defense: 15},
		[]SpecialSituation{TargetUnderLightCover}, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Defense != 15 || !out.Hit {
		t.Errorf("a arma Caçadora sofreu a cobertura leve: Defesa %d, acerto %v; a p335 a anula (Defesa 15)",
			out.Defense, out.Hit)
	}
	// O MESTRE LÊ QUE A COBERTURA FOI ANULADA, e por quem: ele pintou a cobertura no tabuleiro, e uma linha que
	// a calasse o faria achar que o app a esqueceu.
	if !slices.Equal(out.Situations, []string{"cobertura leve anulada pela Caçadora"}) {
		t.Errorf("a linha de ataque diz %q; quer a cobertura leve anulada pela Caçadora", out.Situations)
	}
}

func TestTheHuntingWeaponIgnoresConcealment(t *testing.T) {
	for _, s := range []SpecialSituation{TargetUnderLightConcealment, TargetUnderTotalConcealment} {
		// O d10 = 1 desfaria o acerto em qualquer camuflagem; o `4` é o dano, e sem o d10 pedido ele é o único dado.
		out, err := suggestedAttackUnder(hunting(), AttackTarget{Defense: 15}, []SpecialSituation{s}, 10, fixedDice(t, 4))
		if err != nil {
			t.Fatalf("%s: resolver: %v", s, err)
		}
		if !out.Hit {
			t.Errorf("a arma Caçadora errou por %s, e a p335 anula a chance de falha", s)
		}
		if want := SpecialSituationLabel(s) + " anulada pela Caçadora"; !slices.Equal(out.Situations, []string{want}) {
			t.Errorf("a linha de ataque diz %q; quer %q", out.Situations, want)
		}
	}
}

func TestTheHuntingWeaponStillCannotHitThroughTotalCover(t *testing.T) {
	_, err := suggestedAttackUnder(hunting(), AttackTarget{Defense: 15},
		[]SpecialSituation{TargetUnderTotalCover}, 20, fixedDice(t))
	if err == nil {
		t.Fatal("a arma Caçadora atacou um alvo sob cobertura total: a p335 nomeia só a leve")
	}
}

// E A CAÇADORA É DA ARMA QUE A TEM: a adaga comum da outra mão sofre a cobertura como sempre.
func TestTheHuntingEnchantStaysOnItsWeapon(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }
	var oracle struct {
		Char Character `json:"char"`
	}
	readJSON(t, filepath.Join(mustWd(t), "..", "..", "parity", "bardo-versatil-nv7.json"), &oracle)
	ch := oracle.Char
	ch.Items = []CharacterItem{
		{CatalogID: ptr("espada-longa"), Name: "Espada longa", Equipped: ptr("wielded"), Improvements: "[]",
			Enchants: []string{"encanto-cacadora"}},
		{CatalogID: ptr("adaga"), Name: "Adaga", Equipped: ptr("wielded2"), Improvements: "[]"},
	}
	want := []SpecialSituation{TargetUnderLightConcealment, TargetUnderTotalConcealment, TargetUnderLightCover}
	seen := map[string]bool{}
	for _, card := range BookRuleset(catalogs).ComputeWeaponCards(ch, map[string]bool{}) {
		seen[card.Name] = true
		got := []SpecialSituation{}
		for _, ignored := range card.IgnoresSituations {
			got = append(got, ignored.Row)
			if ignored.By != "Caçadora" {
				t.Errorf("a carta %q diz que %s é anulada por %q; quer a Caçadora", card.Name, ignored.Row, ignored.By)
			}
		}
		slices.Sort(got)
		if card.Name == "Espada longa" && !slices.Equal(got, sortedSituations(want)) {
			t.Errorf("a espada Caçadora ignora %v; a p335 nomeia %v", got, want)
		}
		if card.Name != "Espada longa" && len(got) > 0 {
			t.Errorf("a carta %q ignora %v pela Caçadora da espada da outra mão", card.Name, got)
		}
	}
	if !seen["Espada longa"] || !seen["Adaga"] {
		t.Fatalf("a ficha não tem as duas cartas, e o caso não as mede (cartas: %v)", seen)
	}
}

func sortedSituations(s []SpecialSituation) []SpecialSituation {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}
