package sheet

import (
	"encoding/json"
	"fmt"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// AS REGRAS DE ESCOLHA de poder: quantos poderes cabem no nível, quantos
// benefícios a origem dá, quais caminhos e quais deuses cada classe aceita.
//
// Elas moram no `sheet` porque leem o LIVRO (o poder existe? este deus serve a
// esta classe?) e a FICHA (que classes, que nível, que origem) — não cabem no
// `book`, que não pode importar daqui, nem na cena, que é apresentação e é lida
// pela rota JSON.
//
// E são a FRONTEIRA, não um espelho da tela: sem elas um pedido montado à mão
// põe vinte poderes num personagem de nível 1, e o motor soma os modificadores
// de todos. Rodam nas DUAS portas — a rota JSON e os comandos da ficha em
// Datastar —, porque duas validações divergem no dia em que uma regra nova
// chegar e a esquecida aceita o que a outra recusa.

// benefitsOriginLimit são os benefícios que a origem concede: duas perícias e um
// poder é o desenho do livro (p85), e a ficha os trata como DOIS itens de uma
// lista que inclui o poder único.
const BenefitsOriginLimit = 2

// levelWithPowerFirst é o nível em que a primeira vaga de poder abre. Todas as
// catorze classes ganham "um poder por nível a partir do 2º" (p33).
const levelWithPowerFirst = 2

// PowerSlots são as vagas que o nível na classe já abriu.
func PowerSlots(level int64) int {
	if level < levelWithPowerFirst {
		return 0
	}
	return int(level) - levelWithPowerFirst + 1
}

// ChoiceOption é uma opção de escolha da classe: o que se grava e o que se lê.
//
// Ela é o par que a REGRA precisa — o valor decide se a escolha é válida, o
// rótulo é o nome do livro. A cena a converte no tipo de `<select>` dela; o par
// não é de apresentação, é o que o livro oferece.
type ChoiceOption struct {
	Value string
	Label string
}

// classPaths são as escolhas de caminho, e o nível em que elas abrem.
//
// O Arcanista escolhe no 1º (o caminho DEFINE o atributo-chave dele), e o
// Paladino e o Cavaleiro no 5º.
var classPaths = map[string]struct {
	Options  []ChoiceOption
	MinLevel int64
}{
	"Arcanista": {Options: []ChoiceOption{
		{Value: "bruxo", Label: "Bruxo"},
		{Value: "feiticeiro", Label: "Feiticeiro"},
		{Value: "mago", Label: "Mago"},
	}, MinLevel: 1},
	"Paladino": {Options: []ChoiceOption{
		{Value: "egide-sagrada", Label: "Égide Sagrada"},
		{Value: "montaria-sagrada", Label: "Montaria Sagrada"},
	}, MinLevel: 5},
	"Cavaleiro": {Options: []ChoiceOption{
		{Value: "bastiao", Label: "Bastião"},
		{Value: "montaria", Label: "Montaria"},
	}, MinLevel: 5},
}

// LevelPaths são as opções de caminho quando o nível já as abriu.
func LevelPaths(class string, level int64) []ChoiceOption {
	slot, found := classPaths[class]
	if !found || level < slot.MinLevel {
		return nil
	}
	return slot.Options
}

// ClassDevotees são os deuses que a classe aceita, ou nil quando ela não escolhe
// devoto.
//
// As três listas são do livro: o Clérigo serve qualquer deus MAIOR ou o Panteão
// (p57); o Paladino tem a lista de oito mais o "Paladino do Bem" (p82); o Druida
// serve Allihanna, Megalokk ou Oceano (p61), e não tem alternativa fora das
// divindades.
func ClassDevotees(class string) []ChoiceOption {
	_, _, gods := book.CharacterCatalogs()
	switch class {
	case "Clérigo":
		return append(godsThat(gods, func(d book.God) bool { return d.Major }),
			ChoiceOption{Value: "panteao", Label: "Panteão"})
	case "Paladino":
		return append(godsThat(gods, func(d book.God) bool { return d.PaladinEligible }),
			ChoiceOption{Value: "bem", Label: "Paladino do Bem"})
	case "Druida":
		return godsThat(gods, func(d book.God) bool { return d.DruidEligible })
	}
	return nil
}

func godsThat(gods []book.God, accepts func(book.God) bool) []ChoiceOption {
	outside := []ChoiceOption{}
	for _, d := range gods {
		if accepts(d) {
			outside = append(outside, ChoiceOption{Value: d.ID, Label: d.Name})
		}
	}
	return outside
}

// ── A VALIDAÇÃO, que é a fronteira ───────────────────────────────────────────

// WithChoicesValid recusa um conjunto de escolhas que o livro não permite.
//
// Ela vale sobre o RESULTADO, e não sobre a diferença: a escrita tem de deixar a
// ficha inteira válida. É mais estrito que "não acrescente além do limite" — uma
// ficha que já esteja fora da conta não aceita escrita de escolha nenhuma até ser
// arrumada —, e é a decisão do dono.
func WithChoicesValid(dto CharacterDTO) error {
	if err := chosenFitPowers(dto); err != nil {
		return err
	}
	if err := originFitBenefits(dto); err != nil {
		return err
	}
	return classChoiceSelectionsAreValid(dto)
}

// chosenFitPowers confere a conta de vagas e a procedência de cada id.
//
// As vagas são a SOMA das classes: um bárbaro 3/ladino 2 tem as vagas dos dois
// níveis. E cada poder escolhido precisa existir — um poder de classe ELETIVO de
// uma classe que o personagem tem, ou um poder geral. Automático não conta: ele
// não ocupa vaga porque não foi escolhido.
func chosenFitPowers(dto CharacterDTO) error {
	chosen := UnmarshalStrings(dto.ClassPowers)
	slots := 0
	classes := map[string]bool{}
	for _, c := range dto.Classes {
		slots += PowerSlots(c.Level)
		classes[c.ClassName] = true
	}
	for _, id := range chosen {
		if err := chosenExistsPower(id, classes); err != nil {
			return err
		}
	}
	if len(chosen) > slots {
		return fmt.Errorf("são %d poderes escolhidos para %s",
			len(chosen), writtenSlots(slots))
	}
	return nil
}

func chosenExistsPower(id string, classes map[string]bool) error {
	if power, found := book.ClassPowers()[id]; found {
		if power.GrantedAtLevel != nil {
			return fmt.Errorf("%q é automático da classe e não ocupa vaga", power.Name)
		}
		if !classes[power.ClassName] {
			return fmt.Errorf("%q é um poder de %s, e esta ficha não tem a classe",
				power.Name, power.ClassName)
		}
		return nil
	}
	if _, found := book.GeneralPowers()[id]; found {
		return nil
	}
	return fmt.Errorf("o poder %q não existe no livro", id)
}

func writtenSlots(slots int) string {
	if slots == 1 {
		return "1 vaga"
	}
	return strconv.Itoa(slots) + " vagas"
}

// originFitBenefits confere o teto de dois e a procedência.
func originFitBenefits(dto CharacterDTO) error {
	chosen := UnmarshalStrings(dto.OriginChoices)
	if len(chosen) > BenefitsOriginLimit {
		return fmt.Errorf("a origem dá %d benefícios, e foram escolhidos %d",
			BenefitsOriginLimit, len(chosen))
	}
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	fromOrigin := map[string]bool{}
	for _, b := range OriginBenefitsOf(origin) {
		fromOrigin[b.ID] = true
	}
	for _, id := range chosen {
		if !fromOrigin[id] {
			return fmt.Errorf("%q não é um benefício de %s", id, origin.Name)
		}
	}
	return nil
}

// OriginBenefitsOf são os benefícios da origem MAIS o poder único dela, que a
// ficha trata como um item da mesma lista.
func OriginBenefitsOf(origin book.Origin) []book.OriginBenefit {
	outside := append([]book.OriginBenefit{}, origin.Benefits...)
	if origin.UniquePower.ID != "" {
		outside = append(outside, origin.UniquePower)
	}
	return outside
}

// classChoiceSelectionsAreValid confere caminho e devoto contra as opções da
// classe.
func classChoiceSelectionsAreValid(dto CharacterDTO) error {
	choices := ClassChoiceSelections(dto)
	for _, class := range dto.Classes {
		blob := choices[class.ClassName]
		if err := chosenExistsOption(
			"caminho", blob.Path, LevelPaths(class.ClassName, class.Level), class.ClassName,
		); err != nil {
			return err
		}
		if err := chosenExistsOption(
			"devoto", blob.Devotee, ClassDevotees(class.ClassName), class.ClassName,
		); err != nil {
			return err
		}
	}
	return nil
}

// ClassChoiceSelections lê o blob de `classChoices` por nome de classe.
func ClassChoiceSelections(dto CharacterDTO) map[string]engine.ClassChoiceSelections {
	choices := map[string]engine.ClassChoiceSelections{}
	_ = json.Unmarshal([]byte(dto.ClassChoices), &choices)
	return choices
}

// chosenExistsOption recusa um valor fora da lista da classe.
//
// Vazio é caminho normal: quem ainda não escolheu tem uma PENDÊNCIA, e não um
// erro — a ficha existe para ser preenchida aos poucos.
func chosenExistsOption(which, value string, options []ChoiceOption, class string) error {
	if value == "" {
		return nil
	}
	if len(options) == 0 {
		return fmt.Errorf("%s não escolhe %s", class, which)
	}
	for _, o := range options {
		if o.Value == value {
			return nil
		}
	}
	return fmt.Errorf("%q não é um %s de %s", value, which, class)
}
