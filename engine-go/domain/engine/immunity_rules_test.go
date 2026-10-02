package engine

import (
	"slices"
	"testing"
)

// A IMUNIDADE da p228-229, e ela NÃO é dado por criatura.
//
//	"A criatura é imune a um tipo de efeito ou outro elemento (como um tipo de
//	 dano, uma condição ou uma habilidade). Ela não sofre nenhuma consequência
//	 direta daquilo contra a qual ela é imune."
//
// As regras moram nas DESCRIÇÕES dos tipos de efeito (p228), e todas falam de
// CATEGORIA e não de indivíduo:
//
//   - "Construtos e mortos-vivos são imunes a efeitos de cansaço"
//   - "Construtos e mortos-vivos são imunes a efeitos de metabolismo"
//   - "Construtos e mortos-vivos são imunes a venenos"
//   - "Criaturas com Inteligência nula são imunes a medo"
//   - "Criaturas com Inteligência nula são imunes a efeitos mentais"
//
// Por isso isto é DERIVAÇÃO e não transcrição: o bestiário já guarda `tipo` e
// `inteligencia`, e os 34 verbetes que citam imunidade em prosa estão repetindo
// o que as duas colunas já dizem. Transcrevê-los daria uma terceira grafia da
// mesma regra — e a que diverge sozinha é a que ninguém abre.
//
// Os cinco tipos acima são os ÚNICOS dos 18 cuja descrição fala em imunidade.

// A METADE DO TIPO: construto e morto-vivo não cansam, não adoecem, não
// envenenam.
func TestConstructsAndUndeadAreImmuneToTheThreeBodilyEffectTypes(t *testing.T) {
	for _, tipo := range []string{"construto", "morto-vivo"} {
		imune := ImmunitiesOfCreature(tipo, nil)
		for _, efeito := range []string{"cansaco", "metabolismo", "veneno"} {
			if !slices.Contains(imune, efeito) {
				t.Errorf("um %s não é imune a %q, e a p228 diz que é: %v", tipo, efeito, imune)
			}
		}
	}
	// O CONTROLE: um humanoide não é imune a nada por tipo. Sem ele, uma função
	// que devolvesse os três para todo mundo passaria no caso acima.
	if imune := ImmunitiesOfCreature("humanoide", ptrInt(2)); len(imune) != 0 {
		t.Errorf("um humanoide de Inteligência 2 veio imune a %v, e a p228 não lhe dá nada", imune)
	}
}

// A METADE DA MENTE, e o caso que a separa da outra: INTELIGÊNCIA NULA não é
// Inteligência ZERO.
//
// No bestiário a nula é `null` — sete das onze criaturas de construto e
// morto-vivo a têm —, e o zero é um número como outro qualquer. A Aparição é
// morto-vivo com Int 0 e o Vampiro com Int 3: os dois são imunes aos três do
// TIPO e nenhum dos dois é imune a medo. Um código que tratasse 0 como "sem
// mente" passaria no Zumbi e erraria nos dois.
func TestOnlyACreatureWithoutIntelligenceIsImmuneToFearAndMentalEffects(t *testing.T) {
	semMente := ImmunitiesOfCreature("construto", nil)
	for _, efeito := range []string{"medo", "mental"} {
		if !slices.Contains(semMente, efeito) {
			t.Errorf("uma criatura de Inteligência nula não é imune a %q: %v", efeito, semMente)
		}
	}
	comZero := ImmunitiesOfCreature("morto-vivo", ptrInt(0))
	for _, efeito := range []string{"medo", "mental"} {
		if slices.Contains(comZero, efeito) {
			t.Errorf("uma criatura de Inteligência ZERO veio imune a %q — nula é a ausência "+
				"do atributo, e zero é um número: %v", efeito, comZero)
		}
	}
	// E ela continua imune aos três do TIPO, que é o que separa as duas metades.
	if !slices.Contains(comZero, "veneno") {
		t.Errorf("o morto-vivo de Inteligência 0 perdeu a imunidade a veneno, que é do "+
			"TIPO dele e não da mente: %v", comZero)
	}
}

// A CONDIÇÃO É RECUSADA PELA TAG dela, e a recusa diz QUAL tipo de efeito a
// barrou — "imune a veneno" explica, "imune" manda procurar.
func TestAConditionIsRefusedByTheEffectTypeThatBlocksIt(t *testing.T) {
	zumbi := ImmunitiesOfCreature("morto-vivo", nil)

	porque, barrada := ImmuneToCondition(zumbi, []string{"veneno"})
	if !barrada {
		t.Errorf("o Envenenado passou num morto-vivo, e a p228 o barra")
	}
	if porque != "veneno" {
		t.Errorf("a recusa culpou %q, e quem barra é o tipo de efeito \"veneno\"", porque)
	}

	// O CONTROLE: uma condição de tipo que ele NÃO é imune passa. Sem ele, uma
	// função que recusasse tudo passaria na asserção acima.
	if _, barrada := ImmuneToCondition(zumbi, []string{"movimento"}); barrada {
		t.Errorf("o Caído foi barrado num morto-vivo — a p228 não lhe dá imunidade a movimento")
	}
	// E A CONDIÇÃO SEM TAG NENHUMA passa: a maioria das 35 não tem tipo, e
	// barrá-las seria inventar imunidade a partir de um campo vazio.
	if _, barrada := ImmuneToCondition(zumbi, nil); barrada {
		t.Errorf("uma condição sem tipo de efeito foi barrada")
	}
}

func ptrInt(n int) *int { return &n }
