package book

import "testing"

// O DEGRAU DA INSPIRAÇÃO vale também no dano de quem tem a Inspiração Marcial (p45), e só de quem a tem: o degrau
// é gravado SEM a condição da flag, então quem decide a quem ele chega é a posse do poder.
func TestTheStanceStepsReachOnlyTheModifiersOfOwnedPowers(t *testing.T) {
	owns := func(ids ...string) func(string) bool {
		return func(id string) bool {
			for _, each := range ids {
				if each == id {
					return true
				}
			}
			return false
		}
	}
	plain := StanceDegreeModifiers("inspiracao", 2, owns("class.bardo.inspiracao"))
	if len(plain) != 1 || plain[0].Target.K != "expertiseAll" || plain[0].Amount != 3 {
		t.Errorf("o degrau da Inspiração sem a Marcial é +3 nas perícias, e veio %+v", plain)
	}
	martial := StanceDegreeModifiers("inspiracao", 2, owns("class.bardo.inspiracao", "class.bardo.inspiracao-marcial"))
	damage := false
	for _, m := range martial {
		damage = damage || (m.Target.K == "damage" && m.Amount == 3)
	}
	if len(martial) != 2 || !damage {
		t.Errorf("o degrau da Inspiração com a Marcial é +3 nas perícias e no dano, e veio %+v", martial)
	}
	if targets := StanceTargets("inspiracao", owns("class.bardo.inspiracao")); len(targets) != 1 {
		t.Errorf("o chip da Inspiração sem a Marcial move só as perícias, e diz %+v", targets)
	}
}
