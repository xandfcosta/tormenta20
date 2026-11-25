package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O RAIO ARCANO do arcanista (ver GLOSSARY, **raio arcano**): o poder que dispara essência, e os dois que mexem
// nele.
const (
	ArcaneBoltPower    = "class.arcanista.raio-arcano"
	MightyBoltPower    = "class.arcanista.raio-poderoso"
	ElementalBoltPower = "class.arcanista.raio-elemental"
)

// ArcaneBoltElements são os cinco tipos do Raio Elemental, na ordem do livro, com a condição de quem falha no
// teste de Reflexos (p39).
var ArcaneBoltElements = []struct{ Kind, Condition string }{
	{"ácido", "vulnerável por 1 rodada"},
	{"eletricidade", "ofuscado por 1 rodada"},
	{"fogo", "em chamas"},
	{"frio", "lento por 1 rodada"},
	{"trevas", "sem poder curar PV por 1 rodada"},
}

// ArcaneBoltElementPM é o que o Raio Elemental cobra para mudar o tipo (p39).
const ArcaneBoltElementPM = 1

// ArcaneBolt é o Raio Arcano desta ficha: o dado, o alcance, a CD do atributo-chave e se ela muda o tipo.
type ArcaneBolt struct {
	Dice      string
	Range     string
	DC        int
	Elemental bool
}

// ArcaneBoltOf é o Raio Arcano da ficha, ou falso sem o poder: "1d8 pontos de dano de essência […] Esse dano
// aumenta em +1d8 para cada círculo de magia acima do 1º que você puder lançar" (p39), em d12 e alcance médio com
// o Raio Poderoso. A CD é a das magias arcanas do arcanista, porque o raio "conta como uma magia para efeitos de
// habilidades e itens que beneficiem suas magias".
//
//	bolt, has := sheet.ArcaneBoltOf(dto, computed) // 3d8, alcance curto, no arcanista de 10º
func ArcaneBoltOf(dto CharacterDTO, computed engine.ComputedSheet) (ArcaneBolt, bool) {
	owned := OwnedPowers(dto)
	has := func(id string) bool {
		return slices.ContainsFunc(owned, func(p OwnedPower) bool { return p.ID == id })
	}
	if !has(ArcaneBoltPower) {
		return ArcaneBolt{}, false
	}
	faces, reach := 8, "curto"
	if has(MightyBoltPower) {
		faces, reach = 12, "medio"
	}
	attribute := computed.SpellAttributeByClass["Arcanista"]
	at := engine.SpellAt{Tradition: "arcana", Classes: []string{"Arcanista"}}
	return ArcaneBolt{
		Dice:  fmt.Sprintf("%dd%d", max(1, HighestCastableCircle(dto.Classes)), faces),
		Range: reach, DC: computed.SpellCdByAttribute[attribute] + computed.ForSpell("spellDC", at).Total,
		Elemental: has(ElementalBoltPower),
	}, true
}

// Roll é a parcela do raio: de essência, ou do tipo que o Raio Elemental escolheu, com a condição de quem falha
// na nota. O tipo que não é um dos cinco, ou sem o Raio Elemental, é recusado.
func (b ArcaneBolt) Roll(element string) ([]book.SpellRoll, error) {
	if element == "" {
		return []book.SpellRoll{{Dice: b.Dice, Kind: "essência"}}, nil
	}
	if !b.Elemental {
		return nil, fmt.Errorf("o Raio Arcano só muda o tipo do dano com o Raio Elemental (p39)")
	}
	for _, e := range ArcaneBoltElements {
		if e.Kind == element {
			return []book.SpellRoll{{Dice: b.Dice, Kind: e.Kind, Note: "quem falha fica " + e.Condition}}, nil
		}
	}
	return nil, fmt.Errorf("o Raio Elemental causa ácido, eletricidade, fogo, frio ou trevas, e veio %q (p39)", element)
}
