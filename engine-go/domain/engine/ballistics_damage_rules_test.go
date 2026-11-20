package engine

import (
	"strings"
	"testing"
)

// A BALÍSTICA NO DANO (p69): "quando usa uma arma de ataque à distância, pode usar sua Inteligência em
// vez de Destreza nos testes de ataque (e, caso possua o poder Estilo de Disparo, nas rolagens de
// dano)". O Estilo de Disparo soma a Destreza no dano com arma de disparo (p125); com a Balística, a
// Inteligência entra no lugar dela quando é MAIOR — todos dizem "pode", e a carta fica com o maior.
// O caso compara o arco com o interruptor do estilo desligado e ligado, com os números à mão.
func TestTheBallisticsPutsIntelligenceInTheFiredStyleDamage(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		name       string
		ballistics bool
		intel, dex int
		damage     int
	}{
		{"com a Balística e a Inteligência maior", true, 5, 3, 5},
		{"sem a Balística vale a Destreza", false, 5, 3, 3},
		{"com a Balística e a Destreza maior", true, 2, 3, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			powers := `["estilo-de-disparo"]`
			if c.ballistics {
				powers = `["estilo-de-disparo","class.inventor.balistica"]`
			}
			bow := "arco-curto"
			hero := Character{
				Level: 5, Dexterity: c.dex, Intelligence: c.intel, ClassPowers: powers,
				Classes: []CharacterClass{{ClassName: "Inventor", Level: 5}},
				Items:   []CharacterItem{{CatalogID: &bow, Name: bow, Equipped: ptrTo("wielded"), Improvements: "[]"}},
			}
			on := map[string]bool{FlagGroupID("estilo-de-disparo"): true}
			off, lit := world.ComputeWeaponCards(hero, nil)[0], world.ComputeWeaponCards(hero, on)[0]
			if got := lit.DamageBonus - off.DamageBonus; got != c.damage {
				t.Errorf("o Estilo de Disparo somou %+d no dano, e o livro dá %+d", got, c.damage)
			}
			// A PARCELA DIZ POR QUE o número não é o da Destreza, só quando a troca acontece.
			swapped := false
			for _, part := range append(lit.DamageAll.Contributions, lit.DamageOwn.Contributions...) {
				if strings.Contains(part.Note, "Balística: Inteligência no lugar de Destreza") {
					swapped = true
				}
			}
			if want := c.damage == c.intel && c.intel > c.dex; swapped != want {
				t.Errorf("a parcela do Estilo de Disparo diz a troca: %v, e devia: %v", swapped, want)
			}
		})
	}
}
