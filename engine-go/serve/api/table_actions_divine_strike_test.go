package api

import (
	"fmt"
	"html"
	"strings"
	"testing"
)

// O GOLPE DIVINO NA TIRA DO IMPULSO (ALE-652, fatia C). O que a Arma Sagrada e o Vingador Sagrado fazem com o
// dado e com o preço é do motor (`divine_strike_rules_test.go`); aqui se prende a LIGAÇÃO: a tira de cada arma
// diz o dado dela, e o gesto do Vingador muda o que a tira cobra.

const divineStrikePower = "class.paladino.golpe-divino-1d8"

func strikeScreenOf(t *testing.T, f sceneFixture, character int64) string {
	t.Helper()
	return html.UnescapeString(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", character), ""))
}

// p82: com a Arma Sagrada, a espada longa de Khalmyr rola o Golpe Divino em d12, e a espada curta em d8. A
// tira de cada linha diz o dado dela — dizer d8 na espada de Khalmyr seria a tela contando outra conta.
func TestTheSacredWeaponRowSaysD12AndTheOtherRowSaysD8(t *testing.T) {
	f, paladin, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, paladin, "Paladino", 5)
	seedPowerChoices(t, f.s, paladin, `["class.paladino.arma-sagrada"]`, "{}")
	devoteOf(t, f, paladin, "khalmyr", "Dom da Verdade")
	itemSemeia(t, f, paladin, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, paladin, "espada-curta", "Espada curta", "wielded2")

	screen := strikeScreenOf(t, f, paladin)
	if len(boostRowsOffering(t, screen, divineStrikePower)) < 2 {
		t.Fatalf("o Golpe Divino não está nas duas espadas:\n%s", recorte(screen))
	}
	for _, buys := range []string{`["+1d12","+2d12"]`, `["+1d8","+2d8"]`} {
		if !strings.Contains(screen, buys) {
			t.Errorf("nenhuma linha diz %s:\n%s", buys, recorte(screen))
		}
	}
}

// p84: o gesto do Vingador Sagrado põe o Golpe Divino pela metade do custo e com dois dados a mais, até o fim da
// cena. Antes dele a tira cobra o cheio; depois, a metade, e diz os dados a mais.
func TestTheHolyAvengerGestureHalvesTheDivineStrikeOnTheStrip(t *testing.T) {
	f, paladin, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, paladin, "Paladino", 20)
	devoteOf(t, f, paladin, "khalmyr", "Dom da Verdade")
	itemSemeia(t, f, paladin, "espada-curta", "Espada curta", "wielded")

	// Do 20º nível, o Golpe Divino vai de 2 PM a 6 PM, e de 1d8 a 5d8.
	full, halved := `? [2,3,4,5,6]`, `? [1,1,2,2,3]`
	before := strikeScreenOf(t, f, paladin)
	if !strings.Contains(before, full) || !strings.Contains(before, `["+1d8","+2d8","+3d8","+4d8","+5d8"]`) {
		t.Fatalf("fora da forma, a tira não cobra o Golpe Divino cheio:\n%s", recorte(before))
	}
	if refusal := powerCommand(t, f, paladin, "usa/class.paladino.vingador-sagrado", ""); refusal != "" {
		t.Fatalf("usar o Vingador Sagrado: %q", refusal)
	}
	after := strikeScreenOf(t, f, paladin)
	if !strings.Contains(after, halved) || !strings.Contains(after, `["+3d8","+4d8","+5d8","+6d8","+7d8"]`) {
		t.Errorf("na forma de vingador, a tira não cobra a metade nem diz os dois dados a mais:\n%s", recorte(after))
	}
}
