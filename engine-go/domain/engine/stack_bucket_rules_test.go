package engine

import "testing"

// AS DUAS PILHAS DIZEM O MESMO SOBRE QUEM DISPUTA COM QUEM (p226): a das parcelas — que junta as pilhas das
// perícias com a resistência — lê o mesmo balde que a do mundo (`stackBucket`). O aumento do item disputa só
// dentro do item; o mesmo tipo em duas fontes disputa; o `untyped` soma sempre.
func TestThePartsPileReadsTheSameBucketAsTheWorld(t *testing.T) {
	for _, c := range []struct {
		why   string
		parts []Contribution
		want  int
	}{
		{"o aumento do item em dois itens soma", []Contribution{
			{SourceID: "armadura-completa", BonusType: "enhancement", Amount: 1, RaisesItem: true},
			{SourceID: "escudo-pesado", BonusType: "enhancement", Amount: 1, RaisesItem: true},
		}, 2},
		{"o aumento do item no mesmo item disputa", []Contribution{
			{SourceID: "escudo-pesado", BonusType: "enchantment", Amount: 2, RaisesItem: true},
			{SourceID: "escudo-pesado", BonusType: "enchantment", Amount: 4, RaisesItem: true},
		}, 4},
		{"o mesmo tipo em duas fontes disputa", []Contribution{
			{SourceID: "anel", BonusType: "enhancement", Amount: 1},
			{SourceID: "manto", BonusType: "enhancement", Amount: 2},
		}, 2},
		{"o untyped soma sempre", []Contribution{
			{SourceID: "anel", BonusType: "untyped", Amount: 1},
			{SourceID: "anel", BonusType: "untyped", Amount: 1},
		}, 2},
	} {
		if got := resolveStack(c.parts).Total; got != c.want {
			t.Errorf("%s: a pilha das parcelas deu %+d; want %+d", c.why, got, c.want)
		}
	}
}
