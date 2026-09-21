package forge

import (
	"fmt"
	"sort"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/infra/wire"
)

// A FOLHA EM BRANCO DA FORJA — o que a cena desenha.
//
// A cena inteira sai daqui montada: as cartas com o que cada escolha COMPRA (os
// atributos da raça, o PV e o PM da classe), as origens, e o equipamento de
// p140 já estreitado pela classe escolhida. A tela não decide nada — ela não
// sabe que arcanista não veste armadura nem que bardo leva arma marcial; ela
// desenha as opções que este arquivo lhe entrega, e o servidor recusa o que não
// couber (`forgeRefusals`).

// forgeView é a folha inteira.
type forgeView struct {
	Name    string
	Races   []raceCard
	Classes []classCard
	Origins []originOption
	// Equipamento é nil enquanto não há classe escolhida: o kit de p140 só se
	// conhece pela classe, e oferecer uma armadura antes disso seria oferecer a
	// escolha errada para o arcanista.
	Gear   *startingGear
	Errors wire.FieldErrorMap
	// OrphanRefusals são as recusas cujo campo NÃO está na tela.
	//
	// Elas existem porque a folha só desenha o que o kit oferece: mandar uma
	// arma marcial para um ladino é recusado, e o seletor de arma marcial não
	// existe na folha dele — a frase não teria onde aparecer, e a pessoa veria
	// a folha voltar sem uma palavra. Isso só acontece com um POST feito na mão
	// ou com uma folha respondida antes de trocar a classe, e as duas merecem
	// resposta.
	OrphanRefusals []string
}

// raceCard é uma das 17 cartas da linhagem.
type raceCard struct {
	Name         string
	Attributes   string
	Size         string
	Displacement int
	Abilities    string
	Chosen       bool
}

// classCard é uma das 14 cartas do ofício.
type classCard struct {
	Name       string
	PV         int
	PM         int
	Expertises string
	Chosen     bool
}

// originOption é uma linha da lista de origens.
type originOption struct {
	Name    string
	Benefit string
	Chosen  bool
}

// startingGear é o kit de p140 já reduzido ao que ESTA classe recebe.
type startingGear struct {
	SimpleWeapons  []itemOption
	MartialWeapons []itemOption
	Armors         []itemOption
	// Escudo é o rótulo do escudo do kit, ou "" quando a classe não usa escudos.
	Shield       string
	ShieldChosen bool
	// AResolver são as concessões de origem que são ESCOLHA e não item — elas
	// não nascem na mochila, e a folha diz isso em vez de deixar o jogador
	// descobrir que faltou algo.
	ToResolve []string
}

// itemOption é uma entrada de um dos seletores do equipamento.
type itemOption struct {
	ID     string
	Label  string
	Chosen bool
}

// blankForgeSheet monta a cena a partir das respostas que já existem.
//
// Ela é chamada nas três situações e devolve a mesma coisa nas três: a folha
// vazia do primeiro GET, a folha redesenhada quando a classe muda, e a folha
// recusada com os erros por campo.
func blankForgeSheet(sheet forgeAnswers, errs wire.FieldErrorMap) forgeView {
	races, classes, _ := book.CharacterCatalogs()
	v := forgeView{Name: sheet.Name, Errors: errs}
	for _, race := range races {
		v.Races = append(v.Races, raceCardOf(race, sheet.Race))
	}
	for _, class := range classes {
		v.Classes = append(v.Classes, classCardOf(class, sheet.Class))
	}
	v.Origins = originOptions(sheet.Origin)
	if class := classByName(sheet.Class); class != nil {
		v.Gear = startingGearFor(sheet, *class)
	}
	v.OrphanRefusals = orphanRefusals(errs, v.Gear)
	return v
}

// orphanRefusals junta as recusas dos campos que esta folha não desenha.
func orphanRefusals(errs wire.FieldErrorMap, gear *startingGear) []string {
	onScreen := map[string]bool{"name": true, "race": true, "class": true, "origin": true}
	if gear != nil {
		onScreen["weaponSimple"] = true
		onScreen["weaponMartial"] = len(gear.MartialWeapons) > 0
		onScreen["armor"] = len(gear.Armors) > 0
		onScreen["shield"] = gear.Shield != ""
	}
	var orphans []string
	for field, messages := range errs {
		if onScreen[field] {
			continue
		}
		orphans = append(orphans, messages...)
	}
	sort.Strings(orphans)
	return orphans
}

