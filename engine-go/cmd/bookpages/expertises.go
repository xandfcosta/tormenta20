package main

import (
	"fmt"
	"regexp"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
)

// expertiseTablePage é a Tabela 2-1, que traz as duas colunas que a lista de
// criação não tinha: "só treinada" e "penalidade de armadura".
const expertiseTablePage = 115

// expertiseSpan é onde as perícias são DESCRITAS, para achar a página das duas
// que o índice remissivo não lista.
var expertiseSpan = pageSpan{115, 123}

var attributeAbbreviation = map[string]string{
	"strength": "for", "dexterity": "des", "constitution": "con",
	"intelligence": "int", "wisdom": "sab", "charisma": "car",
}

// rebuildExpertises recria as perícias com o que se COLHE, nunca o que se
// transcreve: o atributo vem da lista de criação (que o motor já usa para
// rolar), a página do Índice Remissivo conferida contra o texto, e "só
// treinada" e "penalidade de armadura" da Tabela 2-1.
//
// O CONTROLE que vale mais que os três: o atributo da tabela é comparado com o
// da lista, perícia a perícia. Uma discordância PARA o script, porque uma das
// duas fontes está errada e não se sabe qual.
func (b *book) rebuildExpertises() error {
	table := b.page(expertiseTablePage)
	kept := map[string]string{}
	for _, e := range transcript.Expertises {
		kept[e.ID] = e.UID
	}
	var expertises []transcript.Expertise
	for _, option := range transcript.CreationOptions.Expertises {
		name := option.Name
		m := regexp.MustCompile(regexp.QuoteMeta(fold(name)) + pySpace + `+(for|des|con|int|sab|car)` +
			pySpace + `+(` + pyNonSpace + `+)` + pySpace + `+(` + pyNonSpace + `+)`).FindStringSubmatch(table)
		if m == nil {
			return fatal{fmt.Sprintf("a perícia %s não está na Tabela 2-1 da p%d", pyfmt.Repr(name), expertiseTablePage)}
		}
		attribute, trained, armor := m[1], m[2], m[3]
		if attribute != attributeAbbreviation[option.Attribute] {
			return fatal{fmt.Sprintf("%s: a tabela do livro diz %s e o catálogo diz %s — uma das duas está errada",
				name, attribute, attributeAbbreviation[option.Attribute])}
		}

		var pages []int
		for _, p := range b.byIndex(name, "perícia", "") {
			if b.confirms(name, p) {
				pages = append(pages, p)
			}
		}
		if len(pages) == 0 {
			// Pontaria e Reflexos não estão no índice; a descrição delas está na
			// faixa das perícias, e lá o nome abre a linha. A página da TABELA
			// sai da conta: as 29 aparecem lá, e aceitá-la mandaria essas duas
			// para a lista em vez de para a descrição delas.
			for _, p := range b.byTitle(name, expertiseSpan) {
				if p != expertiseTablePage && b.confirms(name, p) {
					pages = append(pages, p)
				}
			}
		}
		if len(pages) == 0 {
			return fatal{fmt.Sprintf("não achei a página da perícia %s", pyfmt.Repr(name))}
		}
		id := classID(name)
		expertises = append(expertises, transcript.Expertise{ID: id, Name: name, Attribute: option.Attribute,
			TrainedOnly: trained == "sim", ArmorPenalty: armor == "sim", BookPage: pages[0], UID: kept[id]})
	}
	transcript.Expertises = expertises
	trainedOnly, armorPenalty := 0, 0
	for _, e := range expertises {
		if e.TrainedOnly {
			trainedOnly++
		}
		if e.ArmorPenalty {
			armorPenalty++
		}
	}
	fmt.Printf("catálogo de perícias: %d · %d só treinadas · %d com penalidade de armadura · "+
		"atributos conferidos contra a Tabela 2-1\n", len(expertises), trainedOnly, armorPenalty)
	return nil
}
