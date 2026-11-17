package engine

import (
	"slices"
	"testing"
)

// O JOGADOR VÊ AS SITUAÇÕES DO LADO DELE (decisão do dono). A Tabela 5-3 mistura as do
// atacante — as que mexem no ataque dele, ou que são o estado dele — com as do ALVO, que
// mexem na Defesa ou na chance de errar dele. A carta do jogador explica a conta dele com as
// primeiras; as do alvo ficam com o mestre, como a Defesa (p230: o jogador não conhece a
// ficha de outra entidade).
func TestTheAttackerSeesOnlyTheSituationsOfHisSide(t *testing.T) {
	situations := []SpecialSituation{AttackerFlanking, TargetUnderLightCover, TargetInMeleeCombat, TargetUnderLightConcealment}
	out, err := JudgeAttack(aWeapon("1d8", 3, 19, 2, 8), AttackTarget{Defense: 13}, situations, D20(12), func(int) (int, error) { return 3, nil })
	if err != nil {
		t.Fatal(err)
	}
	// O CONTROLE: o mestre lê as quatro.
	for _, want := range []string{"flanqueando", "cobertura leve", "camuflagem leve"} {
		if !slices.ContainsFunc(out.Situations, func(s string) bool { return s == want }) {
			t.Fatalf("o mestre não lê %q entre %v — o caso mediria outra coisa", want, out.Situations)
		}
	}
	for _, want := range []string{"flanqueando"} {
		if !slices.Contains(out.AttackerSituations, want) {
			t.Errorf("o atacante não vê %q, que mexe no ataque dele: %v", want, out.AttackerSituations)
		}
	}
	for _, hidden := range []string{"cobertura leve", "camuflagem leve"} {
		if slices.Contains(out.AttackerSituations, hidden) {
			t.Errorf("o atacante vê %q, que é do alvo: %v", hidden, out.AttackerSituations)
		}
	}
}
