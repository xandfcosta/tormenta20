package engine

import (
	"sort"
	"strconv"

	"t20engine/domain/ecs"
)

// A RESOLUÇÃO EM SISTEMAS: aqui a ENTIDADE é o TERMO (ALE-378, fatia 3).
//
// Na coleta a entidade é a FONTE — um machado, uma raça, uma condição —, e o
// que ela carrega é uma lista de modificadores. A resolução pergunta outra
// coisa: "quanto vale `defense`, e por quê". Essa pergunta é sobre TERMOS, e é
// por isso que eles viram entidades aqui.
//
// # O que muda em relação à função que isto substitui
//
// A função que isto substituiu descartava: o modificador que não cumpria condição, o
// que perdia o empilhamento e o que virava fator saíam da lista e sumiam. Aqui
// eles ficam no mundo com uma TAG (`Suppressed`), e quem soma pergunta quem
// está vivo.
//
// Isso não é estética. A ficha existe para responder "por que 12?", e o número que
// perdeu o empilhamento chega à conta dela riscado, com o motivo (`OutOfTheSum`).
//
// # A ORDEM continua sendo a regra
//
// A ordem das chaves de alvo é a de PRIMEIRA APARIÇÃO, e a das contribuições
// dentro de cada uma é a das fontes. O `ecs.World` varre na ordem de inserção
// justamente para isso — é a propriedade em volta da qual o núcleo foi
// desenhado, porque `map` em Go não a tem.

// Term é UM modificador de uma fonte, como entidade.
type Term struct {
	SourceID string
	Source   string
	Wear     *string
	Mod      Modifier
}

// Suppressed marca o termo que NÃO conta, e diz por quê.
//
// Tag e não remoção: o termo continua no mundo. O `Why` é para quem depurar; o que a
// ficha mostra é o `OutOfTheSum`, que só a parcela descartada recebe — o fator e o
// dado extra também saem da pilha, e não foram descartados.
//
// O PRIMEIRO MOTIVO É O QUE FICA (`suppress`), na resolução e no ataque: um
// sistema que passe depois não troca a razão que já foi escrita.
type Suppressed struct{ Why string }

// Deferred marca o termo que vira OPT-IN: ele não entra na conta, e aparece na
// lista de condicionais para o jogador ligar.
type Deferred struct{}

// SwitchedOn marca o termo condicional que o jogador LIGOU: ele entra na conta como qualquer outro, e
// continua na lista dos oferecidos para a tela desenhar o interruptor aceso.
//
// O interruptor é ENTRADA da resolução, e não uma segunda passada sobre o resultado. Houve uma segunda
// passada, que não existe mais: ela refazia o empilhamento por lista e colhia de novo fatores e
// dados; ela esqueceu um campo duas vezes, e cada regra nova da resolução tinha de ser escrita nas duas.
type SwitchedOn struct{}

// Optional marca o termo que a pessoa PODE ligar: o adiado e o já ligado. É a lista da aba Efeitos.
//
// Marca própria, e não "adiado ou ligado", porque o ligado pode ser calado depois de virar opção — o dado
// extra sai da pilha para a carta — e continua sendo um interruptor aceso.
type Optional struct{}

// sourceOrder é a posição do termo na ordem das fontes. A lista dos oferecidos sai nessa ordem, e não na
// do mundo: o termo ligado nasce no fim (ver `explodeIntoTerms`), e a lista da aba Efeitos não pode pular
// quando alguém liga um interruptor.
type sourceOrder struct{ seq int }

// worldFactors é um dos dois acumuladores da resolução — o outro é o
// `worldFlags`, que mora no `resolve_ecs_flag.go` —, e os dois são guardados
// numa entidade SINGLETON.
//
// É o idioma de "recurso" do ECS: um dado que é do mundo inteiro, e não de uma
// entidade em particular. Sem ele, um sistema teria de escrever numa variável
// capturada — e aí ele deixaria de ser função do mundo, que é o contrato.
type worldFactors struct {
	byTarget map[string]Ratio
	// source é o id de quem pôs o fator que venceu em cada alvo — o mais severo, e no empate o primeiro —,
	// para a tela nomeá-lo ("Alta Arcana corta este custo pela metade").
	source map[string]string
}

