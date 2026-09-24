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
	Source    string
	Wear      *string
	Modifiers []Modifier
}

// ActiveItemsInWorld é o `ActiveItemsFor` rodando no ECS.
//
// @example engine.Catalogs{}.ActiveItemsInWorld(ch) // as mesmas fontes, na mesma ordem
func (c *Catalogs) ActiveItemsInWorld(ch Character) []ActiveItem {
	world := ecs.NewWorld()
	ecs.Run(world, c.collectionSystems(ch)...)

	items := []ActiveItem{}
	ecs.Each(world, func(_ ecs.Entity, g Grants) {
		items = append(items, ActiveItem{Source: g.Source, Equipped: g.Wear, Modifiers: g.Modifiers})
	})
	return items
}

// collectionSystems é a ORDEM, e ela é a regra.
//
// Trocar duas linhas aqui troca a ordem da lista, e o teste de paridade reprova
// — que é o resultado certo. Esta fatia é a única coisa que precisa casar com a
// sequência do `ActiveItemsFor`.
func (c *Catalogs) collectionSystems(ch Character) []ecs.System {
	return []ecs.System{
		c.equippedItems(ch),
		c.appliedEffects(ch),
		spawnAll(func() []ActiveItem { return c.raceActiveItems(ch) }),
		spawnOne(func() *ActiveItem { return c.originActiveItem(ch) }),
		spawnAll(func() []ActiveItem { return c.classActiveItems(ch) }),
		spawnAll(func() []ActiveItem { return c.generalPowerActiveItem(ch) }),
		spawnOne(func() *ActiveItem { return c.tormentaCarismaItem(ch) }),
		spawnOne(func() *ActiveItem { return conditionActiveItem(ch) }),
	}
}

// grant pendura uma fonte no mundo. É o `append` do coletor velho.
func grant(w *ecs.World, item ActiveItem) {
	ecs.Set(w, w.Spawn(), Grants{
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

// equippedItems: o que a pessoa está USANDO. Item sem estado de uso não entra —
// uma espada na mochila não modifica nada.
func (c *Catalogs) equippedItems(ch Character) ecs.System {
	return func(w *ecs.World) {
		proficiencies := parseProficiencySet(ch.Proficiencies)
		for _, it := range ch.Items {
			if it.Equipped == nil {
				continue
			}
			grant(w, c.itemActiveItem(it, proficiencies))
		}
	}
}

// appliedEffects: os efeitos em vigor. Efeito sem modificador NÃO entra — ele
// existiria como linha vazia na decomposição, dizendo que algo contribuiu zero.
func (c *Catalogs) appliedEffects(ch Character) ecs.System {
	return func(w *ecs.World) {
		for _, eff := range ch.ActiveEffects {
			mods := parseEffectModifiers(eff.Modifiers)
			if len(mods) == 0 {
				continue
			}
			grant(w, ActiveItem{
				Source:    fmt.Sprintf("%s (%s)", c.appliedEffectName(eff.CatalogID, mods), DurationLabel(eff.Scope)),
				Equipped:  &vestedWear,
				Modifiers: mods,
			})
		}
	}
}
