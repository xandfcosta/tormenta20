package forge

import (
	"fmt"
	"net/http"
	"strings"
	"t20engine/sheet"

	"t20engine/book"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// O NASCIMENTO DO HERÓI (ALE-272, fatia 9).
//
// A forja curta faz quatro perguntas — nome, raça, classe, origem — mais o
// equipamento inicial de p140, e o que sai dela é um personagem VÁLIDO de 1º
// nível, não um rascunho. O que ela não pergunta são as escolhas que a ficha já
// sabe fazer desde as fatias 2 a 8: os dois benefícios da origem (p85), o treino
// de perícia, os atributos de raça, o caminho, o devoto. Isso vira PENDÊNCIA, e
// pendência não é erro.
//
// A AUTORIDADE É DAQUI. A rota JSON de criar personagem diz por escrito que as
// checagens de catálogo estão adiadas porque "o frontend pré-valida"; esta não
// adia nada — raça, classe, origem e cada peça do equipamento são conferidas
// contra o catálogo antes de virarem linha no banco.

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
func forgeRefusals(folha forgeAnswers) plataforma.FieldErrorMap {
	erros := plataforma.FieldErrorMap{}
	if nome := strings.TrimSpace(folha.Name); nome == "" || len([]rune(nome)) > heroNameMax {
		erros["name"] = []string{fmt.Sprintf(
			"O nome é obrigatório e cabe em %d caracteres.", heroNameMax)}
	}
	if raceByName(folha.Race) == nil {
		erros["race"] = []string{choiceRefusal(folha.Race, "a linhagem", "raça")}
	}
	if _, tem := book.Origins()[folha.Origin]; !tem {
		erros["origin"] = []string{choiceRefusal(folha.Origin, "a origem", "origem")}
	}
	classe := classByName(folha.Class)
	if classe == nil {
		erros["class"] = []string{choiceRefusal(folha.Class, "o ofício", "classe")}
		return erros
	}
	// O equipamento só se confere DEPOIS da classe: é ela que diz quais peças o
	// kit oferece, e conferir contra um kit inventado acusaria o campo errado.
	gearRefusals(folha, engine.StartingKitFor(classe.Name, classe.Proficiencias), erros)
	return erros
}

// choiceRefusal separa "não escolheu" de "escolheu o que não existe".
//
// As duas voltam no mesmo campo e não dizem a mesma coisa: em branco é um passo
// que falta, e um valor desconhecido só chega por POST feito na mão ou por
// catálogo que mudou debaixo de uma folha aberta. Mandar `"" não é uma raça do
// livro` para quem simplesmente ainda não escolheu é responder outra pergunta.
func choiceRefusal(valor, oQueFalta, oQueE string) string {
	if strings.TrimSpace(valor) == "" {
		return "Escolha " + oQueFalta + " do herói."
	}
	return fmt.Sprintf("%q não é uma %s do livro.", valor, oQueE)
}

// gearRefusals confere as escolhas de p140 contra o kit da classe.
//
// As duas metades importam. Faltar o que o kit OFERECE é recusa porque o herói
// nasceria desarmado sem ter escolhido isso; mandar o que o kit NÃO oferece
// também é, porque é o cliente concedendo a si mesmo uma brunea que a classe
// não sabe vestir.
func gearRefusals(folha forgeAnswers, kit engine.StartingKit, erros plataforma.FieldErrorMap) {
	if erro := weaponFitsKit(folha.SimpleWeapon, "weapon-simple", true); erro != "" {
		erros["weaponSimple"] = []string{erro}
	}
	if erro := weaponFitsKit(folha.MartialWeapon, "weapon-martial", kit.MartialWeapon); erro != "" {
		erros["weaponMartial"] = []string{erro}
	}
	if erro := armorFitsKit(folha.Armor, kit.Armors); erro != "" {
		erros["armor"] = []string{erro}
	}
	if folha.Shield && kit.Shield == "" {
		erros["shield"] = []string{"Esta classe não é proficiente com escudos."}
	}
}

// weaponFitsKit devolve a recusa da arma, ou "" quando ela serve. `oferecida`
// diz se o kit dá essa arma: quando não dá, o campo tem de vir vazio.
func weaponFitsKit(id, categoria string, oferecida bool) string {
	if !oferecida {
		if id == "" {
			return ""
		}
		return "Esta classe não começa com arma marcial."
	}
	item := book.ItemByID(id)
	if item == nil {
		return "Escolha a arma com que o herói nasce."
	}
	if item.Category != categoria {
		return fmt.Sprintf("%s não é uma arma da categoria que o kit oferece.", item.Name)
	}
	return ""
}

// armorFitsKit confere a armadura contra as que o kit oferece. Lista vazia
// é o arcanista, que "começa sem armadura" (p140) — e aí escolher uma é recusa.
func armorFitsKit(id string, oferecidas []string) string {
	if len(oferecidas) == 0 {
		if id == "" {
			return ""
		}
		return "Arcanistas começam sem armadura (p140)."
	}
	for _, oferecida := range oferecidas {
		if id == oferecida {
			return ""
		}
	}
	return "Escolha uma das armaduras que o kit oferece."
}

// raceByName acha a raça pelo nome, ou nil.
func raceByName(nome string) *book.Race {
	racas, _, _ := book.CharacterCatalogs()
	for i := range racas {
		if racas[i].Name == nome {
			return &racas[i]
		}
	}
	return nil
}

// classByName acha a classe pelo nome, ou nil.
func classByName(nome string) *book.Class {
	_, classes, _ := book.CharacterCatalogs()
	for i := range classes {
		if classes[i].Name == nome {
			return &classes[i]
		}
	}
	return nil
}

// birthHero cria o herói da folha e devolve o id dele.
//
// Assume a folha JÁ conferida por `forgeRefusals` — quem chama recusa antes.
func (s Scene) birthHero(r *http.Request, ownerID int64, folha forgeAnswers) (int64, error) {
	raca, classe := raceByName(folha.Race), classByName(folha.Class)
	if raca == nil || classe == nil {
		return 0, fmt.Errorf("nascimento com folha não conferida: raça %q, classe %q", folha.Race, folha.Class)
	}
	corpo, err := birthBody(folha, *raca, *classe)
	if err != nil {
		return 0, err
	}
	id, err := s.deps.InsertCharacter(r, ownerID, corpo.Name, corpo, 1,
		book.GrantedProficiencies([]string{classe.Name}), sheet.ToStringSet(corpo.TrainedExpertises))
	if err != nil {
		return 0, err
	}
	return id, s.fillPools(r, id)
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
func birthBody(folha forgeAnswers, raca book.Race, classe book.Class) (sheet.CreateBody, error) {
	tibar, err := birthPurse(folha.Origin)
	if err != nil {
		return sheet.CreateBody{}, err
	}
	kit := engine.StartingKitFor(classe.Name, classe.Proficiencias)
	return sheet.CreateBody{
		Name:              strings.TrimSpace(folha.Name),
		Races:             []string{raca.Name},
		Origin:            folha.Origin,
		Classes:           []sheet.ClassEntry{{ClassName: classe.Name, Level: 1}},
		Tibar:             &tibar,
		Items:             birthItems(folha, kit),
		Size:              raca.Tamanho,
		Displacement:      int64(raca.Deslocamento),
		TrainedExpertises: classe.Pericias,
	}, nil
}

// fillPools deixa o herói recém-nascido com PV e PM CHEIOS.
//
// São dois passos e não um porque o `healVitals` calcula os máximos a partir do
// agregado já gravado — antes de existir linha no banco não há de onde tirar o
// PV da classe. Nascer com o poço no zero seria nascer inconsciente.
func (s Scene) fillPools(r *http.Request, id int64) error {
	row, err := s.deps.Queries().GetCharacter(r.Context(), id)
	if err != nil {
		return err
	}
	dto, err := sheet.Load(r.Context(), s.deps.Queries(), row)
	if err != nil {
		return err
	}
	if err := s.deps.HealVitals(r, id, &dto); err != nil {
		return err
	}
	return s.deps.Queries().SetCharacterVitals(r.Context(), sqlcgen.SetCharacterVitalsParams{
		HpMax: dto.HpMax, HpCurrent: dto.HpMax, MpMax: dto.MpMax, MpCurrent: dto.MpMax,
		UpdatedAt: plataforma.NowISO(), ID: id,
	})
}
