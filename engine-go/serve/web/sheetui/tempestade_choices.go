package sheetui

import (
	"encoding/json"
	"net/url"
	"strconv"

	"t20engine/domain/sheet"
)

// O DIÁLOGO ESCOLHER DA FICHA PARALELA (ALE-473), pelo mockup aprovado. O dado
// é o da ficha de hoje (`choicesPanel`); daqui sai só o que o diálogo novo lê
// diferente: quantas pendências cada aba tem, o filtro dos poderes e os
// endereços dos gestos.

// pendingIn conta as pendências de uma aba do diálogo (`raca`, `origem`,
// `classe`), para o número ao lado do nome dela.
func pendingIn(v View, source string) int {
	n := 0
	for _, p := range v.Choices.Pending {
		if p.Source == source {
			n++
		}
	}
	return n
}

// choicesOpens é o gesto do Escolher: abre na aba da primeira pendência, e
// carrega no sinal a distribuição de atributos que a raça já gravou — sem
// isso as opções nasceriam desmarcadas sobre uma escolha feita.
func choicesOpens(v View) string {
	chosen := []string{}
	for _, card := range v.Choices.Races {
		if card.Attribute != nil && card.Attribute.Kind != "ascendencia" {
			chosen = card.Attribute.Chosen
		}
	}
	written, _ := json.Marshal(chosen)
	return "$power_source = '" + firstPendingSource(v) + "'; $power_pick_filter = ''; $race_attributes = " + string(written)
}

// powerPickShows é o `data-show` de um poder na lista da classe, pelo filtro
// "Todos · Da classe · Gerais · Escolhidos" (decisão do dono). O que é da
// classe e o que está escolhido o servidor já sabe; o filtro só esconde.
func powerPickShows(p powerChoice) string {
	kind := "general"
	if p.Source == "Classe" {
		kind = "class"
	}
	shows := "$power_pick_filter === '' || $power_pick_filter === '" + kind + "'"
	if p.Chosen {
		shows += " || $power_pick_filter === 'chosen'"
	}
	return shows
}

// chosenPowers conta os poderes escolhidos de uma classe, para o rótulo do
// filtro "Escolhidos · N".
func chosenPowers(card classChoiceCard) int {
	n := 0
	for _, p := range card.Powers {
		if p.Chosen {
			n++
		}
	}
	return n
}

// originWritten é o teto da origem: "1 de 2".
func originWritten(card originChoiceCard) string {
	return strconv.Itoa(len(card.Chosen)) + " de " + strconv.Itoa(sheet.BenefitsOriginLimit)
}

// raceAbilityPost e countedPost são os endereços de marcar e tirar uma opção
// de habilidade de raça e de escolha contada.
func raceAbilityPost(v View, block raceAbilityBlock, value string) string {
	return sheetPost(v, "/poderes/raca/"+block.ID+"/"+url.PathEscape(value))
}

func countedPost(v View, id, value string, takesOff bool) string {
	path := "/poderes/escolha/" + id + "/" + url.PathEscape(value)
	if takesOff {
		path += "/tira"
	}
	return sheetPost(v, path)
}

// pendingTitle é o cabeçalho das pendências, como o mockup aprovado o
// escreve: "Faltam 4 escolhas".
func pendingTitle(n int) string {
	if n == 1 {
		return "Falta 1 escolha"
	}
	return "Faltam " + strconv.Itoa(n) + " escolhas"
}
