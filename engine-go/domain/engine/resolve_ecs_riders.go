package engine

import "t20engine/domain/ecs"

// worldRiders são as parcelas que a CARTA DA ARMA aplica, e não a pilha: o dado
// extra de um encanto e o bônus que só existe no crítico. Moram num recurso
// pelo mesmo motivo dos fatores — são do mundo, não de um termo.
type worldRiders struct {
	extra     []ExtraDamage
	critical  map[string]int
	announced []AnnouncedCondition
}

// harvestWeaponRiders tira da pilha o que a CARTA DA ARMA aplica.
//
// São duas coisas que o `Amount` não sabe expressar: o dado extra de um encanto
// (`+1d6 de fogo`) e o bônus que só vale no acerto crítico (`+10` do
// Dilacerante). Nenhuma das duas é parcela somável — a primeira é um dado que
// ainda vai rolar, a segunda depende de um resultado que a ficha não conhece.
//
// O sistema roda ANTES do `deferConditionals` de propósito: o `onCritical` é
// uma condição que NÃO se liga, e deixá-lo chegar lá o ofereceria como
// interruptor na aba Efeitos.
func harvestWeaponRiders(w *ecs.World) {
	riders := ridersOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) {
			return
		}
		if t.Mod.Target.K == "unarmedDamage" {
			// O DADO DA BRIGA não é dano extra de ataque nenhum: ele É o dado
			// do ataque desarmado, que a carta lê dos poderes (`brawlDie`).
			// Colhido como extra, cada degrau somaria em toda arma.
			ecs.Set(w, e, Suppressed{Why: "é o dado do ataque desarmado, e quem o lê é a carta"})
			return
		}
		if t.Mod.Dice != "" {
			// SÓ O INCONDICIONAL SAI AQUI. O dado sob condição que a pessoa
			// LIGA — o "+1d8 em ermo ao ar livre" do Machado silvestre (p338) —
			// segue para o `deferConditionals` e vira interruptor, como
			// qualquer outro condicional. Colhê-lo aqui o fazia valer SEMPRE, e
			// a marca `Suppressed` ainda tirava dele a chance de ser oferecido:
			// o dado valia de graça e não havia como desligá-lo (ALE-457).
			if !isUnconditional(t.Mod) {
				if !switched(w, e) {
					return
				}
				// O DADO LIGADO é interruptor aceso, e sai da pilha aqui como o incondicional: a marca de
				// opção vem antes, porque o `deferConditionals` não o verá vivo.
				ecs.Set(w, e, Optional{})
			}
			riders.extra = append(riders.extra, ExtraDamage{
				Dice: t.Mod.Dice, Type: t.Mod.Target.DamageType, Scope: t.Mod.Target.Scope,
			})
			ecs.Set(w, e, Suppressed{Why: "é dado extra, e dado não soma na pilha"})
			return
		}
		if t.Mod.Target.K == "announcedEffect" {
			// O EFEITO ANUNCIADO segue o caminho da condição anunciada: sai da pilha e vai para a carta, com
			// o texto do livro no lugar do id da condição.
			riders.announced = append(riders.announced, AnnouncedCondition{Effect: t.Mod.Note, Scope: t.Mod.Target.Scope})
			ecs.Set(w, e, Suppressed{Why: "é anúncio de efeito, e anúncio não soma"})
			return
		}
		if t.Mod.Target.K == "announcedCondition" {
			// O ANÚNCIO não é número: ele sai da pilha inteiro e vai para a
			// carta da arma, como o dado extra. Somá-lo daria um modificador
			// que existe e vale zero, que é a forma mais silenciosa de defeito
			// deste motor.
			riders.announced = append(riders.announced, AnnouncedCondition{
				Condition:  t.Mod.Target.Name,
				OnCritical: t.Mod.Condition != nil && t.Mod.Condition.C == "onCritical",
				Note:       t.Mod.Note,
				Scope:      t.Mod.Target.Scope,
			})
			ecs.Set(w, e, Suppressed{Why: "é anúncio de condição, e anúncio não soma"})
			return
		}
		if c := t.Mod.Condition; c != nil && c.C == "onCritical" {
			riders.critical[targetKey(t.Mod.Target)] += t.Mod.Amount
			ecs.Set(w, e, Suppressed{Why: "só vale no crítico, e a pilha não sabe disso"})
		}
	})
	setRiders(w, riders)
}

// ridersOf e setRiders: ao contrário do `worldFactors`, cujo mapa se muta no
// lugar, a lista de parcelas CRESCE — e uma fatia acrescida dentro do `Each`
// não volta para o componente sozinha. Por isso a leitura e a escrita são duas.
func ridersOf(w *ecs.World) worldRiders {
	var found worldRiders
	ecs.Each(w, func(_ ecs.Entity, r worldRiders) { found = r })
	return found
}

func setRiders(w *ecs.World, riders worldRiders) {
	ecs.Each(w, func(e ecs.Entity, _ worldRiders) { ecs.Set(w, e, riders) })
}
