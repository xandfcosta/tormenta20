package sheetui

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// O DIÁLOGO DE ESCOLHER PODERES como dado.
//
// # Por que é um diálogo, e não meia tela
//
// Como MODO da aba Poderes ele abria sozinho sempre que havia pendência — o
// estado normal de quem acabou de subir de nível —, e o cromo dele comia quase
// metade do painel no telefone. Escolher poder acontece uma vez por nível; virar
// meia tela para isso é caro demais.
//
// As três abas de FONTE ficam aqui dentro, e aqui elas fazem sentido: no preparo
// a pergunta é "de onde vem o que eu ainda posso escolher". Na mesa a pergunta é
// outra, e por isso a lista de trás não tem abas.

// choicesPanel é o diálogo inteiro.
type choicesPanel struct {
	Pending []pendencia
	Races   []raceChoiceCard
	Origin  *originChoiceCard
	Classes []classChoiceCard
}

// raceChoiceCard é o que uma raça pede: o bônus de atributo e as variantes.
type raceChoiceCard struct {
	Race string
	// Attribute existe quando a raça distribui bônus — o `+1 ×3` do humano — ou
	// escolhe ascendência, como o suraggel.
	Attribute *attributeChoice
	Variants  []variantChoice
}

// attributeChoice é o bônus de atributo que a raça deixa escolher.
type attributeChoice struct {
	// Kind é `floating` (N atributos distintos) ou `ascendencia`.
	Kind string
	// Count são quantos atributos se escolhe, no `floating`.
	Count int
	Value int
	// Exclude é o atributo PROIBIDO, quando a raça tem um.
	Exclude  string
	Options  []filterOption
	Chosen   []string
	Complete bool
}

// variantChoice é uma habilidade de raça com opções — a Resistência Elemental
// do qareen, a Herança Divina do suraggel.
type variantChoice struct {
	AbilityID string
	Name      string
	Options   []filterOption
	Chosen    string
}

// originChoiceCard são os benefícios da origem, com o teto de dois.
type originChoiceCard struct {
	Name    string
	Options []filterOption
	Chosen  []string
	Left    int
}

// classChoiceCard é o que uma classe pede: as vagas de poder, o caminho e o
// devoto.
type classChoiceCard struct {
	ClassName string
	Level     int64
	// Slots são as vagas do nível e quantas já foram usadas.
	Slots int
	Used  int
	// Powers são os poderes ELETIVOS da classe mais os gerais, já marcados.
	Powers  []powerChoice
	Path    *pickerChoice
	Devotee *pickerChoice
}

// powerChoice é um poder que dá para escolher.
type powerChoice struct {
	ID     string
	Name   string
	Detail string
	// Source é "Classe" ou "Geral", para a lista dizer de onde o poder vem.
	Source string
	Chosen bool
}

// pickerChoice é uma escolha de valor único — caminho ou devoto.
type pickerChoice struct {
	Options []filterOption
	Chosen  string
}

// choicesPanelOf monta o diálogo.
func (s Scene) choicesPanelOf(dto sheet.CharacterDTO, search string) choicesPanel {
	panel := choicesPanel{Pending: s.sheetPendings(dto)}
	panel.Races = s.raceChoices(dto)
	panel.Origin = originChoice(dto)
	panel.Classes = classChoiceCards(dto, search)
	return panel
}

func (s Scene) raceChoices(dto sheet.CharacterDTO) []raceChoiceCard {
	cards := []raceChoiceCard{}
	for _, r := range dto.Races {
		card := raceChoiceCard{Race: r.Race, Variants: raceVariants(dto, r.Race)}
		card.Attribute = s.attributeRaceBonus(dto, r.Race)
		if card.Attribute == nil && len(card.Variants) == 0 {
			continue
		}
		cards = append(cards, card)
	}
	return cards
}

