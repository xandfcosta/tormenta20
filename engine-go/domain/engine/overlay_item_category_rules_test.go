package engine

import (
	"path/filepath"
	"testing"
)

// A SOBREPOSIÇÃO RAMIFICA PELO HOSPEDEIRO (p166).
//
// "Armadura e Escudo. Fornece redução de dano: armaduras leves e escudos, RD 2;
// armaduras pesadas, RD 5" — o adamante diz DOIS números, e qual deles vale
// depende da peça em que ele foi forjado.
//
// É filtro de COLETA e não condição de jogo: a pergunta já está respondida
// quando o item é lido, e uma condição sobreviveria até a resolução oferecendo
// um interruptor para o que ninguém escolhe.
func TestAnOverlayOnlyLandsOnTheHostFamilyItNames(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }
	adamante := "material-adamante"

	rd := func(item, equipped string) int {
		ch := Character{Items: []CharacterItem{{
			CatalogID: ptr(item), Name: item, Equipped: ptr(equipped),
			Improvements: "[]", Material: &adamante,
		}}}
		effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(ch))
		return StatFor(effects, ModifierTarget{K: "damageReduction"}).Total
	}

	// ARMADURA PESADA: 5. ARMADURA LEVE e ESCUDO: 2. São os números da p166, e
	// o par é o caso inteiro — um só deles passaria verde com o filtro que
	// aceita tudo.
	if got := rd("armadura-completa", "vested"); got != 5 {
		t.Errorf("a armadura completa de adamante dá RD %d e a p166 diz 5 para pesada", got)
	}
	if got := rd("armadura-couro", "vested"); got != 2 {
		t.Errorf("a armadura de couro de adamante dá RD %d e a p166 diz 2 para leve", got)
	}
	if got := rd("escudo-pesado", "wielded"); got != 2 {
		t.Errorf("o escudo pesado de adamante dá RD %d e a p166 diz 2 para escudo", got)
	}
	// E A ARMA NÃO GANHA RD NENHUMA: a metade de arma do verbete anda o dado
	// (ALE-435), e nada ali fala de redução.
	if got := rd("espada-longa", "wielded"); got != 0 {
		t.Errorf("a espada longa de adamante deu RD %d: a p166 não dá RD a arma nenhuma", got)
	}
}