// resolveInWorld dobra as fontes colhidas em `ItemEffects`, em sistemas, com os interruptores que o
// jogador ligou: pelo endereço do termo (`TermID`) ou pelo do grupo da flag (`FlagGroupID`).
func resolveInWorld(items []ActiveItem, switches map[string]bool) ItemEffects {
	w, resource := resolutionWorld()
	ecs.Run(w, resolutionSystems(items, switches)...)
	return effectsFromWorld(w, resource)
}

// resolutionWorld é o mundo vazio da resolução, com a entidade dos acumuladores.
func resolutionWorld() (*ecs.World, ecs.Entity) {
	w := ecs.NewWorld()
	resource := w.Spawn()
	ecs.Set(w, resource, worldFlags{set: map[string]bool{}, source: map[string]string{}})
	ecs.Set(w, resource, worldFactors{byTarget: map[string]Ratio{}, source: map[string]string{}})
	ecs.Set(w, resource, worldRiders{critical: map[string]int{}})
	return w, resource
}

// resolutionSystems é a ordem dos sistemas da resolução, que é a regra.
func resolutionSystems(items []ActiveItem, switches map[string]bool) []ecs.System {
	return []ecs.System{
		explodeIntoTerms(items, switches),
		raiseFlags,
		// DEPOIS de acender e ANTES de qualquer um que leia flag: uma bandeira
		// apagada não pode continuar calando o que ela calava.
		eraseExemptFlags,
		// O `flagOff` só pode ser julgado DEPOIS de todas as flags estarem no ar
		// — era a "passada prévia" da função antiga, e aqui ela é um sistema com
		// nome. A dependência deixou de ser um comentário e virou posição.
		dropWhatTheFlagTurnsOff,
		// O ESPELHO do de cima, e no mesmo lugar da sequência: quem EXIGE a
		// flag também só pode ser julgado com todas elas no ar.
		dropWhatTheFlagRequires,
		// ANTES do `deferConditionals`: o dado extra e o bônus de crítico saem
		// da pilha aqui, ou virariam interruptor na aba Efeitos — e crítico não
		// se liga.
		harvestWeaponRiders,
		deferConditionals,
		dropWhatTheConditionRefuses,
		harvestFactors,
		harvestFlags,
		// DEPOIS do `harvestFactors`, porque ele precisa do mapa pronto, e
		// ANTES do empilhamento, porque o que a Fúria Titânica dobra é a
		// parcela e não o vencedor: com o degrau pago, o balde `morale` tem o
		// +2 da base e o +5 do degrau, e os dois entram dobrados para o
		// não-empilhamento escolher o maior (p42).
		scaleWhatTheSourceFactorHits,
		resolveStacking,
	}
}

// explodeIntoTerms transforma cada fonte colhida em N entidades de termo.
//
// A fonte sem `Wear` não entra: item que não está vestido nem empunhado não
// concede nada, e essa porta é a primeira da função antiga.
//
// O TERMO LIGADO NASCE DEPOIS DE TODOS OS OUTROS, e a ordem é a regra: a contribuição dele entra no fim
// de cada alvo, o dado extra dele depois dos incondicionais, e no empate do empilhamento quem já estava
// vence. É a ordem que a segunda passada de antes produzia, e o oráculo a guarda.
func explodeIntoTerms(items []ActiveItem, switches map[string]bool) ecs.System {
	return func(w *ecs.World) {
		type late struct {
			term Term
			seq  int
		}
		later := []late{}
		seq := 0
		for _, item := range items {
			if item.Equipped == nil {
				continue
			}
			for _, m := range item.Modifiers {
				t := Term{SourceID: item.SourceID, Source: item.Source, Wear: item.Equipped, Mod: m}
				seq++
				if switchedOn(item.SourceID, m, switches) {
					later = append(later, late{t, seq})
					continue
				}
				e := w.Spawn()
				ecs.Set(w, e, t)
				ecs.Set(w, e, sourceOrder{seq: seq})
			}
		}
		for _, l := range later {
			e := w.Spawn()
			ecs.Set(w, e, l.term)
			ecs.Set(w, e, sourceOrder{seq: l.seq})
			ecs.Set(w, e, SwitchedOn{})
		}
	}
}

