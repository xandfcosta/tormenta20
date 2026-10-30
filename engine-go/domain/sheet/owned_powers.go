package sheet

import (
	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O ACERVO DE PODERES: o que o personagem TEM, nas seis procedências.
//
// Ele morava em `serve/web/sheetui` como função privada, e isso tinha preço: a
// POSSE também é pergunta do CASO DE USO — usar um poder que não é seu, receber
// a concessão de um poder que não é seu —, e o `app/` não alcança o `serve/`
// (ALE-426). Aqui ele tem dois leitores e uma resposta só.
//
// O TEXTO continua sendo do catálogo e a decoração é da tela: esta lista diz o
// que o personagem tem e de onde veio; "Classe · Bárbaro" é frase, e frase mora
// na cena.

// PowerSource é de onde o poder vem.
type PowerSource string

const (
	PowerFromRace     PowerSource = "race"
	PowerFromOrigin   PowerSource = "origin"
	PowerFromClass    PowerSource = "class"
	PowerFromGeneral  PowerSource = "general"
	PowerFromTormenta PowerSource = "tormenta"
	PowerFromGod      PowerSource = "god"
	// PowerFromItem é a SEXTA, e ela entrou com os itens mágicos (ALE-433): o
	// encanto ATIVÁVEL de um item equipado é uma coisa que se faz, e o lugar
	// onde se procura o que fazer é este acervo.
	PowerFromItem PowerSource = "item"
)

// OwnedPower é um poder que o personagem tem.
type OwnedPower struct {
	ID     string
	Name   string
	Detail string
	Source PowerSource
	// Owner é quem concede, quando isso tem nome: a raça, a origem ou a classe.
	// Vazio no poder geral e no da Tormenta, que não pertencem a ninguém, e no
	// divino, cuja fonte já vem escrita no catálogo.
	Owner string
	Page  int
}

// OwnedPowers junta as procedências na ordem em que a ficha as mostra.
//
// @example OwnedPowers(dto) // [{ID: "dahllan-armadura-de-allihanna", Source: "race", …}, …]
func OwnedPowers(dto CharacterDTO) []OwnedPower {
	outside := []OwnedPower{}
	outside = append(outside, raceAbilitiesOf(dto)...)
	outside = append(outside, raceHeldPowersOf(dto)...)
	outside = append(outside, borrowedAbilitiesOf(dto)...)
	outside = append(outside, chosenOriginBenefitsOf(dto)...)
	outside = append(outside, automaticClassPowersOf(dto)...)
	outside = append(outside, chosenPowersOf(dto)...)
	outside = append(outside, godPowerOf(dto)...)
	return append(outside, itemEnchantActivationsOf(dto)...)
}

// itemEnchantActivationsOf são os encantos ATIVÁVEIS dos itens EQUIPADOS.
//
// As duas restrições são regra, e cada uma responde uma pergunta diferente:
//
//   - ATIVÁVEL: o acervo é o que a pessoa pode FAZER, e um encanto passivo não
//     se faz — a Flamejante soma +1d6 de fogo sozinha, e ela já aparece no
//     cartão da arma e na Mochila. Listá-la aqui seria a mesma coisa duas
//     vezes. Quem responde "este tem gesto?" é o registro de ativações;
//   - EQUIPADO: espada guardada na mochila não dança. O gesto pede o item na
//     mão ou vestido, e é o mesmo critério que os modificadores do encanto já
//     usam (`condition: wielded`).
//
// O ENCANTO É DO ITEM, e por isso o `Owner` é o nome dele: duas espadas
// encantadas com a mesma coisa viram duas linhas, e sem o dono elas seriam
// iguais na tela.
func itemEnchantActivationsOf(dto CharacterDTO) []OwnedPower {
	outside := []OwnedPower{}
	for _, item := range dto.Items {
		if item.Equipped == nil || *item.Equipped == "" {
			continue
		}
		// O ITEM COMPOSTO tem gesto PRÓPRIO, e ele não é de encanto nenhum: a
		// Avalanche (p337) diz "gaste uma ação padrão e 6 PM para invocar uma
		// tempestade de gelo", e isso é do verbete.
		if catalog := catalogOf(item); catalog != nil && catalog.ComposedOf != nil {
			outside = append(outside, activationsOfItem(item, *catalog)...)
		}
		for _, id := range enchantsOf(item) {
			enchant := book.ItemByID(id)
			if enchant == nil {
				continue
			}
			outside = append(outside, activationsOfItem(item, *enchant)...)
		}
	}
	return outside
}

// catalogOf é o verbete do item da ficha, ou nil para o item caseiro.
func catalogOf(item ItemDTO) *book.Item {
	if item.CatalogID == nil || *item.CatalogID == "" {
		return nil
	}
	return book.ItemByID(*item.CatalogID)
}

// enchantsOf são os encantos EFETIVOS: os que o mestre pôs na ficha mais os que
// o verbete já traz (p337, p340).
//
// Os dois na mesma lista porque, para quem joga, não há diferença: a arma está
// encantada, e o gesto que o encanto oferece se faz do mesmo jeito.
func enchantsOf(item ItemDTO) []string {
	catalog := catalogOf(item)
	if catalog == nil || catalog.ComposedOf == nil {
		return item.Enchants
	}
	return append(append([]string{}, catalog.ComposedOf.Enchants...), item.Enchants...)
}

// activationsOfItem é o gesto que um verbete oferece, se ele tiver um.
//
// O `Owner` é o nome do ITEM da ficha, sempre: duas espadas com o mesmo encanto
// viram duas linhas, e sem o dono elas seriam iguais na tela.
func activationsOfItem(item ItemDTO, entry book.Item) []OwnedPower {
	spec := book.ActivationOf("item."+entry.ID, entry.Name)
	if spec == nil {
		return nil
	}
	return []OwnedPower{{
		ID: entry.ID, Name: spec.Name, Detail: entry.Description,
		Source: PowerFromItem, Owner: item.Name, Page: entry.BookPage,
	}}
}

// OwnsActivation diz se a ficha tem o poder daquela ativação.
//
// PELO ID OU PELO NOME, e o nome não é folga: cada catálogo escreve o mesmo
// poder com uma grafia — a Armadura de Allihanna é `dahllan-armadura-de-
// allihanna` no de raças e `race.dahllan.armadura-de-allihanna` no de
// ativações, e o Dom da Verdade é `divino.dom-da-verdade` contra
// `deus.khalmyr.dom-da-verdade`. É o mesmo elo que o `book.ActivationOf` usa na
// ida, e o que o guarda do poder único de origem já prende.
func OwnsActivation(dto CharacterDTO, spec book.Activation) bool {
	return OwnedIndexOf(dto).Has(spec)
}

// OwnedIndex são os poderes da ficha montados UMA vez, para perguntar várias
// ativações em seguida. Remontar o `OwnedPowers` a cada pergunta, num laço
// sobre o registro de ativações e outro sobre as perícias, custava ~46% da
// página de Perícias (ALE-520).
//
//	owned := sheet.OwnedIndexOf(dto)
//	for _, a := range book.Activations() { if owned.Has(a) { … } }
type OwnedIndex struct {
	ids, names map[string]bool
}

func OwnedIndexOf(dto CharacterDTO) OwnedIndex {
	powers := OwnedPowers(dto)
	index := OwnedIndex{ids: make(map[string]bool, len(powers)), names: make(map[string]bool, len(powers))}
	for _, owned := range powers {
		index.ids[owned.ID] = true
		index.names[owned.Name] = true
	}
	return index
}

// Has diz se a ficha tem o poder da ativação — pelo id ou pelo nome, como o
// `OwnsActivation` explica.
func (o OwnedIndex) Has(spec book.Activation) bool {
	return o.ids[spec.ID] || o.names[spec.Name]
}

// borrowedAbilitiesOf são as habilidades que a Memória Póstuma tomou de outra
// raça (p29, ALE-504): da ficha, com a raça de origem como dona.
func borrowedAbilitiesOf(dto CharacterDTO) []OwnedPower {
	outside := []OwnedPower{}
	for _, b := range borrowedAbilities(dto) {
		outside = append(outside, OwnedPower{
			ID: b.ability.ID, Name: b.ability.Name, Detail: b.ability.Description,
			Source: PowerFromRace, Owner: b.from.Name + " (Memória Póstuma)",
		})
	}
	book.SortByName(outside, func(p OwnedPower) string { return p.Name })
	return outside
}

// raceHeldPowersOf são os poderes que uma habilidade de raça deu: o da Tormenta
// que a Deformidade trocou por um bônus (p24, ALE-493), o geral da Memória
// Póstuma (p29, ALE-502). A ficha os TEM, e o dono é a habilidade.
func raceHeldPowersOf(dto CharacterDTO) []OwnedPower {
	outside := []OwnedPower{}
	abilities := raceAbilitiesWithChoice(dto)
	for id, picks := range RaceAbilityPicks(dto) {
		ability, known := abilities[id]
		if !known {
			continue
		}
		for _, pick := range picks {
			power, found := book.GeneralPowers()[pick]
			if !found {
				continue
			}
			source := PowerFromGeneral
			if power.Kind == "tormenta" {
				source = PowerFromTormenta
			}
			outside = append(outside, OwnedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: source, Owner: ability.Name, Page: power.BookPage,
			})
		}
	}
	return outside
}

