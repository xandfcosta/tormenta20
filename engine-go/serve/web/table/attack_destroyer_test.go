package table

import (
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O DESTRUIDOR NA CARTA (ALE-652, fatia D): o botão é de quem rolou, no dano rolado e ainda não confirmado, com
// um campo da mão por dado da arma que saiu 1 ou 2 — e some depois de usado. A carta do mestre diz o que mudou.
func TestTheDestroyerIsOfferedToWhoRolledWithOneFieldPerLowDie(t *testing.T) {
	rolled := func() *live.SessionRuntimeState {
		st := live.EmptyRuntimeState()
		st.PendingAttack = &live.PendingAttack{ByUserID: 7, Ruling: live.RulingHit, DamageRolled: true,
			RerollsLowDice: true, Dice: []int{1, 6, 2}, Faces: 6, RawDamage: 12, Damage: 12}
		return st
	}
	// Pelo RECORTE do jogador, que é como a carta chega a quem rolou: o `forPlayer` copia campo a campo, e um
	// campo que ele não copia some da tela com o teste do `attackProposalOf` verde.
	st := rolled()
	mine := attackProposalOf(st, 7).forPlayer(*st.PendingAttack)
	if len(mine.Destroyer) != 1 || mine.Destroyer[0].Label != "2d6" || len(mine.Destroyer[0].Signals) != 2 {
		t.Fatalf("quem rolou recebeu %+v, want um grupo 2d6 com dois campos (o 1 e o 2)", mine.Destroyer)
	}
	if other := attackProposalOf(rolled(), 9); other.Destroyer != nil {
		t.Errorf("quem não rolou recebeu o Destruidor: %+v", other.Destroyer)
	}
	noLow := rolled()
	noLow.PendingAttack.Dice = []int{3, 6, 4}
	if got := attackProposalOf(noLow, 7); got.Destroyer != nil {
		t.Errorf("sem 1 nem 2 na arma, o Destruidor foi oferecido: %+v", got.Destroyer)
	}
	plain := rolled()
	plain.PendingAttack.RerollsLowDice = false
	if got := attackProposalOf(plain, 7); got.Destroyer != nil {
		t.Errorf("a arma sem o Destruidor o ofereceu: %+v", got.Destroyer)
	}

	used := rolled()
	used.PendingAttack.Dice, used.PendingAttack.DamageRerolledFrom = []int{5, 6, 1}, []int{1, 6, 2}
	after := attackProposalOf(used, 7).forPlayer(*used.PendingAttack)
	if after.Destroyer != nil {
		t.Errorf("o Destruidor continuou oferecido depois de usado: %+v", after.Destroyer)
	}
	if after.DamageRerolled != "1 → 5, 2 → 1" {
		t.Errorf("o que o Destruidor mudou saiu %q, want \"1 → 5, 2 → 1\"", after.DamageRerolled)
	}
	var row *duelRow
	for i, each := range damageRows(*used.PendingAttack) {
		if each.Label == "Destruidor" {
			row = &damageRows(*used.PendingAttack)[i]
		}
	}
	if row == nil || !strings.Contains(row.Sum, "1 → 5, 2 → 1") {
		t.Errorf("a carta do mestre não tem a linha do Destruidor com o que mudou: %+v", damageRows(*used.PendingAttack))
	}
}
