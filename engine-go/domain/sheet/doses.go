package sheet

import (
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// AS DOSES DA INJEÇÃO ALQUÍMICA (p165): "um ataque que acerte causa seu dano normal e
// libera uma carga de um preparado [...] ou de água benta, que atinge o alvo
// automaticamente. A melhoria tem espaço para 2 doses."
//
// A dose fica escrita, e não entra no dano da arma: quem aplica o efeito é o mestre.
// O livro também não diz se "atinge automaticamente" dispensa o teste do item, então
// a faixa mostra o teste com a CD, e a palavra é dele (decisão do dono).

const (
	AlchemicalInjection     = "melhoria-injecao-alquimica"
	AlchemicalInjectionName = "Injeção alquímica"
	MaxDoses                = 2
)

// DoseEffectOf é o efeito do item na Injeção alquímica, ou nil se ele não carrega.
func DoseEffectOf(catalogID string) *book.DoseEffect {
	item := book.ItemByID(catalogID)
	if item == nil || item.Consumable == nil {
		return nil
	}
	return item.Consumable.Dose
}

// HoldsDoses diz se a coluna de melhorias da arma tem a Injeção alquímica.
func HoldsDoses(improvements string) bool {
	for _, id := range UnmarshalStrings(improvements) {
		if id == AlchemicalInjection {
			return true
		}
	}
	return false
}

// WrittenDose é o nome do item e o efeito dele como a mesa lê, com a CD de quem atacou
// (p227): 10 + metade do nível + o atributo.
//
//	WrittenDose("acido", 8, computed.Attributes) // "Ácido", "2d4 de ácido (Reflexos CD 17 reduz à metade)"
func WrittenDose(catalogID string, level int64, attributes map[string]engine.AttributeBreakdown) (name, effect string) {
	dose := DoseEffectOf(catalogID)
	if dose == nil {
		return catalogID, ""
	}
	name = book.ItemByID(catalogID).Name
	// Sem a ficha computada não há CD a dizer, e um número feito com atributo zero
	// pareceria certo.
	attribute, known := attributes[dose.SaveAttribute]
	if !known {
		return name, dose.Effect + " (" + dose.Save + " " + dose.OnSave + ")"
	}
	save := engine.EffectSaveDc(int(level), attribute.Total)
	return name, dose.Effect + " (" + dose.Save + " CD " + strconv.Itoa(save) + " " + dose.OnSave + ")"
}
