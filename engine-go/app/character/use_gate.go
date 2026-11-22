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
}

// UseGateOf monta o portão de uma ficha.
func UseGateOf(dto sheet.CharacterDTO) UseGate {
	gate := UseGate{dto: dto, flags: map[string]bool{}, uses: PowerUses(dto)}
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
	used := g.uses[spec.ID]
	return book.UseDecision(spec, book.UseContext{
		CurrentPM: int(g.dto.ManaAvailable()), Cost: g.rules, MaxSteps: PowerMaxSteps(g.dto, spec),
		UsedThisScene: used.Scene, UsedToday: used.Day, Flags: g.flags,
	}, use)
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
