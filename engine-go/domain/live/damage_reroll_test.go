package live

import (
	"slices"
	"strings"
	"testing"
)

// O DESTRUIDOR NA MESA (ALE-652, fatia D): o dano já rolado se rola de novo UMA vez, antes de o mestre
// confirmar, por quem atacou ou pelo mestre — e o provisório guarda os dados da arma de antes, para a carta do
// mestre dizer o que mudou.
func TestTheRolledDamageIsRerolledOnceBeforeTheConfirmation(t *testing.T) {
	rolled := func() *SessionRuntimeState {
		st := EmptyRuntimeState()
		st.PendingAttack = &PendingAttack{ByUserID: 7, Ruling: RulingHit, DamageRolled: true, Dice: []int{1, 6}, Damage: 10}
		return st
	}
	player, stranger := Attacker{UserID: 7, Role: "player"}, Attacker{UserID: 9, Role: "player"}

	st := rolled()
	again := *st.PendingAttack
	again.Dice, again.Damage, again.DamageRerolledFrom = []int{5, 6}, 14, []int{1, 6}
	if err := RerollAttackDamage(st, player, again); err != nil {
		t.Fatalf("quem atacou rolar de novo o dano rolado: %v", err)
	}
	if !slices.Equal(st.PendingAttack.Dice, []int{5, 6}) || !st.PendingAttack.DamageRolled {
		t.Errorf("o provisório ficou com %+v, want os dados novos e o dano ainda rolado", st.PendingAttack)
	}
	if err := RerollAttackDamage(st, player, again); err == nil || !strings.Contains(err.Error(), "já") {
		t.Errorf("a segunda vez deu %v, want a recusa: vale o segundo resultado (p226)", err)
	}

	if err := RerollAttackDamage(rolled(), stranger, again); err == nil {
		t.Error("outro jogador rolou de novo o dano de quem atacou")
	}
	if err := RerollAttackDamage(rolled(), Attacker{UserID: 1, Role: "gm"}, again); err != nil {
		t.Errorf("o mestre rolar de novo, como ele pode rolar o dano: %v", err)
	}
	notYet := rolled()
	notYet.PendingAttack.DamageRolled = false
	if err := RerollAttackDamage(notYet, player, again); err == nil {
		t.Error("rolou de novo um dano que ainda não tinha sido rolado")
	}
}
