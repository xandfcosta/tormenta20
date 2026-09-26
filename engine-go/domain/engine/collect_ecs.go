package engine

import (
	"fmt"

	"t20engine/domain/ecs"
)

// A COLETA, EM SISTEMAS (ALE-382).
//
// Este arquivo é o caminho novo do `ActiveItemsFor`, e ele existe ao lado do
// velho de propósito: durante a ALE-378 os dois motores rodam e o oráculo das 18
// fichas prova que concordam — que é justamente o que o cabeçalho do
// `genoracle` diz não conseguir provar enquanto houver um só.
//
// # Por que isto encaixa sem forçar
//
// O `ActiveItemsFor` já era oito coletores numa sequência fixa, e a ORDEM do
// resultado é a ordem em que eles rodam. A `ecs.Run` é essa sequência, e a
// garantia de ordem de inserção do armazenamento entrega a mesma lista. Nenhum
// dos dois lados precisou ceder.
//
// # O componente é UM, e isso é resposta a uma pergunta que se faz
//
// O `ActiveItem` tem três campos e TODA entrada tem os três — partir em
// `Source`, `Wear` e `Modifiers` daria um componente em três pedaços, sem
// nenhuma consulta que peça um sem o outro. O que varia é a ENTRADA: é lá que
// nascem os componentes com consulta, na fatia que decompõe o equipamento.
type Grants struct {
	// SourceID é o id ESTÁVEL do verbete — ver `ActiveItem` (ALE-386).
	SourceID  string
	Source    string
	Wear      *string
	Modifiers []Modifier
}

// ActiveItemsFor colhe toda fonte de modificador ativa num `[]ActiveItem` — a
// entrada que a resolução consome.
//
// A ORDEM é significativa: o despejo de paridade a compara byte a byte, e quem
// a prende é o `TestEveryCollectorLandsInTheDeclaredOrder`.
//
// Desde a ALE-378 esta é a ÚNICA coleta. A lista de sistemas é a especificação
// dela, e regra nova entra ali — não em duas funções que alguém mantém iguais.
//
// @example engine.BookRuleset(catalogs).ActiveItemsFor(ch)
func (r *Ruleset) ActiveItemsFor(ch Character) []ActiveItem {
	world := ecs.NewWorld()
	ecs.Run(world, r.collectionSystems(ch)...)

	items := []ActiveItem{}
	ecs.Each(world, func(_ ecs.Entity, g Grants) {
		items = append(items, ActiveItem{SourceID: g.SourceID, Source: g.Source, Equipped: g.Wear, Modifiers: g.Modifiers})
	})
	return items
}

// collectionSystems é a ORDEM, e ela é a regra.
//
// Trocar duas linhas aqui troca a ordem da lista, e o teste de paridade reprova
// — que é o resultado certo. Esta fatia é a única coisa que precisa casar com a
// sequência do `ActiveItemsFor`.
func (r *Ruleset) collectionSystems(ch Character) []ecs.System {
	systems := []ecs.System{}
	// O EQUIPAMENTO é o primeiro, e ele são SEIS passadas e não uma (ALE-385):
	// as entidades nascem na primeira e as cinco seguintes acrescentam o que
	// cada capacidade concede. As sete fontes abaixo continuam de uma passada
	// só, porque o coletor delas já devolve `ActiveItem` pronto.
	systems = append(systems, r.itemSystems(ch)...)
	return append(systems,
		r.appliedEffects(ch),
		spawnAll(func() []ActiveItem { return r.raceActiveItems(ch) }),
		spawnOne(func() *ActiveItem { return r.originActiveItem(ch) }),
		spawnAll(func() []ActiveItem { return r.classActiveItems(ch) }),
		spawnAll(func() []ActiveItem { return r.generalPowerActiveItem(ch) }),
		spawnOne(func() *ActiveItem { return r.godPowerActiveItem(ch) }),
		spawnOne(func() *ActiveItem { return r.tormentaCarismaItem(ch) }),
		spawnOne(func() *ActiveItem { return conditionActiveItem(ch) }),
		spawnAll(func() []ActiveItem { return r.campaignGrants(ch) }),
		// O SILÊNCIO é o último sistema, e tem de ser: ele age sobre o que os
		// anteriores penduraram, inclusive sobre uma concessão da própria mesa.
		r.silenceSystem(ch),
	)
}

// grant pendura uma fonte no mundo. É o `append` do coletor velho.
func grant(w *ecs.World, item ActiveItem) {
	ecs.Set(w, w.Spawn(), Grants{
		SourceID:  item.SourceID,
		Source:    item.Source,
		Wear:      item.Equipped,
		Modifiers: item.Modifiers,
	})
}

// spawnAll e spawnOne embrulham os seis coletores que já devolvem `ActiveItem`
// pronto. Eles NÃO ganham nada de virar sistema hoje — o que ganham é o
// endereço: quando a fatia seguinte decompuser o equipamento, um coletor novo
// entra nesta lista em vez de dentro de uma função de 30 linhas.
func spawnAll(collect func() []ActiveItem) ecs.System {
	return func(w *ecs.World) {
		for _, item := range collect() {
			grant(w, item)
		}
	}
}

func spawnOne(collect func() *ActiveItem) ecs.System {
	return func(w *ecs.World) {
		if item := collect(); item != nil {
			grant(w, *item)
		}
	}
}

// appliedEffects: os efeitos em vigor. Efeito sem modificador NÃO entra — ele
// existiria como linha vazia na decomposição, dizendo que algo contribuiu zero.
func (r *Ruleset) appliedEffects(ch Character) ecs.System {
	return func(w *ecs.World) {
		for _, eff := range ch.ActiveEffects {
			mods := parseEffectModifiers(eff.Modifiers)
			if len(mods) == 0 {
				continue
			}
			grant(w, ActiveItem{
				SourceID:  eff.CatalogID,
				Source:    fmt.Sprintf("%s (%s)", r.appliedEffectName(eff.CatalogID, mods), DurationLabel(eff.Scope)),
				Equipped:  &vestedWear,
				Modifiers: mods,
			})
		}
	}
}

// silenceSystem tira do mundo os termos que a mesa calou.
//
// Ele reusa o `applySilences` do coletor legado em vez de varrer as entidades
// por conta: a regra de quem cala quem é UMA, e duas implementações dela
// divergiriam na primeira vez que o endereço de um termo mudasse.
//
// Entidade que fica sem termo nenhum é REMOVIDA, e não deixada vazia: o
// `ecs.Each` varre na ordem de inserção, e uma fonte vazia viraria linha de
// decomposição dizendo que algo contribuiu zero.
func (r *Ruleset) silenceSystem(ch Character) ecs.System {
	return func(w *ecs.World) {
		if len(r.mesa.Silences) == 0 {
			return
		}
		ecs.Each(w, func(e ecs.Entity, g Grants) {
			kept := applySilences(r.mesa.Silences, ch, []ActiveItem{{
				SourceID: g.SourceID, Source: g.Source, Equipped: g.Wear, Modifiers: g.Modifiers,
			}})
			if len(kept) == 0 {
				ecs.Remove[Grants](w, e)
				return
			}
			g.Modifiers = kept[0].Modifiers
			ecs.Set(w, e, g)
		})
	}
}
