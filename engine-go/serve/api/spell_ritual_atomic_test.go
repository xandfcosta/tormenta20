package api

import "testing"

// A magia e o desligar do interruptor saem juntos: se o segundo falha, o primeiro não cobra, e tentar de
// novo não cobra duas vezes.
func TestARitualThatFailsToEndChargesNothing(t *testing.T) {
	f := ritualCaster(t, `["celebrar-ritual"]`)
	learnSpell(t, f, "luz")
	if _, err := f.s.db.ExecContext(t.Context(),
		`CREATE TRIGGER falha BEFORE DELETE ON character_conditionals BEGIN SELECT RAISE(ABORT, 'o disco recusou'); END`); err != nil {
		t.Fatal(err)
	}
	refusal, spent := castWith(t, f, "luz", `{}`)
	if refusal == "" {
		t.Fatal("o controle falhou: a gravação devia ter falhado")
	}
	if spent != 0 {
		t.Errorf("a magia falhou e cobrou %d PM", spent)
	}
	if !ritualIsOn(t, f) {
		t.Error("o interruptor desligou numa conjuração que não saiu")
	}
}