// attributeRaceBonus descreve a escolha de atributo, ou nil quando a raça
// não pede nenhuma (as doze de bônus fixo).
func (s Scene) attributeRaceBonus(dto sheet.CharacterDTO, name string) *attributeChoice {
	mod := attributeRaceMod(name)
	if mod == nil || mod.Kind == "fixed" {
		return nil
	}
	choice := &attributeChoice{
		Kind: mod.Kind, Count: mod.Count, Value: mod.Value, Exclude: mod.Exclude,
		Chosen: attributeSavedChoices(dto.RaceAttributeChoices),
	}
	if s.deps.Catalogs() != nil {
		choice.Complete = s.deps.Catalogs().RaceAttributeChoiceIsComplete(name, dto.RaceAttributeChoices)
	}
	if mod.Kind == "floating" {
		choice.Options = thatFitAttributes(mod.Exclude)
		return choice
	}
	choice.Kind = "ascendencia"
	choice.Options = raceAncestries(name)
	if a := savedAncestry(dto.RaceAttributeChoices); a != "" {
		choice.Chosen = []string{a}
	}
	return choice
}

// thatFitAttributes são os seis do livro, menos o proibido da raça.
func thatFitAttributes(forbidden string) []filterOption {
	outside := []filterOption{}
	for _, a := range book.AttributeOrder {
		if a.Key == forbidden {
			continue
		}
		outside = append(outside, filterOption{Value: a.Key, Label: a.Abbreviation})
	}
	return outside
}

// raceVariants são as habilidades de raça que pedem uma escolha.
func raceVariants(dto sheet.CharacterDTO, name string) []variantChoice {
	race := withVariantsRace(name)
	if race == nil {
		return nil
	}
	chosen := sheet.UnmarshalStrings(dto.RaceAbilityChoices)
	outside := []variantChoice{}
	for _, ability := range race.Abilities {
		if len(ability.Variants) == 0 {
			continue
		}
		choice := variantChoice{AbilityID: ability.ID, Name: ability.Name}
		for _, v := range ability.Variants {
			active := contemTraco(chosen, v.ID)
			if active {
				choice.Chosen = v.ID
			}
			choice.Options = append(choice.Options,
				filterOption{Value: v.ID, Label: v.Name, Active: active})
		}
		outside = append(outside, choice)
	}
	return outside
}

// originChoice monta o cartão da origem.
func originChoice(dto sheet.CharacterDTO) *originChoiceCard {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	chosen := sheet.UnmarshalStrings(dto.OriginChoices)
	card := &originChoiceCard{
		Name: origin.Name, Chosen: chosen,
		Left: sheet.BenefitsOriginLimit - len(chosen),
	}
	for _, b := range sheet.OriginBenefitsOf(origin) {
		card.Options = append(card.Options, filterOption{
			Value: b.ID, Label: b.Name, Active: contemTraco(chosen, b.ID),
		})
	}
	return card
}

// classChoiceCards monta um cartão por classe da ficha.
func classChoiceCards(dto sheet.CharacterDTO, search string) []classChoiceCard {
	chosen := sheet.UnmarshalStrings(dto.ClassPowers)
	choices := sheet.ClassChoiceSelections(dto)
	cards := []classChoiceCard{}
	for _, class := range dto.Classes {
		card := classChoiceCard{
			ClassName: class.ClassName, Level: class.Level,
			Slots:  sheet.PowerSlots(class.Level),
			Used:   len(chosen),
			Powers: thatChoosePowers(class.ClassName, chosen, search),
		}
		if options := sheet.LevelPaths(class.ClassName, class.Level); len(options) > 0 {
			card.Path = markedPicker(options, choices[class.ClassName].Path)
		}
		if options := sheet.ClassDevotees(class.ClassName); len(options) > 0 {
			card.Devotee = markedPicker(options, choices[class.ClassName].Devotee)
		}
		cards = append(cards, card)
	}
	return cards
}

// markedPicker vira as opções do LIVRO nas opções da TELA, marcando a escolhida.
//
// A conversão existe porque as duas listas respondem perguntas diferentes: o
// `sheet.ChoiceOption` é o que o livro oferece (valor e nome), e o
// `filterOption` é o que o `<select>` desenha — com o `Ativo` que só a tela tem.
func markedPicker(options []sheet.ChoiceOption, chosen string) *pickerChoice {
	marked := make([]filterOption, 0, len(options))
	for _, o := range options {
		marked = append(marked, filterOption{
			Value: o.Value, Label: o.Label, Active: o.Value == chosen,
		})
	}
	return &pickerChoice{Options: marked, Chosen: chosen}
}

