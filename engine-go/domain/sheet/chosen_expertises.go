package sheet

import (
	"fmt"
	"slices"
	"sort"

	"t20engine/domain/book"
)

// AS PERÍCIAS DO ESPECIALISTA (p73, ALE-501): "Escolha um número de perícias
// treinadas igual a sua Inteligência, exceto bônus temporários (mínimo 1)."
//
// A escolha mora em `powerChoices` como as outras, e não gasta vaga. A ficha que
// sairia dela — Inteligência abaixo do que foi escolhido, perícia escolhida sem
// treino — é RECUSADA em todo gesto que a produziria, e não consertada sozinha:
// qual perícia sai é decisão de quem joga (decisão do dono).

// ChoiceLimit é quantas opções a escolha contada por atributo aceita: o
// atributo sem temporários, mínimo 1.
//
//	ChoiceLimit(dto, *book.ClassPowers()["class.ladino.especialista"].Choice) // 2
func ChoiceLimit(dto CharacterDTO, choice book.PowerChoice) (int, error) {
	value, err := countAttributeValue(dto, choice)
	return max(1, value), err
}

// countAttributeValue é o atributo que conta a escolha, sem temporários.
func countAttributeValue(dto CharacterDTO, choice book.PowerChoice) (int, error) {
	if dto.Ruleset == nil {
		return 0, fmt.Errorf("a ficha %d chegou sem o mundo dela; quem a montou não passou pelo `sheet.Load`", dto.ID)
	}
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return 0, err
	}
	return dto.Ruleset.PermanentAttribute(ec, choice.CountAttribute), nil
}

// ChosenExpertisesFit confere as escolhas que nomeiam perícias contra a ficha
// como ela está: as contadas por atributo cabem no limite e são treinadas, as
// do Foco em Perícia são treinadas (ALE-510), e as
// das habilidades de raça apontam perícia e Ofício que existem (ALE-493).
func ChosenExpertisesFit(dto CharacterDTO) error {
	picks := PowerPicks(dto.PowerChoices)
	ids := make([]string, 0, len(picks))
	for id := range picks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		power, known := book.PowerWithChoice(id)
		if !known {
			continue
		}
		if power.Choice.CountAttribute != "" {
			if err := expertisePicksFit(dto, power, picks[id]); err != nil {
				return err
			}
			continue
		}
		if power.Choice.Kind == "trained-expertise" {
			if err := trainedPicksFit(dto, power, picks[id]); err != nil {
				return err
			}
		}
	}
	// As habilidades de raça também nomeiam perícias: o Ofício do
	// Vanguardista, as da Deformidade (ALE-493).
	return raceAbilityPicksFit(dto)
}

func expertisePicksFit(dto CharacterDTO, power book.ClassPower, picks []string) error {
	for i, pick := range picks {
		if slices.Contains(picks[:i], pick) {
			return fmt.Errorf("%s escolheu %s duas vezes", power.Name, pick)
		}
		if !trainedIn(dto, pick) {
			return fmt.Errorf("%s escolheu %s, e ela não seria perícia treinada desta ficha: tire-a do %s antes",
				power.Name, pick, power.Name)
		}
	}
	limit, err := ChoiceLimit(dto, *power.Choice)
	if err != nil {
		return err
	}
	if len(picks) > limit {
		return fmt.Errorf("%s escolhe %s (%s sem bônus temporários, mínimo 1), e ficariam %d: tire %d do %s antes",
			power.Name, writtenSkills(limit), book.AttributeLabel(power.Choice.CountAttribute), len(picks), len(picks)-limit, power.Name)
	}
	return nil
}

// trainedPicksFit confere a escolha de perícia que se repete: cada vez é uma
// perícia diferente, e treinada — o pré-requisito do Foco em Perícia (p130).
func trainedPicksFit(dto CharacterDTO, power book.ClassPower, picks []string) error {
	for i, pick := range picks {
		if slices.Contains(picks[:i], pick) {
			return fmt.Errorf("%s já escolheu %s; cada vez é uma perícia diferente", power.Name, pick)
		}
		if !trainedIn(dto, pick) {
			return fmt.Errorf("%s escolheu %s, e ela não seria treinada nesta ficha: tire-a do %s antes",
				power.Name, pick, power.Name)
		}
	}
	return nil
}

func writtenSkills(n int) string {
	if n == 1 {
		return "1 perícia"
	}
	return fmt.Sprintf("%d perícias", n)
}

