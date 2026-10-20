package sheetui

import (
	"encoding/json"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O CATÁLOGO DE ITENS lido pela Mochila.
//
// Tudo aqui responde a uma pergunta sobre uma linha da ficha contra o
// `items.json`: que entrada do livro é esta, o que ela concede, que melhorias
// ela está usando, e em que chip da grade ela cai.

// catalogItem acha a entrada do livro de um item da ficha.
func catalogItem(item sheet.ItemDTO) *book.Item {
	if item.CatalogID == nil || *item.CatalogID == "" {
		return nil
	}
	return book.ItemByID(*item.CatalogID)
}

// itemOverlays são os NOMES das melhorias e do material aplicados.
func itemOverlays(item sheet.ItemDTO) []string {
	names := []string{}
	for _, entry := range bookOverlays(item) {
		names = append(names, entry.Name)
	}
	// O ENCANTO ENTRA AQUI e não no `bookOverlays`, e a divisão é do DESTINO:
	// o crachá diz tudo o que a arma É, porque é ele que o jogador olha em
	// combate; o bloco "Melhorias & material" da ficha do item tem irmão
	// próprio para o encanto, e pô-lo nos dois o escreveria duas vezes.
	for _, id := range item.Enchants {
		if entry := book.ItemByID(id); entry != nil {
			names = append(names, entry.Name)
		}
	}
	return names
}

// bookOverlays resolve melhorias + material contra o catálogo.
//
// Id desconhecido é PULADO em vez de virar erro: a coluna guarda um blob de
// texto, e uma linha torta não pode impedir a mochila inteira de abrir.
func bookOverlays(item sheet.ItemDTO) []book.Item {
	ids := savedImprovements(item.Improvements)
	if item.Material != nil && *item.Material != "" {
		ids = append(ids, *item.Material)
	}
	outside := []book.Item{}
	for _, id := range ids {
		if entry := book.ItemByID(id); entry != nil {
			outside = append(outside, *entry)
		}
	}
	return outside
}

// savedImprovements lê o blob JSON da coluna `improvements`.
func savedImprovements(blob string) []string {
	var ids []string
	if json.Unmarshal([]byte(blob), &ids) != nil {
		return nil
	}
	return ids
}

// thatGrantsItem são os crachás do que o item dá enquanto equipado.
//
// A Defesa base de armadura e escudo sai UMA vez: o catálogo traz o número em
// `armor.defense` E como modificador de Defesa do mesmo valor, e desenhar os
// dois daria "Defesa +2 · Defesa +2" em toda armadura.
func thatGrantsItem(item sheet.ItemDTO, exempt wearerExemptions) []itemChip {
	catalog := catalogItem(item)
	if catalog == nil {
		return nil
	}
	chips := []itemChip{}
	if base := baseItemDefense(*catalog); base != "" {
		chips = append(chips, itemChip{Text: base})
	}
	if catalog.Weapon != nil {
		chips = append(chips, itemChip{Text: "Dano " + weaponDieOfItem(item, *catalog)})
	}
	for _, m := range catalog.Modifiers {
		chips = append(chips, itemChip{Text: modifierBadge(m)})
		// A habilidade que anula entra como crachá PRÓPRIO, logo depois do que
		// ela anulou. O par lado a lado é o que ensina: sem ele, ou o número
		// mente (era o defeito) ou ele some e ninguém fica sabendo que existe
		// uma isenção agindo a favor do personagem.
		if why := exemptedBy(m, exempt); why != "" {
			chips[len(chips)-1].Inactive = true
			// "isento por" e não só o nome da habilidade: TODO crachá deste
			// cartão descreve o ITEM, e um que diz "Devagar e Sempre" sozinho
			// se lê como se a armadura concedesse isso. Visto na tela — o risco
			// no vizinho não basta para desfazer a leitura.
			chips = append(chips, itemChip{Text: "isento por " + why, Reason: true})
		}
	}
	return chipsSem(chips)
}

// weaponDieOfItem é o dado da arma DEPOIS do que as sobreposições dela andam.
//
// O crachá descreve o ITEM, então só os passos DELE entram — o material, a
// melhoria. O passo que vem de quem empunha (o Campeão do guerreiro, p66) fica
// na carta de Combate, que é a que fala do ataque: uma espada de adamante causa
// 1d10 na mão de qualquer um, e o passo do capstone é do personagem.
//
// Quem soma é o `engine.DamageStepsOf`, o MESMO da carta: duas somas de passo
// dariam duas respostas para o mesmo dado, e a ALE-319 já cobrou essa fatura
// uma vez.
func weaponDieOfItem(item sheet.ItemDTO, catalog book.Item) string {
	steps := 0
	for _, overlay := range bookOverlays(item) {
		steps += engine.DamageStepsOf(overlay.Modifiers, item.Equipped)
	}
	return engine.DamageStep(catalog.Weapon.Damage, steps)
}

// itemChip é um crachá do cartão. `Inactive` é o item concedendo algo que NÃO
// alcança este portador — o crachá fica, riscado, porque ele descreve o item.
type itemChip struct {
	Text     string
	Inactive bool
	// Reason marca o crachá que EXPLICA o riscado ao lado. Ele é o único que
	// pode crescer em altura, e o campo existe para a tela saber disso sem
	// farejar o texto.
	Reason bool
}

// wearerExemptions são as isenções do PORTADOR: flag → nome da habilidade que a
// concede ("displacement-ignores-armor-and-load" → "Devagar e Sempre").
//
// O nome vem do catálogo e não do motor porque o motor resolve as flags num
// CONJUNTO e perde a procedência. Ver `book.RaceAbility.Modifiers`.
type wearerExemptions map[string]string

// exemptedBy devolve a habilidade que anula este modificador para este
// portador, ou vazio quando ele vale.
//
// HOJE É UM PAR SÓ — a redução de deslocamento por armadura contra o "Devagar e
// Sempre" do anão (p20) —, e o `if` está escrito à mão de propósito: uma tabela
// com uma linha esconde a regra em vez de mostrá-la. Quando o segundo par
// chegar, isto vira mapa de `targetKey` para flag.
func exemptedBy(m engine.Modifier, exempt wearerExemptions) string {
	if m.Target.K != "displacement" || m.Target.Scope != "armor" {
		return ""
	}
	return exempt[engine.DisplacementIgnoresArmor]
}

// chipsSem tira os repetidos pelo TEXTO, preservando o primeiro — o mesmo que o
// `repetidosSem` faz com as listas de string.
func chipsSem(list []itemChip) []itemChip {
	seen := map[string]bool{}
	out := []itemChip{}
	for _, chip := range list {
		if chip.Text == "" || seen[chip.Text] {
			continue
		}
		seen[chip.Text] = true
		out = append(out, chip)
	}
	return out
}

func baseItemDefense(catalog book.Item) string {
	protection := catalog.Armor
	if protection == nil {
		protection = catalog.Shield
	}
	if protection == nil || protection.Defense == 0 {
		return ""
	}
	for _, m := range catalog.Modifiers {
		if m.Target.K == "defense" && m.Amount == protection.Defense {
			return ""
		}
	}
	return "Defesa " + book.WithSign(protection.Defense)
}

// modifierBadge escreve o que um modificador de item concede.
//
// Alvo de FLAG é booleano: ele não leva número, e escrever "Fadiga ao dormir
// +1" faria a tela prometer uma quantidade que não existe.
func modifierBadge(m engine.Modifier) string {
	label := targetLabel(m.Target)
	if m.Target.K == "flag" || m.Amount == 0 {
		return label
	}
	// O DESLOCAMENTO é contado em QUADRADOS no motor e lido em METROS na mesa
	// (ALE-390). O crachá é tela, então ele converte de volta — sem isso a
	// armadura completa diria "−2" onde o livro diz "−3m".
	if m.Target.K == "displacement" {
		return label + " " + book.MetresWithSign(float64(m.Amount)*engine.SquareMetres)
	}
	return label + " " + book.WithSign(m.Amount)
}

func repetidosSem(list []string) []string {
	seen := map[string]bool{}
	outside := []string{}
	for _, text := range list {
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		outside = append(outside, text)
	}
	return outside
}

// itemCatalog é o id de catálogo de uma linha do banco, ou "".
func itemCatalog(item sqlcgen.GetItemRow) string {
	if !item.Catalogid.Valid {
		return ""
	}
	return item.Catalogid.String
}

// howEngineItem traduz a entrada do livro para a forma que as validações do
// motor esperam. Só os campos que elas leem — eixo, id e nome —, porque um
// tradutor completo prometeria que os dois lados têm a mesma forma, e não têm.
func howEngineItem(catalog *book.Item) *engine.CatalogItem {
	if catalog == nil {
		return nil
	}
	return &engine.CatalogItem{
		ID: catalog.ID, Name: catalog.Name, Category: catalog.Category,
		Equip: catalog.Equip, Slots: catalog.Slots,
	}
}

// exemptionsOf colhe as isenções que a RAÇA e os PODERES do personagem
// concedem.
//
// Aqui estava escrito "só raça hoje, e quando uma classe ou poder isentar
// alguma coisa esta função cresce". O dia chegou com o Fanático (p128), que é
// poder GERAL e isenta o deslocamento da armadura pesada — então ela cresceu,
// e exatamente até onde a medição pediu: raça e poder geral. Classe e origem
// concedem flag pelo mesmo mecanismo e nenhuma isenta nada hoje; varrer os
// catálogos delas seria custo por nada.
func exemptionsOf(dto sheet.CharacterDTO) wearerExemptions {
	exempt := wearerExemptions{}
	for _, entry := range dto.Races {
		race, found := book.RaceTraitsByKey()[entry.Race]
		if !found {
			continue
		}
		for _, ability := range race.Abilities {
			collectFlagNames(exempt, ability.Modifiers, ability.Name)
		}
	}
	byPower := map[string][]engine.Modifier{}
	for _, p := range book.GeneralPowersWithModifiers() {
		byPower[p.ID] = p.Modifiers
	}
	for _, owned := range ownedPowersOf(dto) {
		collectFlagNames(exempt, byPower[owned.ID], owned.Name)
	}
	return exempt
}

// collectFlagNames guarda, para cada flag que estes modificadores levantam, o
// NOME de quem a levantou — é essa procedência que a Mochila escreve ao riscar
// um modificador, e o motor a perde ao resolver as flags num conjunto.
func collectFlagNames(into wearerExemptions, mods []engine.Modifier, name string) {
	for _, m := range mods {
		if m.Target.K == "flag" && m.Target.Name != "" {
			into[m.Target.Name] = name
		}
	}
}
