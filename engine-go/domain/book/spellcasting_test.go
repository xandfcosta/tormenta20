package book

import (
	"sort"
	"strconv"
	"testing"
)

// A PROGRESSÃO DE CÍRCULO conferida contra o livro, uma classe por página.
//
// Os números abaixo foram transcritos À MÃO das tabelas de classe e das
// habilidades "Magias", e NÃO do `classes.json`, que é o que está sob teste
// (ALE-338). Isto restaura a segunda opinião que morreu com a SPA: havia um
// teste do front comparando as duas cópias da tabela, e quando a cópia ficou uma
// só a tabela passou a não ter ninguém do outro lado.
//
// As três armadilhas valem mais que os vinte e cinco números:
//
//  1. O Bardo e o Druida PARAM no 4º círculo, e o 5º deles é NULO — não um nível
//     alto. "Ainda não alcançou" e "nunca alcança" são coisas diferentes, e um 21
//     escrito no lugar do nulo se comportaria igual até alguém chegar ao 20º.
//  2. O Paladino não tem habilidade "Magias". O 1º círculo dele vem do PODER
//     Orar (p83), e poder de paladino começa no 2º nível (Tabela 1-18, p82) — por
//     isso o 1º círculo abre no 2º e não no 1º, que é a única linha da tabela
//     inteira que não abre no nível da própria classe.
//  3. As duas cadências não são a mesma. Arcanista e Clérigo abrem de quatro em
//     quatro a partir do 1º (1/5/9/13/17); Bardo e Druida saltam do 1º para o 6º
//     e seguem de quatro em quatro (1/6/10/14).
//
// O livro NÃO TEM círculo 0: a Tabela 4-1 (p170) vai do 1º ao 5º, e "truque" é um
// APRIMORAMENTO que zera o custo em PM de uma magia — a magia continua no círculo
// dela. Por isso a conferência abaixo é do CONJUNTO de chaves e não só dos
// valores: um círculo que o livro não tem reprova com o nome dele.
func TestTheCircleProgressionMatchesTheBook(t *testing.T) {
	level := func(n int) *int { return &n }
	const unreachable = "nunca"

	book := []struct {
		class     string
		page      int
		list      string
		attribute string
		max       int
		opens     map[int]*int
	}{
		// Tabela 1-5 e "Magias", p37: "Você pode lançar magias arcanas de 1º
		// círculo. A cada quatro níveis, pode lançar magias de um círculo maior
		// (2º círculo no 5º nível, 3º círculo no 9º nível e assim por diante)".
		//
		// Inteligência é o que o catálogo guarda, mas o livro diz que o
		// atributo-chave do arcanista é definido pelo CAMINHO: é o de dois dos
		// três (bruxo e mago), e o feiticeiro lança com Carisma. Quem resolve o
		// caminho é o `spellcastingAttributeFor`, no `engine`.
		{"Arcanista", 37, "arcana", "intelligence", 5, map[int]*int{
			1: level(1), 2: level(5), 3: level(9), 4: level(13), 5: level(17),
		}},
		// Tabela 1-7 e "Magias", p44: "2º círculo no 6º nível, 3º círculo no 10º
		// nível e 4º círculo no 14º nível". A tabela para no 4º.
		{"Bardo", 44, "arcana", "charisma", 4, map[int]*int{
			1: level(1), 2: level(6), 3: level(10), 4: level(14), 5: nil,
		}},
		// Tabela 1-11 e "Magias", p57: mesma cadência do arcanista, lista divina.
		{"Clérigo", 57, "divina", "wisdom", 5, map[int]*int{
			1: level(1), 2: level(5), 3: level(9), 4: level(13), 5: level(17),
		}},
		// Tabela 1-12 e "Magias", p61: mesma cadência do bardo, lista divina.
		{"Druida", 61, "divina", "wisdom", 4, map[int]*int{
			1: level(1), 2: level(6), 3: level(10), 4: level(14), 5: nil,
		}},
		// O poder Orar, p83: "Você aprende e pode lançar uma magia divina de 1º
		// círculo a sua escolha. Seu atributo-chave para esta magia é Sabedoria".
		// Poder de paladino começa no 2º nível (Tabela 1-18, p82).
		{"Paladino", 83, "divina", "wisdom", 1, map[int]*int{
			1: level(2), 2: nil, 3: nil, 4: nil, 5: nil,
		}},
	}

	table := SpellProgressions()
	if len(table) != len(book) {
		t.Fatalf("o catálogo ofereceu %d classes conjuradoras, quer %d", len(table), len(book))
	}

	checkedCircles := 0
	for _, want := range book {
		found, casts := table[want.class]
		if !casts {
			t.Errorf("%s não trouxe progressão nenhuma (livro p%d)", want.class, want.page)
			continue
		}
		if found.List != want.list {
			t.Errorf("%s lança da lista %q, quer %q (p%d)", want.class, found.List, want.list, want.page)
		}
		if found.Attribute != want.attribute {
			t.Errorf("%s lança com %q, quer %q (p%d)", want.class, found.Attribute, want.attribute, want.page)
		}
		if found.MaxCircle != want.max {
			t.Errorf("%s vai até o %dº círculo, quer o %dº (p%d)", want.class, found.MaxCircle, want.max, want.page)
		}

		// O CONJUNTO de chaves antes dos valores: um círculo a mais é um círculo
		// que o livro não tem, e um a menos some sem nunca reprovar num laço que
		// só percorre o esperado.
		var leftover []string
		for key := range found.UnlockLevel {
			c, err := strconv.Atoi(key)
			if _, fromBook := want.opens[c]; err != nil || !fromBook {
				leftover = append(leftover, key)
			}
		}
		sort.Strings(leftover)
		if len(leftover) > 0 {
			t.Errorf("%s conhece o(s) círculo(s) %v, que o livro não tem (Tabela 4-1, p170: 1º ao 5º)", want.class, leftover)
		}

		for circle, wantLevel := range want.opens {
			checkedCircles++
			hasLevel, declared := found.UnlockLevel[strconv.Itoa(circle)]
			if !declared {
				t.Errorf("%s não diz nada sobre o %dº círculo (p%d)", want.class, circle, want.page)
				continue
			}
			if wantLevel == nil {
				if hasLevel != nil {
					t.Errorf(
						"%s abre o %dº círculo no %dº nível, e no livro ele %s abre (p%d)",
						want.class, circle, *hasLevel, unreachable, want.page,
					)
				}
				continue
			}
			if hasLevel == nil {
				t.Errorf(
					"%s nunca abre o %dº círculo, e no livro ele abre no %dº nível (p%d)",
					want.class, circle, *wantLevel, want.page,
				)
				continue
			}
			if *hasLevel != *wantLevel {
				t.Errorf(
					"%s abre o %dº círculo no %dº nível, quer no %dº (p%d)",
					want.class, circle, *hasLevel, *wantLevel, want.page,
				)
			}
		}
	}

	// O DENOMINADOR, porque uma tabela vazia e uma tabela certa se parecem num
	// laço que percorre o que ESPERA: são cinco classes × cinco círculos.
	if checkedCircles != 25 {
		t.Errorf("a conferência olhou %d círculos, e são 25 (5 classes × 5 círculos)", checkedCircles)
	}
}
