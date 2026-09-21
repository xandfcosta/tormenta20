package character

import (
	"testing"

	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
)

// A FRONTEIRA DO CÍRCULO É DO SERVIDOR, e não da tela.
//
// # O buraco que este guarda fecha
//
// 126 dos 486 aprimoramentos do catálogo exigem um círculo mínimo, e até a fatia
// 6 esse limite existia SÓ na interface: a tabela que o decide vivia no
// TypeScript, então o `validateAugments` nem tinha como perguntar. Um pedido
// montado à mão conjurava o que a regra não permite, e nada acusava.
func TestAnAugmentAboveTheCircleIsRefusedByTheServer(t *testing.T) {
	spell, known := catalog.LookupSpell("invisibilidade")
	if !known {
		t.Fatal("a magia `invisibilidade` sumiu do catálogo — o caso mediria o vazio")
	}
	target := -1
	for i, a := range spell.Augments {
		if a.RequiresCircle != nil && *a.RequiresCircle >= 3 {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatal("a Invisibilidade não tem aprimoramento com `requiresCircle` ≥ 3: " +
			"ou o catálogo mudou, ou o campo parou de ser lido — nos dois casos este " +
			"guarda mediria o vazio")
	}
	picks := []sheet.AugmentPick{{AugmentIndex: target, Stacks: 1}}

	// QUEM NÃO ALCANÇA é recusado…
	if _, failure := validateAugments(spell, picks, 2); failure == "" {
		t.Error("um aprimoramento de círculo alto passou para quem alcança só o 2º")
	}
	// …e quem alcança, passa. Sem esta metade o guarda ficaria verde numa
	// validação que recusa TUDO.
	if _, failure := validateAugments(spell, picks, 5); failure != "" {
		t.Errorf("quem alcança o 5º foi recusado: %s", failure)
	}
}
