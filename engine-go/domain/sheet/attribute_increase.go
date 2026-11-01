package sheet

import (
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// attributeIncreasePower é o poder repetível com o limite por patamar (p38). Cada classe tem
// o seu id, e o que os junta é o verbete: a conta atravessa as classes.
const attributeIncreasePower = "Aumento de Atributo"

// characterTier são os patamares da p35, pelo nível do personagem, e quantos ele já alcançou.
func characterTier(level int64) (string, int) {
	switch {
	case level >= 17:
		return "Lenda", 4
	case level >= 11:
		return "Campeão", 3
	case level >= 5:
		return "Veterano", 2
	}
	return "Iniciante", 1
}

// AttributeIncreasesFit confere o "apenas uma vez por patamar para um mesmo atributo" (p38).
//
// A ficha não guarda em que nível cada escolha foi feita, então a conta é pelo teto: um
// atributo tem no máximo uma escolha por patamar já alcançado. O nível sai das classes, e não
// da coluna, para valer também na ficha hipotética de quem vai descer de nível. Duas Forças
// no 4º nível são recusadas, e no 5º já cabem.
func AttributeIncreasesFit(dto CharacterDTO) error {
	count := map[string]int{}
	for id, picks := range PowerPicks(dto.PowerChoices) {
		power, chooses := book.PowerWithChoice(id)
		if !chooses || power.Name != attributeIncreasePower {
			continue
		}
		for _, attribute := range picks {
			count[attribute]++
		}
	}
	var level int64
	for _, c := range dto.Classes {
		level += c.Level
	}
	tier, allowed := characterTier(level)
	// Na ordem do livro, para a mesma ficha dar sempre a mesma recusa.
	for _, attribute := range engine.AttributeKeys {
		if count[attribute] > allowed {
			return fmt.Errorf("%s se escolhe uma vez por patamar para o mesmo atributo (p38): %s teria %d, e o nível %d alcança até o patamar %s",
				attributeIncreasePower, book.AttributeLabel(attribute), count[attribute], level, tier)
		}
	}
	return nil
}
