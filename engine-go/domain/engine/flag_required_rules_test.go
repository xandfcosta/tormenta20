package engine

import (
	"path/filepath"
	"testing"
)

// O MOTOR PRECISA SABER EXIGIR A FLAG, e não só calar por causa dela (ALE-442).
//
// A assimetria que abriu esta fatia:
//
//	Pele de Ferro (p62)  "+4 na Defesa se NÃO estiver usando armadura pesada"
//	Encouraçado (p125)   "se estiver usando uma armadura pesada, +2 na Defesa"
//
// A primeira já se escrevia com `flagOff`; a segunda não tinha como.
//
// # E por que não é o `flagOn`
//
// Porque ele é OPT-IN, e a flag dele é POSTURA — a Fúria, a Inspiração, as seis
// do cavaleiro —, que a pessoa liga na aba Efeitos. A `armadura-pesada` é
// DERIVADA do que está vestido: oferecer um interruptor para ela seria
// perguntar ao jogador o que a ficha já sabe.
func TestTheModifierThatRequiresAFlagOnlyAppliesWhileItIsRaised(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	comArmadura := func(item string) ItemEffects {
		ch := Character{
			Level: 6, ClassPowers: `["encouracado","inexpugnavel"]`,
			Items: []CharacterItem{{
				CatalogID: ptr(item), Name: item, Equipped: ptr("vested"), Improvements: "[]",
			}},
		}
		return ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(ch))
	}

	// COM ARMADURA PESADA: os dois poderes valem. +2 na Defesa (p125) e +2 em
	// todos os testes de resistência (p128).
	pesada := comArmadura("armadura-completa")
	if !pesada.Flags["armadura-pesada"] {
		t.Fatalf("o controle já estava errado: a armadura completa não levantou a flag")
	}
	if got := StatFor(pesada, ModifierTarget{K: "defense"}).Total; got != 12 {
		t.Errorf("a Defesa somou %d: a armadura completa dá +10 e o Encouraçado +2 (p125)", got)
	}
	if got := StatFor(pesada, ModifierTarget{K: "resistance"}).Total; got != 2 {
		t.Errorf("a resistência somou %d e o Inexpugnável dá +2 (p128)", got)
	}

	// COM ARMADURA LEVE: nenhum dos dois vale, e é esta metade que o `flagOn`
	// não conseguiria entregar — ele deixaria os dois como interruptor.
	leve := comArmadura("armadura-couro")
	if leve.Flags["armadura-pesada"] {
		t.Fatalf("o controle já estava errado: a armadura de couro levantou a flag de pesada")
	}
	if got := StatFor(leve, ModifierTarget{K: "defense"}).Total; got != 2 {
		t.Errorf("a Defesa somou %d: a armadura de couro dá +2 e o Encouraçado NÃO vale "+
			"sem armadura pesada", got)
	}
	if got := StatFor(leve, ModifierTarget{K: "resistance"}).Total; got != 0 {
		t.Errorf("a resistência somou %d sem armadura pesada, e o Inexpugnável exige uma", got)
	}
	// E NEM VIRAM INTERRUPTOR: um `flagOn` os ofereceria na aba Efeitos, e a
	// pessoa ligaria um bônus que ela não tem.
	for _, c := range leve.Conditional {
		if c.SourceID == "encouracado" || c.SourceID == "inexpugnavel" {
			t.Errorf("%s virou opt-in, e usar armadura pesada não é escolha de tela: "+
				"é o que está vestido", c.SourceID)
		}
	}
}
