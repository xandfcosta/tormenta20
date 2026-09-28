package engine

import "t20engine/domain/ecs"

// O ATAQUE EM SISTEMAS: aqui a ENTIDADE é a PARCELA DE DANO (ALE-414).
//
// A entidade muda por fase porque a pergunta muda — a FONTE na coleta, o TERMO
// na resolução da ficha. Aqui ela é outra de novo: "de que se soma este dano, e
// o que ficou de fora?". Quem responde isso é a parcela.
//
// # O que a troca compra
//
// O mesmo que a resolução da ficha comprou: o que NÃO entrou para de sumir. O
// dado extra que não multiplicou e o bônus que só existe no crítico saíam da
// conta sem deixar rastro — a mesa recebia o total e nada mais. Agora eles
// ficam no mundo com `Suppressed`, e a tela que quiser dizer "+10 do
// Dilacerante, não aplicado: o ataque não criticou" tem onde buscar.
//
// # A PUREZA fica, e era a condição da fatia
//
// O `d20` continua entrando por parâmetro e os dados por função: os sistemas se
// fecham sobre eles, como o `explodeIntoTerms(items)` já faz. A assinatura do
// `ResolveAttack` não mudou, e é por isso que os casos do livro que o cercam
// continuam valendo sem uma linha tocada — eles afirmam RESULTADO.
//
// # Uma diferença de comportamento, e ela é deliberada
//
// A notação de dano é lida ANTES de se saber se o ataque acertou, porque a
// parcela precisa existir para ser suprimida. Antes ela só era lida no acerto,
// então uma arma com `"1d"` no catálogo ficava calada em todo ataque que errava
// e estourava nos outros. Agora o defeito do catálogo aparece sempre.

// hitVerdict é o d20 julgado: a rolagem, a soma e o veredito.
//
// Ele e os outros acumuladores moram numa entidade SINGLETON — o idioma de
// "recurso" do ECS, para o dado que é do mundo e não de uma parcela. É o mesmo
// desenho do `worldFlags` da resolução da ficha.
type hitVerdict struct {
	Roll  int
	Total int
	Hit   bool
}

// criticalVerdict é a segunda decisão do d20, e ela é separada da primeira
// porque DEPENDE dela: bater a margem num ataque que erra não é crítico coisa
// nenhuma (p231).
type criticalVerdict struct {
	Critical   bool
	Multiplier int
}

// damageDice é a parcela que ROLA, na quantidade que o crítico já pediu.
type damageDice struct {
	Count int
	Faces int
}

// damageFlat é a parcela que já vem em número: o bônus de Força, o de um
// encanto, o de Dilacerante.
type damageFlat struct{ Amount int }

// weaponsOwnDice marca a parcela que É a arma. Só ela multiplica no crítico, e
// só ela vai para o `Dice`/`Faces` da conta — as outras viajam discriminadas.
type weaponsOwnDice struct{}

// damageTypeOf é o tipo da parcela extra ("fogo"). Ausente na arma: o tipo do
// dano dela é o da própria arma, e a carta não o separa.
type damageTypeOf struct{ Name string }

// onlyOnCritical marca a parcela que não existe fora do acerto crítico.
type onlyOnCritical struct{}

// diceRolled é o que o `rollTheDice` escreveu — a única escrita de aleatório
// do mundo inteiro.
type diceRolled struct {
	Values []int
	Total  int
}

// rawTally e absorbedTally são as duas últimas contas, e são duas porque são
// dois sistemas: quem soma as parcelas não sabe de RD, e quem aplica a RD não
// sabe de parcela.
type rawTally struct{ Raw int }
type absorbedTally struct{ Absorbed, Final int }

// attackFault guarda o PRIMEIRO erro e cala os sistemas seguintes.
//
// Um `ecs.System` é `func(*World)` e não devolve erro — a alternativa seria
// mudar a assinatura do núcleo por causa de dois pontos de falha (a notação do
// catálogo e o sorteio). O erro é um componente como outro qualquer, e quem
// chama o pergunta ao mundo no fim.
type attackFault struct{ Err error }