// switchedOn diz se o jogador ligou este termo condicional: pelo endereço dele ou pelo do grupo da flag
// que ele divide. O incondicional não tem interruptor, e um id ligado que casasse com ele não muda nada.
func switchedOn(sourceID string, m Modifier, switches map[string]bool) bool {
	if len(switches) == 0 || isUnconditional(m) {
		return false
	}
	if switches[TermID(sourceID, m)] {
		return true
	}
	c := m.Condition
	return c != nil && c.C == "flagOn" && switches[FlagGroupID(c.Flag)]
}

// deferConditionals separa o que o jogador LIGA do que vale sozinho. O que ele já ligou não é adiado:
// entra na conta.
func deferConditionals(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || isUnconditional(t.Mod) {
			return
		}
		ecs.Set(w, e, Optional{})
		if !switched(w, e) {
			ecs.Set(w, e, Deferred{})
		}
	})
}

// dropWhatTheConditionRefuses cala o termo cuja condição não se cumpre.
func dropWhatTheConditionRefuses(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, t Term) {
		// O LIGADO NÃO É JULGADO: a condição dele é a circunstância que a pessoa disse que vale.
		if dead(w, e) || deferred(w, e) || switched(w, e) || conditionMet(t.Mod, t.Wear) {
			return
		}
		suppress(w, e, "a condição não se cumpre")
	})
}

// harvestFactors tira os fatores da pilha: eles multiplicam o total, e não são
// parcela dele — ver `factor.go`.
func harvestFactors(w *ecs.World) {
	factors := factorsOf(w)
	var sources map[string]string
	ecs.Each(w, func(_ ecs.Entity, f worldFactors) { sources = f.source })
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) || t.Mod.Factor == nil {
			return
		}
		key := targetKey(t.Mod.Target)
		if won := severest(factors[key], *t.Mod.Factor); won != factors[key] {
			factors[key] = won
			sources[key] = t.SourceID
		}
		suppress(w, e, "é fator, e fator não soma")
	})
}

// scaleWhatTheSourceFactorHits multiplica a PARCELA das fontes que têm fator.
//
// O fator de alvo opera no TOTAL, depois da soma; o de FONTE opera na parcela,
// antes dela. A diferença é o que o livro pede: a Fúria Titânica (p42) dobra "o
// bônus que você recebe quando usa Fúria", e não o ataque inteiro — dobrar o
// total levaria Luta, Força e o bônus da arma junto.
//
// O DEFERIDO FICA DE FORA, porque não entra na conta; o LIGADO entra, e é multiplicado aqui, uma vez —
// a Fúria é condicional, e é o caso que o livro dá.
func scaleWhatTheSourceFactorHits(w *ecs.World) {
	factors := factorsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) {
			return
		}
		scaled := sourceFactorOf(factors, t.SourceID)
		if !scaled.isSet() {
			return
		}
		t.Mod.Amount = scaled.Applied(t.Mod.Amount)
		ecs.Set(w, e, t)
	})
}

