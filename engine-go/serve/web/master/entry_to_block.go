package master

import (
	"t20engine/domain/book"
	"t20engine/domain/creature"
)

// O VERBETE DO LIVRO VIRANDO BLOCO DO MESTRE.
//
// É o "editar este ogro", e é o caminho PRINCIPAL de criar um NPC (decisão do
// dono): a maioria dos NPCs de campanha nasce como cópia do bestiário com dois
// ou três números mexidos, e escrever do zero é a exceção. Os dois caminhos
// terminam no mesmo formulário; o que muda é a SEMENTE.
//
// Os campos passam direto, porque o livro modela criatura e NPC do mesmo jeito.
// As duas coisas que NÃO passam direto estão comentadas onde acontecem.

// CopyOfEntry copia um verbete do livro para um bloco editável do mestre.
//
// @example CopyOfEntry(ogro).SourceMonsterID // "ogro"
func CopyOfEntry(v book.Entry) creature.Block {
	return creature.Block{
		ND:         v.ND,
		Kind:       v.Kind,
		Size:       v.Size,
		Initiative: v.Initiative,
		Perception: v.Perception,
		// O PM ATRAVESSA COMO PONTEIRO, e não desreferenciado: a maioria das
		// criaturas não tem a linha, e um zero diria "tem mana e está sem" —
		// que é outro estado. Os dois lados guardam a ausência de propósito.
		PM:        v.PM,
		Defense:   v.Defense,
		Fortitude: v.Fortitude,
		Reflex:    v.Reflex,
		Will:      v.Will,
		HP:        v.HP,
		Speed:     v.Speed,
		// ATRIBUTO AUSENTE VIRA ZERO, e esta é uma PERDA CONHECIDA. O livro
		// escreve TRAVESSÃO onde a criatura não tem o atributo — o Zumbi não tem
		// Inteligência (p297) —, e no bloco do mestre isso vira 0, que "+0"
		// afirma ser a média de um humano.
		//
		// Aceitar a perda é deliberado: o bloco é NUMÉRICO e não sabe dizer "não
		// tem", e ensiná-lo exigiria mexer no struct, no formulário e na
		// validação. A partir da cópia o bloco é DELE e ele edita; quem guarda a
		// ausência de verdade é o CATÁLOGO, que é a fonte.
		Strength:         orZero(v.Strength),
		Dexterity:        orZero(v.Dexterity),
		Constitution:     orZero(v.Constitution),
		Intelligence:     orZero(v.Intelligence),
		Wisdom:           orZero(v.Wisdom),
		Charisma:         orZero(v.Charisma),
		Attacks:          copyAttacks(v.Attacks),
		Skills:           copyExpertises(v.Skills),
		Equipment:        v.Equipment,
		Treasure:         v.Treasure,
		SpecialAbilities: copyPhrases(v.SpecialAbilities),
		SourceMonsterID:  v.ID,
	}
}

func orZero(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

// As três cópias abaixo existem para o bloco do mestre não COMPARTILHAR fatia
// com o catálogo embutido: o catálogo é imutável e servido a todo mundo, e o
// bloco nasce para ser editado. Sem a cópia, mexer num ataque do NPC mexeria no
// verbete que o bestiário desenha para a mesa inteira.
func copyAttacks(de []creature.Attack) []creature.Attack {
	fora := make([]creature.Attack, len(de))
	copy(fora, de)
	return fora
}

func copyExpertises(de []creature.Skill) []creature.Skill {
	fora := make([]creature.Skill, len(de))
	copy(fora, de)
	return fora
}

func copyPhrases(de []string) []string {
	fora := make([]string, len(de))
	copy(fora, de)
	return fora
}