func attackSystems(
	card WeaponCard, target AttackTarget, d20 int, rollDie func(faces int) (int, error),
) []ecs.System {
	return []ecs.System{
		judgeTheHit(card, target, d20),
		judgeTheCritical(card, target, d20),
		spawnTheDamageParcels(card),
		dropEverythingIfItMissed,
		multiplyTheWeaponDice,
		dropWhatIsNotCritical,
		rollTheDice(rollDie),
		sumTheParcels,
		absorbWithDamageReduction(target),
	}
}

// judgeTheHit decide o acerto.
//
// "Se o resultado é igual ou maior que a Defesa do alvo, você acerta" (p230). O
// IGUAL decide todo ataque que empata, e é a metade que um `>` perderia em
// silêncio.
//
// AS DUAS PONTAS DO DADO MANDAM, e elas não estão na página do teste:
//
//	"Ao fazer um teste, um 20 natural sempre é um sucesso, e um 1 natural
//	sempre é uma falha, não importando o valor a ser alcançado." (p221)
//
// Aqui morava o contrário, escrito com citação: "não há 20 automático nem 1
// automático, e a ausência é deliberada (p220)". A p220 define o teste sem
// exceção nenhuma, e ler ali a AUSÊNCIA da regra é o erro — ela mora na página
// seguinte, em "Regras Adicionais de testes". Um 20 natural errava contra
// Defesa alta, e nada acusava.
func judgeTheHit(card WeaponCard, target AttackTarget, d20 int) ecs.System {
	return func(w *ecs.World) {
		total := d20 + card.Attack
		setResource(w, hitVerdict{
			Roll:  d20,
			Total: total,
			Hit:   d20 == 20 || (d20 != 1 && total >= target.Defense),
		})
	}
}

// judgeTheCritical decide o crítico, e só depois do acerto.
//
// "Você faz um acerto crítico quando ACERTA um ataque rolando um valor igual ou
// maior que a margem de ameaça" (p231): são as duas condições.
//
// "Quando nenhuma margem aparece, será 20. Quando nenhum multiplicador aparece,
// será x2" (p230) — por isso o zero do catálogo cai no padrão do livro em vez
// de virar uma arma que nunca critica.
func judgeTheCritical(card WeaponCard, target AttackTarget, d20 int) ecs.System {
	return func(w *ecs.World) {
		margin, multiplier := card.CritRange, card.CritMult
		if margin <= 0 {
			margin = 20
		}
		if multiplier <= 0 {
			multiplier = 2
		}
		// "Um alvo imune a acertos críticos ainda sofre o dano de um ataque
		// normal" (p231): a imunidade tira o crítico, nunca o ataque.
		hit := resourceOf[hitVerdict](w).Hit
		setResource(w, criticalVerdict{
			Critical:   hit && d20 >= margin && !target.CritImmune,
			Multiplier: multiplier,
		})
	}
}

// spawnTheDamageParcels põe no mundo uma entidade por parcela, NA ORDEM em que
// a conta as lê: a arma, os dados extras, o bônus numérico e o de crítico.
//
// A ordem é a da inserção porque é ela que decide a ordem das ROLAGENS, e o
// `ecs.World` varre por inserção justamente para isto.
func spawnTheDamageParcels(card WeaponCard) ecs.System {
	return func(w *ecs.World) {
		count, faces, err := parseDiceNotation(card.Damage)
		if err != nil {
			setResource(w, attackFault{Err: err})
			return
		}
		weapon := w.Spawn()
		ecs.Set(w, weapon, damageDice{Count: count, Faces: faces})
		ecs.Set(w, weapon, weaponsOwnDice{})

		for _, extra := range card.ExtraDamage {
			count, faces, err := parseDiceNotation(extra.Dice)
			if err != nil {
				setResource(w, attackFault{Err: err})
				return
			}
			parcel := w.Spawn()
			ecs.Set(w, parcel, damageDice{Count: count, Faces: faces})
			ecs.Set(w, parcel, damageTypeOf{Name: extra.Type})
		}

		if card.DamageBonus != 0 {
			ecs.Set(w, w.Spawn(), damageFlat{Amount: card.DamageBonus})
		}
		if card.CriticalBonus != 0 {
			parcel := w.Spawn()
			ecs.Set(w, parcel, damageFlat{Amount: card.CriticalBonus})
			ecs.Set(w, parcel, onlyOnCritical{})
		}
	}
}