// resolveStacking cala quem perde o empilhamento (T20 p226: efeitos de itens "acumulam
// com os de outras fontes, mas não entre si", e p333: "use apenas o melhor"). O
// `untyped` é a exceção e acumula.
//
// Ele CALA em vez de descartar, e é a diferença que esta fatia compra: o termo
// que perdeu continua no mundo, com a razão escrita.
func resolveStacking(w *ecs.World) {
	type slot struct {
		best   ecs.Entity
		amount int
	}
	winner := map[string]slot{}
	type loss struct {
		loser ecs.Entity
		key   string
	}
	losses := []loss{}
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) || accumulates(t.Mod.BonusType) {
			return
		}
		key := targetKey(t.Mod.Target) + "::" + t.Mod.BonusType
		held, seen := winner[key]
		if !seen {
			winner[key] = slot{best: e, amount: t.Mod.Amount}
			return
		}
		loser, amount := e, t.Mod.Amount
		if strongerThan(amount, held.amount) {
			loser, amount = held.best, held.amount
			winner[key] = slot{best: e, amount: t.Mod.Amount}
		}
		suppress(w, loser, "bônus de "+t.Mod.BonusType+" não empilha: "+strconv.Itoa(amount)+" perdeu")
		losses = append(losses, loss{loser, key})
	})
	// O MOTIVO NOMEIA O VENCEDOR FINAL, e por isso sai depois da varredura: com +1, +2 e +3 do mesmo tipo,
	// o +1 perdeu primeiro para o +2, que depois perdeu para o +3 — e a conta da ficha diz "não empilha com"
	// quem está nela, não com outro que também ficou de fora.
	for _, l := range losses {
		won, _ := ecs.Get[Term](w, winner[l.key].best)
		ecs.Set(w, l.loser, OutOfTheSum{Note: stackingLossNote(won.Source, won.Mod.BonusType)})
	}
}

// OutOfTheSum marca a parcela que mirava um alvo e não entrou nele, com a frase que a conta da ficha mostra
// (`AggregatedStat.Dropped`). Só a recebe o que a pessoa esperaria ver somado: o que perdeu o empilhamento.
// O dado extra, o fator e a flag também saem da pilha, mas não foram descartados: foram para outro lugar.
type OutOfTheSum struct{ Note string }

// stackingLossNote é o motivo de uma parcela que perdeu o empilhamento. As duas pilhas — a da resolução e a
// das perícias (`resolveStack`) — dizem com a mesma frase.
//
// A CONDIÇÃO tem regra e página próprias: "Condições com os mesmos efeitos não se acumulam; aplique apenas
// os mais severos" (p394). Todas elas vêm de uma fonte só, "Condições", e por isso a frase não nomeia o
// vencedor: "não empilha com Condições" não diria nada. O resto segue a p226, e o vencedor tem nome.
func stackingLossNote(winner, bonusType string) string {
	if bonusType == "condition" {
		return "não entra: condições com os mesmos efeitos não se acumulam, vale a mais severa (p394)"
	}
	return "não entra: não empilha com " + winner + " (mesmo tipo de bônus, p226)"
}

