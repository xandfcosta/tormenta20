package sheet

import (
	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// itemSaveDc é a CD de quem aplica um item (p227): 10 + metade do nível + o atributo que o item
// pede — a Destreza do ácido, a Inteligência do veneno. known é falso sem a ficha computada, e aí
// não há CD a dizer: um número feito com atributo zero pareceria certo. O que soma a ela por cima (o
// próprio veneno, o Veneno Potente) é de quem chama.
func itemSaveDc(level int64, computed engine.ComputedSheet, attribute string) (dc int, known bool) {
	value, known := computed.Attributes[attribute]
	if !known {
		return 0, false
	}
	return engine.EffectSaveDc(int(level), value.Total), true
}

// doseSaveAttribute é o atributo da CD da dose: o que o item pede, ou o que o motor escolheu no lugar
// dele no item ALQUÍMICO (`engine.LicensedAttribute`), com quem autorizou.
func doseSaveAttribute(dose *book.DoseEffect, computed engine.ComputedSheet) (attribute, source string) {
	if licensed, changed := computed.AlchemicalSaveAttributes[dose.SaveAttribute]; dose.Alchemical && changed {
		return licensed.Attribute, licensed.Source
	}
	return dose.SaveAttribute, ""
}
