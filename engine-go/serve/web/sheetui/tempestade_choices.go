package sheetui

import (
	"net/url"
	"strconv"
)

// O DIÁLOGO ESCOLHER DA FICHA (ALE-473), pelo mockup aprovado. O dado
// é o do painel (`choicesPanel`); daqui sai o que os corpos dos passos
// leem diferente: o filtro dos poderes e os endereços dos gestos. A divisão em
// passos é do `tempestade_steps.go`.

// pendingIn conta as pendências de uma origem (`raca`, `origem`, `classe`),
// para o "falta N" do passo da raça.
func pendingIn(v View, source string) int {
	n := 0
	for _, p := range v.Choices.Pending {
		if p.Source == source {
			n++
		}
	}
	return n
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
	return strconv.Itoa(len(card.Chosen)) + " de " + strconv.Itoa(card.Ceiling)
}

// raceAbilityPost e countedPost são os endereços de marcar uma opção
// de habilidade de raça e de escolha contada.
func raceAbilityPost(v View, block raceAbilityBlock, value string) string {
	return sheetPost(v, "/poderes/raca/"+block.ID+"/"+url.PathEscape(value))
}

func countedPost(v View, id, value string) string {
	return sheetPost(v, "/poderes/escolha/"+id+"/"+url.PathEscape(value))
}

// pendingTitle é o cabeçalho das pendências, como o mockup aprovado o
// escreve: "Faltam 4 escolhas".
func pendingTitle(n int) string {
	if n == 1 {
		return "Falta 1 escolha"
	}
	return "Faltam " + strconv.Itoa(n) + " escolhas"
}
