package engine

import (
	"strings"
	"testing"

	"t20engine/domain/ecs"
)

// noDice é o sorteio de quem não vai sortear: julgar o teste não rola dano, e
// um dublê que devolvesse zero calado esconderia uma rolagem indevida.
func noDice(t *testing.T) func(int) (int, error) {
	t.Helper()
	return func(faces int) (int, error) {
		t.Fatalf("ninguém devia rolar dado aqui, e alguém pediu um d%d", faces)
		return 0, nil
	}
}

// TODO SISTEMA DO ATAQUE DEIXA A MARCA DELE NO MUNDO.
//
// # Ele repõe o que a função sequencial dava
//
// Tirar um passo de uma função sequencial apagava uma linha que o compilador
// via: as variáveis seguintes deixavam de existir. Com sistemas, um que não
// rode deixa o componente ausente e quem o lê recebe o valor ZERO — um ataque
// que causa 0 de dano, sem erro e sem log.
//
// # Por que cada sistema traz o CENÁRIO dele
//
// Vários só agem numa metade do dado ou da decisão: o que multiplica e o que
// descarta o só-em-crítico são mutuamente exclusivos por construção, e os da
// Tabela 5-3 dormem num ataque sem situação. Um mundo só mediria os fáceis.
//
// # O DENOMINADOR é o que impede a lista de apodrecer
//
// A lista é escrita à mão, e o que a mantém honesta é a contagem bater com o
// número de sistemas dos DOIS tempos: acrescentar um sistema sem a conferência
// dele reprova aqui, com o número.
func TestEveryAttackSystemWritesItsComponent(t *testing.T) {
	// A espada do `aWeapon` mais um bônus que só existe no crítico: é ele que
	// dá ao `dropWhatIsNotCritical` o que descartar.
	card := aWeapon("1d8", 3, 19, 2, 5)
	card.CriticalBonus = 10
	target := AttackTarget{Defense: 15, DamageReduction: 2}

	julgado := judgeWorld(card, target, nil, 12, noDice(t))
	acerto := damageWorld(card, target, 12, RulingHit, fixedDice(t, 8))
	critico := damageWorld(card, target, 12, RulingCritical, fixedDice(t, 8, 5))

	// A arma de PONTARIA é o que dá ao `dropWhatDoesNotReachThisWeapon` o que
	// matar — com a espada do caso, o flanqueio vale e nada é despachado.
	arco := aWeapon("1d6", 2, 20, 3, 5)
	arco.Skill = "Pontaria"
	elevado := judgeWorld(card, target, []SpecialSituation{AttackerOnHigherGround}, 12, noDice(t))
	disparo := judgeWorld(arco, target, []SpecialSituation{AttackerFlanking}, 12, noDice(t))
	proibido := judgeWorld(card, target, []SpecialSituation{TargetUnderTotalCover}, 20, noDice(t))
	camuflado := judgeWorld(card, target, []SpecialSituation{TargetUnderTotalConcealment}, 12, fixedDice(t, 3))

	conferencias := []struct {
		sistema string
		mundo   *ecs.World
		deixou  func(*ecs.World) bool
	}{
		{"spawnTheSituations", elevado, func(w *ecs.World) bool {
			return countOf[situationLabel](w) == 1 && countOf[shiftsTheAttack](w) == 1
		}},
		{"dropWhatDoesNotReachThisWeapon", disparo, func(w *ecs.World) bool {
			// A entidade do flanqueio nasceu e MORREU: a arma é de disparo, e a
			// p239 diz "apenas para corpo a corpo".
			return countOf[meleeOnly](w) == 0
		}},
		{"forbidTheAttackIfAnyoneSaysSo", proibido, func(w *ecs.World) bool {
			_, has := ecs.Get[attackFault](w, theResource(w))
			return has
		}},
		{"tallyTheSituations", elevado, func(w *ecs.World) bool {
			return resourceOf[situationTally](w).Attack == 2
		}},
		{"judgeTheHit", julgado, func(w *ecs.World) bool {
			_, has := ecs.Get[hitVerdict](w, theResource(w))
			return has
		}},
		{"judgeTheCritical", julgado, func(w *ecs.World) bool {
			_, has := ecs.Get[criticalVerdict](w, theResource(w))
			return has
		}},
		{"undoTheHitOnConcealment", camuflado, func(w *ecs.World) bool {
			// d10 = 3 cai na faixa "1 a 5" da camuflagem total (p239): o acerto que
			// havia é DESFEITO, e é o único sistema do ataque que faz isso.
			return !resourceOf[hitVerdict](w).Hit
		}},
		{"applyTheRuling", critico, func(w *ecs.World) bool {
			// O 12 está fora da margem 19, e o crítico do mestre vale assim mesmo.
			return resourceOf[criticalVerdict](w).Critical
		}},
		{"spawnTheDamageParcels", acerto, func(w *ecs.World) bool {
			return countOf[damageDice](w) == 1 && countOf[damageFlat](w) == 2
		}},
		{"multiplyTheWeaponDice", critico, func(w *ecs.World) bool {
			return weaponDiceOf(w).Count == 2
		}},
		{"dropWhatIsNotCritical", acerto, func(w *ecs.World) bool {
			return len(suppressionReasons(w)) == 1
		}},
		{"rollTheDice", acerto, func(w *ecs.World) bool {
			return countOf[diceRolled](w) == 1
		}},
		{"sumTheParcels", acerto, func(w *ecs.World) bool {
			_, has := ecs.Get[rawTally](w, theResource(w))
			return has
		}},
		{"absorbWithDamageReduction", acerto, func(w *ecs.World) bool {
			_, has := ecs.Get[absorbedTally](w, theResource(w))
			return has
		}},
	}

	sistemas := len(judgeSystems(card, target, nil, D20(12))) + 1 + // a camuflagem
		len(damageSystems(card, target, RulingHit, fixedDice(t)))
	if len(conferencias) != sistemas {
		t.Fatalf("são %d sistemas de ataque e %d conferências.\n"+
			"Sistema novo entra nesta lista JUNTO, com o CENÁRIO em que ele age — "+
			"senão ele pode parar de rodar e ninguém fica sabendo: o componente "+
			"ausente vira valor zero, e num ataque isso é dano que não aplica.",
			sistemas, len(conferencias))
	}

	for _, c := range conferencias {
		if !c.deixou(c.mundo) {
			t.Errorf("o %s não deixou marca no mundo — ele não rodou, e quem lê o "+
				"que ele devia ter escrito vai receber o valor ZERO como se fosse a conta",
				c.sistema)
		}
	}
}

