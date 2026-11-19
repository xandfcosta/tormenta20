package api

import (
	"strings"
	"testing"
)

// O USAR SOMA O BÔNUS FIXO DO ITEM à rolagem que a mesa digita. A Poção de Curar Ferimentos cura
// 2d8+2 (p341): o campo pergunta o dado, e o +2 vem por cima, dito à parte (decisão do dono). O
// Guerreiro de nível 3 com 20 de dano está com 10 de 30 PV.
func TestThePotionAddsItsFixedBonusToTheTableRoll(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Ferido", "Guerreiro", 3, 20, 0)
	seedMember(t, f.s, f.campaignID, id)
	item := itemSemeia(t, f, id, "pocao-de-curar-ferimentos-2d8-2-pv", "Poção de Curar Ferimentos (2d8+2 PV)", "")

	if refusal := use(t, f, id, item, `{"item_roll_hp":5}`); refusal != "" {
		t.Fatalf("usar foi recusado: %q", refusal)
	}
	if pool := poolsOf(t, f.s, id); pool.HpCurrent != 17 {
		t.Errorf("o PV ficou %d, quer 17 (10 + os 5 rolados + os 2 da poção)", pool.HpCurrent)
	}
}

func TestThePotionFieldSaysTheFixedBonus(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Ferido", "Guerreiro", 3, 20, 0)
	seedMember(t, f.s, f.campaignID, id)
	item := itemSemeia(t, f, id, "pocao-de-curar-ferimentos-2d8-2-pv", "Poção de Curar Ferimentos (2d8+2 PV)", "")

	f.charID = id
	if dialog := itemDialogText(t, f, item); !strings.Contains(dialog, "PV rolados (2d8) · +2 do item") {
		t.Errorf("o Usar da poção não diz o +2 do item:\n%s", dialog)
	}
}

// AS REFEIÇÕES DÃO PONTOS TEMPORÁRIOS (p162), e não curam: o Macarrão de Yuvalin, +5 PV
// temporários; o Gorad quente, +2 PM temporários. Os temporários somam por cima, mesmo além do
// máximo, e somem no fim do dia (p106).
func TestTheNoodlesGiveTemporaryHpAndDoNotHeal(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Faminto", "Guerreiro", 3, 20, 0)
	seedMember(t, f.s, f.campaignID, id)
	item := itemSemeia(t, f, id, "macarrao-de-yuvalin", "Macarrão de Yuvalin", "")

	if refusal := use(t, f, id, item, ""); refusal != "" {
		t.Fatalf("comer foi recusado: %q", refusal)
	}
	if pool := poolsOf(t, f.s, id); pool.HpCurrent != 10 {
		t.Errorf("o PV ficou %d, quer 10: o macarrão curou, e ele dá PV temporários", pool.HpCurrent)
	}
	if temp := tempHpOf(t, f, id); temp != 5 {
		t.Errorf("os PV temporários ficaram %d, quer 5", temp)
	}
}

func TestTheGoradGivesTemporaryMp(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Sonolento", "Arcanista", 3, 0, 0)
	seedMember(t, f.s, f.campaignID, id)
	before := poolsOf(t, f.s, id)
	item := itemSemeia(t, f, id, "gorad-quente", "Gorad quente", "")

	if refusal := use(t, f, id, item, ""); refusal != "" {
		t.Fatalf("beber foi recusado: %q", refusal)
	}
	after := poolsOf(t, f.s, id)
	if after.TempMp != 2 {
		t.Errorf("os PM temporários ficaram %d, quer 2", after.TempMp)
	}
	if after.MpCurrent != before.MpCurrent {
		t.Errorf("o PM foi de %d para %d: o gorad dá PM temporários, e não recupera", before.MpCurrent, after.MpCurrent)
	}
}
