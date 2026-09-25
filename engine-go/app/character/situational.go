package character

import (
	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O QUE O JOGADOR PODE ALTERNAR NA PRÓPRIA FICHA (ALE-387).
//
// Jogador não muda REGRA — isso é do mestre. O que ele liga e desliga é o
// opt-in de um condicional que a ficha dele JÁ OFERECE: o bônus contra mortos-
// vivos da lâmina que ele empunha, a penalidade da armadura que ele veste.
//
// Esta partição é a autoridade sobre esse conjunto, e ela tem DOIS leitores de
// propósito: a cena desenha uma linha por grupo, e o caso de uso recusa a chave
// que não estiver nele. Duas implementações divergiriam na primeira vez que a
// regra de agrupamento mudasse — e a divergência apareceria como a tela
// escondendo o que o servidor aceita, que é exatamente o buraco que isto fecha.

// SituationalGroup é UM interruptor da lista de situacionais, com os
// condicionais que ele acende.
type SituationalGroup struct {
	// Key é o que o gesto manda e o que o motor casa: o endereço do TERMO quando
	// o condicional é solto, e o do GRUPO quando eles dividem uma flag.
	Key     string
	Members []engine.ConditionalEffect
}

// SituationalGroupsOf parte os condicionais oferecidos em interruptores.
//
// # As POSTURAS ficam de fora, e não é cosmético
//
// O interruptor delas mora nos Poderes porque entrar CUSTA PM. Deixá-las aqui é
// dar ao jogador a Fúria de graça — medido: um POST com a chave da postura dava
// +3 em ataque e dano com o PM intacto e sem linha de postura gravada.
//
// @example character.SituationalGroupsOf(effects.Conditional)
func SituationalGroupsOf(offered []engine.ConditionalEffect) []SituationalGroup {
	stances := book.StancesFromCatalog()
	groups := []SituationalGroup{}
	byFlag := map[string]int{} // flag -> posição em `groups`
	for _, c := range offered {
		if c.Flag == "" {
			groups = append(groups, SituationalGroup{Key: c.Term, Members: []engine.ConditionalEffect{c}})
			continue
		}
		if _, isStance := stances[c.Flag]; isStance {
			continue
		}
		if at, seen := byFlag[c.Flag]; seen {
			groups[at].Members = append(groups[at].Members, c)
			continue
		}
		byFlag[c.Flag] = len(groups)
		groups = append(groups, SituationalGroup{
			Key: engine.FlagGroupID(c.Flag), Members: []engine.ConditionalEffect{c},
		})
	}
	return groups
}
