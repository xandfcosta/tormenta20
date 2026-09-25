package engine

import (
	"strconv"
	"strings"
)

// CALAR UM TERMO (ALE-387).
//
// O mestre não só acrescenta: ele DESLIGA. E a granularidade que a mesa pede é
// a do TERMO, não a da fonte — "nesta mesa o medalhão não dá o bônus de Luta,
// mas continua dando o limite de PM".
//
// # O endereço de um termo NÃO inclui o valor, e isso é a decisão
//
// `fonte :: alvo :: escala :: condição`. O jeito óbvio seria misturar o valor e
// o tipo de bônus, que é o que o `ConditionalID` faz — e é justamente o que
// torna aquele id frágil: corrigir um número no livro troca o endereço, e o
// silêncio do mestre evapora em silêncio, que é a pior forma de uma regra
// sumir.
//
// Sem o valor, uma errata do livro NÃO quebra o que a mesa escreveu.
//
// # E a unicidade é MEDIDA, não suposta
//
// Os quatro campos endereçam sozinhos os 246 modificadores do catálogo de hoje
// — zero colisões. O que separa "é único hoje" de "é único" é o
// `TestEveryCatalogTermHasAUniqueAddress`: no dia em que alguém autorar dois
// termos no mesmo endereço, a suíte diz o nome da fonte, em vez de o silêncio
// do mestre matar dois termos de uma vez.
//
// Os discriminadores são `scale` e `condition` porque são eles que separam os
// pares que existem: o anão tem `maxPv +2` e `maxPv +1 por nível`, e a Força da
// Natureza do druida tem `pmCost -2` e o mesmo `-2` em terreno natural. A PROSA
// (`note`, `label`) fica de fora: ela é texto que uma errata reescreve.

// Silence é um termo que esta mesa não aplica, e a quem o silêncio alcança.
//
// @example engine.Silence{Applies: engine.EveryoneIn(), Term: "medalhao-de-prata::expertise:Luta::::"}
type Silence struct {
	Applies Selector
	Term    string
}

// TermID é o endereço estável de um termo dentro de uma fonte.
//
// @example engine.TermID("medalhao-de-prata", mod) // "medalhao-de-prata::pmLimit::::wielded"
func TermID(sourceID string, m Modifier) string {
	return strings.Join([]string{
		sourceID, targetKey(m.Target), scaleAddress(m.Scale), conditionAddress(m.Condition),
	}, "::")
}

// scaleAddress é a escala como texto, vazia quando não há.
func scaleAddress(s *VitalScale) string {
	if s == nil {
		return ""
	}
	return strings.Join([]string{s.Per, strconv.Itoa(s.Step), s.Round, s.Attribute}, "/")
}

// conditionAddress é a condição como texto, SEM a prosa.
//
// `note` e `label` ficam de fora de propósito: são a frase que a tela mostra, e
// reescrevê-la é errata, não mudança de regra. Um endereço que as incluísse
// mudaria quando o texto mudasse.
func conditionAddress(c *ModifierCondition) string {
	if c == nil {
		return ""
	}
	return strings.Join([]string{c.C, c.Type, c.Trait, c.Flag}, "/")
}

// applySilences tira de cada fonte os termos que esta mesa calou para ESTE
// personagem.
//
// Fonte que fica sem termo nenhum SAI da lista: ela existiria como linha vazia
// na decomposição, dizendo que algo contribuiu zero — é a mesma regra que o
// coletor de efeitos já aplica.
//
// Sem silêncio nenhum ela devolve a fatia recebida, intocada: quem não está em
// campanha não paga uma cópia por ficha.
func applySilences(silences []Silence, ch Character, items []ActiveItem) []ActiveItem {
	mudos := map[string]bool{}
	for _, s := range silences {
		if s.Term != "" && s.Applies.Matches(ch) {
			mudos[s.Term] = true
		}
	}
	if len(mudos) == 0 {
		return items
	}
	out := make([]ActiveItem, 0, len(items))
	for _, item := range items {
		kept := make([]Modifier, 0, len(item.Modifiers))
		for _, m := range item.Modifiers {
			if !mudos[TermID(item.SourceID, m)] {
				kept = append(kept, m)
			}
		}
		if len(kept) == 0 {
			continue
		}
		item.Modifiers = kept
		out = append(out, item)
	}
	return out
}
