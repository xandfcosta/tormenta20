package engine

import "t20engine/domain/ecs"

// A BANDEIRA NA RESOLUÇÃO: quem a acende, quem a apaga e quem a consulta.
//
// Ela sai do `resolve_ecs.go` porque tem razão própria para mudar, e a razão
// apareceu: a bandeira é a única parcela que a resolução CALA antes de ela
// virar contribuição, então quem precisa saber QUEM a acendeu não tem onde
// buscar — e passou a precisar quando a tela teve de explicar qual peça calou a
// Destreza na Defesa (ALE-444).
//
// São cinco sistemas, e eles rodam em ORDEM: acende, isenta, cala o que a
// bandeira desliga, cobra o que ela exige, e no fim tira os termos de bandeira
// da pilha. A sequência inteira está no `resolveInWorld`.

// worldFlags é o acumulador: o Set das bandeiras acesas e, ao lado, quem
// acendeu cada uma. Ele é o irmão do `worldFactors`, e o `resolve_ecs.go`
// explica o idioma de recurso singleton que os dois usam.
type worldFlags struct {
	set    map[string]bool
	source map[string]string
}

// raiseFlags acende toda flag de termo que cumpre a própria condição.
func raiseFlags(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(_ ecs.Entity, t Term) {
		if t.Mod.Target.K == "flag" && conditionMet(t.Mod, t.Wear) {
			raiseFlagWithSource(flags, t)
		}
	})
}

// raiseFlagWithSource acende a bandeira do termo e guarda quem a acendeu, e a
// PRIMEIRA fonte fica.
//
// Primeira e não última porque a ordem é a dos itens da mochila: trocar o nome
// no meio faria a mesma ficha explicar a mesma Defesa com peças diferentes
// conforme a ordem em que o jogador guardou as coisas.
func raiseFlagWithSource(flags worldFlags, t Term) {
	flags.set[t.Mod.Target.Name] = true
	if _, named := flags.source[t.Mod.Target.Name]; !named {
		flags.source[t.Mod.Target.Name] = t.Source
	}
}

// eraseExemptFlags apaga a bandeira de que uma fonte ISENTA o portador (p340).
//
// Ele é o oposto do `raiseFlags`, e existe porque uma composição só sabia
// ACRESCENTAR: a Cota élfica é uma *"cota de malha defensora de mitral"*, a cota
// de malha é pesada e acende a bandeira que zera a Destreza na Defesa, e o
// verbete desfaz exatamente essa metade — *"permite que você aplique sua
// Destreza na Defesa como se fosse uma armadura leve"*.
//
// # Por que é um SISTEMA e não uma condição
//
// A isenção não fala do modificador dela mesma: ela fala do MUNDO. Escrita como
// condição, ela teria de ser avaliada dentro de cada termo que consulta a
// bandeira — e a ordem entre "acende" e "isenta" viraria a ordem em que os
// termos aparecem, que é a ordem dos itens da ficha.
func eraseExemptFlags(w *ecs.World) {
	exempt := map[string]bool{}
	ecs.Each(w, func(_ ecs.Entity, t Term) {
		if t.Mod.Target.K == "flagExempt" && conditionMet(t.Mod, t.Wear) {
			exempt[t.Mod.Target.Name] = true
		}
	})
	if len(exempt) == 0 {
		return
	}
	flags := flagsOf(w)
	for name := range exempt {
		delete(flags.set, name)
		delete(flags.source, name)
	}
	// E O TERMO QUE A ACENDE SAI JUNTO. Apagar só do mapa não basta: o
	// `harvestFlags`, lá no fim da sequência, varre os termos de novo e
	// REACENDE toda flag que não esteja calada — foi o que este sistema fez na
	// primeira versão, e a bandeira voltava sem ninguém ver.
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if t.Mod.Target.K == "flag" && exempt[t.Mod.Target.Name] {
			ecs.Set(w, e, Suppressed{Why: "a bandeira " + t.Mod.Target.Name + " está isenta"})
		}
	})
}

// dropWhatTheFlagTurnsOff cala o passivo do livro que se desliga enquanto a
// flag está posta.
func dropWhatTheFlagTurnsOff(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if c := t.Mod.Condition; c != nil && c.C == "flagOff" && flags.set[c.Flag] {
			ecs.Set(w, e, Suppressed{Why: "a flag " + c.Flag + " está posta"})
		}
	})
}

// dropWhatTheFlagRequires cala o que só vale ENQUANTO a flag está posta.
//
// É o espelho do `dropWhatTheFlagTurnsOff`, e os dois existem porque o livro
// escreve as duas frases: *"+4 na Defesa se NÃO estiver usando armadura
// pesada"* (Pele de Ferro, p62) e *"se estiver usando uma armadura pesada, você
// recebe +2 na Defesa"* (Encouraçado, p125).
//
// # Por que não dá para usar o `flagOn` que já existe
//
// Ele é OPT-IN, e a flag dele é POSTURA: a Fúria, a Inspiração, as seis do
// cavaleiro — coisas que a pessoa LIGA na aba Efeitos. A `armadura-pesada` é
// derivada do que está vestido, e oferecer um interruptor para ela seria
// perguntar ao jogador o que a ficha já sabe.
func dropWhatTheFlagRequires(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if c := t.Mod.Condition; c != nil && c.C == "flagRequired" && !flags.set[c.Flag] {
			ecs.Set(w, e, Suppressed{Why: "a flag " + c.Flag + " não está posta"})
		}
	})
}

// harvestFlags tira as flags da pilha: elas são interruptor, não número.
func harvestFlags(w *ecs.World) {
	flags := flagsOf(w)
	ecs.Each(w, func(e ecs.Entity, t Term) {
		if dead(w, e) || deferred(w, e) || t.Mod.Target.K != "flag" {
			return
		}
		raiseFlagWithSource(flags, t)
		ecs.Set(w, e, Suppressed{Why: "é flag, e flag não soma"})
	})
}

func flagsOf(w *ecs.World) worldFlags {
	var found worldFlags
	ecs.Each(w, func(_ ecs.Entity, f worldFlags) { found = f })
	return found
}
