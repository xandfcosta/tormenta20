package engine

import (
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
// O `ComputeItemEffects` descartava: o modificador que não cumpria condição, o
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

// worldFlags e worldFactors são os dois acumuladores da resolução, guardados
// numa entidade SINGLETON.
//
// É o idioma de "recurso" do ECS: um dado que é do mundo inteiro, e não de uma
// entidade em particular. Sem ele, um sistema teria de escrever numa variável
// capturada — e aí ele deixaria de ser função do mundo, que é o contrato.
type worldFlags struct{ set map[string]bool }
type worldFactors struct{ byTarget map[string]Ratio }

// resolveInWorld dobra as fontes colhidas em `ItemEffects`, em sistemas.
func resolveInWorld(items []ActiveItem) ItemEffects {
	w := ecs.NewWorld()
	resource := w.Spawn()
	ecs.Set(w, resource, worldFlags{set: map[string]bool{}})
	ecs.Set(w, resource, worldFactors{byTarget: map[string]Ratio{}})

	ecs.Run(w,
		explodeIntoTerms(items),
		raiseFlags,
		// O `flagOff` só pode ser julgado DEPOIS de todas as flags estarem no ar
		// — era a "passada prévia" da função antiga, e aqui ela é um sistema com
		// nome. A dependência deixou de ser um comentário e virou posição.
		dropWhatTheFlagTurnsOff,
		deferConditionals,
		dropWhatTheConditionRefuses,
		harvestFactors,
		harvestFlags,
		resolveStacking,
	)
	return effectsFromWorld(w, resource)
}

// explodeIntoTerms transforma cada fonte colhida em N entidades de termo.
//
// A fonte sem `Wear` não entra: item que não está vestido nem empunhado não
// concede nada, e essa porta é a primeira da função antiga.
func explodeIntoTerms(items []ActiveItem) ecs.System {
	return func(w *ecs.World) {
		for _, item := range items {
			if item.Equipped == nil {
				continue
			}
			for _, m := range item.Modifiers {
				ecs.Set(w, w.Spawn(), Term{
					SourceID: item.SourceID, Source: item.Source, Wear: item.Equipped, Mod: m,
				})
			}
		}
	}
}

// raiseFlags acende toda flag de termo que cumpre a própria condição.
func raiseFlags(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(_ ecs.Entity, t Term) {
		if t.Mod.Target.K == "flag" && conditionMet(t.Mod, t.Wear) {
			flags[t.Mod.Target.Name] = true
		}
	})
}

// dropWhatTheFlagTurnsOff cala o passivo do livro que se desliga enquanto a
// flag está posta.
func dropWhatTheFlagTurnsOff(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if c := t.Mod.Condition; c != nil && c.C == "flagOff" && flags[c.Flag] {
			ecs.Set(w, e, Suppressed{Why: "a flag " + c.Flag + " está posta"})
		}
	})
}

// deferConditionals separa o que o jogador LIGA do que vale sozinho.
func deferConditionals(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || isUnconditional(t.Mod) {
			return
		}
		ecs.Set(w, e, Deferred{})
	})
}

// dropWhatTheConditionRefuses cala o termo cuja condição não se cumpre.
func dropWhatTheConditionRefuses(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) || conditionMet(t.Mod, t.Wear) {
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

// harvestFlags tira as flags da pilha: elas são interruptor, não número.
func harvestFlags(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) || t.Mod.Target.K != "flag" {
			return
		}
		flags[t.Mod.Target.Name] = true
		ecs.Set(w, e, Suppressed{Why: "é flag, e flag não soma"})
	})
}

// resolveStacking cala quem perde o empilhamento (T20 p107: bônus do mesmo tipo
// não se somam, vale o maior; `untyped` é a exceção e acumula).
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

	ecs.Each(w, func(e ecs.Entity, t Term) {
		if deferred(w, e) {
			conditional = append(conditional, conditionalFrom(t))
			return
		}
		if dead(w, e) {
			return
		}
		key := targetKey(t.Mod.Target)
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		c := Contribution{SourceID: t.SourceID, Source: t.Source, BonusType: t.Mod.BonusType, Amount: t.Mod.Amount}
		if t.Mod.Note != "" {
			c.Note = t.Mod.Note
		}
		buckets[key] = append(buckets[key], c)
	})

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
	return ItemEffects{
		ByTarget: byTarget, Flags: flags.set, Factors: factors.byTarget, Conditional: conditional,
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

func flagsOf(w *ecs.World) map[string]bool {
	var found map[string]bool
	ecs.Each(w, func(_ ecs.Entity, f worldFlags) { found = f.set })
	return found
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
	w := ecs.NewWorld()
	resource := w.Spawn()
	ecs.Set(w, resource, worldFlags{set: map[string]bool{}})
	ecs.Set(w, resource, worldFactors{byTarget: map[string]Ratio{}})
	ecs.Run(w, explodeIntoTerms(items), raiseFlags, dropWhatTheFlagTurnsOff,
		deferConditionals, dropWhatTheConditionRefuses, harvestFactors, harvestFlags, resolveStacking)

	total, reasons := 0, []string{}
	ecs.Each(w, func(e ecs.Entity, _ Term) {
		total++
		if s, has := ecs.Get[Suppressed](w, e); has && strings.Contains(s.Why, "não empilha") {
			reasons = append(reasons, s.Why)
		}
	})
	return total, reasons
}
