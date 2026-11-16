package api

import (
	"fmt"
	"html"
	"strings"
	"testing"
)

// O GOLPE DIVINO NA TIRA DO IMPULSO (ALE-652, fatia C). O que a Arma Sagrada faz com o dado é do motor
// (`divine_strike_rules_test.go`); aqui se prende a LIGAÇÃO: a tira de cada arma diz o dado dela.

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
