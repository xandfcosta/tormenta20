package engine

import (
	"path/filepath"
	"testing"
)

// O ITEM QUE O MUNDO NÃO CONHECE CONTRIBUI ZERO (ALE-389).
//
// Regra do dono:
//
//	"A única pessoa que pode modificar as regras do mundo é o Mestre […]
//	 Criação de itens por personagem é somente cosmético, não muda nada na
//	 ficha."
//
// O jogador controla três campos do item caseiro — nome, quantidade e espaços —
// e nenhum deles é modificador. Até aqui a regra valia por AUSÊNCIA: não há
// coluna de modificador em `character_items`, e o verbete não achado não
// contribui.
//
// # A ausência tinha um buraco, e ele é o das SOBREPOSIÇÕES
//
// O `ownItemMods` aplicava melhoria, material e encanto mesmo sem verbete: um
// item caseiro com `material-adamante` e `melhoria-reforcada` gravados entregava
// Defesa +1 e penalidade de armadura −1. A sobreposição é uma linha que o livro
// escreve SOBRE um item — sem o item, ela não tem a que se aplicar, e o
// `overlayModsFor` passava categoria vazia e deixava passar o que não é
// restrito a família nenhuma.
//
// # Por que o guarda mora no MOTOR e não só na cena
//
// A cena já recusa (`fitsItemImprovement`: "um item custom não recebe melhoria:
// ele não tem família no catálogo"), e essa recusa continua sendo a fronteira
// do GESTO. Mas linha antiga não passou por ela — é a mesma forma da ALE-450 —,
// e a regra do dono é sobre o que a FICHA soma, não sobre o que o formulário
// aceita. Travar na UI é UX; a fronteira é o servidor, e a conta é do motor.
func TestAnItemTheWorldDoesNotKnowContributesNothing(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }
	adamante := "material-adamante"

	// O CONTROLE: as MESMAS sobreposições sobre um verbete que o mundo conhece
	// entregam o que o livro manda. Sem ele, um motor que ignorasse toda
	// sobreposição passaria verde no caso de baixo.
	conhecido := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(Character{
		Level: 5,
		Items: []CharacterItem{{
			CatalogID: ptr("armadura-completa"), Name: "Armadura completa",
			Equipped: ptr("vested"), Improvements: `["melhoria-reforcada"]`, Material: &adamante,
		}},
	}))
	if StatFor(conhecido, ModifierTarget{K: "defense"}).Total == 0 {
		t.Fatalf("o controle já estava errado: a armadura completa reforçada de "+
			"adamante não deu Defesa nenhuma — %+v", conhecido.ByTarget)
	}

	// O ITEM CASEIRO: sem `catalogId`, nem a sobreposição vale.
	caseiro := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(Character{
		Level: 5,
		Items: []CharacterItem{{
			Name: "Tábua caseira", Equipped: ptr("vested"),
			Improvements: `["melhoria-reforcada"]`, Material: &adamante,
		}},
	}))
	if len(caseiro.ByTarget) != 0 {
		t.Errorf("o item caseiro entregou %d alvos à ficha, e a regra do dono diz que "+
			"ele é cosmético.\nA sobreposição é uma linha que o livro escreve SOBRE "+
			"um item; sem o item, ela não tem a que se aplicar.\n%+v",
			len(caseiro.ByTarget), caseiro.ByTarget)
	}
	if len(caseiro.Flags) != 0 {
		t.Errorf("o item caseiro acendeu as bandeiras %v", caseiro.FlagList())
	}
}

// E O VERBETE QUE SÓ AQUELA MESA CONHECE também contribui zero, pela mesma
// porta.
//
// É a semântica que a ALE-389 já declarava certa: um herói levado a outra mesa
// perde o item, porque o `catalogId` não resolve fora do mundo dele. O que
// faltava era isso valer com sobreposição gravada.
func TestAnItemFromAnotherWorldContributesNothing(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }
	adamante := "material-adamante"

	effects := ComputeItemEffects(BookRuleset(catalogs).ActiveItemsFor(Character{
		Level: 5,
		Items: []CharacterItem{{
			CatalogID: ptr("espada-da-mesa-do-vizinho"), Name: "Espada do vizinho",
			Equipped: ptr("wielded"), Improvements: `["melhoria-reforcada"]`, Material: &adamante,
		}},
	}))
	if len(effects.ByTarget) != 0 {
		t.Errorf("um item de outro mundo entregou %d alvos: %+v",
			len(effects.ByTarget), effects.ByTarget)
	}
}
