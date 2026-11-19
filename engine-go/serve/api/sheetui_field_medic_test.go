package api

import (
	"context"
	"strings"
	"testing"
)

// MÉDICO DE CAMPO (origem Curandeiro, p89): "você soma sua Sabedoria aos PV restaurados por suas
// habilidades e itens MUNDANOS de cura". O par mágico é a Cura Gentil (p133), e por isso a Poção de
// Curar Ferimentos não ganha nada. No app, a cura mundana com fluxo é o Bálsamo restaurador (p160):
// o Medicina e os cuidados prolongados a mesa rola e o mestre aplica (decisão do dono).
//
// O Guerreiro de nível 3 com 20 de dano está com 10 de 30 PV, e tem Sabedoria 3.
func TestTheFieldMedicAddsWisdomToTheBalm(t *testing.T) {
	f, id := fieldMedic(t, true)
	item := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	if refusal := use(t, f, id, item, `{"item_roll_hp":7}`); refusal != "" {
		t.Fatalf("usar foi recusado: %q", refusal)
	}
	if pool := poolsOf(t, f.s, id); pool.HpCurrent != 20 {
		t.Errorf("o PV ficou %d, quer 20 (10 + os 7 rolados + 3 de Sabedoria)", pool.HpCurrent)
	}
}

// A POÇÃO É MÁGICA, e a cura dela é a mesma com e sem o Médico de Campo. Comparar com a ficha sem o
// poder, e não com um número escrito, é o que deixa este caso fora do bônus fixo da poção.
func TestTheFieldMedicLeavesThePotionAlone(t *testing.T) {
	const potion, name = "pocao-de-curar-ferimentos-2d8-2-pv", "Poção de Curar Ferimentos (2d8+2 PV)"
	healed := func(medic bool) int64 {
		f, id := fieldMedic(t, medic)
		item := itemSemeia(t, f, id, potion, name, "")
		if refusal := use(t, f, id, item, `{"item_roll_hp":5}`); refusal != "" {
			t.Fatalf("usar foi recusado: %q", refusal)
		}
		return poolsOf(t, f.s, id).HpCurrent
	}
	if with, without := healed(true), healed(false); with != without {
		t.Errorf("a poção curou até %d com o Médico de Campo e até %d sem: a Sabedoria entrou na cura mágica", with, without)
	}
}

// O USAR DO BÁLSAMO DIZ A SOMA antes de curar, e de onde ela vem: a conta pronta e decomposta.
func TestTheBalmDialogSaysTheFieldMedicBonus(t *testing.T) {
	f, id := fieldMedic(t, true)
	item := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")

	f.charID = id
	dialog := itemDialogText(t, f, item)
	if !strings.Contains(dialog, "PV rolados (2d4) · +3 Origem: Curandeiro") {
		t.Errorf("o Usar do bálsamo não diz a soma do Médico de Campo:\n%s", dialog)
	}
}

// fieldMedic é um Guerreiro de nível 3, com 20 de dano e Sabedoria 3, Curandeiro com ou sem o
// Médico de Campo.
func fieldMedic(t *testing.T, medic bool) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Curandeira", "Guerreiro", 3, 20, 0)
	seedMember(t, f.s, f.campaignID, id)
	choices := "[]"
	if medic {
		choices = `["origin-curandeiro-unique"]`
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET origin = 'curandeiro', originChoices = ?, wisdom = 3 WHERE id = ?", choices, id); err != nil {
		t.Fatalf("fazer da ficha uma curandeira: %v", err)
	}
	return f, id
}
