package engine

import (
	"slices"

	"t20engine/domain/ecs"
)

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
// `Unproficient` responde uma vez "a pessoa sabe usar isto?", e o sistema que
// roda sobre ele só escreve o modificador. A regra passa a estar na CONSULTA,
// visível, em vez de enterrada num `if` no meio de uma função de trinta linhas.
//
// O homebrew que morava aqui (a Equilibrada e o medalhão vestido) virou regra
// da mesa: o mestre a escreve no editor, e ela não é mais marcador.

// WornItem é a fonte crua: o que a pessoa está usando, e o verbete do livro.
type WornItem struct {
	Item    CharacterItem
	Catalog *CatalogItem
}

// OwnGrants é o ESTÁGIO 1 congelado — catálogo, melhorias e material.
//
// Ele existe separado do `Grants` porque o espelho de ataque de arma lê os
// modificadores ANTES das penalidades. Sem os dois estágios, ele leria o
// acumulado e espelharia a própria penalidade.
type OwnGrants struct{ Modifiers []Modifier }

// Os MARCADORES. Struct vazia é o suficiente: quem responde a pergunta é a
// existência do componente, e a consulta é onde a regra fica legível.
type (
	// Unproficient: a pessoa não sabe usar o que está usando (p142).
	Unproficient struct{}
	// WeaponLike: o verbete declara arma.
	WeaponLike struct{}
)

// itemSystems são as passadas sobre os itens equipados, NA ORDEM.
//
// Trocar duas linhas aqui troca a ordem dos modificadores dentro do item, e o
// teste de paridade reprova — que é o resultado certo.
func (r *Ruleset) itemSystems(ch Character) []ecs.System {
	return []ecs.System{
		r.spawnWornItems(ch),
		penalizeUnproficient,
		mirrorWeaponAttack,
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
				Wear:      r.wornWear(ch, it),
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
	if !HoldsProficiently(catalog, it.Equipped, prof) {
		ecs.Set(w, e, Unproficient{})
	}
	if catalog.Weapon == nil {
		return
	}
	ecs.Set(w, e, WeaponLike{})
}

// ownItemMods é o estágio 1: o que o verbete e as sobreposições concedem.
//
// SEM VERBETE, NADA — nem as sobreposições (ALE-389).
//
// A regra do dono é que o item criado pelo JOGADOR é cosmético, e ela valia por
// ausência: `character_items` não tem coluna de modificador. O buraco da
// ausência era este laço, que aplicava melhoria, material e encanto mesmo com o
// verbete nulo — um item caseiro com `material-adamante` e `melhoria-reforcada`
// gravados entregava Defesa +1 e penalidade de armadura −1.
//
// A sobreposição é uma linha que o livro escreve SOBRE um item: ela diz "+1 na
// Defesa desta armadura", e sem a armadura não há a que ela se aplique. O
// `overlayModsFor` não tinha como recusar sozinho — ele recebe a CATEGORIA do
// hospedeiro, e sem hospedeiro ela vem vazia, o que deixa passar tudo o que não
// é restrito a família nenhuma.
//
// Vale também para o item de OUTRO MUNDO: um herói levado a outra mesa não leva
// o verbete que só a mesa dele conhece, e o `catalogId` que não resolve tem de
// contribuir zero INTEIRO — não zero do verbete e o resto da sobreposição.
//
// A CENA JÁ RECUSA O GESTO (`fitsItemImprovement`: "um item custom não recebe
// melhoria"), e continua sendo a fronteira dele. Esta porta é da CONTA: linha
// antiga não passou por aquela recusa.
func (r *Ruleset) ownItemMods(it CharacterItem, catalog *CatalogItem) []Modifier {
	own := []Modifier{}
	if catalog == nil {
		return own
	}
	own = append(own, catalog.Modifiers...)
	for _, id := range parseStringArray(it.Improvements) {
		own = append(own, overlayModsFor(r.getCatalogItem(id), catalog)...)
		if id == ImprovedItemImprovement {
			own = append(own, ImprovedSkillsOf(catalog.Modifiers, catalog.SkillTies)...)
		}
	}
	if it.Material != nil {
		own = append(own, overlayModsFor(r.getCatalogItem(*it.Material), catalog)...)
	}
	for _, id := range it.Enchants {
		own = append(own, overlayModsFor(r.getCatalogItem(id), catalog)...)
	}
	return onThisWeapon(own, catalog)
}

// onThisWeapon prende à arma o dano, a margem e o multiplicador que ela mesma concede, com o verbete,
// a melhoria, o material ou o encanto: "a arma fornece +1 nas rolagens de dano" (Cruel, p165), "a
// margem de ameaça [desta arma] aumenta" (Precisa, p166). No escopo vazio ou no `this`, toda carta os
// lia, e a Cruel da espada subia a adaga da outra mão e o ataque desarmado. O ANÚNCIO vai junto: "uma
// criatura viva atingida fica sangrando" (Sanguinária, p336) é o golpe desta arma.
//
// O escopo é o do id do catálogo, como o do espelho de ataque: duas adagas iguais na mão o dividem.
// O ataque fica de fora porque `mirrorWeaponAttackMods` já o leva, e o passo de dano porque
// `DamageStepsOf` o lê só dos modificadores deste item.
func onThisWeapon(mods []Modifier, catalog *CatalogItem) []Modifier {
	if catalog.Weapon == nil {
		return mods
	}
	out := slices.Clone(mods)
	for i, m := range out {
		unscoped := m.Target.Scope == "" || m.Target.Scope == "this"
		strike := m.Target.K == "critRange" || m.Target.K == "critMult" ||
			m.Target.K == "announcedCondition" || m.Target.K == "announcedEffect" ||
			(m.Target.K == "damage" && (m.Target.Scope == "this" || m.Dice != ""))
		if unscoped && strike {
			out[i].Target.Scope = WeaponDamageScope(catalog.ID)
		}
	}
	return out
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

// diceOfThisCard são os dados extras que valem nesta carta: os de toda arma (escopo vazio ou `all`, como
// o Ataque Furtivo), e os da arma dela.
func diceOfThisCard(extra []ExtraDamage, scopes []string) []ExtraDamage {
	out := []ExtraDamage{}
	for _, die := range extra {
		if reachesThisCard(die.Scope, scopes) {
			out = append(out, die)
		}
	}
	return out
}

// announcesOfThisCard são os anúncios que valem nesta carta, pela mesma régua do dado extra.
func announcesOfThisCard(announced []AnnouncedCondition, scopes []string) []AnnouncedCondition {
	out := []AnnouncedCondition{}
	for _, a := range announced {
		if reachesThisCard(a.Scope, scopes) {
			out = append(out, a)
		}
	}
	return out
}

func reachesThisCard(scope string, scopes []string) bool {
	return scope == "" || scope == "all" || slices.Contains(scopes, scope)
}
