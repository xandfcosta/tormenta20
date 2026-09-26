package engine

import "t20engine/domain/ecs"

// A DERIVAÇÃO EM SISTEMAS: aqui a entidade é a FICHA (ALE-378, fatia 4).
//
// A coleta pergunta "quem concede?" e a entidade é a fonte; a resolução
// pergunta "quanto vale este alvo?" e a entidade é o termo. A derivação
// pergunta "qual é o número final?", e há UM só de cada — então a entidade é a
// ficha, e os números derivados são componentes dela.
//
// # A ORDEM é o grafo de dependência, e ele é raso
//
// Só existe UMA aresta: a carga alimenta o deslocamento e as perícias. Todo o
// resto lê `(personagem, efeitos)` e nada mais. O escalonador declara isso numa
// lista, onde antes estava implícito em quem chamava quem.
//
// # O que esta fatia CUSTA, medido e registrado
//
// Antes, o compilador era o escalonador: `displacementBreakdown(base, e, load)`
// não compila sem a carga na mão. Com sistemas, uma dependência esquecida vira
// um componente com valor ZERO em tempo de execução, em silêncio.
//
// A troca foi decisão do dono — o critério passou a ser aprender ECS, e não o
// que paga em manutenção (ALE-378). O que repõe a rede é o
// `TestEveryDerivationSystemWritesItsComponent`: ele afirma que cada sistema
// escreveu o componente dele, com denominador. Sem ele, a fatia trocaria um
// erro de compilação por um zero calado.

// sheetInput é a ENTRADA da derivação, e é o único componente que nenhum
// sistema escreve — ele é posto antes de o escalonador rodar.
type sheetInput struct {
	char        Character
	effects     ItemEffects
	raceSquares int
}

// Os componentes DERIVADOS. Cada um é escrito por exatamente um sistema, e o
// agrupamento é por PERGUNTA — o que se lê junto mora junto, que é o critério
// de componente em ECS.
type (
	derivedLoad       struct{ v LoadBreakdown }
	derivedAttributes struct{ v map[string]AttributeBreakdown }
	derivedDefense    struct{ v DefenseBreakdown }
	derivedMovement   struct {
		displacement ValueBreakdown
		flySpeed     int
	}
	derivedMagic struct {
		pmLimit     ValueBreakdown
		bestBaseCd  *int
		byAttribute map[string]int
		dcBonus     TotalContribs
		costMod     TotalContribs
	}
	derivedCombat struct {
		attackAll  TotalContribs
		damageAll  TotalContribs
		reduction  RdBreakdown
		tempHpFury TempHpBreakdown
	}
	derivedExpertises struct {
		rows     []ExpertiseBreakdown
		autoFail []string
	}
)

// derivationSystems é a ORDEM, e ela é o grafo de dependência escrito.
//
// A carga vem primeiro porque o deslocamento e as perícias a leem. Os outros
// cinco não dependem de ninguém, e a ordem entre eles é arbitrária — o que NÃO
// é arbitrário é eles virem depois, e é por isso que a lista tem uma linha em
// branco no meio.
func derivationSystems() []ecs.System {
	return []ecs.System{
		deriveLoad,

		deriveAttributes,
		deriveDefense,
		deriveMovement,
		deriveMagic,
		deriveCombat,
		deriveExpertises,
	}
}

// deriveSheet roda a derivação e monta a ficha decomposta.
func deriveSheet(in sheetInput) ComputedSheet {
	w := ecs.NewWorld()
	ecs.Set(w, w.Spawn(), in)
	ecs.Run(w, derivationSystems()...)
	return sheetFromWorld(w)
}

// onSheet é o laço que todo sistema de derivação usa: ele acha a ficha e a
// entrega junto com a entidade em que escrever.
//
// Existe para os sistemas não repetirem a varredura, e para nenhum deles poder
// escrever numa entidade que não seja a da ficha.
func onSheet(w *ecs.World, visit func(e ecs.Entity, in sheetInput)) {
	ecs.Each(w, visit)
}

