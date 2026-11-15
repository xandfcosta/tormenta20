package sheetui

import (
	"fmt"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A MANOBRA EM SEGUIDA NAS AÇÕES (ALE-651, `engine.FollowUpManeuver`). O jogador a DECLARA antes de rolar
// (decisão do dono): o gesto que a abre — o Agarrar do Lutador de Chão (p77), o Atacar da Mão na Boca (p74) —
// ganha a folha com o interruptor, como o Golpe Relâmpago tem o Golpe Imprudente. A linha da manobra livre
// espera o ✓ do mestre, que a mesa escreve em `$follow_up_open`.

// followUpOff desliga a declaração antes de rolar sem a folha: um interruptor esquecido ligado não pode viajar
// num gesto que não o mostrou.
const followUpOff = "$follow_up = false; "

// declaresTheFollowUp pendura no gesto `at` da linha a folha da manobra em seguida que um poder abre depois
// dele (`after`: a manobra, ou "" para o golpe). A da Mão na Boca só aparece com o Ataque Furtivo ligado, que é
// quando ela vale.
func declaresTheFollowUp(row *actionRow, at int, offers []engine.FollowUpManeuver, after string) {
	for _, offer := range offers {
		if offer.After == after && offer.Ready {
			hangsTheFollowUpSheet(row, at, offer)
			return
		}
	}
}

// switchesTheFollowUpOff desliga a declaração nos gestos sem a folha dela, para quem tem manobra em seguida.
func switchesTheFollowUpOff(rows []actionRow, offers []engine.FollowUpManeuver) {
	if len(offers) == 0 {
		return
	}
	for r := range rows {
		for i := range rows[r].Rolls {
			if roll := &rows[r].Rolls[i]; roll.Dialog == "" {
				roll.Before = followUpOff + roll.Before
			}
		}
	}
}

// hangsTheFollowUpSheet põe a folha da declaração no gesto `at` da linha. O gesto da linha abre a folha e
// desliga o interruptor; o Rolar da folha manda o que ficou ligado. O id leva a ARMA porque a Mão na Boca
// pendura uma folha em cada linha de Atacar, e o desarmado já é a segunda.
func hangsTheFollowUpSheet(row *actionRow, at int, offer engine.FollowUpManeuver) {
	roll := row.Rolls[at]
	// A MANOBRA se nomeia pelo botão ("Agarrar"), e o golpe pela linha ("Atacar · Adaga"): o botão dele é "Rolar".
	title := row.Name
	if !roll.Primary {
		title = roll.Label
	}
	sheet := &rollSheet{
		ID: fmt.Sprintf("follow-up-%d-%s", row.Weapon, offer.PowerID), Title: title, Detail: row.Detail, Roll: roll,
		Switch: rollSwitch{Label: offer.Name, Signal: "follow_up", Detail: followUpSentence(offer)},
	}
	sheet.Roll.Before, sheet.After = "", rollAfter(*row)
	row.Rolls[at].Before = followUpOff
	row.Rolls[at].Dialog = sheet.ID
	row.Sheet = sheet
}

// followUpSentence é o que a declaração faz: "derrubar em seguida, como ação livre, por 1 PM, contra o mesmo
// alvo, depois do ✓ do mestre".
func followUpSentence(offer engine.FollowUpManeuver) string {
	price := ""
	if offer.PM > 0 {
		price = ", por " + strconv.Itoa(offer.PM) + " PM"
	}
	return offer.Maneuver + " em seguida, como ação livre" + price + ", contra o mesmo alvo, depois do ✓ do mestre"
}

// followUpRows são as manobras livres que o ✓ do mestre abre, uma por poder: esperam a abertura do poder
// delas, na vez de quem a declarou.
func followUpRows(offers []engine.FollowUpManeuver, melee int) []actionRow {
	rows := make([]actionRow, 0, len(offers))
	for _, offer := range offers {
		after := "golpe"
		if offer.After != "" {
			after = offer.After
		}
		cost := ""
		if offer.PM > 0 {
			cost = strconv.Itoa(offer.PM) + " PM"
		}
		rows = append(rows, actionRow{
			Name:   capitalized(offer.Maneuver) + " · " + offer.Name,
			Detail: "teste oposto de Luta · depois do " + after + " declarado",
			Value:  book.WithSign(melee),
			Cost:   cost,
			Can:    true,
			Moves:  []string{"attack"},
			Rolls: []actionRoll{{
				Label: "Rolar", Path: fmt.Sprintf("/acoes/manobra/%s/em-seguida/%s", offer.Maneuver, offer.PowerID),
				NeedsTarget: true, Primary: true, Before: followUpOff,
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
