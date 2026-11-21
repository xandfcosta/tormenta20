package sheetui

import (
	"testing"

	"t20engine/domain/engine"
)

// A PARCELA DO DANO TROCADO DIZ O ATRIBUTO QUE A CARTA TROUXE (ALE-672). A licença do catálogo aceita qualquer
// atributo; a tela não pode supor a Inteligência. Uma licença de Sabedoria tem de sair "SAB" na linha compacta e
// "Sabedoria" na conta, com a frase da licença.
func TestTheSwappedDamageRowNamesTheAttributeTheCardCarries(t *testing.T) {
	card := engine.WeaponCard{Skill: "Luta", AttributeDamage: 3, DamageBonus: 3,
		DamageSwapAttribute: "wisdom", DamageSwapNote: "Exemplo: Sabedoria no lugar de Força"}
	if _, damage := weaponRows(card); len(damage) == 0 || damage[0].Label != "SAB" || damage[0].Value != "+3" {
		t.Errorf("a primeira parcela do dano trocado pela Sabedoria não é SAB +3: %+v", damage)
	}
	if _, terms := weaponSumTerms(card); len(terms) == 0 || terms[0].Label != "Sabedoria" || terms[0].Note != card.DamageSwapNote {
		t.Errorf("a conta do dano trocado pela Sabedoria não diz Sabedoria com a frase da licença: %+v", terms)
	}
}