func deriveLoad(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		ecs.Set(w, e, derivedLoad{v: loadBreakdownOf(in.char, inventorySlotsTotal(in.char, in.effects))})
	})
}

func deriveAttributes(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		attrs := make(map[string]AttributeBreakdown, len(AttributeKeys))
		for _, a := range AttributeKeys {
			attrs[a] = attributeBreakdown(in.char, a, in.effects)
		}
		ecs.Set(w, e, derivedAttributes{v: attrs})
	})
}

func deriveDefense(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		ecs.Set(w, e, derivedDefense{v: defenseBreakdown(in.char, in.effects)})
	})
}

// deriveMovement LÊ a carga, e é uma das duas pontas da única aresta do grafo.
func deriveMovement(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		load, _ := ecs.Get[derivedLoad](w, e)
		ecs.Set(w, e, derivedMovement{
			displacement: displacementBreakdown(in.raceSquares, in.effects, load.v),
			flySpeed:     flySpeedTotal(in.effects),
		})
	})
}

func deriveMagic(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		ecs.Set(w, e, derivedMagic{
			pmLimit:     pmLimitBreakdown(in.char, in.effects),
			bestBaseCd:  bestBaseSpellCd(in.char, in.effects),
			byAttribute: spellCdByAttribute(in.char, in.effects),
			dcBonus:     spellDCBonus(in.effects),
			costMod:     pmCostMod(in.effects),
		})
	})
}

func deriveCombat(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		ecs.Set(w, e, derivedCombat{
			attackAll:  totalContribsFor(in.effects, ModifierTarget{K: "attack", Scope: "all"}),
			damageAll:  totalContribsFor(in.effects, ModifierTarget{K: "damage", Scope: "all"}),
			reduction:  characterDamageReduction(in.char, in.effects),
			tempHpFury: tempHpFromPowers(in.char, in.effects, true),
		})
	})
}

// deriveExpertises é a outra ponta da aresta: a penalidade de carga entra em
// três perícias.
func deriveExpertises(w *ecs.World) {
	onSheet(w, func(e ecs.Entity, in sheetInput) {
		load, _ := ecs.Get[derivedLoad](w, e)
		rows := []ExpertiseBreakdown{}
		for _, ex := range in.char.Expertises {
			rows = append(rows, expertiseBreakdown(in.char, ex, in.effects, load.v))
		}
		ecs.Set(w, e, derivedExpertises{rows: rows, autoFail: autoFailExpertises(in.effects)})
	})
}

// sheetFromWorld lê os componentes derivados e monta a ficha.
func sheetFromWorld(w *ecs.World) ComputedSheet {
	var out ComputedSheet
	ecs.Each(w, func(e ecs.Entity, _ sheetInput) {
		load, _ := ecs.Get[derivedLoad](w, e)
		attrs, _ := ecs.Get[derivedAttributes](w, e)
		def, _ := ecs.Get[derivedDefense](w, e)
		mov, _ := ecs.Get[derivedMovement](w, e)
		magic, _ := ecs.Get[derivedMagic](w, e)
		combat, _ := ecs.Get[derivedCombat](w, e)
		skills, _ := ecs.Get[derivedExpertises](w, e)
		out = ComputedSheet{
			Defense:            def.v,
			Displacement:       mov.displacement,
			FlySpeed:           mov.flySpeed,
			Load:               load.v,
			Attributes:         attrs.v,
			PmLimit:            magic.pmLimit,
			BestBaseSpellCd:    magic.bestBaseCd,
			SpellCdByAttribute: magic.byAttribute,
			SpellDCBonus:       magic.dcBonus,
			PmCostMod:          magic.costMod,
			AttackAll:          combat.attackAll,
			DamageAll:          combat.damageAll,
			DamageReduction:    combat.reduction,
			TempHpFury:         combat.tempHpFury,
			Expertises:         skills.rows,
			AutoFailExpertises: skills.autoFail,
		}
	})
	return out
}
