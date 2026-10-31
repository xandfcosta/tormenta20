package engine

import "t20engine/domain/ecs"

// O MESTRE QUE CONCORDA COM O LIVRO: o teste é julgado e, quando o livro diz
// que acertou, o dano é rolado com a decisão que ele sugere. É como os casos do
// livro se escrevem — o exemplo da p142 decide o crítico pelo d20 —, e é o que
// os testes de regra de dano precisam: a decisão contrária do mestre tem os
// casos dela no `attack_ruling_rules_test.go`.

func suggestedAttack(
	card WeaponCard, target AttackTarget, d20 int, rollDie func(int) (int, error),
) (AttackOutcome, error) {
	return suggestedAttackOf(card, target, nil, D20(d20), rollDie)
}

func suggestedAttackUnder(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	d20 int, rollDie func(int) (int, error),
) (AttackOutcome, error) {
	return suggestedAttackOf(card, target, situations, D20(d20), rollDie)
}

func suggestedAttackOf(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	die AttackDie, rollDie func(int) (int, error),
) (AttackOutcome, error) {
	judged, err := JudgeAttack(card, target, situations, die, rollDie)
	if err != nil || !judged.Hit {
		return judged, err
	}
	ruling := RulingHit
	if judged.Critical {
		ruling = RulingCritical
	}
	return RollAttackDamage(card, target, situations, die, ruling, rollDie)
}

// judgeWorld e damageWorld são os MUNDOS dos dois tempos, para os guardas que
// leem o que a saída não mostra.
func judgeWorld(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	d20 int, rollDie func(int) (int, error),
) *ecs.World {
	return runAttackWorld(append(judgeSystems(card, target, situations, D20(d20)),
		undoTheHitOnConcealment(rollDie)))
}

func damageWorld(
	card WeaponCard, target AttackTarget, d20 int, ruling Ruling, rollDie func(int) (int, error),
) *ecs.World {
	return runAttackWorld(append(judgeSystems(card, target, nil, D20(d20)),
		damageSystems(card, target, ruling, rollDie)...))
}
