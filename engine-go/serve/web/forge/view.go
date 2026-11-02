package forge

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
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
	Name  string
	Races []raceCard
	// Classes são as cartas da folha de escolher, sem as que o herói já tem.
	Classes []classCard
	// ClassLines são as classes do herói, com o nível de cada uma.
	ClassLines []classLine
	TotalLevel int64
	// LevelCapped é o 20º nível: os `+` se apagam e não há outra classe a oferecer (p35).
	LevelCapped bool
	Origins     []originOption
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
	// O ID é o que o rádio SUBMETE e o Name é o que a pessoa lê. Eram o mesmo
	// campo até a ALE-404, e por isso renomear a raça na tela renomeava a
	// chave que a ficha guarda.
	ID           string
	Name         string
	Attributes   string
	Size         string
	Displacement int
	Abilities    string
	Chosen       bool
	// Find é o texto que a busca da folha casa, já dobrado (`search.Fold`).
	Find string
}

// classCard é uma das 14 cartas do ofício.
type classCard struct {
	Name       string
	PV         int
	PM         int
	Expertises string
	Find       string
}

// classLine é uma classe que o herói já tem.
type classLine struct {
	Name  string
	Meta  string
	Level int64
}

// originOption é uma linha da lista de origens.
type originOption struct {
	ID      string
	Name    string
	Benefit string
	Chosen  bool
	Find    string
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
	ID    string
	Label string
	// Meta é a linha do que o item é: o dano e o crítico da arma, a Defesa e a
	// penalidade da armadura.
	Meta   string
	Chosen bool
	Find   string
}

// blankForgeSheet monta a cena a partir das respostas que já existem.
//
// Ela é chamada nas três situações e devolve a mesma coisa nas três: a folha
// vazia do primeiro GET, a folha redesenhada quando uma escolha muda, e a folha
// recusada com os erros por campo.
func blankForgeSheet(answers forgeAnswers, errs wire.FieldErrorMap) forgeView {
	races, classes, _ := book.CharacterCatalogs()
	v := forgeView{Name: answers.Name, Errors: errs}
	for _, race := range races {
		v.Races = append(v.Races, raceCardOf(race, answers.Race))
	}
	for _, class := range classes {
		if !slices.ContainsFunc(answers.Classes, func(c sheet.ClassEntry) bool { return c.ClassName == class.Name }) {
			v.Classes = append(v.Classes, classCardOf(class))
		}
	}
	v.ClassLines = classLinesOf(answers.Classes)
	v.TotalLevel = totalLevelOf(answers.Classes)
	v.LevelCapped = v.TotalLevel >= heroLevelMax
	v.Origins = originOptions(answers.Origin)
	if class := answers.firstClass(); class != nil {
		v.Gear = startingGearFor(answers, *class)
	}
	v.OrphanRefusals = orphanRefusals(errs, v.Gear)
	return v
}

// classLinesOf escreve o que cada classe dá: a primeira, o que ela dá por inteiro; as outras, o que somam
// por nível, porque "não ganha as perícias treinadas ou proficiências da nova classe" (p35).
func classLinesOf(classes []sheet.ClassEntry) []classLine {
	lines := make([]classLine, 0, len(classes))
	for i, entry := range classes {
		class := classByName(entry.ClassName)
		if class == nil {
			continue
		}
		line := classLine{Name: class.Name, Level: entry.Level}
		if i == 0 {
			pv, pm, _ := engine.ClassStartingVitals(class.Name)
			line.Meta = fmt.Sprintf("1ª classe · PV %d · PM %d · %s", pv, pm, cardExpertisesLine(*class))
		} else {
			pv, pm, _ := engine.ClassLevelGains(class.Name)
			line.Meta = fmt.Sprintf("+%d PV e +%d PM por nível · sem perícias nem proficiências", pv, pm)
		}
		lines = append(lines, line)
	}
	return lines
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
	card := raceCard{
		ID: race.ID, Name: race.Name, Attributes: race.AttributeMod.Escrito(),
		Size: race.Size, Displacement: race.Speed,
		Abilities: strings.Join(names, ", "), Chosen: race.ID == chosen,
	}
	card.Find = search.Fold(card.Name + " " + card.Attributes + " " + card.Abilities)
	return card
}

func classCardOf(class book.Class) classCard {
	pv, pm, _ := engine.ClassStartingVitals(class.Name)
	card := classCard{Name: class.Name, PV: pv, PM: pm, Expertises: cardExpertisesLine(class)}
	card.Find = search.Fold(card.Name + " " + card.Expertises)
	return card
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
	ids := make([]string, 0, len(origins))
	for id := range origins {
		ids = append(ids, id)
	}
	// Ordenado pelo RÓTULO, que é o que a pessoa lê na lista — o slug põe
	// "Herói Camponês" entre "Guarda" e "Herdeiro" no mesmo lugar, mas
	// `Assistente de Laboratório` e `assistente-de-laboratorio` não ordenam
	// igual em toda origem.
	book.SortByName(ids, func(id string) string { return origins[id].Name })

	list := make([]originOption, 0, len(ids))
	for _, id := range ids {
		benefit := benefitsLine(origins[id])
		list = append(list, originOption{
			ID: id, Name: origins[id].Name, Benefit: benefit,
			Chosen: id == chosen, Find: search.Fold(origins[id].Name + " " + benefit),
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
func startingGearFor(answers forgeAnswers, class book.Class) *startingGear {
	kit := engine.StartingKitFor(class.Name, class.Proficiencies)
	eq := &startingGear{
		SimpleWeapons: itemOptionsInCategory("weapon-simple", answers.SimpleWeapon),
		Armors:        itemOptionsByID(kit.Armors, answers.Armor),
		ToResolve:     grantsToResolve(answers.Origin),
	}
	if kit.MartialWeapon {
		eq.MartialWeapons = itemOptionsInCategory("weapon-martial", answers.MartialWeapon)
	}
	if item := book.ItemByID(kit.Shield); item != nil {
		eq.Shield, eq.ShieldChosen = item.Name, answers.Shield
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
	option := itemOption{ID: item.ID, Label: item.Name, Meta: itemMeta(item), Chosen: item.ID == chosen}
	option.Find = search.Fold(option.Label + " " + option.Meta)
	return option
}

// itemMeta escreve o que a opção do equipamento é, na grafia da ficha:
// "1d8 · 19/×2" para a arma, "Defesa +2 · penalidade −1" para a armadura.
func itemMeta(item book.Item) string {
	switch {
	case item.Weapon != nil:
		crit := fmt.Sprintf("×%d", item.Weapon.CritMult)
		if item.Weapon.CritRange > 0 && item.Weapon.CritRange < 20 {
			crit = fmt.Sprintf("%d/%s", item.Weapon.CritRange, crit)
		}
		return item.Weapon.Damage + " · " + crit
	case item.Armor != nil:
		// A penalidade já vem NEGATIVA do catálogo.
		if item.Armor.Penalty == 0 {
			return "Defesa " + book.WithSign(item.Armor.Defense) + " · sem penalidade"
		}
		return "Defesa " + book.WithSign(item.Armor.Defense) + " · penalidade " + book.WithSign(item.Armor.Penalty)
	}
	return ""
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
