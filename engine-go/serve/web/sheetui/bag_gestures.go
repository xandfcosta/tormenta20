package sheetui

import (
	"strconv"
	"strings"
)

// Os gestos e as frases dos diálogos da Mochila (`tempestade_bag_dialogs.templ`).

// writtenWeight é "2 × 0,5 espaço = 1", que é o que a carga conta.
func writtenWeight(sheet itemSheet) string {
	return strconv.FormatInt(sheet.Quantity, 10) + " × " + sheet.Slots + " espaço = " + sheet.Total
}

// equipCommand escreve o `@post` do lugar escolhido — guardar tem rota
// própria, porque guardar não passa por teto nenhum.
func equipCommand(v View, sheet itemSheet, place equipChoice) string {
	if place.Slot == "" {
		return sheetPost(v, "/itens/"+sheet.Command+"/guarda")
	}
	return sheetPost(v, "/itens/"+sheet.Command+"/equipa/"+place.Slot)
}

// improvementSignal é a expressão que diz se a melhoria está escolhida AGORA.
func improvementSignal(choice overlayChoice) string {
	return "$item_improvements.includes('" + choice.ID + "')"
}

// thatTogglesImprovementGesture liga ou desliga uma melhoria na lista do sinal.
//
// A lista é COPIADA antes de ser mexida (`[...$item_improvements]`): o sinal é um
// proxy, e escrever dentro dele item a item é a armadilha que o guia do Go
// registra — ler um índice que não existe o CRIA.
func thatTogglesImprovementGesture(choice overlayChoice) string {
	return "const escolhidas = [...$item_improvements]; const onde = escolhidas.indexOf('" + choice.ID + "'); " +
		"if (onde >= 0) { escolhidas.splice(onde, 1) } else { escolhidas.push('" + choice.ID + "') }; " +
		"$item_improvements = escolhidas"
}

func improvementWrittenList(sheet itemSheet) string {
	ids := []string{}
	for _, choice := range sheet.Improvements {
		if choice.Active {
			ids = append(ids, "'"+choice.ID+"'")
		}
	}
	return "[" + strings.Join(ids, ", ") + "]"
}

func chosenMaterial(sheet itemSheet) string {
	for _, choice := range sheet.Materials {
		if choice.Active {
			return choice.ID
		}
	}
	return ""
}
