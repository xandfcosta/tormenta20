package engine

import "t20engine/domain/ecs"

// O ITEM EQUIPADO, DECOMPOSTO EM SISTEMAS (ALE-385).
//
// O `itemActiveItem` é cinco transformações empilhadas numa função, e a ORDEM
// delas é o que o oráculo julga. Aqui elas viram cinco sistemas, e a ordem cai
// de graça: cada um ACRESCENTA ao `Grants` da entidade, e a `ecs.Run` roda na
// sequência escrita. Não há nada a coordenar à mão.
//
// # A capacidade é COMPONENTE, e é isso que muda a leitura
//
// Os marcadores carregam a CLASSIFICAÇÃO, e o sistema para de reperguntar: o
// `UnbalancedFixed` funde três condições — é arma, tem o traço `desbalanceada`,
// tem a melhoria — numa passada, e o sistema que roda sobre ele só escreve o
// modificador. A regra passa a estar na CONSULTA, visível, em vez de enterrada
// num `if` no meio de uma função de trinta linhas.
//
// O que isso compra: uma regra da mesa nova é um marcador, uma linha na
// classificação, um sistema e uma linha na lista. **Nenhuma edição em função
// existente** — que é a diferença entre estender e mexer.

// WornItem é a fonte crua: o que a pessoa está usando, e o verbete do livro.
type WornItem struct {
	Item    CharacterItem
	Catalog *CatalogItem
}

// OwnGrants é o ESTÁGIO 1 congelado — catálogo, melhorias e material.
//
// Ele existe separado do `Grants` porque duas das cinco transformações leem os
// modificadores ANTES das penalidades: o espelho de ataque de arma e o
// esotérico vestido. Sem os dois estágios, elas leriam o acumulado e
// espelhariam a própria penalidade.
type OwnGrants struct{ Modifiers []Modifier }

// Os MARCADORES. Struct vazia é o suficiente: quem responde a pergunta é a
// existência do componente, e a consulta é onde a regra fica legível.
type (
	// Unproficient: a pessoa não sabe usar o que está usando (p142).
	Unproficient struct{}
	// WeaponLike: o verbete declara arma.
	WeaponLike struct{}
	// UnbalancedFixed: arma desbalanceada COM a melhoria que anula o traço.
	UnbalancedFixed struct{}
	// VestedEsoteric: esotérico VESTIDO que a regra opcional deixa manter o
	// bônus de empunhadura.
	VestedEsoteric struct{}
)

// itemSystems são as seis passadas sobre os itens equipados, NA ORDEM.
//
// Trocar duas linhas aqui troca a ordem dos modificadores dentro do item, e o
// teste de paridade reprova — que é o resultado certo.
func (r *Ruleset) itemSystems(ch Character) []ecs.System {
	return []ecs.System{
		r.spawnWornItems(ch),
		penalizeUnproficient,
		mirrorWeaponAttack,
		grantEquilibradaHomebrew,
		grantVestedEsotericHomebrew,
	}
}

// spawnWornItems cria uma entidade por item em uso e a CLASSIFICA.
//
// Item sem estado de uso não entra: uma espada na mochila não modifica nada.
func (r *Ruleset) spawnWornItems(ch Character) ecs.System {
	return func(w *ecs.World) {
		proficiencies := parseProficiencySet(ch.Proficiencies)
		for _, it := range ch.Items {
			if it.Equipped == nil {
				continue
			}
			var catalog *CatalogItem
			if it.CatalogID != nil {
				catalog = r.getCatalogItem(*it.CatalogID)
			}
			own := r.ownItemMods(it, catalog)

			e := w.Spawn()
			ecs.Set(w, e, WornItem{Item: it, Catalog: catalog})
			ecs.Set(w, e, OwnGrants{Modifiers: own})
			// O `Grants` NASCE com os mods próprios, e os sistemas seguintes
			// acrescentam. A cópia é obrigatória: sem ela os dois componentes
			// dividiriam o mesmo arranjo e o primeiro `append` que couber na
			// capacidade sobrescreveria o que o `OwnGrants` guarda.
			ecs.Set(w, e, Grants{
				SourceID:  catalogItemID(it),
				Source:    it.Name,
				Wear:      it.Equipped,
				Modifiers: append([]Modifier{}, own...),
			})

			r.classifyWornItem(w, e, it, catalog, proficiencies)
		}
	}
}

