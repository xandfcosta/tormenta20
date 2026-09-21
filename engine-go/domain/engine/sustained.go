package engine

import "errors"

// A HABILIDADE SUSTENTADA COBRA TODO TURNO, e é a única duração que cobra.
//
// "A habilidade precisa de um fluxo constante de mana. O personagem deve gastar
// 1 PM como uma ação livre no início de cada turno seu para manter o efeito
// ativo. Se não o fizer, a habilidade termina. Você pode manter diversas
// habilidades sustentadas, pagando o custo de cada uma, mas apenas uma magia
// sustentada por vez." (p227)
//
// AÇÃO LIVRE, então a manutenção NÃO toca no orçamento do turno: quem sustenta
// Velocidade continua com a padrão e a de movimento inteiras (p233).

// SustainedUpkeepPM é o que UMA sustentada cobra por turno. Uma constante e não
// um número solto no meio da conta porque ela é do livro, e o dia em que uma
// habilidade cobrar outro valor a pergunta vai ser "onde está o 1?".
const SustainedUpkeepPM = 1

// SustainedUpkeep é o que a manutenção de um turno decidiu.
type SustainedUpkeep struct {
	// Paid continuam de pé, e foram pagos.
	Paid []string
	// Dropped acabaram por falta de mana. Eles são NOMEADOS e não contados
	// porque a mesa precisa saber qual efeito sumiu da ficha.
	Dropped []string
	// Cost é o que sai do poço: um por sustentada paga.
	Cost int
	// Unconscious diz POR QUE tudo caiu, e a mesa lê a diferença: sem mana é
	// uma escolha que acabou; a 0 PV é um personagem no chão.
	Unconscious bool
}

// PaySustained decide a manutenção do início do turno (T20 p227).
//
// A ORDEM da lista é a ordem de pagamento, e ela importa só quando o mana não
// cobre todas. O livro deixa a escolha com o jogador — "se não o fizer" é
// vontade, não falta —, e este é o caminho automático: paga-se da mais antiga
// para a mais nova, que é estável e explicável. Quem quiser outra escolha
// encerra a que preferir, o que também é ação livre.
//
// INCONSCIENTE NÃO SUSTENTA, e o mana cheio não muda isso. Quem decide não é um
// `if` daqui: manter a habilidade é uma AÇÃO LIVRE no início do turno, e é o
// `UsableNow` que sabe o que uma livre exige do instante (p233). É por aí que a
// cláusula da p227 — a morte encerra as sustentadas, e só elas — chega a um app
// cujo poço de PV tem piso em zero: lá dentro, quem morreu e quem está
// sangrando ocupam o mesmo número, e os dois deixam de pagar.
//
// @example PaySustained([]string{"velocidade", "oracao"}, 1, ActionMoment{OnTurn: true, CanAct: true})
//
//	// Paid: ["velocidade"], Dropped: ["oracao"], Cost: 1
func PaySustained(sustentados []string, pmAtual int, quando ActionMoment) SustainedUpkeep {
	if err := UsableNow(ActionFree, quando); err != nil {
		if len(sustentados) == 0 {
			return SustainedUpkeep{}
		}
		// A RAZÃO vem da recusa e não de um segundo `if`: a mesa lê frases
		// diferentes para "está no chão" e para o resto.
		return SustainedUpkeep{Dropped: sustentados, Unconscious: errors.Is(err, ErrCannotAct)}
	}
	sobra := max(pmAtual, 0)
	var feito SustainedUpkeep
	for _, id := range sustentados {
		if sobra < SustainedUpkeepPM {
			feito.Dropped = append(feito.Dropped, id)
			continue
		}
		sobra -= SustainedUpkeepPM
		feito.Cost += SustainedUpkeepPM
		feito.Paid = append(feito.Paid, id)
	}
	return feito
}