// dropEverythingIfItMissed: quem erra não rola dano.
//
// SUPRIMIR e não apagar — a parcela fica, e é o que permite à mesa ler o que o
// ataque teria causado. Quem não rola é o `rollTheDice`, que pula o suprimido.
func dropEverythingIfItMissed(w *ecs.World) {
	if resourceOf[hitVerdict](w).Hit {
		return
	}
	suppressEvery[damageDice](w, "o ataque errou")
	suppressEvery[damageFlat](w, "o ataque errou")
}

// multiplyTheWeaponDice aplica o multiplicador da arma.
//
// "Multiplica os DADOS de dano do ataque (incluindo quaisquer aumentos por
// passos) pelo multiplicador da arma. Bônus numéricos de dano, assim como dados
// extras, não são multiplicados" (p231). O exemplo trabalhado é da p142: um
// dano de 1d8+3 torna-se 2d8+3 — mais DADOS, e o +3 uma vez só.
//
// É a marca `weaponsOwnDice` que carrega a exceção: o 1d6 de fogo de uma espada
// flamejante tem dados e não multiplica, e num crítico x2 ela causa 2d8 + 1d6.
//
// ELE VEM ANTES DO `rollTheDice`, e essa é a aresta do grafo: depois dele, a
// multiplicação encontraria um `diceRolled` já escrito e mudaria uma contagem
// que ninguém mais lê — o total sairia plausível e errado.
func multiplyTheWeaponDice(w *ecs.World) {
	verdict := resourceOf[criticalVerdict](w)
	if !verdict.Critical {
		return
	}
	ecs.Each2(w, func(e ecs.Entity, dice damageDice, _ weaponsOwnDice) {
		ecs.Set(w, e, damageDice{Count: dice.Count * verdict.Multiplier, Faces: dice.Faces})
	})
}

// dropWhatIsNotCritical tira as parcelas que só existem no crítico
// (Dilacerante, p336).
//
// ELE FICA ANTES DO `rollTheDice` e HOJE ISSO NÃO É LOAD-BEARING: toda parcela
// só-em-crítico que a carta produz é o `CriticalBonus`, que é um número e não
// rola. Trocar os dois de lugar não muda nada, e foi medido.
//
// A posição é a que sobrevive ao dia em que um encanto der 1d6 SÓ no crítico:
// lá, depois da rolagem, o sorteio já teria consumido dados que não entram na
// conta — e, com um sorteio de verdade, o ataque seguinte sairia com outros
// números sem nada acusar. Não há guarda porque não há como montar o caso a
// partir de uma `WeaponCard`, e teste sobre o que ninguém constrói é dívida.
func dropWhatIsNotCritical(w *ecs.World) {
	if resourceOf[criticalVerdict](w).Critical {
		return
	}
	ecs.Each(w, func(e ecs.Entity, _ onlyOnCritical) {
		suppress(w, e, "o ataque não foi um acerto crítico")
	})
}

// rollTheDice é o único sistema que sorteia, e o sorteio entra FECHADO nele.
func rollTheDice(rollDie func(faces int) (int, error)) ecs.System {
	return func(w *ecs.World) {
		if faulted(w) {
			return
		}
		ecs.Each(w, func(e ecs.Entity, parcel damageDice) {
			if dead(w, e) || faulted(w) {
				return
			}
			rolled := diceRolled{}
			for i := 0; i < parcel.Count; i++ {
				value, err := rollDie(parcel.Faces)
				if err != nil {
					setResource(w, attackFault{Err: err})
					return
				}
				rolled.Values = append(rolled.Values, value)
				rolled.Total += value
			}
			ecs.Set(w, e, rolled)
		})
	}
}

// sumTheParcels soma o que sobrou vivo.
func sumTheParcels(w *ecs.World) {
	raw := 0
	ecs.Each(w, func(e ecs.Entity, rolled diceRolled) {
		if !dead(w, e) {
			raw += rolled.Total
		}
	})
	ecs.Each(w, func(e ecs.Entity, parcel damageFlat) {
		if !dead(w, e) {
			raw += parcel.Amount
		}
	})
	setResource(w, rawTally{Raw: raw})
}

