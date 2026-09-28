package engine

import (
	"strings"
	"testing"

	"t20engine/domain/ecs"
)

// noDice é o sorteio de quem não vai sortear: o mundo do erro não rola nada, e
// um dublê que devolvesse zero calado esconderia uma rolagem indevida.
func noDice(t *testing.T) func(int) (int, error) {
	t.Helper()
	return func(faces int) (int, error) {
		t.Fatalf("o ataque errou e alguém pediu um d%d mesmo assim", faces)
		return 0, nil
	}
}

// TODO SISTEMA DO ATAQUE DEIXA A MARCA DELE NO MUNDO (ALE-414).
//
// # Ele repõe o que a função sequencial dava
//
// Antes, tirar um passo do `ResolveAttack` apagava uma linha que o compilador
// via: as variáveis seguintes deixavam de existir. Com sistemas, um que não
// rode deixa o componente ausente e quem o lê recebe o valor ZERO — um ataque
// que causa 0 de dano, sem erro e sem log.
//
// # Por que cada sistema traz o CENÁRIO dele
//
// Três dos nove só agem numa metade do dado: o que cala tudo no erro não roda
// no acerto, e o que multiplica e o que descarta o só-em-crítico são
// mutuamente exclusivos por construção. Um mundo só mediria os seis fáceis e
// daria verde sobre os três que importam.
//
// # O DENOMINADOR é o que impede a lista de apodrecer
//
// Não há como perguntar ao `ecs` quantos componentes uma entidade tem sem
// inventar API para isto, então a lista é escrita à mão. O que a mantém honesta
// é a contagem bater com o número de sistemas: acrescentar um sistema sem
// acrescentar a conferência dele reprova aqui, com o número.
func TestEveryAttackSystemWritesItsComponent(t *testing.T) {
	// A espada do `aWeapon` mais um bônus que só existe no crítico: é ele que
	// dá ao `dropWhatIsNotCritical` o que descartar.
	card := aWeapon("1d8", 3, 19, 2, 5)
	card.CriticalBonus = 10
	target := AttackTarget{Defense: 15, DamageReduction: 2}

	acerto := attackWorld(card, target, 12, fixedDice(t, 8))
	erro := attackWorld(card, target, 2, noDice(t))
	critico := attackWorld(card, target, 19, fixedDice(t, 8, 5))

	conferencias := []struct {
		sistema string
		mundo   *ecs.World
		deixou  func(*ecs.World) bool
	}{
		{"judgeTheHit", acerto, func(w *ecs.World) bool {
			_, has := ecs.Get[hitVerdict](w, theResource(w))
			return has
		}},
		{"judgeTheCritical", acerto, func(w *ecs.World) bool {
			_, has := ecs.Get[criticalVerdict](w, theResource(w))
			return has
		}},
		{"spawnTheDamageParcels", acerto, func(w *ecs.World) bool {
			return countOf[damageDice](w) == 1 && countOf[damageFlat](w) == 2
		}},
		{"dropEverythingIfItMissed", erro, func(w *ecs.World) bool {
			return len(suppressionReasons(w)) == 3
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

	sistemas := attackSystems(card, target, 12, fixedDice(t, 8))
	if len(conferencias) != len(sistemas) {
		t.Fatalf("são %d sistemas de ataque e %d conferências.\n"+
			"Sistema novo entra nesta lista JUNTO, com o CENÁRIO em que ele age — "+
			"senão ele pode parar de rodar e ninguém fica sabendo: o componente "+
			"ausente vira valor zero, e num ataque isso é dano que não aplica.",
			len(sistemas), len(conferencias))
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
// É o que a fatia comprou, e ela não aparece na saída: o `AttackOutcome` de um
// acerto normal é byte a byte o mesmo de antes, com ou sem a tag. Medir a conta
// não diria nada sobre isto.
//
// Sem a razão escrita, a tela que um dia mostrar "+10 do Dilacerante, não
// aplicado" teria de redescobrir o motivo — e redescobrir é reimplementar a
// regra numa segunda camada.
func TestTheParcelThatDidNotCountSaysWhy(t *testing.T) {
	card := aWeapon("1d8", 3, 19, 2, 5)
	card.CriticalBonus = 10

	// 12+5 = 17 contra Defesa 15: acerta, e não critica.
	acerto := attackWorld(card, AttackTarget{Defense: 15}, 12, fixedDice(t, 8))
	razoes := suppressionReasons(acerto)
	if len(razoes) != 1 || !strings.Contains(razoes[0], "crítico") {
		t.Fatalf("as parcelas caladas do acerto normal disseram %q, e o esperado é uma só, "+
			"dizendo que o ataque não foi crítico", razoes)
	}

	erro := attackWorld(card, AttackTarget{Defense: 30}, 12, noDice(t))
	for _, razao := range suppressionReasons(erro) {
		if !strings.Contains(razao, "errou") {
			t.Errorf("uma parcela do ataque que ERROU foi calada dizendo %q, e a razão "+
				"do erro é a que a mesa vai ler", razao)
		}
	}
	if len(suppressionReasons(erro)) != 3 {
		t.Errorf("o ataque que errou calou %d parcelas, e a arma tem 3 (1d8, o +3 e o +10 "+
			"do crítico). Parcela que some da lista some da tela", len(suppressionReasons(erro)))
	}
}

// NOTAÇÃO QUEBRADA NO CATÁLOGO FALHA MESMO QUANDO O ATAQUE ERRA.
//
// É uma mudança de comportamento da ALE-414, e ela é deliberada. Antes a
// notação só era lida depois do acerto, então um `"1d"` no catálogo ficava
// calado em todo ataque que errava — o defeito aparecia numa fração das
// rolagens, que é a pior maneira de um defeito aparecer.
func TestABrokenDamageNotationFailsEvenOnAMiss(t *testing.T) {
	quebrada := aWeapon("1d", 0, 20, 2, 0)

	if _, err := ResolveAttack(quebrada, AttackTarget{Defense: 30}, 2, noDice(t)); err == nil {
		t.Error("a arma com notação de dano inválida errou o ataque e não reclamou — " +
			"o defeito do catálogo só apareceria quando alguém acertasse")
	}
	if _, err := ResolveAttack(quebrada, AttackTarget{Defense: 1}, 20, noDice(t)); err == nil {
		t.Error("a arma com notação de dano inválida acertou e não reclamou")
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
