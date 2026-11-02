package sheet

import (
	"fmt"
	"strconv"
	"strings"
)

// attributeValueLimit não é regra do livro, que não dá teto: é o que barra o erro de digitação.
const attributeValueLimit = 99

// AttributeValueRead lê o valor que a pessoa escreveu. Vazio é zero, o ponto de partida do livro (p17), e
// o sinal de menos tipográfico vale, porque é assim que a tela escreve.
//
//	AttributeValueRead("−2") // -2, nil
func AttributeValueRead(written string) (int, error) {
	text := strings.ReplaceAll(strings.TrimSpace(written), "−", "-")
	if text == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil || value < -attributeValueLimit || value > attributeValueLimit {
		return 0, fmt.Errorf("%q não serve: o valor é um número inteiro de −%d a %d", written, attributeValueLimit, attributeValueLimit)
	}
	return value, nil
}

// AttributeMinimumEffect é o aviso do que o livro prevê para um total menor que −5 (p17), ou "". É aviso e
// não recusa: quem decide o que acontece é o mestre.
//
//	AttributeMinimumEffect("constitution", -6) // "Abaixo de −5: morre (p17)"
func AttributeMinimumEffect(key string, total int) string {
	if total >= -5 {
		return ""
	}
	switch key {
	case "strength", "dexterity":
		return "Abaixo de −5: paralisado (p17)"
	case "constitution":
		return "Abaixo de −5: morre (p17)"
	case "intelligence", "wisdom":
		return "Abaixo de −5: inconsciente (p17)"
	case "charisma":
		return "Abaixo de −5: torna-se um NPC (p17)"
	}
	return ""
}