// effectsFromWorld lê o mundo resolvido e monta os `ItemEffects`.
//
// A ordem das chaves é a de PRIMEIRA APARIÇÃO, e a das contribuições é a das
// fontes — as duas saem de graça da varredura em ordem de inserção.
func effectsFromWorld(w *ecs.World, resource ecs.Entity) ItemEffects {
	order := []string{}
	buckets := map[string][]Contribution{}
	conditional := []ConditionalEffect{}

	type offer struct {
		seq int
		ce  ConditionalEffect
	}
	offers := []offer{}
	dropped := map[string][]Contribution{}
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if out, has := ecs.Get[OutOfTheSum](w, e); has && !deferred(w, e) {
			key := targetKey(t.Mod.Target)
			dropped[key] = append(dropped[key], Contribution{
				SourceID: t.SourceID, Source: t.Source, BonusType: t.Mod.BonusType, Amount: t.Mod.Amount, Note: out.Note,
			})
		}
	})
	ecs.Each(w, func(e ecs.Entity, t Term) {
		on := switched(w, e)
		if _, optional := ecs.Get[Optional](w, e); optional {
			at, _ := ecs.Get[sourceOrder](w, e)
			offers = append(offers, offer{at.seq, conditionalFrom(t)})
		}
		if deferred(w, e) || dead(w, e) {
			return
		}
		key := targetKey(t.Mod.Target)
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		source := t.Source
		if on {
			// O LIGADO SE DIZ LIGADO na conta: a tela mostra de onde veio a parcela, e "(cond.)" é o que
			// separa a Fúria acesa de um bônus que vale sempre.
			source += " (cond.)"
		}
		c := Contribution{SourceID: t.SourceID, Source: source, BonusType: t.Mod.BonusType, Amount: t.Mod.Amount,
			StacksWith: t.Mod.StacksWith, StacksWithAll: t.Mod.StacksWithAll}
		if note := t.Mod.Note; on {
			// A NOTA DO LIGADO é a do interruptor ("Em Fúria", "no subterrâneo"): é ela que diz na conta por
			// que a parcela vale agora.
			c.Note = conditionalFrom(t).Note
		} else if note != "" {
			c.Note = note
		}
		buckets[key] = append(buckets[key], c)
	})

	sort.SliceStable(offers, func(a, b int) bool { return offers[a].seq < offers[b].seq })
	for _, o := range offers {
		conditional = append(conditional, o.ce)
	}

	// O ALVO DE CADA CHAVE, de todo termo do mundo: o fator sai da pilha antes de virar parcela, e a
	// chave dele também precisa dizer o que alcança.
	targets := []keyedTarget{}
	known := map[string]bool{}
	ecs.Each(w, func(_ ecs.Entity, t Term) {
		if key := targetKey(t.Mod.Target); !known[key] {
			known[key] = true
			targets = append(targets, keyedTarget{key: key, target: t.Mod.Target})
		}
	})

	byTarget := map[string]AggregatedStat{}
	for _, key := range order {
		total := 0
		for _, c := range buckets[key] {
			total += c.Amount
		}
		byTarget[key] = AggregatedStat{Total: total, Contributions: buckets[key], Dropped: dropped[key]}
	}
	flags, _ := ecs.Get[worldFlags](w, resource)
	factors, _ := ecs.Get[worldFactors](w, resource)
	riders := ridersOf(w)
	return ItemEffects{
		ByTarget: byTarget, Flags: flags.set, FlagSources: flags.source,
		Factors: factors.byTarget, FactorSources: factors.source,
		ExtraDamage: riders.extra, CriticalBonus: riders.critical,
		Announced:   riders.announced,
		Conditional: conditional,
		targets:     targets,
	}
}

// conditionalFrom monta a linha de opt-in de um termo adiado.
func conditionalFrom(t Term) ConditionalEffect {
	ce := ConditionalEffect{
		Term:      TermID(t.SourceID, t.Mod),
		SourceID:  t.SourceID,
		Source:    t.Source,
		BonusType: t.Mod.BonusType,
		Amount:    t.Mod.Amount,
		Note:      firstNonEmpty(describeCondition(t.Mod), t.Mod.Note),
		Target:    t.Mod.Target,
		Factor:    t.Mod.Factor,
		Dice:      t.Mod.Dice,
	}
	if c := t.Mod.Condition; c != nil && c.C == "flagOn" {
		ce.Flag = c.Flag
	}
	return ce
}

// dead e deferred são as duas perguntas de TAG da resolução. Elas existem como
// função para o `if` de cada sistema se ler como a frase que ele é.
func dead(w *ecs.World, e ecs.Entity) bool {
	_, has := ecs.Get[Suppressed](w, e)
	return has
}

func deferred(w *ecs.World, e ecs.Entity) bool {
	_, has := ecs.Get[Deferred](w, e)
	return has
}

func switched(w *ecs.World, e ecs.Entity) bool {
	_, has := ecs.Get[SwitchedOn](w, e)
	return has
}

func factorsOf(w *ecs.World) map[string]Ratio {
	var found map[string]Ratio
	ecs.Each(w, func(_ ecs.Entity, f worldFactors) { found = f.byTarget })
	return found
}
