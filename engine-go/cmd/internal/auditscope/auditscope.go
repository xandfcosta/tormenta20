// Package auditscope diz qual auditor do livro (`cmd/audit/<nome>`) mede cada item do catálogo.
//
// Cada auditor confere uma família de tabelas do livro, e o item é de UM deles. A regra mora aqui, e não em cada
// auditor, porque o `equipment` conta à parte o que é dos irmãos e aponta o órfão: com a seleção espalhada, um
// irmão que mudasse a sua deixava essa conta mentindo. Os auditores que escolhem pelo CATÁLOGO selecionam por
// `Of`; os que escolhem pela TABELA do livro (o `specificitems`, que também resolve as peças de cada composição)
// consultam o catálogo por nome, e a regra diz de quem é o item, não por onde ele é achado.
package auditscope

import "t20engine/domain/catalog/transcript"

// Auditor é o nome do auditor em `cmd/audit/`.
type Auditor string

const (
	Equipment     Auditor = "equipment"
	Materials     Auditor = "materials"
	Enchants      Auditor = "enchants"
	SpecificItems Auditor = "specificitems"
	Potions       Auditor = "potions"
	Accessories   Auditor = "accessories"
	Artifacts     Auditor = "artifacts"
	// Nobody é o item que auditor nenhum alcança — o que o `TestEveryCatalogItemHasTheAuditorThatMeasuresIt` reprova.
	Nobody Auditor = "nenhum"
)

// All são os auditores que medem itens, sem o `Nobody`.
var All = []Auditor{Equipment, Materials, Enchants, SpecificItems, Potions, Accessories, Artifacts}

// As páginas do capítulo 3 no LIVRO: o que está fora delas é de um auditor do capítulo 8.
const (
	chapterThreeFirst = 138
	chapterThreeLast  = 167
)

// Of é o auditor que mede o item. A PÁGINA decide primeiro, porque o capítulo 3 também tem `apparel` e o capítulo 8
// também tem arma; dentro do capítulo 8, o tipo do verbete diz qual tabela o tem. O artefato vem antes da
// composição: a Espada-Deus é composta e é artefato, e a p346 tira o artefato das tabelas de sorteio.
//
//	auditscope.Of(transcript.Item{Category: "material", BookPage: 166}) // Materials
func Of(it transcript.Item) Auditor {
	if it.BookPage >= chapterThreeFirst && it.BookPage <= chapterThreeLast {
		if it.Category == "material" {
			return Materials
		}
		return Equipment
	}
	switch {
	case it.Category == "weapon-enchant" || it.Category == "armor-enchant":
		return Enchants
	case it.Artifact:
		return Artifacts
	case it.ComposedOf != nil:
		return SpecificItems
	case it.Consumable != nil && it.Consumable.Spell != "":
		return Potions
	case it.Category == "apparel":
		return Accessories
	}
	return Nobody
}
