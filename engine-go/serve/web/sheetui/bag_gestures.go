package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
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

// improvementsTakenExpr conta as vagas que o diálogo ocupa AGORA: as melhorias
// ligadas e o material, que é uma delas (p165).
const improvementsTakenExpr = "($item_improvements.length + ($item_material ? 1 : 0))"

// improvementsFull é a expressão que diz que as quatro vagas (p164) estão
// tomadas. A recusa é do servidor (`book.FitsImprovementCount`); isto é a tela
// não oferecer o que ele recusaria.
var improvementsFull = improvementsTakenExpr + " >= " + strconv.Itoa(book.MaxItemImprovements)

// improvementsOfMax é o fim do "3 de 4 melhorias", igual nas duas escritas abaixo.
var improvementsOfMax = " de " + strconv.Itoa(book.MaxItemImprovements) + " melhorias · o material conta como uma"

// improvementsCountText é o "3 de 4 melhorias" vivo do diálogo.
var improvementsCountText = improvementsTakenExpr + " + '" + improvementsOfMax + "'"

// writtenImprovementsCount é a mesma frase escrita pelo servidor, para o
// diálogo não nascer vazio antes de o Datastar avaliar.
func writtenImprovementsCount(sheet itemSheet) string {
	taken := 0
	for _, choice := range sheet.Improvements {
		if choice.Active {
			taken++
		}
	}
	if chosenMaterial(sheet) != "" {
		taken++
	}
	return strconv.Itoa(taken) + improvementsOfMax
}

// improvementLocked trava o interruptor DESLIGADO quando as vagas acabaram. O
// ligado não trava: ele precisa poder sair.
func improvementLocked(choice overlayChoice) string {
	return "!" + improvementSignal(choice) + " && " + improvementsFull
}

// materialLocked trava o material quando não há material e as quatro vagas
// são de melhorias. Trocar um material por outro não muda a conta.
var materialLocked = "$item_material === '' && " + improvementsFull
