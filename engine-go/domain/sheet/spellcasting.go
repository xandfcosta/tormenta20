package sheet

import (
	"strconv"

	"t20engine/domain/book"
)

// AS CONTAS DE CONJURAÇÃO QUE NÃO DEPENDEM DA MAGIA: o custo de cada círculo e o círculo que uma classe
// alcança no nível dela. O que depende da magia — quem a lança, até que círculo, se o mago a memorizou — é
// do conjurador (`caster.go`).

// SpellBasePmCost é a Tabela 4-1, "Custo de Magias" (livro p170).
var SpellBasePmCost = map[int]int{0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15}

// HighestCastableCircle é o maior círculo que estas classes alcançam, 0 sem classe que conjure. O
// círculo de uma magia é do conjurador (`SpellOnTheSheet.CircleReach`), que escolhe as classes e põe o piso.
func HighestCastableCircle(classes []ClassDTO) int {
	best := 0
	progressions := book.SpellProgressions()
	for _, entry := range classes {
		prog, casts := progressions[entry.ClassName]
		if !casts {
			continue
		}
		for circle := 1; circle <= prog.MaxCircle; circle++ {
			level := prog.UnlockLevel[strconv.Itoa(circle)]
			if level == nil || entry.Level < int64(*level) {
				continue
			}
			if circle > best {
				best = circle
			}
		}
	}
	return best
}

// IsCasterClass diz se a classe conjura.
func IsCasterClass(name string) bool {
	_, casts := book.SpellProgressions()[name]
	return casts
}