// thatChoosePowers são os ELETIVOS da classe mais os gerais.
//
// "Você sempre pode substituir um poder de classe por um poder geral" (p33), e
// por isso as duas listas viram uma só — a vaga é a mesma.
func thatChoosePowers(class string, chosen []string, query string) []powerChoice {
	term := search.Fold(strings.TrimSpace(query))
	outside := []powerChoice{}
	for _, p := range book.ClassPowers() {
		if p.ClassName != class || p.GrantedAtLevel != nil || !casaComABusca(p.Name, term) {
			continue
		}
		outside = append(outside, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Source: "Classe",
			Chosen: contemTraco(chosen, p.ID),
		})
	}
	for _, p := range book.GeneralPowers() {
		if !casaComABusca(p.Name, term) {
			continue
		}
		outside = append(outside, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Source: powerGeneralSource(p),
			Chosen: contemTraco(chosen, p.ID),
		})
	}
	sort.SliceStable(outside, func(a, b int) bool {
		if outside[a].Chosen != outside[b].Chosen {
			return outside[a].Chosen
		}
		return outside[a].Name < outside[b].Name
	})
	return outside
}

func casaComABusca(name, term string) bool {
	return term == "" || strings.Contains(search.Fold(name), term)
}

// ── o que a TELA escreve ─────────────────────────────────────────────────────

// classWrittenSlots é "3 de 4 vagas".
func classWrittenSlots(card classChoiceCard) string {
	return strconv.Itoa(card.Used) + " de " + strconv.Itoa(card.Slots) + " vagas"
}

// attributeWrittenChoice descreve o que a raça pede.
func attributeWrittenChoice(choice attributeChoice) string {
	if choice.Kind == "ascendencia" {
		return "Escolha a ascendência"
	}
	text := "Distribua +" + strconv.Itoa(choice.Value) + " em " +
		strconv.Itoa(choice.Count) + " atributos diferentes"
	if choice.Exclude != "" {
		return text + " (exceto " + book.AttributeAbbrev(choice.Exclude) + ")"
	}
	return text
}

// sourceThree são as abas do diálogo, na ordem em que o livro monta um
// personagem: raça, origem, classe.
var sourceThree = []string{"raca", "origem", "classe"}

// firstPendingSource é a aba em que o diálogo ABRE.
//
// Quem abriu veio pela pendência, e fazê-lo caçar a aba certa é gastar o clique
// que ele acabou de dar. Sem pendência nenhuma, ele abre na Raça.
func firstPendingSource(v View) string {
	if len(v.Choices.Pending) > 0 {
		return v.Choices.Pending[0].Source
	}
	return "raca"
}

// choiceChip é a classe de um chip que liga e desliga.
func choiceChip(active bool) string {
	base := "rounded-full border px-2 py-0.5 text-3xs uppercase tracking-wider outline-none transition-colors"
	if active {
		return base + " border-grimorio-gold/60 bg-accent text-grimorio-gold"
	}
	return base + " border-grimorio-iron text-muted-foreground hover:text-foreground"
}

// choiceClassCommand escreve o `@post` do caminho ou do devoto.
func choiceClassCommand(v View, class, choice, value string) string {
	return sheetPost(v, "/poderes/classe/"+url.PathEscape(class)+"/"+choice+"/"+value)
}

// thatTogglesAttributeGesture liga ou desliga um atributo na distribuição.
//
// A lista é COPIADA antes de ser mexida: o sinal é um proxy, e escrever dentro
// dele item a item é a armadilha que o guia do Go registra.
func thatTogglesAttributeGesture(attribute string) string {
	return "const escolhidos = [...$race_attributes]; const onde = escolhidos.indexOf('" + attribute + "'); " +
		"if (onde >= 0) { escolhidos.splice(onde, 1) } else { escolhidos.push('" + attribute + "') }; " +
		"$race_attributes = escolhidos"
}
