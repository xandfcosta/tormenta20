package character

import (
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// UseGate é a DECISÃO DE USAR desta ficha, com as entradas juntadas UMA vez: o PM, as regras de custo, as flags
// acesas e os usos já gastos. A tela, o gesto de usar e a entrada na postura perguntam a ele, e por isso
// respondem a mesma coisa — antes eram três montagens, duas fontes de flag e uma entrada na postura que pulava as
// recusas de flag e de limite (ALE-679). A regra em si é a `book.UseDecision`; o portão só a alimenta.
//
//	gate := character.UseGateOf(dto)
//	price, refusal := gate.Decide(spec, book.Use{Steps: 1, OnAlly: true})
type UseGate struct {
	dto   sheet.CharacterDTO
	rules engine.AbilityCostRules
	flags map[string]bool
	uses  map[string]PowerUse
	// maxSteps guarda o teto de cada habilidade já perguntada: ele pode pedir a ficha computada (o teto pelo
	// atributo), e a tela e a decisão o leem na mesma linha.
	maxSteps map[string]int
}

// UseGateOf monta o portão de uma ficha.
func UseGateOf(dto sheet.CharacterDTO) UseGate {
	gate := UseGate{dto: dto, flags: map[string]bool{}, uses: PowerUses(dto), maxSteps: map[string]int{}}
	effects, _, err := sheet.CharacterEffects(dto)
	if err != nil {
		return gate
	}
	gate.rules = effects.AbilityCostRules()
	// AS FLAGS SÃO OS CONDICIONAIS LIGADOS (decisão do dono): as mesmas que o motor aplica nos números. A postura
	// paga acende a dela ali, então o gesto que lia as posturas e a tela que lia os condicionais concordavam — por
	// coincidência, e não por construção.
	//
	// E a flag não está no banco: o banco guarda os condicionais ligados, e é o motor quem diz que flag cada um
	// acende — a pergunta "a Fúria está em pé?" é uma junção dos dois. Sem regras montáveis o mapa sai vazio, e
	// o poder de gatilho não é oferecido, que é o lado seguro.
	on := sheet.ToStringSet(dto.Conditionals)
	for _, conditional := range effects.Conditional {
		if conditional.Flag != "" && on[engine.FlagGroupID(conditional.Flag)] {
			gate.flags[conditional.Flag] = true
		}
	}
	return gate
}

// Decide é o preço do uso e a recusa, vazia quando pode (ver `book.UseDecision`).
func (g UseGate) Decide(spec book.Activation, use book.Use) (int, string) {
	used := g.Used(spec)
	return book.UseDecision(spec, book.UseContext{
		CurrentPM: int(g.dto.ManaAvailable()), Cost: g.rules, MaxSteps: g.MaxSteps(spec),
		UsedThisScene: used.Scene, UsedToday: used.Day, Flags: g.flags,
	}, use)
}

// MaxSteps é o teto de degraus da habilidade (ver `PowerMaxSteps`): o MESMO que a decisão confere, para a tela
// não oferecer um contador que o gesto recusa.
func (g UseGate) MaxSteps(spec book.Activation) int {
	if max, known := g.maxSteps[spec.ID]; known {
		return max
	}
	max := PowerMaxSteps(g.dto, spec)
	g.maxSteps[spec.ID] = max
	return max
}

// Used são os usos já gastos da habilidade, os que a decisão confere contra o limite.
func (g UseGate) Used(spec book.Activation) PowerUse {
	return g.uses[spec.ID]
}

// CostRules são as regras de custo da ficha, para quem escreve o preço sem decidir (o rótulo, a prévia).
func (g UseGate) CostRules() engine.AbilityCostRules {
	return g.rules
}

// decidedPrice é o preço do uso, ou a recusa como erro com o nome da habilidade.
func decidedPrice(dto sheet.CharacterDTO, spec book.Activation, use book.Use) (int, error) {
	price, refusal := UseGateOf(dto).Decide(spec, use)
	if refusal != "" {
		return 0, fmt.Errorf("%s: %s", spec.Name, refusal)
	}
	return price, nil
}