func trainedIn(dto CharacterDTO, skill string) bool {
	return slices.ContainsFunc(dto.Expertises, func(e ExpertiseDTO) bool { return e.Name == skill && e.Trained })
}

// HasExpertiseChoice diz se o `powerChoices` guarda alguma escolha que depende
// do treino ou do atributo da ficha — a contada por atributo e a do Foco em
// Perícia. É o atalho que poupa carregar a ficha
// inteira nos gestos que só PODERIAM tirá-la da escolha.
func HasExpertiseChoice(blob string) bool {
	for id := range PowerPicks(blob) {
		power, known := book.PowerWithChoice(id)
		if !known {
			continue
		}
		kind := power.Choice.Kind
		if power.Choice.CountAttribute != "" || kind == "trained-expertise" {
			return true
		}
	}
	return false
}

// ChangeKeepsChosenExpertises aplica a mudança numa cópia da ficha e confere
// o resultado ANTES de gravar: destreinar, guardar um item de +Int, baixar o
// atributo. A cópia é da ficha recém-carregada, que o chamador descarta.
//
//	err := ChangeKeepsChosenExpertises(dto, func(d *CharacterDTO) { d.Items = nil })
func ChangeKeepsChosenExpertises(dto CharacterDTO, change func(*CharacterDTO)) error {
	dto.Expertises = slices.Clone(dto.Expertises)
	dto.Items = slices.Clone(dto.Items)
	change(&dto)
	return ChosenExpertisesFit(dto)
}

// CountedChoice é uma escolha contada que a ficha TEM — por atributo (as
// perícias do Especialista) ou por nível (os terrenos do Explorador): o poder,
// o que foi escolhido, o limite e o que se pode escolher.
type CountedChoice struct {
	Power book.ClassPower
	Picks []string
	// Value é o que dá o limite — o atributo sem temporários, ou o nível da
	// classe —, e Basis o nome dele na tela ("Int", "nível").
	Value int
	Basis string
	Limit int
	// Offered são as opções, e Labels o nome de cada uma quando ele não é o
	// próprio id (os terrenos).
	Offered []string
	Labels  map[string]string
	// Repeats diz que a mesma opção pode ser escolhida de novo (o terreno
	// repetido soma +2, p50); Noun é o que se escolhe ("perícia", "terreno").
	Repeats bool
	Noun    string
}

// CountedChoices são as escolhas contadas por atributo dos poderes que a ficha
// tem, para o diálogo e a pendência. Ficha sem o mundo não tem limite a dizer,
// e fica sem elas.
func CountedChoices(dto CharacterDTO) []CountedChoice {
	picks := PowerPicks(dto.PowerChoices)
	trained := []string{}
	for _, e := range dto.Expertises {
		if e.Trained {
			trained = append(trained, e.Name)
		}
	}
	out := []CountedChoice{}
	for _, owned := range OwnedPowers(dto) {
		power, known := book.ClassPowers()[owned.ID]
		if owned.Source != PowerFromClass || !known || power.Choice == nil {
			continue
		}
		if len(power.Choice.CountLevels) > 0 {
			out = append(out, levelCountedChoice(dto, power, picks[power.ID]))
			continue
		}
		if power.Choice.CountAttribute == "" {
			continue
		}
		value, err := countAttributeValue(dto, *power.Choice)
		if err != nil {
			continue
		}
		out = append(out, CountedChoice{
			Power: power, Picks: picks[power.ID], Value: value, Limit: max(1, value), Offered: trained,
			Basis: book.AttributeAbbrev(power.Choice.CountAttribute), Noun: "perícia",
		})
	}
	return out
}

// levelCountedChoice monta a escolha contada por nível: as opções que o nível
// da classe já alcança, com o nome de cada uma.
func levelCountedChoice(dto CharacterDTO, power book.ClassPower, picks []string) CountedChoice {
	limit, level := levelChoiceLimit(dto, power)
	choice := CountedChoice{
		Power: power, Picks: picks, Value: level, Limit: limit, Basis: "nível",
		Labels: map[string]string{}, Repeats: true, Noun: "terreno",
	}
	for _, o := range power.Choice.Options {
		if o.MinLevel > level {
			continue
		}
		choice.Offered = append(choice.Offered, o.ID)
		choice.Labels[o.ID] = o.Name
	}
	return choice
}