// absorbWithDamageReduction aplica a RD do alvo.
//
// "Se uma criatura com RD 5 sofre um ataque que causa 8 pontos de dano, perde
// apenas 3 PV" (p229).
//
// O PISO EM ZERO é decisão desta casa: o livro só dá o caso em que sobra dano, e
// "ignora parte do dano que sofre" não descreve um ataque que devolve PV. Sem o
// piso, uma RD alta viraria cura — e a cura tem regra própria, que não é esta.
func absorbWithDamageReduction(target AttackTarget) ecs.System {
	return func(w *ecs.World) {
		raw := resourceOf[rawTally](w).Raw
		absorbed := target.DamageReduction
		if absorbed > raw {
			absorbed = raw
		}
		setResource(w, absorbedTally{Absorbed: absorbed, Final: raw - absorbed})
	}
}

// outcomeFromWorld remonta a conta que a mesa lê.
func outcomeFromWorld(w *ecs.World) AttackOutcome {
	hit := resourceOf[hitVerdict](w)
	out := AttackOutcome{
		Roll:      hit.Roll,
		Total:     hit.Total,
		Hit:       hit.Hit,
		Critical:  resourceOf[criticalVerdict](w).Critical,
		RawDamage: resourceOf[rawTally](w).Raw,
		Absorbed:  resourceOf[absorbedTally](w).Absorbed,
		Damage:    resourceOf[absorbedTally](w).Final,
	}
	ecs.Each2(w, func(e ecs.Entity, _ weaponsOwnDice, parcel damageDice) {
		if dead(w, e) {
			return
		}
		// As faces viajam mesmo quando a arma não rolou nada, porque a mesa lê a
		// NOTAÇÃO: "2d8+3" diz de onde os números vieram, e "8+5+3" faz quem
		// olha reconstruir a arma de cabeça.
		out.Faces = parcel.Faces
		if rolled, ok := ecs.Get[diceRolled](w, e); ok {
			out.Dice = rolled.Values
		}
	})
	ecs.Each2(w, func(e ecs.Entity, rolled diceRolled, kind damageTypeOf) {
		if dead(w, e) {
			return
		}
		faces := 0
		if parcel, ok := ecs.Get[damageDice](w, e); ok {
			faces = parcel.Faces
		}
		out.Extra = append(out.Extra, ExtraRoll{
			Dice: rolled.Values, Faces: faces, Type: kind.Name, Total: rolled.Total,
		})
	})
	return out
}

// resourceOf e setResource: o recurso é UMA entidade, e cada acumulador é um
// componente dela. Ler pelo TIPO em vez de guardar a entidade é o que deixa o
// sistema ser função do mundo e de mais nada.
func resourceOf[C any](w *ecs.World) C {
	var found C
	ecs.Each(w, func(_ ecs.Entity, c C) { found = c })
	return found
}

func setResource[C any](w *ecs.World, c C) {
	ecs.Set(w, theResource(w), c)
}

// theAttackResource marca a entidade singleton dos acumuladores.
//
// A marca é um componente e não o id 1, ainda que a primeira entidade cunhada
// seja sempre ela: um sistema que dependa do NÚMERO da entidade depende da
// ordem em que o mundo foi montado, e essa é a dependência que não aparece em
// lugar nenhum quando alguém a quebra.
type theAttackResource struct{}

func theResource(w *ecs.World) ecs.Entity {
	var found ecs.Entity
	ecs.Each(w, func(e ecs.Entity, _ theAttackResource) { found = e })
	return found
}

func faulted(w *ecs.World) bool {
	_, has := ecs.Get[attackFault](w, theResource(w))
	return has
}

// suppressEvery marca com a mesma razão tudo que carrega C.
func suppressEvery[C any](w *ecs.World, why string) {
	ecs.Each(w, func(e ecs.Entity, _ C) { suppress(w, e, why) })
}

// suppress cala a parcela, e A PRIMEIRA RAZÃO É A QUE FICA.
//
// Mais de um sistema alcança a mesma parcela, e sem esta porta a razão que a
// mesa lê é a do ÚLTIMO a passar. O bônus de crítico de um ataque que errou
// saía dizendo "não foi um acerto crítico" — verdade que não explica nada: ele
// não contou porque o ataque errou, e a distinção é a diferença entre "role de
// novo" e "role melhor".
func suppress(w *ecs.World, e ecs.Entity, why string) {
	if dead(w, e) {
		return
	}
	ecs.Set(w, e, Suppressed{Why: why})
}
