package sheetui

import (
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O ATRIBUTO NO LUGAR DO DA PERÍCIA aparece nas duas leituras dela: a nota debaixo do nome diz DES, e
// não o FOR gravado na ficha, e a conta diz quem pôs a Destreza ali (o Pequeno e Rechonchudo, p28).
func TestTheSwappedAttributeShowsInTheRowAndInTheSum(t *testing.T) {
	computed := engine.ComputedSheet{Expertises: []engine.ExpertiseBreakdown{{
		Name: "Atletismo", Attribute: "dexterity", AttributeSource: "Pequeno e Rechonchudo", AttrValue: 4, Total: 4,
	}}}
	row := expertiseRowFor(0, sheet.ExpertiseDTO{Name: "Atletismo", Attribute: "strength"}, computed)
	if got := expertiseHint(row); got != "DES" {
		t.Errorf("a nota do Atletismo diz %q; a perícia rola com a Destreza", got)
	}
	if row.Attribute != "strength" {
		t.Errorf("o seletor do atributo marca %q; ele mostra o GRAVADO, e salvar não pode trocá-lo", row.Attribute)
	}
	terms := expertiseSumTerms(computed.Expertises[0])
	if terms[0].Label != "Destreza" || terms[0].Note != "Pequeno e Rechonchudo, no lugar do atributo da perícia" {
		t.Errorf("a parcela do atributo saiu %+v; ela diz quem pôs a Destreza ali", terms[0])
	}
}
