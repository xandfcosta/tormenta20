package engine

import (
	"path/filepath"
	"reflect"
	"testing"
)

// O MOTOR EM ECS COLHE AS MESMAS FONTES, NA MESMA ORDEM (ALE-382).
//
// Este é o teste que o cabeçalho do `genoracle` dizia não existir: *"o que ele
// NÃO prova é que dois motores concordam, porque só existe um"*. Agora existem
// dois, e o oráculo — congelado na ALE-379, com o hash fixado para ninguém
// regenerá-lo no lugar de consertar — julga os dois com o mesmo dado.
//
// O que se prende é a lista INTEIRA: os mesmos itens, com os mesmos
// modificadores, NA MESMA ORDEM. Um `reflect.DeepEqual` sobre fatia reprova por
// ordem, e isso é a metade que importa — o núcleo de ECS foi desenhado em volta
// dela, porque a iteração de `map` em Go é aleatória por construção.
func TestActiveItemsInWorldMatchesTheOracle(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	slugs := parityOracleSlugs(t, dir)

	if len(slugs) == 0 {
		t.Fatal("nenhuma ficha no oráculo: o teste passaria sobre nada")
	}

	for _, slug := range slugs {
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char        Character `json:"char"`
				ActiveItems any       `json:"activeItems"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ActiveItemsInWorld(oracle.Char))
			if !reflect.DeepEqual(got, oracle.ActiveItems) {
				diffReport(t, "activeItems (ECS)", got, oracle.ActiveItems)
			}
		})
	}
	t.Logf("fichas conferidas contra o caminho em ECS: %d", len(slugs))
}

// E OS DOIS CAMINHOS CONCORDAM ENTRE SI.
//
// Não é o mesmo teste acima com outra roupa. Aquele compara cada motor com um
// ARQUIVO; este compara os dois entre si, e é o que continua respondendo no dia
// em que a regra do livro mudar e o oráculo for levantado de propósito: os dois
// motores têm de andar juntos, não só cada um com a foto de ontem.
//
// Ele morre junto com o motor velho, na fatia que apaga a coleta antiga — e aí
// terá feito o trabalho dele.
func TestBothCollectionPathsAgree(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	slugs := parityOracleSlugs(t, dir)

	for _, slug := range slugs {
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char Character `json:"char"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			legacy := roundTrip(t, catalogs.ActiveItemsFor(oracle.Char))
			world := roundTrip(t, catalogs.ActiveItemsInWorld(oracle.Char))
			if !reflect.DeepEqual(world, legacy) {
				diffReport(t, "ECS contra o coletor antigo", world, legacy)
			}
		})
	}
}

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

	for _, path := range []struct {
		name    string
		collect func(Character) []ActiveItem
	}{
		{"coletor antigo", catalogs.ActiveItemsFor},
		{"em ECS", catalogs.ActiveItemsInWorld},
	} {
		t.Run(path.name, func(t *testing.T) {
			got := path.collect(ch)
			if len(got) != 1 {
				t.Fatalf("colheu %d fontes, esperava 1 — o efeito vazio entrou, ou o que concede ficou de fora.\nColhido: %+v", len(got), got)
			}
			if len(got[0].Modifiers) != 1 {
				t.Fatalf("a fonte colhida veio com %d modificadores, esperava 1: %+v", len(got[0].Modifiers), got[0])
			}
		})
	}
}
