package engine

// ManeuverReading é o desfecho que os números de uma manobra SUGEREM (p234). É
// leitura para o mestre, não aplicação: nada no motor muda por causa dela.
type ManeuverReading string

const (
	ManeuverSucceeds  ManeuverReading = "sucesso"
	ManeuverFails     ManeuverReading = "falha"
	ManeuverRollAgain ManeuverReading = "rolar de novo"
)

// ReadManeuver lê o teste oposto pela regra da p234.
//
//	ReadManeuver(5, 12, 8, 12) // ManeuverSucceeds: empate, e quem tenta tem +7 contra +4
func ReadManeuver(attackerRoll, attackerTotal, defenderRoll, defenderTotal int) ManeuverReading {
	if attackerTotal != defenderTotal {
		if attackerTotal > defenderTotal {
			return ManeuverSucceeds
		}
		return ManeuverFails
	}
	// O EMPATE decide pelo BÔNUS, que é o total sem o d20.
	attackerBonus, defenderBonus := attackerTotal-attackerRoll, defenderTotal-defenderRoll
	switch {
	case attackerBonus > defenderBonus:
		return ManeuverSucceeds
	case attackerBonus < defenderBonus:
		return ManeuverFails
	}
	return ManeuverRollAgain
}
