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
// O `d20` entra por parâmetro e os dados por função: os sistemas se fecham
// sobre eles, como o `explodeIntoTerms(items)` já faz. É o que deixa cada caso
// do livro virar um teste com o número escrito à mão.
//
// # Dois mundos, porque são dois tempos
//
// O TESTE (`judgeSystems` e a camuflagem) devolve a sugestão do livro e não lê
// a notação de dano. O DANO (`damageSystems`) só roda depois da decisão do
// mestre, e começa trocando a sugestão pela decisão — por isso não há sistema
// que cale as parcelas de um ataque que errou: um erro não chega até aqui.

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

// criticalExtra marca a parcela INTEIRA como extra de crítico — é o caso do
// bônus que só existe nele (a Dilacerante, p336).
//
// Ela existe porque o livro dá a um alvo "chance de ignorar o dano extra de
// acertos críticos" (p166, p338) e a decisão da casa é INFORMAR essa chance em
// vez de rolá-la: para o mestre ter o que tirar, o número precisa ser
// recortável do total.
type criticalExtra struct{}

// criticalMultiplied guarda quantos dados a parcela tinha ANTES de o crítico
// multiplicar. O que passar de `Base` nas rolagens é extra de crítico.
type criticalMultiplied struct{ Base int }

// diceRolled é o que o `rollTheDice` escreveu — a única escrita de aleatório
// do mundo inteiro.
type diceRolled struct {
	Values []int
	Total  int
}

// rawTally e absorbedTally são as duas últimas contas, e são duas porque são
// dois sistemas: quem soma as parcelas não sabe de RD, e quem aplica a RD não
// sabe de parcela.
type rawTally struct {
	Raw int
	// ByType é o cru POR TIPO de dano, e ele existe para a RD contra um tipo
	// (p229) saber o que alcança. A soma dele é o `Raw` — toda parcela tem tipo,
	// nem que seja o da própria arma.
	ByType map[string]int
	// CriticalExtra é quanto do cru veio do crítico — ver `criticalExtra`.
	CriticalExtra int
}
type absorbedTally struct{ Absorbed, Final int }

// attackFault guarda o PRIMEIRO erro e cala os sistemas seguintes.
//
// Um `ecs.System` é `func(*World)` e não devolve erro — a alternativa seria
// mudar a assinatura do núcleo por causa de dois pontos de falha (a notação do
// catálogo e o sorteio). O erro é um componente como outro qualquer, e quem
// chama o pergunta ao mundo no fim.
type attackFault struct{ Err error }

// judgeSystems são o TESTE: a Tabela 5-3, o acerto e o crítico que o livro
// sugere. Eles rodam nos dois tempos — no do dano, para a conta ecoar o mesmo
// total e a mesma Defesa que o mestre leu.
func judgeSystems(
	card WeaponCard, target AttackTarget, situations []SpecialSituation, die AttackDie,
) []ecs.System {
	return []ecs.System{
		spawnTheSituations(situations),
		dropWhatDoesNotReachThisWeapon(card),
		forbidTheAttackIfAnyoneSaysSo,
		tallyTheSituations,
		judgeTheHit(card, target, die),
		judgeTheCritical(card, target, die.Natural),
	}
}

// damageSystems são o DANO, sobre a decisão do mestre.
func damageSystems(
	card WeaponCard, target AttackTarget, ruling Ruling, rollDie func(faces int) (int, error),
) []ecs.System {
	return []ecs.System{
		applyTheRuling(card, ruling),
		spawnTheDamageParcels(card),
		multiplyTheWeaponDice,
		dropWhatIsNotCritical,
		rollTheDice(rollDie),
		sumTheParcels(card),
		absorbWithDamageReduction(target),
	}
}

