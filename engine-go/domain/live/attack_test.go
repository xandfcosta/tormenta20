package live

import "testing"

// O PROVISÓRIO DE UM ATAQUE, com a mesma divisa do movimento no tabuleiro: o
// jogador PROPÕE e o mestre CONFIRMA. A regra de quem pode está aqui; a conta
// do ataque é do `engine` e chega pronta.

func aQueue(t *testing.T) *SessionRuntimeState {
	t.Helper()
	st := EmptyRuntimeState()
	st.Initiative = []InitiativeEntry{
		{ID: "heroi", Label: "Arwen", Type: "character", Initiative: 18,
			HpCurrent: PtrInt64(40), HpMax: PtrInt64(40)},
		{ID: "ogro", Label: "Ogro", Type: "npc", Initiative: 9,
			HpCurrent: PtrInt64(30), HpMax: PtrInt64(30)},
	}
	StartScene(st, SceneAction)
	st.TurnIndex = 0
	return st
}

func umGolpe() PendingAttack {
	return PendingAttack{
		AttackerEntryID: "heroi", TargetEntryID: "ogro", Weapon: "Espada longa",
		Roll: 14, Total: 19, Hit: true, Dice: []int{8}, RawDamage: 11, Damage: 11,
		Ruling: RulingHit, DamageRolled: true, ByUserID: 7,
	}
}

// PROPOR NÃO MOVE PV NENHUM. O provisório é o que a mesa VÊ antes de alguém
// decidir — se ele já tivesse aplicado, confirmar não significaria nada.
func TestProposingAnAttackDoesNotTouchTheVitals(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, umGolpe()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	if st.PendingAttack == nil {
		t.Fatal("o provisório tem de ficar guardado no estado, senão ele não sobrevive ao F5")
	}
	if got := DerefOr(st.Initiative[1].HpCurrent, 0); got != 30 {
		t.Errorf("o PV do ogro = %d, e propor não tira nada: ele tinha 30", got)
	}
}

// SÓ O MESTRE PÕE O DANO NA FICHA, a mesma divisa do `CommitMove`.
func TestOnlyTheGameMasterConfirmsAnAttack(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, umGolpe()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	if _, err := AttackToCommit(st, Attacker{UserID: 7, Role: "player"}); err == nil {
		t.Error("o jogador que propôs NÃO confirma o próprio ataque")
	}
	if st.PendingAttack == nil {
		t.Error("uma confirmação recusada deixa o provisório de pé")
	}
}

// O MESTRE CONFIRMA E RECEBE A CONTA, e nada de PV se move aqui: quem aplica é
// o store, que sabe distinguir a linha com ficha da linha sem.
func TestTheGameMasterGetsTheAttackToApply(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, umGolpe()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	attack, err := AttackToCommit(st, Attacker{UserID: 1, Role: "gm"})
	if err != nil {
		t.Fatalf("confirmar: %v", err)
	}
	if attack.Damage != 11 || attack.TargetEntryID != "ogro" {
		t.Errorf("veio %d de dano no alvo %q, e o golpe era 11 no ogro", attack.Damage, attack.TargetEntryID)
	}
	if got := DerefOr(st.Initiative[1].HpCurrent, 0); got != 30 {
		t.Errorf("o PV do ogro = %d: o regime NÃO aplica dano, quem aplica é o store", got)
	}
	ClearPendingAttack(st)
	if st.PendingAttack != nil {
		t.Error("limpo, o provisório some — senão ele seria confirmado duas vezes")
	}
}

// CANCELAR desfaz sem mexer em ninguém.
func TestCancelingAnAttackLeavesTheVitalsAlone(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, umGolpe()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	if err := CancelAttack(st, Attacker{UserID: 1, Role: "gm"}); err != nil {
		t.Fatalf("cancelar: %v", err)
	}
	if st.PendingAttack != nil {
		t.Error("cancelado, o provisório some")
	}
	if got := DerefOr(st.Initiative[1].HpCurrent, 0); got != 30 {
		t.Errorf("o PV do ogro = %d: cancelar não é dano", got)
	}
}

// UM ALVO QUE SAIU DA FILA entre propor e confirmar não pode derrubar a mesa.
func TestConfirmingAgainstAGoneTargetRefuses(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, umGolpe()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	st.Initiative = st.Initiative[:1]
	if _, err := AttackToCommit(st, Attacker{UserID: 1, Role: "gm"}); err == nil {
		t.Error("confirmar contra um alvo que não está mais na fila tem de RECUSAR")
	}
}

// PROPOR CONTRA QUEM NÃO ESTÁ NA FILA é recusado na porta.
func TestProposingAgainstSomeoneOutsideTheQueueRefuses(t *testing.T) {
	st := aQueue(t)
	out := umGolpe()
	out.TargetEntryID = "ninguem"
	if err := ProposeAttack(st, out); err == nil {
		t.Error("propor contra quem não está na fila tem de recusar")
	}
}

// aStrikeWithoutRuling é o golpe como ele nasce, antes de o mestre falar.
func aStrikeWithoutRuling() PendingAttack {
	attack := umGolpe()
	attack.Ruling, attack.DamageToRoll, attack.DamageRolled = "", nil, false
	return attack
}

// A conferência sob a trava: o "acertou" do mestre pode chegar entre o jogador pedir o
// rolar de novo e o dado novo ser gravado (p226).
func TestAReplacedRollWaitsForNoRuling(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, aStrikeWithoutRuling()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	second := aStrikeWithoutRuling()
	second.Roll = 9
	second.Rerolls = []DieReroll{{Name: "Sortudo", PM: 3}}
	if err := ReplaceAttackRoll(st, second); err != nil {
		t.Fatalf("rolar de novo antes da decisão: %v", err)
	}
	if st.PendingAttack.Roll != 9 {
		t.Errorf("o dado novo não ficou: %d", st.PendingAttack.Roll)
	}
	third := second
	third.Rerolls = []DieReroll{{Name: "Sortudo", PM: 3}, {Name: "Sortudo", PM: 3}}
	if err := RuleAttack(st, Attacker{UserID: 1, Role: "gm"}, RulingMiss, nil); err != nil {
		t.Fatalf("decidir: %v", err)
	}
	if err := ReplaceAttackRoll(st, third); err == nil {
		t.Error("rolou de novo depois da decisão do mestre")
	}
}

// Propor sobre o erro vale, porque ele já encerrou o golpe.
func TestANewProposalReplacesTheMiss(t *testing.T) {
	st := aQueue(t)
	if err := ProposeAttack(st, aStrikeWithoutRuling()); err != nil {
		t.Fatalf("propor: %v", err)
	}
	if err := RuleAttack(st, Attacker{UserID: 1, Role: "gm"}, RulingMiss, nil); err != nil {
		t.Fatalf("decidir: %v", err)
	}
	if err := ProposeAttack(st, aStrikeWithoutRuling()); err != nil {
		t.Errorf("o erro encerrou o golpe, e a proposta nova foi recusada: %v", err)
	}
}
