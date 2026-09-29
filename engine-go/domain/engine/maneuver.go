package engine

import (
	"fmt"
	"sort"
)

// A MANOBRA DE COMBATE (T20 p234).
//
// Ela é a primeira regra do motor que NÃO é "teste ≥ CD": o livro manda um teste
// OPOSTO, e os dois lados rolam. Todo o resto do combate mede um total contra um
// número parado — a Defesa, a CD de uma magia —, e aqui o número do outro lado
// também é um dado.
//
// # O que entra pronto, e por quê
//
// Os DOIS `d20` chegam por parâmetro, pela mesma razão do `ResolveAttack`: a
// regra não sorteia, ela decide o que se faz COM a rolagem. É o que deixa o
// empate — que é o caso interessante desta página — virar um teste com o número
// escrito à mão em vez de mil execuções esperando coincidência.

// ManeuverSide é um lado do teste, do ponto de vista da regra.
//
// O `Bonus` é o valor de LUTA já somado ao bônus da manobra, e quem o monta é a
// camada de cima — a regra não sabe de perícia nem de item. O `Ranged` existe
// porque a página proíbe a manobra de quem ataca à distância, e só de quem
// ATACA: *"mesmo que ela esteja usando uma arma de ataque à distância, deve
// fazer o teste usando seu valor de Luta"*.
type ManeuverSide struct {
	Bonus  int
	Ranged bool
}

// ManeuverOutcome é a conta inteira, e não só quem venceu.
//
// A MARGEM viaja porque ela é REGRA e não enfeite: *"se você vencer o teste
// oposto por 5 pontos ou mais"* dá efeito extra ao Derrubar e ao Desarmar, e o
// Empurrar ganha 1,5m por cada 5 pontos. Ela é a DIFERENÇA e pode ser negativa
// — quem perdeu perdeu por quanto.
type ManeuverOutcome struct {
	Kind          string `json:"kind"`
	AttackerRoll  int    `json:"attackerRoll"`
	AttackerTotal int    `json:"attackerTotal"`
	DefenderRoll  int    `json:"defenderRoll"`
	DefenderTotal int    `json:"defenderTotal"`
	Won           bool   `json:"won"`
	Margin        int    `json:"margin"`
	// Reroll é o empate que a página manda repetir: *"se os bônus forem iguais,
	// outro teste deve ser feito"*. Ele não é derrota de quem ataca — tratá-lo
	// assim inventaria uma regra que favorece sempre o mesmo lado.
	Reroll bool `json:"reroll,omitempty"`
	// Refused diz por que a manobra não aconteceu, em português: a recusa é da
	// REGRA e chega a uma pessoa.
	Refused string `json:"refused,omitempty"`
}

// maneuversOfTheBook são as cinco da p234, e a lista é fechada porque a página
// a fecha.
//
// Fechada também protege o CATÁLOGO: o bônus de manobra casa pelo `name` do
// modificador, e um nome escrito errado viraria um bônus que nunca encontra a
// manobra — calado, como todo defeito de casamento por texto.
var maneuversOfTheBook = map[string]bool{
	"agarrar": true, "derrubar": true, "desarmar": true,
	"empurrar": true, "quebrar": true,
}

// ResolveManeuver resolve o teste oposto de uma manobra.
//
//	fora := ResolveManeuver("derrubar", ManeuverSide{Bonus: 4}, ManeuverSide{Bonus: 2}, 15, 8)
//	// fora.Won == true, fora.Margin == 9 — e 9 ≥ 5, então o alvo também é empurrado
func ResolveManeuver(
	kind string, attacker, defender ManeuverSide, attackerD20, defenderD20 int,
) ManeuverOutcome {
	out := ManeuverOutcome{
		Kind:         kind,
		AttackerRoll: attackerD20, AttackerTotal: attackerD20 + attacker.Bonus,
		DefenderRoll: defenderD20, DefenderTotal: defenderD20 + defender.Bonus,
	}
	if !maneuversOfTheBook[kind] {
		out.Refused = fmt.Sprintf("%q não é uma manobra de combate: a p234 tem agarrar, "+
			"derrubar, desarmar, empurrar e quebrar", kind)
		return out
	}
	// "Não é possível fazer manobras de combate com ataques à distância" (p234).
	// Só de quem ATACA: o defensor de arco rola Luta e segue no teste.
	if attacker.Ranged {
		out.Refused = "manobra é ataque corpo a corpo, e a arma empunhada é de ataque à distância (p234)"
		return out
	}

	out.Margin = out.AttackerTotal - out.DefenderTotal
	if out.Margin != 0 {
		out.Won = out.Margin > 0
		return out
	}
	// "Em caso de empate, o personagem com o maior bônus vence. Se os bônus
	// forem iguais, outro teste deve ser feito" (p234).
	switch {
	case attacker.Bonus > defender.Bonus:
		out.Won = true
	case attacker.Bonus < defender.Bonus:
		out.Won = false
	default:
		out.Reroll = true
	}
	return out
}

// ManeuverOffense e ManeuverDefense são os dois lados de um teste oposto, e eles
// existem como tipo porque o CATÁLOGO os distingue: o `Desejo de Liberdade` da
// origem Escravo dá +5 em `agarrar` com escopo de DEFESA, e somar os dois lados
// no mesmo balde foi o defeito que a ALE-406 consertou.
type ManeuverRole string

const (
	ManeuverOffense ManeuverRole = ""
	ManeuverDefense ManeuverRole = "defense"
)

// ManeuverBonus é o bônus que os modificadores dão a UMA manobra, de UM lado.
//
//	ManeuverBonus(efeitos, "derrubar", ManeuverOffense) // o +2 do Derrubar Aprimorado
func ManeuverBonus(effects ItemEffects, kind string, role ManeuverRole) int {
	return StatFor(effects, ModifierTarget{
		K: "maneuver", Name: kind, Scope: string(role),
	}).Total
}

// ManeuversOfTheBook são as cinco da p234, em ordem alfabética.
//
// Exportada porque quem monta o combatente precisa percorrer as cinco por NOME:
// um mapa montado a partir dos modificadores que existem devolveria zero para a
// manobra sem bônus e para a manobra escrita errada do mesmo jeito.
func ManeuversOfTheBook() []string {
	fora := make([]string, 0, len(maneuversOfTheBook))
	for manobra := range maneuversOfTheBook {
		fora = append(fora, manobra)
	}
	sort.Strings(fora)
	return fora
}
