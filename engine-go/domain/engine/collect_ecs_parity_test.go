package engine

import (
	"path/filepath"
	"testing"
)

// AQUI MORAVAM DOIS CASOS DE PARIDADE CRUZADA (ALE-378).
//
// O `TestActiveItemsByEcsMatchesTheOracle` comparava o coletor de ECS com o
// oráculo, e o `TestBothCollectionPathsAgree` comparava os dois coletores entre
// si. Os dois saíram junto com o coletor legado.
//
// O primeiro virou DUPLICATA: com uma coleta só, ele e o `TestActiveItemsParity`
// passaram a chamar a mesma função contra o mesmo arquivo. Ficou o de nome
// genérico.
//
// O segundo foi previsto pelo próprio autor — "ele morre junto com o motor
// velho, na fatia que apaga a coleta antiga, e aí terá feito o trabalho dele".
// O que ele dava de graça e NÃO morre com ele é a ORDEM: quem a prende agora é
// o `TestEveryCollectorLandsInTheDeclaredOrder`, escrito ANTES desta remoção.

// EFEITO SEM MODIFICADOR NÃO ENTRA NA COLETA, E O ORÁCULO NÃO PROVA ISSO.
//
// Este teste existe por uma medição, não por zelo. Ao provar o caminho em ECS
// vermelho, eu tirei o `if len(mods) == 0 { continue }` e **os dois motores
// seguiram verdes** — porque nenhuma das 18 fichas tem efeito sem modificador.
// Contado: 7 fichas com efeito ativo, 7 efeitos ao todo, ZERO sem modificador.
//
// É a armadilha do "um guarda só mede o que ele VISITA", na forma mais barata
// de cair: o oráculo é enorme e não passa por aqui. A regra que ele deixa de
// fora é de produto — um efeito que não concede nada viraria linha na
// decomposição da ficha dizendo que algo contribuiu ZERO.
//
// O CONTROLE POSITIVO está no mesmo caso: o efeito que TEM modificador precisa
// aparecer. Sem ele, um coletor que não roda passaria por "filtrou certo".
func TestEffectWithoutModifiersIsNotCollected(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)

	ch := Character{ActiveEffects: []ActiveEffectRow{
		{CatalogID: "vazio", Scope: "scene", Modifiers: "[]"},
		{CatalogID: "concede", Scope: "scene", Modifiers: `[{"target":{"k":"defense"},"amount":2,"bonusType":"untyped"}]`},
	}}

	got := BookRuleset(catalogs).ActiveItemsFor(ch)
	if len(got) != 1 {
		t.Fatalf("colheu %d fontes, esperava 1 — o efeito vazio entrou, ou o que concede "+
			"ficou de fora.\nColhido: %+v", len(got), got)
	}
	if len(got[0].Modifiers) != 1 {
		t.Fatalf("a fonte colhida veio com %d modificadores, esperava 1: %+v", len(got[0].Modifiers), got[0])
	}
}
