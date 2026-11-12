package engine

import (
	"sort"
	"strconv"
	"strings"

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
// Isso não é estética. A ficha existe para responder "por que 12?", e hoje ela
// não consegue dizer "−2 de X, não aplicado por não empilhar" — o número que
// perdeu não chega à tela. Com a tag, ele está lá para quando a tela quiser.
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
// Tag e não remoção: o termo continua no mundo, e é isso que permite a uma tela
// futura mostrar o que foi descartado. O `Why` é para ela, e para a mensagem de
// falha de quem depurar.
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
type worldFactors struct{ byTarget map[string]Ratio }

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
	ecs.Set(w, resource, worldFactors{byTarget: map[string]Ratio{}})
	ecs.Set(w, resource, worldRiders{critical: map[string]int{}})
	return w, resource
}

// resolutionSystems é a ordem dos sistemas da resolução, que é a regra. Uma lista só: o caso que mede o
// mundo por dentro (`worldAfterResolving`) a copiava, e a cópia já tinha perdido três sistemas.
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
		ecs.Set(w, e, Suppressed{Why: "a condição não se cumpre"})
	})
}

// harvestFactors tira os fatores da pilha: eles multiplicam o total, e não são
// parcela dele — ver `factor.go`.
func harvestFactors(w *ecs.World) {
	factors := factorsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) || t.Mod.Factor == nil {
			return
		}
		key := targetKey(t.Mod.Target)
		factors[key] = severest(factors[key], *t.Mod.Factor)
		ecs.Set(w, e, Suppressed{Why: "é fator, e fator não soma"})
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
		ecs.Set(w, loser, Suppressed{Why: "bônus de " + t.Mod.BonusType + " não empilha: " + strconv.Itoa(amount) + " perdeu"})
	})
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
		c := Contribution{SourceID: t.SourceID, Source: source, BonusType: t.Mod.BonusType, Amount: t.Mod.Amount}
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

	byTarget := map[string]AggregatedStat{}
	for _, key := range order {
		total := 0
		for _, c := range buckets[key] {
			total += c.Amount
		}
		byTarget[key] = AggregatedStat{Total: total, Contributions: buckets[key]}
	}
	flags, _ := ecs.Get[worldFlags](w, resource)
	factors, _ := ecs.Get[worldFactors](w, resource)
	riders := ridersOf(w)
	return ItemEffects{
		ByTarget: byTarget, Flags: flags.set, FlagSources: flags.source,
		Factors:     factors.byTarget,
		ExtraDamage: riders.extra, CriticalBonus: riders.critical,
		Announced:   riders.announced,
		Conditional: conditional,
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

// worldAfterResolving devolve quantos termos o mundo tem e as razões dos que o
// EMPILHAMENTO calou.
//
// Existe para o caso que prende a tag `Suppressed`: a saída da resolução é
// byte a byte igual à de antes, então medir a saída não diria nada sobre ela.
func worldAfterResolving(items []ActiveItem) (int, []string) {
	w, _ := resolutionWorld()
	ecs.Run(w, resolutionSystems(items, nil)...)

	total, reasons := 0, []string{}
	ecs.Each(w, func(e ecs.Entity, _ Term) {
		total++
		if s, has := ecs.Get[Suppressed](w, e); has && strings.Contains(s.Why, "não empilha") {
			reasons = append(reasons, s.Why)
		}
	})
	return total, reasons
}