// applyTheRuling troca a sugestão do livro pela decisão do mestre. O total e a
// rolagem ficam: o que muda é o que eles QUEREM DIZER.
//
// O crítico do mestre vale mesmo contra quem o livro diz imune (p231) e mesmo
// fora da margem: a decisão é dele, e o app informa a sugestão antes dela.
func applyTheRuling(card WeaponCard, ruling Ruling) ecs.System {
	return func(w *ecs.World) {
		verdict := resourceOf[hitVerdict](w)
		verdict.Hit = true
		setResource(w, verdict)
		_, multiplier := CriticalOf(card)
		setResource(w, criticalVerdict{Critical: ruling == RulingCritical, Multiplier: multiplier})
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
func judgeTheHit(card WeaponCard, target AttackTarget, die AttackDie) ecs.System {
	return func(w *ecs.World) {
		// A Tabela 5-3 já somou, e ela entra nos DOIS lados: a posição elevada
		// no ataque, a cobertura na Defesa (p239).
		situations := resourceOf[situationTally](w)
		total := die.Rolled + card.Attack + situations.Attack
		defense := target.Defense + situations.Defense
		setResource(w, effectiveDefense{Value: defense})
		setResource(w, hitVerdict{
			Roll:  die.Rolled,
			Total: total,
			Hit:   die.Natural == 20 || (die.Natural != 1 && total >= defense),
		})
	}
}

// effectiveDefense é a Defesa que ESTE ataque enfrentou, depois da Tabela 5-3.
// Ela viaja até a mesa porque "errei por 1" e "errei por 1 porque ele está
// atrás da carroça" são leituras diferentes do mesmo número.
type effectiveDefense struct{ Value int }

// judgeTheCritical decide o crítico, e só depois do acerto.
//
// "Você faz um acerto crítico quando ACERTA um ataque rolando um valor igual ou
// maior que a margem de ameaça" (p231): são as duas condições.
// O zero do catálogo cai no padrão do livro (`CriticalOf`) em vez de virar uma
// arma que nunca critica.
func judgeTheCritical(card WeaponCard, target AttackTarget, d20 int) ecs.System {
	return func(w *ecs.World) {
		margin, multiplier := CriticalOf(card)
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
			// Ele é EXTRA DE CRÍTICO por definição: fora do crítico não existe.
			ecs.Set(w, parcel, criticalExtra{})
		}
	}
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
		// QUANTOS DADOS ERAM ANTES, para o extra do crítico ser recortável.
		//
		// A marca fica na MESMA parcela, e isso é escolha: uma parcela nova com
		// os dados acrescentados daria o mesmo total e mudaria a ORDEM do
		// sorteio — o 1d6 de fogo do encanto passaria a rolar antes do segundo
		// d8 da espada. Ordem de sorteio é o que a mesa lê ("a espada deu 8 e
		// 5, o fogo deu 4"), e trocá-la por uma conveniência de recorte seria
		// mexer no que a pessoa vê para facilitar a vida de quem soma.
		ecs.Set(w, e, criticalMultiplied{Base: dice.Count})
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
func sumTheParcels(card WeaponCard) ecs.System {
	return func(w *ecs.World) {
		raw := 0
		byType := map[string]int{}
		// O TIPO DA PARCELA é o dela quando ela traz um, e o da ARMA quando não:
		// os dados da espada e o bônus de Força são o corte da espada. Sem este
		// recuo, uma redução de corte não alcançaria o dano da arma de corte —
		// que é o caso da Armadura Óssea do osteon (RD 5 a corte, frio e
		// perfuração).
		extra := 0
		soma := func(e ecs.Entity, amount int) {
			raw += amount
			if _, inteira := ecs.Get[criticalExtra](w, e); inteira {
				extra += amount
			}
			tipo := card.DamageType
			if own, has := ecs.Get[damageTypeOf](w, e); has {
				tipo = own.Name
			}
			if tipo != "" {
				byType[tipo] += amount
			}
		}
		ecs.Each(w, func(e ecs.Entity, rolled diceRolled) {
			if dead(w, e) {
				return
			}
			soma(e, rolled.Total)
			// O EXCEDENTE da parcela que o crítico multiplicou: o que passa da
			// contagem de antes é o que ele acrescentou.
			if before, multiplicada := ecs.Get[criticalMultiplied](w, e); multiplicada {
				for i := before.Base; i < len(rolled.Values); i++ {
					extra += rolled.Values[i]
				}
			}
		})
		ecs.Each(w, func(e ecs.Entity, parcel damageFlat) {
			if !dead(w, e) {
				soma(e, parcel.Amount)
			}
		})
		setResource(w, rawTally{Raw: raw, ByType: byType, CriticalExtra: extra})
	}
}

// absorbWithDamageReduction aplica a RD do alvo, e são DUAS famílias.
//
//	"Se uma criatura com RD 5 sofre um ataque que causa 8 pontos de dano, perde
//	 apenas 3 PV. A redução pode ser contra um ou mais tipos de dano
//	 específicos. Assim, uma criatura com redução de fogo 10 ignora 10 pontos de
//	 dano de fogo, mas sofre dano de outros tipos normalmente." (p229)
//
// A DO TIPO VEM PRIMEIRO, e a ordem é decisão desta casa — o livro não diz o
// que fazer quando as duas alcançam o mesmo ataque. É a única ordem bem
// definida: a geral come de um total, e depois dela ninguém sabe quanto do que
// sobrou era fogo. Cada uma age UMA VEZ por ataque, que é como o livro conta
// ("ignora 10 pontos de dano de fogo", não "de cada parcela").
//
// O PISO EM ZERO é decisão desta casa: o livro só dá o caso em que sobra dano, e
// "ignora parte do dano que sofre" não descreve um ataque que devolve PV. Sem o
// piso, uma RD alta viraria cura — e a cura tem regra própria, que não é esta.
func absorbWithDamageReduction(target AttackTarget) ecs.System {
	return func(w *ecs.World) {
		tally := resourceOf[rawTally](w)
		absorbed := 0
		for tipo, rd := range target.DamageReductionByType {
			absorbed += min(rd, tally.ByType[tipo])
		}
		absorbed += min(target.DamageReduction, tally.Raw-absorbed)
		setResource(w, absorbedTally{Absorbed: absorbed, Final: tally.Raw - absorbed})
	}
}

// outcomeFromWorld remonta a conta que a mesa lê.
func outcomeFromWorld(w *ecs.World) AttackOutcome {
	hit := resourceOf[hitVerdict](w)
	out := AttackOutcome{
		Roll:       hit.Roll,
		Total:      hit.Total,
		Hit:        hit.Hit,
		Defense:    resourceOf[effectiveDefense](w).Value,
		Situations: labelsOfTheSituations(w),
		Critical:   resourceOf[criticalVerdict](w).Critical,
		RawDamage:  resourceOf[rawTally](w).Raw,

		CriticalExtra: resourceOf[rawTally](w).CriticalExtra,
		Absorbed:      resourceOf[absorbedTally](w).Absorbed,
		Damage:        resourceOf[absorbedTally](w).Final,
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

// suppress cala a parcela, e A PRIMEIRA RAZÃO É A QUE FICA: um sistema que
// passe depois não troca o motivo que a mesa lê pelo dele.
func suppress(w *ecs.World, e ecs.Entity, why string) {
	if dead(w, e) {
		return
	}
	ecs.Set(w, e, Suppressed{Why: why})
}
