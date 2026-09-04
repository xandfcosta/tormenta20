package sheetui

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"t20engine/book"
	"t20engine/sheet"
)

// O DIÁLOGO DE ESCOLHER PODERES como dado (ALE-272, fatia 8).
//
// # Por que é um diálogo, e não meia tela
//
// Ele era um MODO da aba Poderes, e o modo é o que a tornava difícil de usar
// (ALE-217): ele ABRIA sozinho sempre que havia pendência — o estado normal de
// quem acabou de subir de nível —, e o cromo dele comia 44% do painel no
// telefone. Escolher poder acontece uma vez por nível; virar meia tela para isso
// é caro demais.
//
// As três abas de FONTE ficam aqui dentro, e aqui elas fazem sentido: no preparo
// a pergunta é "de onde vem o que eu ainda posso escolher". Na mesa a pergunta é
// outra, e por isso a lista de trás não tem abas.

// choicesPanel é o diálogo inteiro.
type choicesPanel struct {
	Pendencias []pendencia
	Races      []raceChoiceCard
	Origin     *originChoiceCard
	Classes    []classChoiceCard
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
	Caminho *pickerChoice
	Devoto  *pickerChoice
}

// powerChoice é um poder que dá para escolher.
type powerChoice struct {
	ID     string
	Name   string
	Detail string
	// Fonte é "Classe" ou "Geral", para a lista dizer de onde o poder vem.
	Fonte  string
	Chosen bool
}

// pickerChoice é uma escolha de valor único — caminho ou devoto.
type pickerChoice struct {
	Options []filterOption
	Chosen  string
}

// choicesPanelOf monta o diálogo.
func (s Scene) choicesPanelOf(dto sheet.CharacterDTO, busca string) choicesPanel {
	panel := choicesPanel{Pendencias: s.sheetPendings(dto)}
	panel.Races = s.raceChoices(dto)
	panel.Origin = originChoice(dto)
	panel.Classes = classChoiceCards(dto, busca)
	return panel
}

func (s Scene) raceChoices(dto sheet.CharacterDTO) []raceChoiceCard {
	cartoes := []raceChoiceCard{}
	for _, r := range dto.Races {
		cartao := raceChoiceCard{Race: r.Race, Variants: raceVariants(dto, r.Race)}
		cartao.Attribute = s.attributeRaceBonus(dto, r.Race)
		if cartao.Attribute == nil && len(cartao.Variants) == 0 {
			continue
		}
		cartoes = append(cartoes, cartao)
	}
	return cartoes
}

// attributeRaceBonus descreve a escolha de atributo, ou nil quando a raça
// não pede nenhuma (as doze de bônus fixo).
func (s Scene) attributeRaceBonus(dto sheet.CharacterDTO, nome string) *attributeChoice {
	mod := attributeRaceMod(nome)
	if mod == nil || mod.Kind == "fixed" {
		return nil
	}
	escolha := &attributeChoice{
		Kind: mod.Kind, Count: mod.Count, Value: mod.Value, Exclude: mod.Exclude,
		Chosen: attributeSavedChoices(dto.RaceAttributeChoices),
	}
	if s.deps.Catalogs() != nil {
		escolha.Complete = s.deps.Catalogs().RaceAttributeChoiceIsComplete(nome, dto.RaceAttributeChoices)
	}
	if mod.Kind == "floating" {
		escolha.Options = thatFitAttributes(mod.Exclude)
		return escolha
	}
	escolha.Kind = "ascendencia"
	escolha.Options = raceAncestries(nome)
	if a := savedAncestry(dto.RaceAttributeChoices); a != "" {
		escolha.Chosen = []string{a}
	}
	return escolha
}

// thatFitAttributes são os seis do livro, menos o proibido da raça.
func thatFitAttributes(proibido string) []filterOption {
	fora := []filterOption{}
	for _, a := range book.AttributeOrder {
		if a.Chave == proibido {
			continue
		}
		fora = append(fora, filterOption{Valor: a.Chave, Rotulo: a.Sigla})
	}
	return fora
}

