package forge

import (
	"fmt"
	"net/http"
	"strings"
	"t20engine/domain/sheet"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/infra/wire"
)

// O NASCIMENTO DO HERÓI.
//
// A forja curta faz quatro perguntas — nome, raça, classe, origem — mais o
// equipamento inicial de p140, e o que sai dela é um personagem VÁLIDO de 1º
// nível, não um rascunho. O que ela não pergunta são as escolhas que a ficha já
// sabe fazer: os dois benefícios da origem (p85), o treino de perícia, os
// atributos de raça, o caminho, o devoto. Isso vira PENDÊNCIA, e pendência não
// é erro.
//
// A AUTORIDADE É DAQUI: raça, classe, origem e cada peça do equipamento são
// conferidas contra o catálogo antes de virarem linha no banco, sem adiar nada
// para uma pré-validação do cliente.

// heroNameMax é o teto do nome. O mesmo do nome de campanha: é a coluna de
// texto de uma tela, e duas medidas diferentes para a mesma coisa só produzem a
// pergunta de qual é a certa.
const heroNameMax = 120

// forgeAnswers são as respostas da folha em branco. Os três campos de
// equipamento carregam IDs do catálogo; `Escudo` é um sim/não porque o kit só
// oferece um escudo, o leve (p140).
type forgeAnswers struct {
	Name          string
	Race          string
	Class         string
	Origin        string
	SimpleWeapon  string
	MartialWeapon string
	Armor         string
	Shield        bool
}

// forgeRefusals confere a folha inteira e devolve uma mensagem por campo.
//
// Devolve TODAS as recusas de uma vez, e não a primeira: quem preencheu a folha
// inteira merece ver tudo o que falta numa passada, e não descobrir o segundo
// erro depois de consertar o primeiro.
func forgeRefusals(sheet forgeAnswers) wire.FieldErrorMap {
	errs := wire.FieldErrorMap{}
	if name := strings.TrimSpace(sheet.Name); name == "" || len([]rune(name)) > heroNameMax {
		errs["name"] = []string{fmt.Sprintf(
			"O nome é obrigatório e cabe em %d caracteres.", heroNameMax)}
	}
	if raceByName(sheet.Race) == nil {
		errs["race"] = []string{choiceRefusal(sheet.Race, "a linhagem", "raça")}
	}
	if _, found := book.Origins()[sheet.Origin]; !found {
		errs["origin"] = []string{choiceRefusal(sheet.Origin, "a origem", "origem")}
	}
	class := classByName(sheet.Class)
	if class == nil {
		errs["class"] = []string{choiceRefusal(sheet.Class, "o ofício", "classe")}
		return errs
	}
	// O equipamento só se confere DEPOIS da classe: é ela que diz quais peças o
	// kit oferece, e conferir contra um kit inventado acusaria o campo errado.
	gearRefusals(sheet, engine.StartingKitFor(class.Name, class.Proficiencies), errs)
	return errs
}

// choiceRefusal separa "não escolheu" de "escolheu o que não existe".
//
// As duas voltam no mesmo campo e não dizem a mesma coisa: em branco é um passo
// que falta, e um valor desconhecido só chega por POST feito na mão ou por
// catálogo que mudou debaixo de uma folha aberta. Mandar `"" não é uma raça do
// livro` para quem simplesmente ainda não escolheu é responder outra pergunta.
func choiceRefusal(value, missing, whatItIs string) string {
	if strings.TrimSpace(value) == "" {
		return "Escolha " + missing + " do herói."
	}
	return fmt.Sprintf("%q não é uma %s do livro.", value, whatItIs)
}

// gearRefusals confere as escolhas de p140 contra o kit da classe.
//
// As duas metades importam. Faltar o que o kit OFERECE é recusa porque o herói
// nasceria desarmado sem ter escolhido isso; mandar o que o kit NÃO oferece
// também é, porque é o cliente concedendo a si mesmo uma brunea que a classe
// não sabe vestir.
func gearRefusals(sheet forgeAnswers, kit engine.StartingKit, errs wire.FieldErrorMap) {
	if failure := weaponFitsKit(sheet.SimpleWeapon, "weapon-simple", true); failure != "" {
		errs["weaponSimple"] = []string{failure}
	}
	if failure := weaponFitsKit(sheet.MartialWeapon, "weapon-martial", kit.MartialWeapon); failure != "" {
		errs["weaponMartial"] = []string{failure}
	}
	if failure := armorFitsKit(sheet.Armor, kit.Armors); failure != "" {
		errs["armor"] = []string{failure}
	}
	if sheet.Shield && kit.Shield == "" {
		errs["shield"] = []string{"Esta classe não é proficiente com escudos."}
	}
}

