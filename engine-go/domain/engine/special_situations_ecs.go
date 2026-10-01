package engine

import "t20engine/domain/ecs"

// A SITUAÇÃO ESPECIAL É UMA ENTIDADE, e cada mecanismo da Tabela 5-3 é um
// sistema que só olha os componentes que lhe interessam.
//
// A alternativa era um `switch` sobre a situação dentro do `judgeTheHit`, com
// um ramo por linha e o "apenas para corpo a corpo" repetido em dois deles.
// Aqui o qualificador é um COMPONENTE, e quem o cobra é um sistema só — que
// vale para a linha que existir amanhã sem ser editado.
//
// É também o que deixa a PROCEDÊNCIA de graça: a entidade carrega o rótulo, e a
// mesa pode perguntar "de onde veio esse +2?" sem o motor guardar uma segunda
// lista para responder.

// shiftsTheAttack e shiftsTheDefense são o mecanismo que o motor já tinha: um
// número que entra antes de comparar.
type shiftsTheAttack struct{ Amount int }
type shiftsTheDefense struct{ Amount int }

// missChance é a faixa de d% que DESFAZ um acerto que já aconteceu. Ela é o
// mecanismo novo, e a única regra do motor que age depois do veredito.
type missChance struct{ UpTo int }

// forbidsTheAttack é a cobertura total. Separada do `shiftsTheDefense` porque
// não é Defesa nenhuma — ver o cabeçalho do `special_situations.go`.
type forbidsTheAttack struct{}

// meleeOnly é o "(apenas para corpo a corpo)" de duas linhas da tabela.
type meleeOnly struct{}

// situationLabel é a procedência que a mesa lê.
type situationLabel struct{ Text string }

// situationTally é o que a tabela inteira somou, e é o que o `judgeTheHit` lê.
type situationTally struct {
	Attack  int
	Defense int
}

// spawnTheSituations traz cada linha que vale AGORA para o mundo.
//
// Linha desconhecida é PULADA em silêncio de propósito: a lista vem da borda
// (o tabuleiro, e um dia a tela), e uma situação que não existe não pode
// derrubar o ataque. Quem garante que a lista é a da tabela é o guarda de
// varredura, no teste.
func spawnTheSituations(situations []SpecialSituation) ecs.System {
	return func(w *ecs.World) {
		for _, s := range situations {
			rule, known := specialSituationTable[s]
			if !known {
				continue
			}
			e := w.Spawn()
			ecs.Set(w, e, situationLabel{Text: rule.Label})
			if rule.Attack != 0 {
				ecs.Set(w, e, shiftsTheAttack{Amount: rule.Attack})
			}
			if rule.Defense != 0 {
				ecs.Set(w, e, shiftsTheDefense{Amount: rule.Defense})
			}
			if rule.MissUpTo != 0 {
				ecs.Set(w, e, missChance{UpTo: rule.MissUpTo})
			}
			if rule.Forbids {
				ecs.Set(w, e, forbidsTheAttack{})
			}
			if rule.MeleeOnly {
				ecs.Set(w, e, meleeOnly{})
			}
		}
	}
}

// dropWhatDoesNotReachThisWeapon mata a situação que só vale corpo a corpo
// quando a arma é de disparo.
//
// DESPAWN e não um `if` em cada sistema que lê: a entidade deixa de existir, e
// os três sistemas seguintes não precisam saber que ela existiu. É o mesmo
// movimento da redação do tabuleiro — o que não vale SOME INTEIRO.
func dropWhatDoesNotReachThisWeapon(card WeaponCard) ecs.System {
	return func(w *ecs.World) {
		if card.Skill != "Pontaria" {
			return
		}
		ecs.Each(w, func(e ecs.Entity, _ meleeOnly) { w.Despawn(e) })
	}
}

// forbidTheAttackIfAnyoneSaysSo para antes de qualquer rolagem.
func forbidTheAttackIfAnyoneSaysSo(w *ecs.World) {
	proibido := false
	ecs.Each(w, func(e ecs.Entity, _ forbidsTheAttack) {
		if w.Alive(e) {
			proibido = true
		}
	})
	if proibido {
		setResource(w, attackFault{Err: errTotalCover})
	}
}

// tallyTheSituations soma o que sobrou, e é o único lugar em que a soma
// acontece: as situações da p239 ACUMULAM entre si, ao contrário dos bônus de
// item da p226 — a tabela diz "cumulativos com outras condições" na linha do
// Caído, e nenhuma outra linha dá tipo a que competir.
func tallyTheSituations(w *ecs.World) {
	tally := situationTally{}
	ecs.Each(w, func(e ecs.Entity, shift shiftsTheAttack) {
		if w.Alive(e) {
			tally.Attack += shift.Amount
		}
	})
	ecs.Each(w, func(e ecs.Entity, shift shiftsTheDefense) {
		if w.Alive(e) {
			tally.Defense += shift.Amount
		}
	})
	setResource(w, tally)
}

// undoTheHitOnConcealment é a chance de falha, e ela roda DEPOIS do veredito.
//
// A ordem é a do livro: a camuflagem não deixa o ataque mais difícil, ela deixa
// o acerto incerto. Modelá-la como Defesa faria um 20 natural atravessar a
// escuridão, e faria a margem de ameaça mentir.
//
// Só a MAIOR faixa vale: "aplique apenas o mais severo" é a regra das condições
// (p394) e aqui ela cai bem — duas camuflagens não dão 70%.
func undoTheHitOnConcealment(rollDie func(faces int) (int, error)) ecs.System {
	return func(w *ecs.World) {
		if !resourceOf[hitVerdict](w).Hit {
			return
		}
		pior := 0
		ecs.Each(w, func(e ecs.Entity, chance missChance) {
			if w.Alive(e) && chance.UpTo > pior {
				pior = chance.UpTo
			}
		})
		if pior == 0 {
			return
		}
		rolled, err := rollDie(100)
		if err != nil {
			setResource(w, attackFault{Err: err})
			return
		}
		if rolled > pior {
			return
		}
		verdict := resourceOf[hitVerdict](w)
		verdict.Hit = false
		setResource(w, verdict)
		setResource(w, criticalVerdict{})
	}
}
