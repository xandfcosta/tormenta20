package engine

import (
	"path/filepath"
	"testing"
)

// A MANOBRA É UM TESTE OPOSTO, E OS DOIS LADOS NÃO SÃO O MESMO BÔNUS.
//
//	"Faça um teste de manobra (um teste de ataque corpo a corpo) OPOSTO com a
//	 criatura. Mesmo que ela esteja usando uma arma de ataque à distância, deve
//	 fazer o teste usando seu valor de Luta."                            — p234
//
// Quem ataca e quem defende rolam Luta, e por isso a primeira modelagem do
// "Desejo de Liberdade" (+5 contra agarrar, p89) foi um bônus em
// `expertise:Luta`. Ela estava errada e a medição mostrou como: em T20 o ATAQUE
// também é um teste de Luta, então o +5 de quem se defende subia o ataque da
// arma empunhada — a Adaga ia de -5 para 0 ao ligar o interruptor.
//
// O alvo `maneuver` com `scope: "defense"` separa os dois usos da perícia. Ele
// não alimenta número nenhum da ficha (nem o de ofensa alimenta): a manobra é um
// bônus que a PESSOA aplica na mesa, e o que a ficha deve é dizer qual é.
func TestTheDefensiveManeuverBonusNeverReachesTheAttack(t *testing.T) {
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))
	adaga, empunhada := "adaga", "wielded"
	base := Character{
		Level:      1,
		Expertises: []CharacterExpertise{{Name: "Luta", Attribute: "strength"}},
		Items: []CharacterItem{{
			CatalogID: &adaga, Name: "Adaga", Quantity: 1, Equipped: &empunhada,
		}},
	}
	comPoder := base
	comPoder.Origin = "escravo"
	comPoder.OriginChoices = `["origin-escravo-unique"]`

	ataqueDe := func(ch Character) int {
		cards := world.ComputeWeaponCards(ch, nil)
		if len(cards) != 1 {
			t.Fatalf("esperava uma carta de arma e vieram %d — sem ela o caso não "+
				"mede o vazamento que existe para pegar", len(cards))
		}
		return cards[0].Attack
	}

	semPoder := ataqueDe(base)
	if got := ataqueDe(comPoder); got != semPoder {
		t.Errorf("o ataque da Adaga é %d sem o Desejo de Liberdade e %d com ele. O "+
			"+5 da p89 é do teste de quem DEFENDE de uma manobra; ele chegar ao "+
			"ataque é o bônus valendo para o lado errado do teste oposto (p234)",
			semPoder, got)
	}

	// E o CONTROLE: o bônus existe mesmo — sem esta metade, um modificador que
	// nunca chegou ao motor passaria por "não vaza".
	efeitos := ComputeItemEffects(world.ActiveItemsFor(comPoder))
	defesa := ModifierTarget{K: "maneuver", Name: "agarrar", Scope: "defense"}
	if got := StatFor(efeitos, defesa).Total; got != 5 {
		t.Errorf("resistir a agarrar vale %d e o livro dá +5 (p89) — o modificador "+
			"não chegou, e o caso acima estaria medindo a ausência dele", got)
	}
	// Os dois lados do teste oposto são baldes DIFERENTES. Sem o escopo na chave
	// eles somariam, e um personagem com os dois poderes teria +7 nos dois lados.
	ofensa := ModifierTarget{K: "maneuver", Name: "agarrar"}
	if got := StatFor(efeitos, ofensa).Total; got != 0 {
		t.Errorf("agarrar OFENSIVO vale %d por causa de um poder que é de defesa — "+
			"o escopo não está entrando na chave do efeito", got)
	}
}
