package sheetui

import (
	"net/url"
	"slices"
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
	// Devotion é o passo da devoção, que toda ficha tem.
	Devotion devotionChoice
	// Training é o passo das perícias da primeira classe, ou nil sem classe.
	Training *sheet.ClassTraining
}

// devotionChoice é o deus da ficha e os poderes que ele concede.
type devotionChoice struct {
	Name     string
	Required bool
	Gods     pickerChoice
	// Powers são os poderes do deus escolhido, e NoGrant diz quem escolheu uma devoção sem poder.
	Powers  []grantedPowerRow
	NoGrant string
	// Limit é quantos poderes o deus dá a esta ficha, e Held quantos dele foram escolhidos.
	Limit, Held int
	// OldPowers são os guardados que não são do deus de agora: só uma ficha de antes de a troca tirá-los
	// chega assim.
	OldPowers []string
	// Changes são os avisos de troca, um por deus que se pode escolher. Sem deus não há o que avisar.
	Changes []devotionChangeNotice
}

// Written é a ponta da lista dos concedidos: "1 de 2", ou "1 de 1 · +1 em vaga" quando há mais marcados do
// que a devoção dá, e o que passa ocupa vaga de poder da classe (p124). A ponta é curta porque divide a
// linha com o título a 390px; o rodapé (`SlotHint`) diz que vaga é essa.
func (d devotionChoice) Written() string {
	if d.Held <= d.Limit {
		return strconv.Itoa(d.Held) + " de " + strconv.Itoa(d.Limit)
	}
	limit := strconv.Itoa(d.Limit)
	return limit + " de " + limit + " · +" + strconv.Itoa(d.Held-d.Limit) + " em vaga"
}

// SlotHint é o rodapé da lista dos concedidos: quantos a devoção dá, e o que custa um a mais.
func (d devotionChoice) SlotHint() string {
	given := "A devoção dá 1"
	if d.Limit != 1 {
		given = "A devoção dá " + strconv.Itoa(d.Limit)
	}
	return given + ". Um a mais é um poder geral e ocupa uma vaga de poder da classe (p124)."
}

// devotionChangeNotice é o que muda na ficha ao trocar para Target. O Target `nenhum` é ficar sem deus.
type devotionChangeNotice struct {
	Target string
	Lines  []string
}

type grantedPowerRow struct {
	Name, Detail string
	Active       bool
}

// raceChoiceCard é o que uma raça pede: o bônus de atributo e as variantes.
type raceChoiceCard struct {
	// O RÓTULO, que é o que o `<h4>` do diálogo mostra. A ficha guarda o slug
	// desde a ALE-404; quem precisa da chave lê `r.Race`.
	Race string
	// Attribute existe quando a raça distribui bônus — o `+1 ×3` do humano — ou
	// escolhe ascendência, como o suraggel.
	Attribute *attributeChoice
	Variants  []variantChoice
	// Abilities são as habilidades com escolha — a Deformidade, o
	// Vanguardista (ALE-493).
	Abilities []raceAbilityBlock
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
// do qareen.
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
	// Ceiling é quantos benefícios a origem cobra (`originBenefitCeiling`).
	Ceiling int
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
	Powers []powerChoice
	Path   *pickerChoice
	// Counted são as escolhas contadas por atributo dos poderes que a classe
	// CONCEDEU — as perícias do Especialista (ALE-501).
	Counted []countedChoice
}

// powerChoice é um poder que dá para escolher.
type powerChoice struct {
	ID     string
	Name   string
	Detail string
	// Requires é o pré-requisito por extenso. Ele informa e não trava: quem decide é a mesa.
	Requires string
	// Source é "Classe" ou "Geral", para a lista dizer de onde o poder vem.
	Source string
	Chosen bool
	// Options são as escolhas que o poder pede, com quantas vezes cada uma já
	// foi feita (ALE-491). Vazio no poder sem escolha, que segue como
	// interruptor.
	Options []powerOption
}

// powerOption é uma opção da escolha de um poder — "Força" no Aumento.
type powerOption struct {
	Value string
	Label string
	Count int
}

// Written é o rótulo da ficha: "Força", ou "Força ×2" quando o poder repetível
// a tomou duas vezes.
func (o powerOption) Written() string {
	if o.Count > 1 {
		return o.Label + " ×" + strconv.Itoa(o.Count)
	}
	return o.Label
}

// pickerChoice é uma escolha de valor único.
type pickerChoice struct {
	Options []filterOption
	Chosen  string
}

