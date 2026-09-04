package sheet

import (
	"encoding/json"
	"fmt"
	"strconv"

	"t20engine/book"
	"t20engine/engine"
)

// AS REGRAS DE ESCOLHA de poder (ALE-272, fatia 8; movidas para cá na ALE-278).
//
// # Elas eram só da TELA, e o servidor gravava qualquer coisa
//
// Quantos poderes cabem no nível, quantos benefícios a origem dá, quais caminhos
// e quais deuses cada classe aceita: tudo isso vivia em 363 linhas de
// `shared/rules/abilities-*.ts`, e o `handleUpdateAbilities` gravava os cinco
// blobs sem conferir NADA. Um pedido montado à mão punha vinte poderes num
// personagem de nível 1 — e o motor somava os modificadores de todos.
//
// # Por que elas moram no `sheet`
//
// Elas leem o LIVRO (o poder existe? este deus serve a esta classe?) e a FICHA
// (que classes, que nível, que origem), e por isso não cabiam nem no `book`, que
// não pode importar daqui, nem na cena, que é apresentação e é lida pela rota
// JSON. Foi a entrada do `book` na lista do guarda desta pasta — decisão do
// dono, ALE-278 — e a razão inteira está escrita lá.
//
// Elas rodam nas DUAS portas: o endpoint JSON e os comandos da ficha em
// Datastar. Duas validações divergiriam no dia em que uma regra nova chegasse, e
// a esquecida aceitaria o que a outra recusa.

// benefitsOriginLimit são os benefícios que a origem concede: duas perícias e um
// poder é o desenho do livro (p85), e a ficha os trata como DOIS itens de uma
// lista que inclui o poder único.
const BenefitsOriginLimit = 2

// levelWithPowerFirst é o nível em que a primeira vaga de poder abre. Todas as
// catorze classes ganham "um poder por nível a partir do 2º" (p33).
const levelWithPowerFirst = 2

