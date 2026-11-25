package table

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/live"
)

// A MANOBRA ESCRITA NA CARTA DO ATAQUE (p234): o nome da manobra tentada e a conta do teste oposto. Ela muda
// quando muda o jeito de a mesa ler a manobra, e não o golpe.

// maneuverSeal é o NOME da manobra em curso ("Derrubar"), que a cabeça das
// duas faixas mostra. O desfecho não mora aqui: para o mestre ele é o
// `maneuverVerdict`, e o jogador não o vê (ALE-532).
func maneuverSeal(m live.ManeuverRoll) string {
	if nome, conhecida := maneuverNames[m.Kind]; conhecida {
		return nome
	}
	return "Manobra"
}

// maneuverNames são as cinco da p234 como a mesa as lê, no INFINITIVO.
//
// Eram pares — "Derrubou" / "não derrubou" —, e o passado afirmava um desfecho
// que o sistema não tem mais como saber. O infinitivo diz o que foi TENTADO, que
// é o que de fato aconteceu.
var maneuverNames = map[string]string{
	"agarrar":  "Agarrar",
	"derrubar": "Derrubar",
	"desarmar": "Desarmar",
	"empurrar": "Empurrar",
	"quebrar":  "Quebrar",
}

// maneuverLine escreve a conta do teste OPOSTO, e ela é a resposta inteira: dois
// totais e a diferença entre eles.
//
// A linha é a CONTA, e ela é do mestre: o selo ao lado diz o desfecho que os
// números sugerem (`maneuverVerdict`), e a diferença aparece porque é a
// subtração que ele faria de cabeça.
//
//	maneuverLine(...) // "d20 (15) + 4 = 19 vs d20 (4) + 6 = 10 → diferença +9 | vitória deixa caído"
func maneuverLine(pa live.PendingAttack) string {
	m := *pa.Maneuver
	// O NOME DA MANOBRA NÃO ENTRA AQUI: o crachá ao lado já o diz, e repeti-lo
	// gastaria o começo da linha — que a 390px é o pedaço mais caro dela.
	line := fmt.Sprintf("%s = %d vs %s = %d → diferença\u00a0%+d",
		rollNotation(pa.Roll, pa.Total), pa.Total, rollNotation(m.OpposedRoll, m.Opposed), m.Opposed, m.Margin)
	// O QUE O LIVRO PREVÊ, dito SEMPRE e não só na vitória: o sistema não sabe
	// quem venceu, e esta linha existe justamente para o mestre decidir com a
	// consequência à vista. Nem toda manobra deixa condição — o desarmar derruba
	// um item —, e por isso ela só aparece quando há.
	//
	// O NOME ESCRITO e não o id: "fica caido" é o identificador vazando para a
	// tela. Quem o traduz é o catálogo, que é onde o nome é autorado.
	if m.ConditionOnAWin != "" {
		line += notationStep + "vitória deixa " + strings.ToLower(book.ConditionName(m.ConditionOnAWin))
	}
	return line
}
