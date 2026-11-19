package sheet

import (
	"strconv"
	"strings"

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
// (p227): 10 + metade do nível + o atributo. O dano do PREPARADO alquímico soma o que a ficha dá a
// ele (o Alquimista de Batalha soma a Inteligência, p68), e a CD dele pode vir de outro atributo,
// quando é maior (o Granadeiro, p69) — e aí a frase diz qual e por quê.
//
//	WrittenDose("acido", 8, computed) // "Ácido", "2d4 de ácido (Reflexos CD 17 reduz à metade)"
func WrittenDose(catalogID string, level int64, computed engine.ComputedSheet) (name, effect string) {
	dose := DoseEffectOf(catalogID)
	if dose == nil {
		return catalogID, ""
	}
	name = book.ItemByID(catalogID).Name
	effect = doseEffectPhrase(dose, computed.AlchemicalDamage)
	attribute, source := doseSaveAttribute(dose, computed)
	save, known := itemSaveDc(level, computed, attribute)
	if !known {
		return name, effect + " (" + dose.Save + " " + dose.OnSave + ")"
	}
	if source != "" {
		return name, effect + " (" + dose.Save + " CD " + strconv.Itoa(save) + ", pela " +
			book.AttributeLabel(attribute) + " do " + source + ", " + dose.OnSave + ")"
	}
	return name, effect + " (" + dose.Save + " CD " + strconv.Itoa(save) + " " + dose.OnSave + ")"
}

// namedParts são as parcelas que ENTRARAM num total, como a frase as diz: "+5 Veneno Potente". As
// descartadas não somaram, e dizê-las mentiria a conta.
func namedParts(sum *engine.TotalContribs) []string {
	parts := make([]string, 0, len(sum.Contributions))
	for _, c := range sum.Contributions {
		if !c.Dropped {
			parts = append(parts, book.WithSign(c.Amount)+" "+c.Source)
		}
	}
	return parts
}

// doseEffectPhrase monta o dano e o resto do efeito: "2d4+4 de ácido (+4 Alquimista de Batalha)",
// "6d6 de impacto em toda criatura a até 3m do alvo". A soma só vale para o PREPARADO, e diz de onde
// vem: a conta pronta e decomposta.
func doseEffectPhrase(dose *book.DoseEffect, alchemical *engine.TotalContribs) string {
	if dose.Damage == nil {
		return dose.Effect
	}
	dice, sources := dose.Damage.Dice, ""
	if dose.Alchemical && alchemical != nil {
		dice += "+" + strconv.Itoa(alchemical.Total)
		sources = " (" + strings.Join(namedParts(alchemical), ", ") + ")"
	}
	phrase := dice + " de " + dose.Damage.Type
	if dose.Effect != "" {
		phrase += " " + dose.Effect
	}
	return phrase + sources
}