// A PARCELA QUE NÃO CONTOU CONTINUA NO MUNDO, E DIZ POR QUÊ.
//
// Ela não aparece na saída: o `AttackOutcome` de um acerto normal é o mesmo com
// ou sem a marca. Sem a razão escrita, a tela que um dia mostrar "+10 do
// Dilacerante, não aplicado" teria de redescobrir o motivo — e redescobrir é
// reimplementar a regra numa segunda camada.
func TestTheParcelThatDidNotCountSaysWhy(t *testing.T) {
	card := aWeapon("1d8", 3, 19, 2, 5)
	card.CriticalBonus = 10

	acerto := damageWorld(card, AttackTarget{Defense: 15}, 12, RulingHit, fixedDice(t, 8))
	razoes := suppressionReasons(acerto)
	if len(razoes) != 1 || !strings.Contains(razoes[0], "crítico") {
		t.Fatalf("as parcelas caladas do acerto normal disseram %q, e o esperado é uma só, "+
			"dizendo que o ataque não foi crítico", razoes)
	}
}

// JULGAR NÃO LÊ A NOTAÇÃO DE DANO, e o dano falha nela.
//
// O teste do ataque não sabe de dados de dano: um `"1d"` no catálogo não
// impede o mestre de ler o d20 contra a Defesa. Quem tropeça nele é o tempo do
// dano — e tropeça sempre, porque um golpe sem dano não chega até lá.
func TestABrokenDamageNotationFailsWhenTheDamageIsRolled(t *testing.T) {
	quebrada := aWeapon("1d", 0, 20, 2, 0)

	if _, err := JudgeAttack(quebrada, AttackTarget{Defense: 1}, nil, D20(15), noDice(t)); err != nil {
		t.Errorf("julgar o teste reclamou da notação de dano, que ele não lê: %v", err)
	}
	if _, err := RollAttackDamage(quebrada, AttackTarget{Defense: 1}, nil, D20(15), RulingHit, noDice(t)); err == nil {
		t.Error("a arma com notação de dano inválida rolou o dano e não reclamou")
	}
	if _, err := DamageToRoll(quebrada, RulingHit); err == nil {
		t.Error("a arma com notação de dano inválida disse o que rolar e não reclamou")
	}
}

func countOf[C any](w *ecs.World) int {
	found := 0
	ecs.Each(w, func(_ ecs.Entity, _ C) { found++ })
	return found
}

func suppressionReasons(w *ecs.World) []string {
	reasons := []string{}
	ecs.Each(w, func(_ ecs.Entity, s Suppressed) { reasons = append(reasons, s.Why) })
	return reasons
}

func weaponDiceOf(w *ecs.World) damageDice {
	var found damageDice
	ecs.Each2(w, func(e ecs.Entity, _ weaponsOwnDice, parcel damageDice) { found = parcel })
	return found
}