// weaponFitsKit devolve a recusa da arma, ou "" quando ela serve. `oferecida`
// diz se o kit dá essa arma: quando não dá, o campo tem de vir vazio.
func weaponFitsKit(id, category string, offered bool) string {
	if !offered {
		if id == "" {
			return ""
		}
		return "Esta classe não começa com arma marcial."
	}
	item := book.ItemByID(id)
	if item == nil {
		return "Escolha a arma com que o herói nasce."
	}
	if item.Category != category {
		return fmt.Sprintf("%s não é uma arma da categoria que o kit oferece.", item.Name)
	}
	return ""
}

// armorFitsKit confere a armadura contra as que o kit oferece. Lista vazia
// é o arcanista, que "começa sem armadura" (p140) — e aí escolher uma é recusa.
func armorFitsKit(id string, offered []string) string {
	if len(offered) == 0 {
		if id == "" {
			return ""
		}
		return "Arcanistas começam sem armadura (p140)."
	}
	for _, available := range offered {
		if id == available {
			return ""
		}
	}
	return "Escolha uma das armaduras que o kit oferece."
}

// raceByName acha a raça pelo nome, ou nil.
func raceByName(name string) *book.Race {
	races, _, _ := book.CharacterCatalogs()
	for i := range races {
		if races[i].Name == name {
			return &races[i]
		}
	}
	return nil
}

// classByName acha a classe pelo nome, ou nil.
func classByName(name string) *book.Class {
	_, classes, _ := book.CharacterCatalogs()
	for i := range classes {
		if classes[i].Name == name {
			return &classes[i]
		}
	}
	return nil
}

// birthHero cria o herói da folha e devolve o id dele.
//
// Assume a folha JÁ conferida por `forgeRefusals` — quem chama recusa antes.
func (s Scene) birthHero(r *http.Request, ownerID int64, stylesheet forgeAnswers) (int64, error) {
	race, class := raceByName(stylesheet.Race), classByName(stylesheet.Class)
	if race == nil || class == nil {
		return 0, fmt.Errorf("nascimento com folha não conferida: raça %q, classe %q", stylesheet.Race, stylesheet.Class)
	}
	body, err := birthBody(stylesheet, *race, *class)
	if err != nil {
		return 0, err
	}
	id, err := s.births.Create(r.Context(), ownerID, body.Name, body, 1,
		book.GrantedProficiencies([]string{class.Name}), sheet.ToStringSet(body.TrainedExpertises))
	if err != nil {
		return 0, err
	}
	// NADA de encher poço aqui: um personagem sem linha em `character_damage` JÁ
	// está cheio, porque o atual é `máximo derivado − dano` e a ausência de linha
	// é o zero. Aqui morava o passo que enchia os poços à força, e ele existia
	// para escrever as quatro colunas de espelho — que saíram na 00015 (ALE-355).
	return id, nil
}

// birthBody monta o herói de 1º nível que a folha descreve.
//
// Três coisas não são perguntas da folha e mesmo assim ficam decididas aqui,
// porque são consequência e não escolha: o TAMANHO e o DESLOCAMENTO saem da
// raça, e as perícias treinadas são as FIXAS da classe. As que a classe manda
// escolher — o "Luta ou Pontaria" do guerreiro e as duas do bolo — são
// pendência da ficha, junto com os dois benefícios da origem (p85).
//
// Os seis atributos nascem em ZERO, que é o ponto de partida da compra de
// pontos (p17): distribuí-los é a segunda cena da forja.
func birthBody(stylesheet forgeAnswers, race book.Race, class book.Class) (sheet.CreateBody, error) {
	tibar, err := birthPurse(stylesheet.Origin)
	if err != nil {
		return sheet.CreateBody{}, err
	}
	kit := engine.StartingKitFor(class.Name, class.Proficiencies)
	return sheet.CreateBody{
		Name:              strings.TrimSpace(stylesheet.Name),
		Races:             []string{race.Name},
		Origin:            stylesheet.Origin,
		Classes:           []sheet.ClassEntry{{ClassName: class.Name, Level: 1}},
		Tibar:             &tibar,
		Items:             birthItems(stylesheet, kit),
		Size:              race.Size,
		Displacement:      int64(race.Speed),
		TrainedExpertises: class.Expertises,
	}, nil
}