// raceVariants são as habilidades de raça que pedem uma escolha.
func raceVariants(dto sheet.CharacterDTO, nome string) []variantChoice {
	raca := withVariantsRace(nome)
	if raca == nil {
		return nil
	}
	escolhidas := sheet.UnmarshalStrings(dto.RaceAbilityChoices)
	fora := []variantChoice{}
	for _, hab := range raca.Abilities {
		if len(hab.Variants) == 0 {
			continue
		}
		escolha := variantChoice{AbilityID: hab.ID, Name: hab.Name}
		for _, v := range hab.Variants {
			ativa := contemTraco(escolhidas, v.ID)
			if ativa {
				escolha.Chosen = v.ID
			}
			escolha.Options = append(escolha.Options,
				filterOption{Valor: v.ID, Rotulo: v.Name, Ativo: ativa})
		}
		fora = append(fora, escolha)
	}
	return fora
}

// originChoice monta o cartão da origem.
func originChoice(dto sheet.CharacterDTO) *originChoiceCard {
	origem, tem := book.Origins()[dto.Origin]
	if !tem {
		return nil
	}
	escolhidos := sheet.UnmarshalStrings(dto.OriginChoices)
	cartao := &originChoiceCard{
		Name: origem.Name, Chosen: escolhidos,
		Left: sheet.BenefitsOriginLimit - len(escolhidos),
	}
	for _, b := range sheet.OriginBenefitsOf(origem) {
		cartao.Options = append(cartao.Options, filterOption{
			Valor: b.ID, Rotulo: b.Name, Ativo: contemTraco(escolhidos, b.ID),
		})
	}
	return cartao
}

// classChoiceCards monta um cartão por classe da ficha.
func classChoiceCards(dto sheet.CharacterDTO, busca string) []classChoiceCard {
	escolhidos := sheet.UnmarshalStrings(dto.ClassPowers)
	escolhas := sheet.ClassChoiceSelections(dto)
	cartoes := []classChoiceCard{}
	for _, classe := range dto.Classes {
		cartao := classChoiceCard{
			ClassName: classe.ClassName, Level: classe.Level,
			Slots:  sheet.PowerSlots(classe.Level),
			Used:   len(escolhidos),
			Powers: thatChoosePowers(classe.ClassName, escolhidos, busca),
		}
		if opcoes := sheet.LevelPaths(classe.ClassName, classe.Level); len(opcoes) > 0 {
			cartao.Caminho = markedPicker(opcoes, escolhas[classe.ClassName].Caminho)
		}
		if opcoes := sheet.ClassDevotees(classe.ClassName); len(opcoes) > 0 {
			cartao.Devoto = markedPicker(opcoes, escolhas[classe.ClassName].Devoto)
		}
		cartoes = append(cartoes, cartao)
	}
	return cartoes
}

// markedPicker vira as opções do LIVRO nas opções da TELA, marcando a escolhida.
//
// A conversão existe porque as duas listas respondem perguntas diferentes: o
// `sheet.ChoiceOption` é o que o livro oferece (valor e nome), e o
// `filterOption` é o que o `<select>` desenha — com o `Ativo` que só a tela tem.
func markedPicker(opcoes []sheet.ChoiceOption, escolhido string) *pickerChoice {
	marcadas := make([]filterOption, 0, len(opcoes))
	for _, o := range opcoes {
		marcadas = append(marcadas, filterOption{
			Valor: o.Value, Rotulo: o.Label, Ativo: o.Value == escolhido,
		})
	}
	return &pickerChoice{Options: marcadas, Chosen: escolhido}
}

