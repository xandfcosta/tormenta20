package engine

import (
	"path/filepath"
	"testing"
)

// TestTheConditionalGodPowersOfferTheirSwitch (ALE-399).
//
// A ALE-397 pôs na ficha os SETE poderes concedidos de bônus fixo. Os que
// sobraram são os CONDICIONAIS — o livro dá o bônus só em certa circunstância:
//
//	"AFINIDADE COM A TORMENTA. +10 em testes de resistência contra efeitos da
//	 Tormenta […]"                                                      — p132
//	"ARMAS DA AMBIÇÃO. +1 em testes de ataque e na margem de ameaça com armas
//	 nas quais é proficiente."                                          — p132
//	"ARSENAL DAS PROFUNDEZAS. +2 nas rolagens de dano com azagaias, lanças e
//	 tridentes e seu multiplicador de crítico com essas armas aumenta em +1."
//	                                                                    — p132
//	"TRADIÇÃO DE LIN-WU. […] recebe +1 na margem de ameaça com ela [a katana]."
//	                                                                    — p135
//	"REJEIÇÃO DIVINA. Você recebe resistência a magia divina +5."        — p135
//	"SANGUE OFÍDICO. Você recebe resistência a veneno +5 […]"            — p135
//
// # O que faz deles uma família
//
// Nenhum precisou de vocabulário novo: o motor já tem `context` (o interruptor
// que a pessoa liga), `against` (contra o quê) e os alvos `attack`, `damage`,
// `critRange`, `critMult` e `resistance`. "Resistência a X +N" é bônus em
// TESTE e não redução de dano, e quem decide isso é o livro: "uma criatura com
// resistência a magia +2 recebe +2 em testes" (p226).
//
// # As duas asserções que importam são opostas
//
// DESLIGADO por padrão — um bônus circunstancial que vale sempre é um bônus
// errado — e LIGADO quando a pessoa aciona o termo. Um caso que só medisse a
// segunda passaria com um modificador incondicional, que é justamente o defeito
// que ele existe para impedir.
func TestTheConditionalGodPowersOfferTheirSwitch(t *testing.T) {
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))

	for _, caso := range []struct {
		poder  string
		alvo   ModifierTarget
		quero  int
		pagina string
	}{
		{"Afinidade com a Tormenta", ModifierTarget{K: "resistance"}, 10, "p132"},
		{"Armas da Ambição", ModifierTarget{K: "attack", Scope: "all"}, 1, "p132"},
		{"Armas da Ambição", ModifierTarget{K: "critRange"}, 1, "p132"},
		{"Arsenal das Profundezas", ModifierTarget{K: "damage", Scope: "all"}, 2, "p132"},
		{"Arsenal das Profundezas", ModifierTarget{K: "critMult"}, 1, "p132"},
		{"Tradição de Lin-Wu", ModifierTarget{K: "critRange"}, 1, "p135"},
		{"Rejeição Divina", ModifierTarget{K: "resistance"}, 5, "p135"},
		{"Sangue Ofídico", ModifierTarget{K: "resistance"}, 5, "p135"},
	} {
		desligado := ComputeItemEffects(world.ActiveItemsFor(
			Character{Level: 1, GodPower: caso.poder}))

		if got := StatFor(desligado, caso.alvo).Total; got != 0 {
			t.Errorf("%s: %s vale %d com o interruptor DESLIGADO, e o livro o dá só "+
				"na circunstância (%s) — um bônus circunstancial que vale sempre é "+
				"um bônus errado", caso.poder, caso.alvo.K, got, caso.pagina)
		}

		// O termo sai do próprio efeito: a `Condition` se perde na conversão
		// para `ConditionalEffect`, e é o `Term` que o opt-in da pessoa usa.
		ligados := map[string]bool{}
		for _, c := range desligado.Conditional {
			ligados[c.Term] = true
		}
		if len(ligados) == 0 {
			t.Errorf("%s: não ofereceu interruptor nenhum — o modificador não chegou "+
				"ou não é condicional", caso.poder)
			continue
		}

		ligado := ApplyActiveConditionals(desligado, ligados)
		if got := StatFor(ligado, caso.alvo).Total; got != caso.quero {
			t.Errorf("%s: %s deu %d com o interruptor ligado e o livro dá +%d (%s)",
				caso.poder, caso.alvo.K, got, caso.quero, caso.pagina)
		}
	}
}
