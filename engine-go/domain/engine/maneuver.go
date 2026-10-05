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
// — e é CONTA, não veredicto: a subtração que o mestre faria de cabeça.
type ManeuverOutcome struct {
	Kind          string `json:"kind"`
	AttackerRoll  int    `json:"attackerRoll"`
	AttackerTotal int    `json:"attackerTotal"`
	DefenderRoll  int    `json:"defenderRoll"`
	DefenderTotal int    `json:"defenderTotal"`
	// Margin é a DIFERENÇA entre os dois totais, e ela é CONTA e não veredicto:
	// é a subtração que o mestre faria de cabeça, e a p234 a usa — *"se você
	// vencer o teste oposto por 5 pontos ou mais"* dá efeito extra ao derrubar e
	// ao desarmar. Negativa quando quem atacou somou menos.
	Margin int `json:"margin"`
	// ConditionOnAWin é a condição que o LIVRO diz que a vitória deixa (p234), e
	// ela viaja SEMPRE — não só quando alguém vence.
	//
	// Isto é informação e não aplicação, e a distinção é a regra da casa: o
	// sistema INFORMA, o mestre DECIDE. Antes este campo se chamava `Imposes` e
	// a confirmação do ataque punha a condição no alvo sozinha; hoje ele diz ao
	// mestre o que o livro prevê, e quem aplica é ele, no gesto que já existe.
	ConditionOnAWin string `json:"conditionOnAWin,omitempty"`
	// Refused diz por que a manobra não aconteceu, em português: a recusa é da
	// REGRA e chega a uma pessoa.
	Refused string `json:"refused,omitempty"`
}

// conditionLeftBy são as manobras que deixam CONDIÇÃO, e são duas das cinco.
//
// As outras três não estão aqui porque o livro não lhes dá condição, e não
// porque alguém esqueceu: o desarmar derruba o item que a criatura segura, o
// empurrar a move 1,5m e o quebrar atinge um item. Escrever uma condição ali
// seria inventar regra.
var conditionLeftBy = map[string]string{
	// "Uma criatura agarrada fica desprevenida e imóvel, sofre –2 nos testes de
	// ataque e só pode atacar com armas leves" (p234).
	"agarrar": "agarrado",
	// "Você deixa o alvo caído" (p234).
	"derrubar": "caido",
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
// Ela NÃO diz quem venceu — ver o corpo. Devolve os dois totais, a diferença, e a
// condição que o livro prevê para a vitória; quem decide é o mestre.
//
//	fora := ResolveManeuver("derrubar", ManeuverSide{Bonus: 4}, ManeuverSide{Bonus: 2}, 15, 8)
//	// fora.Margin == 9, fora.ConditionOnAWin == "caido"
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
	// QUEM VENCEU NÃO SAI DAQUI. O livro tem o desempate — *"em caso de empate, o
	// personagem com o maior bônus vence; se os bônus forem iguais, outro teste
	// deve ser feito"* (p234) —, e ele continua sendo verdade; o que mudou é
	// quem o aplica.
	//
	// Esta função já devolveu `Won`, `Reroll` e `Imposes`, e a confirmação punha
	// a condição no alvo sozinha. Isso é um embate de dados com o sistema
	// anunciando o vencedor e aplicando o efeito, que é a forma que o
	// `CLAUDE.md` da raiz recusa por inteiro. O que sobra é o que a mesa precisa
	// para decidir: os dois totais, a diferença entre eles, e o que o livro diz
	// que a vitória deixaria.
	out.ConditionOnAWin = conditionLeftBy[kind]
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