// classifyWornItem responde, de uma vez, as perguntas que os sistemas fariam.
func (r *Ruleset) classifyWornItem(w *ecs.World, e ecs.Entity, it CharacterItem, catalog *CatalogItem, prof map[string]bool) {
	if catalog == nil {
		return
	}
	if required := requiredProficiency(catalog); required != "" && !prof[required] {
		ecs.Set(w, e, Unproficient{})
	}
	// O ESOTÉRICO NÃO É ARMA, e por isso ele vem ANTES do retorno cedo.
	// Escrevê-lo depois foi o defeito que o oráculo pegou: o medalhão de prata
	// é `esoteric`, caía no `return` de "não é arma", e os dois bônus de
	// empunhadura sumiam da ficha do Kharvos.
	if it.Equipped != nil && *it.Equipped == "vested" && homebrewVestedOK[catalog.ID] {
		ecs.Set(w, e, VestedEsoteric{})
	}
	if catalog.Weapon == nil {
		return
	}
	ecs.Set(w, e, WeaponLike{})
	if contains(catalog.Weapon.Traits, "desbalanceada") &&
		contains(parseStringArray(it.Improvements), "melhoria-equilibrada") {
		ecs.Set(w, e, UnbalancedFixed{})
	}
}

// ownItemMods é o estágio 1: o que o verbete e as sobreposições concedem.
func (r *Ruleset) ownItemMods(it CharacterItem, catalog *CatalogItem) []Modifier {
	own := []Modifier{}
	if catalog != nil {
		own = append(own, catalog.Modifiers...)
	}
	for _, id := range parseStringArray(it.Improvements) {
		own = append(own, overlayModsWithProvenance(r.getCatalogItem(id))...)
	}
	if it.Material != nil {
		own = append(own, overlayModsWithProvenance(r.getCatalogItem(*it.Material))...)
	}
	return own
}

// noProficiencies é o conjunto vazio que os sistemas passam para uma função
// que repergunta o que a classificação já respondeu. Nomeado para o leitor não
// achar que alguém esqueceu de preencher.
var noProficiencies = map[string]bool{}

// grantMore acrescenta ao que a entidade já concede.
func grantMore(w *ecs.World, e ecs.Entity, more []Modifier) {
	if len(more) == 0 {
		return
	}
	g, ok := ecs.Get[Grants](w, e)
	if !ok {
		return
	}
	g.Modifiers = append(g.Modifiers, more...)
	ecs.Set(w, e, g)
}

// ─── os quatro sistemas de regra ──────────────────────────────────────
//
// Cada um roda sobre quem TEM a marca, e nenhum repergunta a condição — ela foi
// respondida na classificação. Quem não tem a marca não é visitado.

// penalizeUnproficient: a penalidade da p142 por usar o que não se sabe usar.
func penalizeUnproficient(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, _ Unproficient) {
		item, _ := ecs.Get[WornItem](w, e)
		// O CONJUNTO VAZIO é deliberado. A marca já diz que falta a
		// proficiência — a `nonProficiencyPenalties` repergunta isso no começo
		// dela, e o conjunto vazio faz a resposta ser a mesma que a
		// classificação já deu. Carregar o conjunto real na entidade só para
		// ser reperguntado é o que esta fatia existe para não fazer.
		grantMore(w, e, nonProficiencyPenalties(item.Catalog, noProficiencies))
	})
}

// mirrorWeaponAttack espelha em ATAQUE o que a arma dá na perícia dela.
func mirrorWeaponAttack(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, _ WeaponLike) {
		item, _ := ecs.Get[WornItem](w, e)
		own, _ := ecs.Get[OwnGrants](w, e)
		grantMore(w, e, mirrorWeaponAttackMods(item.Catalog, own.Modifiers))
	})
}

// grantEquilibradaHomebrew: a melhoria que anula o traço desbalanceada.
func grantEquilibradaHomebrew(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, _ UnbalancedFixed) {
		item, _ := ecs.Get[WornItem](w, e)
		grantMore(w, e, equilibradaHomebrewMods(item.Catalog, parseStringArray(item.Item.Improvements)))
	})
}

// grantVestedEsotericHomebrew: o esotérico vestido mantém o bônus de empunhar.
func grantVestedEsotericHomebrew(w *ecs.World) {
	ecs.Each(w, func(e ecs.Entity, _ VestedEsoteric) {
		item, _ := ecs.Get[WornItem](w, e)
		own, _ := ecs.Get[OwnGrants](w, e)
		grantMore(w, e, vestedEsotericHomebrewMods(item.Item.Equipped, item.Catalog, own.Modifiers))
	})
}
