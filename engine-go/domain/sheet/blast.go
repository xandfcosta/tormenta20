package sheet

import "t20engine/domain/engine"

// BlastSaveDC é a CD da resistência ao que o encanto dispara, com o maior dos atributos que ele deixa
// escolher: "CD For ou Des, à sua escolha" (a Flamejante, p335), e quem escolhe escolhe o maior. A conta é a
// da p227, 10 + metade do nível + o atributo; zero quando a ficha não tem nenhum dos atributos computado.
//
//	BlastSaveDC(6, computed.Attributes, []string{"strength", "dexterity"}) // 17 com Força 4
func BlastSaveDC(level int64, attributes map[string]engine.AttributeBreakdown, keys []string) int {
	best, found := 0, false
	for _, key := range keys {
		attribute, known := attributes[key]
		if !known {
			continue
		}
		if !found || attribute.Total > best {
			best, found = attribute.Total, true
		}
	}
	if !found {
		return 0
	}
	return engine.EffectSaveDc(int(level), best)
}