func raceCardOf(race book.Race, chosen string) raceCard {
	names := make([]string, 0, 2)
	for _, ability := range race.Abilities {
		if len(names) == 2 {
			break
		}
		names = append(names, ability.Name)
	}
	return raceCard{
		Name: race.Name, Attributes: race.AttributeMod.Escrito(),
		Size: race.Size, Displacement: race.Speed,
		Abilities: strings.Join(names, ", "), Chosen: race.Name == chosen,
	}
}

func classCardOf(class book.Class, chosen string) classCard {
	pv, pm, _ := engine.ClassStartingVitals(class.Name)
	return classCard{
		Name: class.Name, PV: pv, PM: pm,
		Expertises: cardExpertisesLine(class), Chosen: class.Name == chosen,
	}
}

// cardExpertisesLine escreve a linha "Perícias" do bloco da classe: as que vêm
// treinadas de saída e quantas ainda se escolhem.
//
//	"Fortitude · mais 2 a escolher"
func cardExpertisesLine(class book.Class) string {
	fixed := strings.Join(class.Expertises, ", ")
	if class.Chooses == 0 {
		return fixed
	}
	choice := fmt.Sprintf("mais %d a escolher", class.Chooses)
	if fixed == "" {
		return choice
	}
	return fixed + " · " + choice
}

// originOptions são as 35 origens com uma linha do que elas dão.
func originOptions(chosen string) []originOption {
	origins := book.Origins()
	names := make([]string, 0, len(origins))
	for name := range origins {
		names = append(names, name)
	}
	book.SortByName(names, func(n string) string { return n })

	list := make([]originOption, 0, len(names))
	for _, name := range names {
		list = append(list, originOption{
			Name: name, Benefit: benefitsLine(origins[name]),
			Chosen: name == chosen,
		})
	}
	return list
}

// benefitsLine resume a lista de benefícios da origem. São dois a
// escolher (p85), e a escolha é da ficha — aqui é só o que a origem oferece.
func benefitsLine(origin book.Origin) string {
	names := make([]string, 0, len(origin.Benefits))
	for _, benefit := range origin.Benefits {
		names = append(names, benefit.Name)
	}
	return strings.Join(names, ", ")
}

// startingGearFor monta os seletores do kit desta classe.
func startingGearFor(sheet forgeAnswers, class book.Class) *startingGear {
	kit := engine.StartingKitFor(class.Name, class.Proficiencies)
	eq := &startingGear{
		SimpleWeapons: itemOptionsInCategory("weapon-simple", sheet.SimpleWeapon),
		Armors:        itemOptionsByID(kit.Armors, sheet.Armor),
		ToResolve:     grantsToResolve(sheet.Origin),
	}
	if kit.MartialWeapon {
		eq.MartialWeapons = itemOptionsInCategory("weapon-martial", sheet.MartialWeapon)
	}
	if item := book.ItemByID(kit.Shield); item != nil {
		eq.Shield, eq.ShieldChosen = item.Name, sheet.Shield
	}
	return eq
}

// itemOptionsInCategory são todos os itens de uma categoria do catálogo, na ordem
// em que o acervo já os guarda.
func itemOptionsInCategory(category, chosen string) []itemOption {
	var options []itemOption
	for _, item := range book.Catalogs().Items {
		if item.Category == category {
			options = append(options, itemOptionOf(item, chosen))
		}
	}
	return options
}

// itemOptionsByID são os itens que o kit nomeia, na ordem do kit — as armaduras
// leves antes da brunea, como o livro as escreve.
func itemOptionsByID(ids []string, chosen string) []itemOption {
	options := make([]itemOption, 0, len(ids))
	for _, id := range ids {
		if item := book.ItemByID(id); item != nil {
			options = append(options, itemOptionOf(*item, chosen))
		}
	}
	return options
}

func itemOptionOf(item book.Item, chosen string) itemOption {
	return itemOption{ID: item.ID, Label: item.Name, Chosen: item.ID == chosen}
}

// grantsToResolve são as linhas "Itens" da origem que pedem uma decisão —
// "Estojo de disfarces OU gazua", "Arma marcial", "Um item estrangeiro (até T$
// 100)". Elas não nascem na mochila; a folha as anuncia e a Mochila as resolve.
func grantsToResolve(origin string) []string {
	var labels []string
	for _, grant := range originGrants(origin) {
		switch grant.Kind {
		case engine.OriginItemFixed, engine.OriginItemMoney:
			continue
		default:
			labels = append(labels, grant.Label)
		}
	}
	return labels
}
