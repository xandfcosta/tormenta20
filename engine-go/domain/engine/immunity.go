package engine

import "slices"

// A IMUNIDADE A TIPO DE EFEITO (p228-229).
//
//	"A criatura é imune a um tipo de efeito ou outro elemento (como um tipo de
//	 dano, uma condição ou uma habilidade). Ela não sofre nenhuma consequência
//	 direta daquilo contra a qual ela é imune. Ela ainda pode ser afetada
//	 indiretamente — por exemplo, uma criatura imune a efeitos mágicos ainda é
//	 afetada por terreno difícil criado por magias."
//
// # Por que isto é DERIVAÇÃO e não um campo no bestiário
//
// As regras não estão num verbete de criatura: elas estão nas DESCRIÇÕES dos
// tipos de efeito, e todas falam de CATEGORIA. Cinco dos dezoito tipos têm
// cláusula de imunidade, e as cinco se resolvem com duas colunas que o bestiário
// já guarda — o `tipo` e a `inteligencia`.
//
// Trinta e quatro verbetes do bestiário citam imunidade em PROSA ("imunidades de
// morto-vivo (cansaço/metabólicos/trevas/veneno)"), e essa prosa está repetindo o
// que as duas colunas já dizem. Transcrevê-la daria uma terceira grafia da mesma
// regra, e a que diverge sozinha é a que ninguém abre.
//
// O que a prosa tem A MAIS — a imunidade a trevas do Zumbi, por exemplo — é do
// VERBETE e não do tipo, e fica de fora: ela precisaria de um campo por criatura,
// que é a transcrição que esta leitura evita.
//
// # Não é ECS, pela mesma razão das escadas do objeto
//
// *ECS para o que COMPÕE, função para o que calcula.* Não há entidade: a
// imunidade de uma criatura É a soma de duas tabelas fechadas lidas pelo tipo e
// pela mente dela.

// immunityByCreatureType são as três cláusulas que falam de TIPO (p228).
//
// Construto e morto-vivo apontam para a mesma lista porque o livro os nomeia
// juntos nas três descrições — e separá-los em duas listas iguais seria a mesma
// regra escrita duas vezes, com uma delas livre para divergir.
var immunityByCreatureType = map[string][]string{
	// "Construtos e mortos-vivos são imunes a efeitos de cansaço."
	// "Construtos e mortos-vivos são imunes a efeitos de metabolismo."
	// "Construtos e mortos-vivos são imunes a venenos."
	"construto":  {"cansaco", "metabolismo", "veneno"},
	"morto-vivo": {"cansaco", "metabolismo", "veneno"},
}

// immunityOfTheMindless são as duas cláusulas que falam de MENTE (p228).
//
//   - "Criaturas com Inteligência nula são imunes a medo."
//   - "Criaturas com Inteligência nula são imunes a efeitos mentais."
var immunityOfTheMindless = []string{"medo", "mental"}

// ImmunitiesOfCreature são os tipos de efeito a que esta criatura é imune.
//
// A INTELIGÊNCIA É PONTEIRO porque NULA e ZERO são coisas diferentes, e o livro
// as separa: nula é a ausência do atributo — a criatura não tem mente —, e zero
// é um número como outro qualquer. No bestiário a nula é `null`, e sete das onze
// criaturas de construto e morto-vivo a têm. A Aparição é morto-vivo com Int 0 e
// o Vampiro com Int 3: os dois sentem medo, e os dois continuam imunes a veneno.
//
//	engine.ImmunitiesOfCreature("morto-vivo", nil) // → cansaço, metabolismo, veneno, medo, mental
func ImmunitiesOfCreature(creatureType string, intelligence *int) []string {
	out := slices.Clone(immunityByCreatureType[creatureType])
	if intelligence == nil {
		out = append(out, immunityOfTheMindless...)
	}
	return out
}

// ImmuneToCondition diz se a criatura barra esta condição, e por QUAL tipo de
// efeito — "imune a veneno" explica a recusa; "imune" manda procurar.
//
// A condição chega pelas TAGS dela, que é como o catálogo liga as 35 condições
// aos 18 tipos de efeito. Dezesseis delas carregam alguma tag que uma imunidade
// da p228 alcança; o resto não tem tipo, e condição sem tipo nunca é barrada —
// inventar imunidade a partir de um campo vazio é pior que não ter a regra.
//
//	engine.ImmuneToCondition(imunidades, []string{"veneno"}) // → "veneno", true
func ImmuneToCondition(immunities, conditionTags []string) (string, bool) {
	for _, tag := range conditionTags {
		if slices.Contains(immunities, tag) {
			return tag, true
		}
	}
	return "", false
}

// A CONDIÇÃO QUE UM PODER BARRA (ALE-650). A imunidade acima é do TIPO de efeito; esta é de UMA condição, e
// vem de um poder da ficha: "Você nunca fica surpreendido" (Esquiva Sobrenatural, p42, e a do ladino, p74).
//
// O poder acende uma flag, e não um alvo próprio, porque a pergunta é de sim ou não e a flag já viaja com o
// nome de quem a acendeu (`FlagSources`) — que é o que a recusa diz ao mestre.
const NeverSurprisedFlag = "nunca-surpreendido"

// conditionBarredByFlag é a condição que cada flag barra.
var conditionBarredByFlag = map[string]string{
	NeverSurprisedFlag: "surpreendido",
}

// ConditionBarredByAPower diz se os efeitos da ficha barram a condição, e o nome do poder que a barra.
//
//	engine.ConditionBarredByAPower(efeitos, "surpreendido") // → "Esquiva Sobrenatural", true
func ConditionBarredByAPower(effects ItemEffects, condition string) (string, bool) {
	for flag, barred := range conditionBarredByFlag {
		if barred == condition && effects.Flags[flag] {
			return effects.FlagSources[flag], true
		}
	}
	return "", false
}
