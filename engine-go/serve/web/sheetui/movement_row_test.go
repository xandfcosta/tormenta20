package sheetui

import (
	"testing"

	"t20engine/domain/engine"
)

// A LINHA MOVER diz os outros deslocamentos que a ficha tem, além do terrestre: o voo já era
// calculado (a magia Voo, o Vingador Sagrado, a Forma Selvagem) e nenhuma tela o mostrava.
func TestTheMoveRowNamesEveryOtherDisplacement(t *testing.T) {
	computed := engine.ComputedSheet{ClimbSpeed: 9, SwimSpeed: 12, FlySpeed: 18}
	computed.Displacement.Total = 6
	row := movementRow(computed)
	if row.Value != "9m" {
		t.Errorf("o valor da linha é %q, e o deslocamento terrestre é 9m", row.Value)
	}
	want := "percorrer o seu deslocamento · p233 · escalada\u00a09m · natação\u00a012m · voo\u00a018m"
	if row.Detail != want {
		t.Errorf("a linha diz %q, e deveria dizer %q", row.Detail, want)
	}
	if plain := movementRow(engine.ComputedSheet{}); plain.Detail != "percorrer o seu deslocamento · p233" {
		t.Errorf("sem outro deslocamento, a linha ganhou %q", plain.Detail)
	}
}

// O GOBLIN LENTO anda 4,5m e escala 4,5m: a escalada É o terrestre (p23), com o meio metro.
func TestTheSlowGoblinClimbsFourAndAHalfMetres(t *testing.T) {
	computed := engine.ComputedSheet{ClimbsAsItWalks: true}
	computed.Displacement.Total = 3
	row := movementRow(computed)
	if row.Value != "4,5m" || row.Detail != "percorrer o seu deslocamento · p233 · escalada\u00a04,5m" {
		t.Errorf("o goblin lento anda %q e a linha diz %q; a escalada devia ser 4,5m", row.Value, row.Detail)
	}
}
