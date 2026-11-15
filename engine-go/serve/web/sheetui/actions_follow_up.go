package sheetui

import (
	"fmt"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A MANOBRA EM SEGUIDA NAS AÇÕES (ALE-651, ALE-659, `engine.FollowUpManeuver`). Ela abre pela decisão do mestre
// sobre o gesto que vem antes — o Sucesso no agarrar do Lutador de Chão (p77), o acerto do golpe com o Ataque
// Furtivo da Mão na Boca (p74) —, e a linha dela espera essa abertura, que a mesa escreve em `$follow_up_open`.
// O gesto anterior rola direto: não há o que declarar antes (decisão do dono).

// followUpRows são as manobras livres que a decisão do mestre abre, uma por poder: a linha diz de que decisão
// ela depende, e espera a abertura do poder dela na vez.
func followUpRows(offers []engine.FollowUpManeuver, melee int) []actionRow {
	rows := make([]actionRow, 0, len(offers))
	for _, offer := range offers {
		cost := ""
		if offer.PM > 0 {
			cost = strconv.Itoa(offer.PM) + " PM"
		}
		rows = append(rows, actionRow{
			Name:   capitalized(offer.Maneuver) + " · " + offer.Name,
			Detail: "teste oposto de Luta · " + offer.Condition(),
			Value:  book.WithSign(melee),
			Cost:   cost,
			Can:    true,
			Moves:  []string{"attack"},
			Rolls: []actionRoll{{
				Label: "Rolar", Path: fmt.Sprintf("/acoes/manobra/%s/em-seguida/%s", offer.Maneuver, offer.PowerID),
				NeedsTarget: true, Primary: true,
				AlsoWaits: fmt.Sprintf("!$my_turn || $follow_up_open !== '%s'", offer.PowerID),
			}},
		})
	}
	return rows
}

// followUpsOf são as manobras em seguida da ficha, com o interruptor que cada uma pede conferido.
func followUpsOf(dto sheet.CharacterDTO) []engine.FollowUpManeuver {
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil || dto.Ruleset == nil {
		return nil
	}
	return dto.Ruleset.FollowUpManeuvers(ec, sheet.ToStringSet(dto.Conditionals))
}
