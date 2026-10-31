package table

import (
	"fmt"
	"strings"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// QUEM VÊ O QUÊ NA FAIXA DO ATAQUE (ALE-532, decisão do dono).
//
// Acertar ou errar é sempre interpretação do MESTRE. A faixa dele mostra as
// rolagens, o desfecho que os números sugerem e a consequência que o livro
// prevê, para facilitar a leitura; a palavra final é dele, em voz. O JOGADOR
// não vê esse julgamento, nem a Defesa do alvo, nem a rolagem do outro lado:
// no papel do personagem, ele não conhece a ficha de outra entidade.

// forPlayer recorta a faixa para quem não é o mestre. Quem rolou fica com a
// própria conta; os outros jogadores, só com quem ataca quem.
//
// O DANO sai junto: hoje ele só é rolado quando o ataque acerta, e a presença
// dele entregaria o veredito. Na mesa física o jogador rola o dano depois que o
// mestre fala.
func (a attackProposal) forPlayer(pa live.PendingAttack) *attackProposal {
	out := attackProposal{
		Attacker: a.Attacker, Target: a.Target, TargetEntryID: a.TargetEntryID,
		Weapon: a.Weapon, Mine: a.Mine, Maneuver: a.Maneuver, Action: a.Action,
		Class: "mesa-ataque-comum",
	}
	if !a.Mine {
		return &out
	}
	out.Tally = ownRollLine(pa)
	out.Boost, out.Guard, out.Rerolled = a.Boost, a.Guard, a.Rerolled
	return &out
}

// ownRollLine é a conta de quem rolou, sem o outro lado: "d20 14 + 7 = 21".
func ownRollLine(pa live.PendingAttack) string {
	bonus := pa.Total - pa.Roll
	sign := "+"
	if bonus < 0 {
		sign, bonus = "−", -bonus
	}
	return fmt.Sprintf("d20 %d %s %d = %d", pa.Roll, sign, bonus, pa.Total)
}

// maneuverVerdict é a leitura do embate para o MESTRE, pela regra da p234
// (`engine.ReadManeuver`). O selo é curto — ele divide a linha com Detalhes, ✕ e
// ✓ —, e o que o explica (o efeito extra da margem de 5, o outro teste do
// empate) vai para a conta, nos Detalhes.
func maneuverVerdict(pa live.PendingAttack) (verdict, tone, note string) {
	m := *pa.Maneuver
	switch engine.ReadManeuver(pa.Roll, pa.Total, m.OpposedRoll, m.Opposed) {
	case engine.ManeuverSucceeds:
		verdict, tone = "Sucesso", "good"
		if m.Margin >= 5 {
			verdict = "Sucesso por 5+"
			if extra := maneuverExtra(m.Kind, m.Margin); extra != "" {
				note = "por 5+: " + extra
			}
		}
	case engine.ManeuverFails:
		verdict, tone = "Falha", "danger"
	default:
		verdict, note = "Empate", "bônus iguais: outro teste (p234)"
	}
	return verdict, tone, note
}

// maneuverExtra é o que a p234 dá a quem vence por 5 ou mais: o derrubar e o
// desarmar empurram um quadrado, e o empurrar anda 1,5m a cada 5 pontos.
func maneuverExtra(kind string, margin int) string {
	if margin < 5 {
		return ""
	}
	switch kind {
	case "derrubar":
		return "também empurra 1 quadrado"
	case "desarmar":
		return "o item voa 1 quadrado"
	case "empurrar":
		metres := 1.5 * float64(1+margin/5)
		return "empurra " + strings.Replace(fmt.Sprintf("%.1fm", metres), ".", ",", 1)
	}
	return ""
}
