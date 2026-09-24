package ecs_test

import (
	"testing"

	"t20engine/domain/ecs"
)

// Componentes de mentira. Eles não são de Tormenta de propósito: se uma palavra
// do livro aparecer neste pacote, o desenho vazou.
type label struct{ text string }
type weight struct{ grams int }

// A ORDEM DA CONSULTA É A DE INSERÇÃO, E ISTO É CORREÇÃO, NÃO ESTILO.
//
// O oráculo de paridade compara byte a byte, e o coletor que esta fatia vai
// substituir avisa que "order is significativa". A iteração de `map` em Go é
// ALEATÓRIA por construção — o runtime sorteia o ponto de partida justamente
// para ninguém depender dela. Um armazenamento ingênuo faria a fatia 2 falhar
// de forma intermitente, com cara de defeito de regra.
//
// `-count` alto no comando não basta para provar isto: o sorteio é por
// iteração, então CEM entidades numa corrida já dão ao `map` chance de sobra de
// embaralhar. O teste falha na primeira vez que alguém trocar a fatia por mapa.
func TestEachVisitsInInsertionOrder(t *testing.T) {
	world := ecs.NewWorld()

	want := make([]ecs.Entity, 0, 100)
	for i := range 100 {
		e := world.Spawn()
		ecs.Set(world, e, label{text: string(rune('a' + i%26))})
		want = append(want, e)
	}

	got := make([]ecs.Entity, 0, 100)
	ecs.Each(world, func(e ecs.Entity, _ label) { got = append(got, e) })

	if len(got) != len(want) {
		t.Fatalf("visitou %d entidades, esperava %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("a ordem quebrou na posição %d: visitou %d, criei %d\n"+
				"A consulta tem de devolver na ordem de INSERÇÃO — se o armazenamento "+
				"virou `map`, a fatia 2 vai falhar intermitente contra o oráculo.",
				i, got[i], want[i])
		}
	}
}

// O PREDICADO DO Each2 É A REGRA, E É ELE QUE SE PRENDE.
//
// Arranjar só o resultado diria o que acontece COM as entidades achadas e nada
// sobre QUAIS são elas. Aqui o caso põe as quatro combinações na mesa — os
// dois, só A, só B, e nenhum — e afirma que exatamente uma passa.
func TestEach2VisitsOnlyWhoHasBothComponents(t *testing.T) {
	world := ecs.NewWorld()

	both := world.Spawn()
	ecs.Set(world, both, label{text: "os dois"})
	ecs.Set(world, both, weight{grams: 10})

	onlyLabel := world.Spawn()
	ecs.Set(world, onlyLabel, label{text: "só rótulo"})

	onlyWeight := world.Spawn()
	ecs.Set(world, onlyWeight, weight{grams: 20})

	world.Spawn() // nenhum componente

	visited := []ecs.Entity{}
	ecs.Each2(world, func(e ecs.Entity, l label, w weight) {
		visited = append(visited, e)
		if l.text != "os dois" || w.grams != 10 {
			t.Errorf("entidade %d veio com componentes trocados: %+v %+v", e, l, w)
		}
	})

	if len(visited) != 1 || visited[0] != both {
		t.Fatalf("Each2 visitou %v, esperava só a entidade %d (a que tem os DOIS).\n"+
			"Quem tem só um componente não pode entrar: o predicado é a regra.",
			visited, both)
	}
}

