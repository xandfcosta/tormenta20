package api

import (
	"context"
	"testing"

	"t20engine/domain/board"
)

// O ALCANCE DA ESCARAMUÇA MEDIDO PELO TABULEIRO (ALE-652, decisão do dono): o +1d8 vale em ataque à distância
// só com o alvo em alcance curto (9m, p51). Com as duas peças no mapa, o golpe com o arco contra o Goblin a três
// quadrados leva o dado, e a oito, não — e a conta diz por quê.
func TestTheSkirmishDieFollowsTheDistanceOnTheBoard(t *testing.T) {
	for _, c := range []struct {
		name   string
		apart  int
		withIt bool
	}{
		{"a três quadrados, em alcance curto", 3, true},
		{"a oito quadrados, além do curto", 8, false},
	} {
		f, id, goblin := styleOnTurn(t, "class.cacador.escaramuca", false, "arco-curto")
		seedClasse(t, f.s, id, "Caçador", 6)
		turnSwitchOn(t, f.s, id, "escaramuca")
		f.seedOpenBoard(t, "stone")
		hunter := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Furioso")
		for _, token := range []board.BoardToken{
			{Label: "Furioso", Kind: "character", X: 0, Y: 0, Footprint: 1, EntryID: &hunter},
			{Label: "Goblin", Kind: "npc", X: c.apart, Y: 0, Footprint: 1, EntryID: &goblin},
		} {
			if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab, token); err != nil {
				t.Fatalf("pôr %q: %v", token.Label, err)
			}
		}
		if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
			t.Fatalf("%s: o disparo foi recusado: %q", c.name, refusal)
		}
		said := false
		for _, s := range pendingOf(t, f).Situations {
			said = said || s == "além do alcance curto"
		}
		if said == c.withIt {
			t.Errorf("%s: a conta diz \"além do alcance curto\" = %t", c.name, said)
		}
		gmRules(t, f, "acertou")
		eight := false
		for _, group := range pendingOf(t, f).DamageToRoll.Dice {
			eight = eight || group.Faces == 8
		}
		if eight != c.withIt {
			t.Errorf("%s: o dano a rolar leva o d8 da Escaramuça = %t, want %t", c.name, eight, c.withIt)
		}
	}
}