// choicesPanelOf monta o diálogo.
func (s Scene) choicesPanelOf(dto sheet.CharacterDTO, search string) choicesPanel {
	panel := choicesPanel{Pending: s.sheetPendings(dto)}
	panel.Races = s.raceChoices(dto, search)
	panel.Origin = originChoice(dto)
	panel.Classes = classChoiceCards(dto, search)
	panel.Devotion = devotionChoiceOf(dto)
	return panel
}

func (s Scene) raceChoices(dto sheet.CharacterDTO, query string) []raceChoiceCard {
	cards := []raceChoiceCard{}
	for _, r := range dto.Races {
		card := raceChoiceCard{
			Race: book.RaceLabel(r.Race), Variants: raceVariants(dto, r.Race),
			Abilities: raceAbilityBlocksOf(dto, r.Race, query),
		}
		card.Attribute = s.attributeRaceBonus(dto, r.Race)
		if card.Attribute == nil && len(card.Variants) == 0 && len(card.Abilities) == 0 {
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
	ceiling := originBenefitCeiling(origin)
	card := &originChoiceCard{
		Name: origin.Name, Chosen: chosen,
		Ceiling: ceiling, Left: ceiling - len(chosen),
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
			Slots:   sheet.PowerSlots(class.Level),
			Used:    sheet.UsedPowerSlots(dto),
			Powers:  thatChoosePowers(class.ClassName, chosen, sheet.PowerPicks(dto.PowerChoices), ownedOptionsOf(dto), search),
			Counted: countedChoicesOf(dto, class.ClassName),
		}
		if options := sheet.LevelPaths(class.ClassName, class.Level); len(options) > 0 {
			card.Path = markedPicker(options, choices[class.ClassName].Path)
		}
		cards = append(cards, card)
	}
	return cards
}

// devotionChoiceOf monta o passo da devoção.
func devotionChoiceOf(dto sheet.CharacterDTO) devotionChoice {
	god := ""
	if dto.God != nil {
		god = *dto.God
	}
	options := sheet.DevotionOptions(dto.Classes)
	choice := devotionChoice{
		Name: "nenhuma", Required: sheet.DevotionRequired(dto.Classes),
		Gods: *markedPicker(options, god), Limit: sheet.GrantedPowerLimit(dto, god),
	}
	held := sheet.UnmarshalStrings(dto.GodPowers)
	offered := sheet.GrantedPowersFor(dto, god)
	ofTheGod := sheet.GrantedPowerNames(dto, god)
	for _, power := range held {
		if god != "" && slices.Contains(ofTheGod, power) {
			choice.Held++
		} else {
			choice.OldPowers = append(choice.OldPowers, power)
		}
	}
	if god == "" {
		return choice
	}
	choice.Name = book.DevotionName(god)
	for _, power := range offered {
		choice.Powers = append(choice.Powers, grantedPowerRow{
			Name: power.Name, Detail: power.Description, Active: slices.Contains(held, power.Name),
		})
	}
	if len(choice.Powers) == 0 {
		choice.NoGrant = book.DevotionWritten(god) + " não recebe poder concedido."
	}
	choice.Changes = append(choice.Changes, devotionChangeNotice{Target: noDevotion, Lines: sheet.DevotionChange(dto, "")})
	for _, option := range options {
		if option.Value != god {
			choice.Changes = append(choice.Changes, devotionChangeNotice{Target: option.Value, Lines: sheet.DevotionChange(dto, option.Value)})
		}
	}
	return choice
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
func thatChoosePowers(
	class string, chosen []string, picks map[string][]string, owned map[string]bool, query string,
) []powerChoice {
	term := search.Fold(strings.TrimSpace(query))
	outside := []powerChoice{}
	for _, p := range book.ClassPowers() {
		if p.ClassName != class || p.GrantedAtLevel != nil || !casaComABusca(p.Name, term) {
			continue
		}
		outside = append(outside, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Requires: book.PowerRequirements(p.ID), Source: "Classe",
			Chosen: contemTraco(chosen, p.ID), Options: optionsOf(p, picks[p.ID], owned),
		})
	}
	for _, p := range book.GeneralPowers() {
		if !casaComABusca(p.Name, term) {
			continue
		}
		choice := powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Requires: book.PowerRequirements(p.ID), Source: powerGeneralSource(p),
			Chosen: contemTraco(chosen, p.ID) || len(picks[p.ID]) > 0,
		}
		// O poder geral que pede escolha (Foco em Perícia, Foco em Arma) se
		// escolhe pela opção, como o de classe (ALE-510).
		if withChoice, chooses := book.PowerWithChoice(p.ID); chooses {
			choice.Options = optionsOf(withChoice, picks[p.ID], owned)
		}
		outside = append(outside, choice)
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

// choiceClassCommand escreve o `@post` do caminho.
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