// raceAbilitiesOf são as habilidades de cada raça da ficha.
//
// Quem lê o catálogo é o `RaceTraitsByKey`, indexado por id E por nome, porque
// o personagem guarda a raça por um dos dois.
func raceAbilitiesOf(dto CharacterDTO) []OwnedPower {
	outside := []OwnedPower{}
	for i, r := range dto.Races {
		race, found := book.RaceTraitsByKey()[r.Race]
		if !found {
			continue
		}
		ancestry := raceAncestry(dto, i, r.Race)
		for _, ability := range race.Abilities {
			// A habilidade da OUTRA ascendência não é da ficha (p31, ALE-505).
			if ability.Ancestry != "" && ability.Ancestry != ancestry {
				continue
			}
			outside = append(outside, OwnedPower{
				ID: ability.ID, Name: ability.Name, Detail: ability.Description,
				Source: PowerFromRace, Owner: race.Name,
			})
		}
	}
	return outside
}

// chosenOriginBenefitsOf são só os ESCOLHIDOS: a origem oferece mais do que o
// personagem leva, então listar todos daria como possuído o que ninguém pegou.
func chosenOriginBenefitsOf(dto CharacterDTO) []OwnedPower {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	chosen := map[string]bool{}
	for _, id := range UnmarshalStrings(dto.OriginChoices) {
		chosen[id] = true
	}
	outside := []OwnedPower{}
	for _, b := range OriginBenefitsOf(origin) {
		if !chosen[b.ID] {
			continue
		}
		outside = append(outside, OwnedPower{
			ID: b.ID, Name: b.Name, Detail: b.Description,
			Source: PowerFromOrigin, Owner: origin.Name, Page: origin.BookPage,
		})
	}
	return outside
}