// PowerSlots são as vagas que o nível na classe já abriu.
func PowerSlots(nivel int64) int {
	if nivel < levelWithPowerFirst {
		return 0
	}
	return int(nivel) - levelWithPowerFirst + 1
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
func LevelPaths(classe string, nivel int64) []ChoiceOption {
	slot, tem := classPaths[classe]
	if !tem || nivel < slot.MinLevel {
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
func ClassDevotees(classe string) []ChoiceOption {
	_, _, deuses := book.CharacterCatalogs()
	switch classe {
	case "Clérigo":
		return append(godsThat(deuses, func(d book.God) bool { return d.Major }),
			ChoiceOption{Value: "panteao", Label: "Panteão"})
	case "Paladino":
		return append(godsThat(deuses, func(d book.God) bool { return d.PaladinoEligible }),
			ChoiceOption{Value: "bem", Label: "Paladino do Bem"})
	case "Druida":
		return godsThat(deuses, func(d book.God) bool { return d.DruidaEligible })
	}
	return nil
}

func godsThat(deuses []book.God, aceita func(book.God) bool) []ChoiceOption {
	fora := []ChoiceOption{}
	for _, d := range deuses {
		if aceita(d) {
			fora = append(fora, ChoiceOption{Value: d.ID, Label: d.Name})
		}
	}
	return fora
}

// ── A VALIDAÇÃO, que é a fronteira ───────────────────────────────────────────

// WithChoicesValid recusa um conjunto de escolhas que o livro não permite.
//
// Ela vale sobre o RESULTADO, e não sobre a diferença: a escrita tem de deixar a
// ficha inteira válida. É mais estrito que "não acrescente além do limite" — uma
// ficha que já esteja fora da conta não aceita escrita de escolha nenhuma até
// ser arrumada — e é a decisão do dono. O projeto ainda não foi usado numa mesa
// real, então não há ficha antiga fora da conta para proteger.
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
	escolhidos := UnmarshalStrings(dto.ClassPowers)
	vagas := 0
	classes := map[string]bool{}
	for _, c := range dto.Classes {
		vagas += PowerSlots(c.Level)
		classes[c.ClassName] = true
	}
	for _, id := range escolhidos {
		if err := chosenExistsPower(id, classes); err != nil {
			return err
		}
	}
	if len(escolhidos) > vagas {
		return fmt.Errorf("são %d poderes escolhidos para %s",
			len(escolhidos), writtenSlots(vagas))
	}
	return nil
}

func chosenExistsPower(id string, classes map[string]bool) error {
	if poder, tem := book.ClassPowers()[id]; tem {
		if poder.GrantedAtLevel != nil {
			return fmt.Errorf("%q é automático da classe e não ocupa vaga", poder.Name)
		}
		if !classes[poder.ClassName] {
			return fmt.Errorf("%q é um poder de %s, e esta ficha não tem a classe",
				poder.Name, poder.ClassName)
		}
		return nil
	}
	if _, tem := book.GeneralPowers()[id]; tem {
		return nil
	}
	return fmt.Errorf("o poder %q não existe no livro", id)
}

func writtenSlots(vagas int) string {
	if vagas == 1 {
		return "1 vaga"
	}
	return strconv.Itoa(vagas) + " vagas"
}

// originFitBenefits confere o teto de dois e a procedência.
func originFitBenefits(dto CharacterDTO) error {
	escolhidos := UnmarshalStrings(dto.OriginChoices)
	if len(escolhidos) > BenefitsOriginLimit {
		return fmt.Errorf("a origem dá %d benefícios, e foram escolhidos %d",
			BenefitsOriginLimit, len(escolhidos))
	}
	origem, tem := book.Origins()[dto.Origin]
	if !tem {
		return nil
	}
	daOrigem := map[string]bool{}
	for _, b := range OriginBenefitsOf(origem) {
		daOrigem[b.ID] = true
	}
	for _, id := range escolhidos {
		if !daOrigem[id] {
			return fmt.Errorf("%q não é um benefício de %s", id, origem.Name)
		}
	}
	return nil
}

// OriginBenefitsOf são os benefícios da origem MAIS o poder único dela, que a
// ficha trata como um item da mesma lista.
func OriginBenefitsOf(origem book.Origin) []book.OriginBenefit {
	fora := append([]book.OriginBenefit{}, origem.Benefits...)
	if origem.PoderUnico.ID != "" {
		fora = append(fora, origem.PoderUnico)
	}
	return fora
}

// classChoiceSelectionsAreValid confere caminho e devoto contra as opções da
// classe.
func classChoiceSelectionsAreValid(dto CharacterDTO) error {
	escolhas := ClassChoiceSelections(dto)
	for _, classe := range dto.Classes {
		blob := escolhas[classe.ClassName]
		if err := chosenExistsOption(
			"caminho", blob.Caminho, LevelPaths(classe.ClassName, classe.Level), classe.ClassName,
		); err != nil {
			return err
		}
		if err := chosenExistsOption(
			"devoto", blob.Devoto, ClassDevotees(classe.ClassName), classe.ClassName,
		); err != nil {
			return err
		}
	}
	return nil
}

// ClassChoiceSelections lê o blob de `classChoices` por nome de classe.
func ClassChoiceSelections(dto CharacterDTO) map[string]engine.ClassChoiceSelections {
	escolhas := map[string]engine.ClassChoiceSelections{}
	_ = json.Unmarshal([]byte(dto.ClassChoices), &escolhas)
	return escolhas
}

// chosenExistsOption recusa um valor fora da lista da classe.
//
// Vazio é caminho normal: quem ainda não escolheu tem uma PENDÊNCIA, e não um
// erro — a ficha existe para ser preenchida aos poucos.
func chosenExistsOption(qual, valor string, opcoes []ChoiceOption, classe string) error {
	if valor == "" {
		return nil
	}
	if len(opcoes) == 0 {
		return fmt.Errorf("%s não escolhe %s", classe, qual)
	}
	for _, o := range opcoes {
		if o.Value == valor {
			return nil
		}
	}
	return fmt.Errorf("%q não é um %s de %s", valor, qual, classe)
}