// Remove COMPACTA o armazenamento e a ordem dos sobreviventes fica de pé.
//
// É o que separa este armazenamento de um que troca com o último: a troca seria
// O(1) e REORDENARIA, e a ordem aqui é correção, não estilo.
//
// O caso tira a do MEIO de propósito — tirar a última esconderia um erro de
// índice, porque não há ninguém depois dela para escorregar.
func TestRemoveCompactsAndKeepsTheOrderOfTheSurvivors(t *testing.T) {
	world := ecs.NewWorld()

	first := world.Spawn()
	middle := world.Spawn()
	last := world.Spawn()
	for _, e := range []ecs.Entity{first, middle, last} {
		ecs.Set(world, e, label{text: "presente"})
	}

	ecs.Remove[label](world, middle)

	if _, ok := ecs.Get[label](world, middle); ok {
		t.Error("o componente sobreviveu ao Remove")
	}

	got := []ecs.Entity{}
	ecs.Each(world, func(e ecs.Entity, _ label) { got = append(got, e) })

	want := []ecs.Entity{first, last}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("depois do Remove a consulta devolveu %v, esperava %v — "+
			"a ordem dos sobreviventes tem de ficar de pé", got, want)
	}

	// O ÍNDICE de quem veio DEPOIS tem de ter escorregado junto. Sem isto o
	// `Get` do sobrevivente leria a posição errada da fatia — e com dois
	// elementos iguais o teste de ordem acima passaria mesmo assim.
	ecs.Set(world, last, label{text: "reescrito"})
	if got, _ := ecs.Get[label](world, last); got.text != "reescrito" {
		t.Fatalf("o índice não escorregou: escrevi em %d e li %q", last, got.text)
	}
	if got, _ := ecs.Get[label](world, first); got.text != "presente" {
		t.Fatalf("escrever no último sujou o primeiro: %q", got.text)
	}
}

// Set duas vezes SUBSTITUI e não duplica — e não reordena.
//
// Se o segundo Set empurrasse a entidade para o fim, a ordem da consulta
// passaria a depender de QUANDO o componente foi escrito pela última vez, e não
// de quando a entidade nasceu. Na fatia 2 isso trocaria a ordem dos
// `ActiveItem` conforme a ordem das regras que os tocam.
func TestSetTwiceReplacesWithoutReordering(t *testing.T) {
	world := ecs.NewWorld()
	first := world.Spawn()
	second := world.Spawn()
	ecs.Set(world, first, label{text: "um"})
	ecs.Set(world, second, label{text: "dois"})

	ecs.Set(world, first, label{text: "um, de novo"})

	got := []string{}
	ecs.Each(world, func(_ ecs.Entity, l label) { got = append(got, l.text) })

	if len(got) != 2 {
		t.Fatalf("o segundo Set duplicou: %v", got)
	}
	if got[0] != "um, de novo" || got[1] != "dois" {
		t.Fatalf("sobrescrever reordenou: %v — a ordem é a de INSERÇÃO da entidade, "+
			"não a da última escrita", got)
	}
}

// Run roda os sistemas NA ORDEM dada, porque a ordem É o grafo de dependência.
func TestRunAppliesSystemsInOrder(t *testing.T) {
	world := ecs.NewWorld()
	e := world.Spawn()
	ecs.Set(world, e, label{text: ""})

	appendTo := func(s string) ecs.System {
		return func(w *ecs.World) {
			current, _ := ecs.Get[label](w, e)
			ecs.Set(w, e, label{text: current.text + s})
		}
	}

	ecs.Run(world, appendTo("a"), appendTo("b"), appendTo("c"))

	got, _ := ecs.Get[label](world, e)
	if got.text != "abc" {
		t.Fatalf("os sistemas rodaram fora de ordem: %q, esperava %q", got.text, "abc")
	}
}

// Componentes de tipos DIFERENTES não se atropelam no mesmo mundo.
func TestStoresOfDifferentTypesDoNotCollide(t *testing.T) {
	world := ecs.NewWorld()
	e := world.Spawn()
	ecs.Set(world, e, label{text: "texto"})
	ecs.Set(world, e, weight{grams: 7})

	if l, ok := ecs.Get[label](world, e); !ok || l.text != "texto" {
		t.Errorf("o rótulo se perdeu: %+v (achou=%v)", l, ok)
	}
	if w, ok := ecs.Get[weight](world, e); !ok || w.grams != 7 {
		t.Errorf("o peso se perdeu: %+v (achou=%v)", w, ok)
	}
}