// automaticClassPowersOf são as que o NÍVEL concede, sem escolha.
//
// Quem decide a posse é o MOTOR, com a mesma regra que a derivação usa para
// somar os modificadores. A lista de ESCOLHIDOS entra vazia de propósito: quem
// escolheu já sai em `chosenPowersOf`, e passá-la aqui daria o poder duas vezes.
func automaticClassPowersOf(dto CharacterDTO) []OwnedPower {
	choices := ClassChoiceSelections(dto)
	outside := []OwnedPower{}
	for _, class := range dto.Classes {
		for _, power := range book.ClassPowers() {
			if power.ClassName != class.ClassName {
				continue
			}
			if !engine.OwnsClassPower(power.ForEngine(), int(class.Level), nil, choices[class.ClassName]) {
				continue
			}
			outside = append(outside, OwnedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: PowerFromClass, Owner: class.ClassName, Page: power.BookPage,
			})
		}
	}
	book.SortByName(outside, func(p OwnedPower) string { return p.Name })
	return outside
}

// chosenPowersOf são os ids da coluna `classPowers` — poder de classe, poder
// geral ou poder da Tormenta, nessa ordem de busca.
//
// Blob torto vira lista vazia: a ficha não pode deixar de abrir porque uma linha
// do banco está errada.
func chosenPowersOf(dto CharacterDTO) []OwnedPower {
	outside := []OwnedPower{}
	for _, id := range UnmarshalStrings(dto.ClassPowers) {
		if power, found := book.ClassPowers()[id]; found {
			outside = append(outside, OwnedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: PowerFromClass, Owner: power.ClassName, Page: power.BookPage,
			})
			continue
		}
		if power, found := book.GeneralPowers()[id]; found {
			source := PowerFromGeneral
			if power.Kind == "tormenta" {
				source = PowerFromTormenta
			}
			outside = append(outside, OwnedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: source, Page: power.BookPage,
			})
		}
	}
	return outside
}

// godPowerOf é o poder que a devoção concede — UM, guardado pelo NOME.
//
// Ele não aparecia no acervo da tela, e a falta ficava calada: o Dom da Verdade
// (p133) está na ficha de quem é devoto de Khalmyr e não estava na lista que diz
// o que o personagem tem.
func godPowerOf(dto CharacterDTO) []OwnedPower {
	if dto.GodPower == "" {
		return nil
	}
	for _, power := range book.DivinePowers() {
		if power.Name != dto.GodPower {
			continue
		}
		return []OwnedPower{{
			ID: power.ID, Name: power.Name, Detail: power.Description,
			Source: PowerFromGod, Owner: power.Source, Page: power.BookPage,
		}}
	}
	return nil
}