// thatChoosePowers são os ELETIVOS da classe mais os gerais.
//
// "Você sempre pode substituir um poder de classe por um poder geral" (p33), e
// por isso as duas listas viram uma só — a vaga é a mesma.
func thatChoosePowers(classe string, escolhidos []string, busca string) []powerChoice {
	termo := foldAccents(strings.TrimSpace(busca))
	fora := []powerChoice{}
	for _, p := range book.ClassPowers() {
		if p.ClassName != classe || p.GrantedAtLevel != nil || !casaComABusca(p.Name, termo) {
			continue
		}
		fora = append(fora, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Fonte: "Classe",
			Chosen: contemTraco(escolhidos, p.ID),
		})
	}
	for _, p := range book.GeneralPowers() {
		if !casaComABusca(p.Name, termo) {
			continue
		}
		fora = append(fora, powerChoice{
			ID: p.ID, Name: p.Name, Detail: p.Description, Fonte: powerGeneralSource(p),
			Chosen: contemTraco(escolhidos, p.ID),
		})
	}
	sort.SliceStable(fora, func(a, b int) bool {
		if fora[a].Chosen != fora[b].Chosen {
			return fora[a].Chosen
		}
		return fora[a].Name < fora[b].Name
	})
	return fora
}

func casaComABusca(nome, termo string) bool {
	return termo == "" || strings.Contains(foldAccents(nome), termo)
}

// ── o que a TELA escreve ─────────────────────────────────────────────────────

// classWrittenSlots é "3 de 4 vagas".
func classWrittenSlots(cartao classChoiceCard) string {
	return strconv.Itoa(cartao.Used) + " de " + strconv.Itoa(cartao.Slots) + " vagas"
}

// attributeWrittenChoice descreve o que a raça pede.
func attributeWrittenChoice(escolha attributeChoice) string {
	if escolha.Kind == "ascendencia" {
		return "Escolha a ascendência"
	}
	texto := "Distribua +" + strconv.Itoa(escolha.Value) + " em " +
		strconv.Itoa(escolha.Count) + " atributos diferentes"
	if escolha.Exclude != "" {
		return texto + " (exceto " + book.AttributeAbbrev(escolha.Exclude) + ")"
	}
	return texto
}

// sourceThree são as abas do diálogo, na ordem em que o livro monta um
// personagem: raça, origem, classe.
var sourceThree = []string{"raca", "origem", "classe"}

// firstPendingSource é a aba em que o diálogo ABRE.
//
// Quem abriu veio pela pendência, e fazê-lo caçar a aba certa é gastar o clique
// que ele acabou de dar. Sem pendência nenhuma, ele abre na Raça.
func firstPendingSource(v View) string {
	if len(v.Choices.Pendencias) > 0 {
		return v.Choices.Pendencias[0].Fonte
	}
	return "raca"
}

// choiceChip é a classe de um chip que liga e desliga.
func choiceChip(ativo bool) string {
	base := "rounded-full border px-2 py-0.5 text-3xs uppercase tracking-wider outline-none transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
	if ativo {
		return base + " border-grimorio-gold/60 bg-accent text-grimorio-gold"
	}
	return base + " border-grimorio-iron text-muted-foreground hover:text-foreground"
}

// choiceClassCommand escreve o `@post` do caminho ou do devoto.
func choiceClassCommand(v View, classe, escolha, valor string) string {
	return sheetPost(v, "/poderes/classe/"+url.PathEscape(classe)+"/"+escolha+"/"+valor)
}

// thatTogglesAttributeGesture liga ou desliga um atributo na distribuição.
//
// A lista é COPIADA antes de ser mexida: o sinal é um proxy, e escrever dentro
// dele item a item é a armadilha que o guia do Go registra.
func thatTogglesAttributeGesture(atributo string) string {
	return "const escolhidos = [...$racaatributos]; const onde = escolhidos.indexOf('" + atributo + "'); " +
		"if (onde >= 0) { escolhidos.splice(onde, 1) } else { escolhidos.push('" + atributo + "') }; " +
		"$racaatributos = escolhidos"
}
