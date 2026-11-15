package sheetui

import (
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// O ROLAR DA FOLHA LIMPA O QUE O DA LINHA LIMPA: o impulso escolhido na tira e a dose. Sem isso, o impulso do
// golpe que saiu pela folha continua escolhido, e viaja no próximo golpe da mesma arma sem ninguém o ver.
func TestTheSheetRollClearsTheBoostLikeTheRowRoll(t *testing.T) {
	offered := []attackBoost{{PowerID: "class.guerreiro.ataque-especial", Name: "Ataque Especial", Can: true}}
	unarmed := weaponTile{Name: engine.UnarmedName, Damage: "1d3"}
	lightning := engine.PaidExtraStrike{PowerID: engine.ImpulsiveStrikePowerID, Name: "Golpe Relâmpago", PM: 1, Unarmed: true}

	sheets := map[string]*rollSheet{
		"Golpe Relâmpago": paidExtraRow(0, unarmed, lightning, offered, true).Sheet,
	}
	row := attackRow(0, unarmed, [][]attackBoost{offered})
	declaresTheFollowUp(&row, 0, []engine.FollowUpManeuver{{
		PowerID: "class.ladino.mao-na-boca", Name: "Mão na Boca", Maneuver: "agarrar", Ready: true,
	}}, "")
	sheets["Mão na Boca"] = row.Sheet

	for name, sheet := range sheets {
		if sheet == nil {
			t.Fatalf("a folha do %s não foi pendurada", name)
		}
		if !strings.Contains(sheet.After, rollClearsTheBoost()) {
			t.Errorf("o Rolar da folha do %s não limpa o impulso escolhido: depois dele vem %q", name, sheet.After)
		}
	}
}
