package sheetui

import (
	"encoding/json"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O ACERVO DE PODERES de um personagem, numa lista só: as habilidades da RAÇA,
// os benefícios de ORIGEM escolhidos, as habilidades AUTOMÁTICAS de classe (as
// que o nível concede), e os poderes ESCOLHIDOS — de classe, gerais e da
// Tormenta.
//
// # Por que o nome e o texto NÃO vêm do motor
//
// O `ClassPower` do motor é um subconjunto deliberado: ele carrega o que a
// derivação lê — id, classe, nível, modificadores — e não a descrição. Quem
// tem o texto é o catálogo cru, que a vitrine do mestre já lê
// (`FlattenedPowers`). Então a POSSE é pergunta do motor e o TEXTO é do
// catálogo; misturar as duas fontes numa struct só é o que este arquivo faz.

// ownedPower é um poder que o personagem TEM, pronto para a tela.
type ownedPower struct {
	ID   string
	Name string
	// Source é a procedência escrita ("Classe · Bárbaro", "Raça · Anão").
	Source string
	// Detail é o texto da regra.
	Detail string
	Page   int
}

// ownedPowersOf junta as cinco procedências, na ordem em que a tela as mostra.
func ownedPowersOf(dto sheet.CharacterDTO) []ownedPower {
	outside := []ownedPower{}
	outside = append(outside, raceAbilities(dto)...)
	outside = append(outside, originBenefits(dto)...)
	outside = append(outside, automaticAbilities(dto)...)
	outside = append(outside, chosenPowers(dto)...)
	return outside
}

// raceAbilities são as habilidades de cada raça da ficha.
//
// Quem lê o catálogo é o `RaceTraitsByKey` do dossiê — o MESMO `race-defs.json`, já
// indexado por id E por nome, porque o personagem guarda a raça por um dos dois.
// Um segundo leitor aqui seria uma terceira cópia da mesma decisão.
func raceAbilities(dto sheet.CharacterDTO) []ownedPower {
	outside := []ownedPower{}
	for _, r := range dto.Races {
		race, found := book.RaceTraitsByKey()[r.Race]
		if !found {
			continue
		}
		for _, ability := range race.Abilities {
			outside = append(outside, ownedPower{
				ID: ability.ID, Name: ability.Name, Detail: ability.Description,
				Source: "Raça · " + race.Name,
			})
		}
	}
	return outside
}

// originBenefits são só os ESCOLHIDOS.
//
// A origem oferece mais benefícios do que o personagem leva (duas perícias e um
// poder, de uma lista maior), então listar todos mostraria como possuído o que
// ninguém escolheu.
func originBenefits(dto sheet.CharacterDTO) []ownedPower {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	chosen := map[string]bool{}
	var ids []string
	if json.Unmarshal([]byte(dto.OriginChoices), &ids) == nil {
		for _, id := range ids {
			chosen[id] = true
		}
	}
	outside := []ownedPower{}
	for _, b := range sheet.OriginBenefitsOf(origin) {
		if !chosen[b.ID] {
			continue
		}
		outside = append(outside, ownedPower{
			ID: b.ID, Name: b.Name, Detail: b.Description,
			Source: "Origem · " + origin.Name, Page: origin.BookPage,
		})
	}
	return outside
}

// automaticAbilities são as que o NÍVEL concede, sem escolha.
//
// Quem decide a posse é o MOTOR (`engine.OwnedClassPowerIDs`), com a mesma
// regra que a derivação usa para somar os modificadores: nível alcançado, id
// escolhido, ou concedido por uma escolha de classe (o caminho do arcanista, o
// deus do clérigo). Uma segunda leitura aqui daria uma tela que mostra um poder
// que a ficha não soma — ou o contrário.
func automaticAbilities(dto sheet.CharacterDTO) []ownedPower {
	choices := sheet.ClassChoiceSelections(dto)
	outside := []ownedPower{}
	for _, class := range dto.Classes {
		for _, power := range book.ClassPowers() {
			if power.ClassName != class.ClassName || !automaticOwnership(power, class, choices) {
				continue
			}
			outside = append(outside, ownedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: "Classe · " + class.ClassName, Page: power.BookPage,
			})
		}
	}
	book.SortByName(outside, func(p ownedPower) string { return p.Name })
	return outside
}

// automaticOwnership pergunta ao MOTOR se a classe concede este poder.
//
// A lista de ESCOLHIDOS entra vazia de propósito: quem escolheu já aparece em
// `chosenPowers`, e passá-la aqui listaria o mesmo poder duas vezes.
func automaticOwnership(
	power book.ClassPower, class sheet.ClassDTO, choices map[string]engine.ClassChoiceSelections,
) bool {
	doMotor := &engine.ClassPower{
		ID: power.ID, ClassName: power.ClassName, Name: power.Name,
		GrantedAtLevel: power.GrantedAtLevel,
	}
	if power.GrantedByChoice != nil {
		doMotor.GrantedByChoice = &engine.GrantedByChoice{
			Field: power.GrantedByChoice.Field, Value: power.GrantedByChoice.Value,
		}
	}
	return engine.OwnsClassPower(doMotor, int(class.Level), nil, choices[class.ClassName])
}

// chosenPowers são os ids da coluna `classPowers` — poder de classe, poder
// geral ou poder da Tormenta, nessa ordem de busca.
//
// Blob torto vira lista vazia: a aba não pode deixar de abrir porque uma linha
// do banco está errada.
func chosenPowers(dto sheet.CharacterDTO) []ownedPower {
	var ids []string
	if json.Unmarshal([]byte(dto.ClassPowers), &ids) != nil {
		return nil
	}
	outside := []ownedPower{}
	for _, id := range ids {
		if power, found := book.ClassPowers()[id]; found {
			outside = append(outside, ownedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: "Classe · " + power.ClassName, Page: power.BookPage,
			})
			continue
		}
		if power, found := book.GeneralPowers()[id]; found {
			outside = append(outside, ownedPower{
				ID: power.ID, Name: power.Name, Detail: power.Description,
				Source: powerGeneralSource(power), Page: power.BookPage,
			})
		}
	}
	return outside
}

// powerGeneralSource separa o poder da TORMENTA do poder geral comum: eles
// moram no mesmo catálogo e a mesa os trata como coisas diferentes.
func powerGeneralSource(power book.GeneralPower) string {
	if power.Kind == "tormenta" {
		return "Poder da Tormenta"
	}
	return "Poder geral"
}

// ── os catálogos, lidos uma vez ──────────────────────────────────────────────

// attributeRaceMod é como a raça mexe nos atributos: fixo, distribuído ou
// por ascendência. Nil quando a raça não está no catálogo.
//
// Ele vem do `races.json` — a vitrine do mestre —, que é o único catálogo que
// carrega o `atributoMod`. O `race-defs.json` traz as habilidades com id e
// texto; os dois são lidos por esta aba, cada um pelo que só ele tem.
func attributeRaceMod(name string) *book.RaceAttribute {
	races, _, _ := book.CharacterCatalogs()
	for i, r := range races {
		if r.Name == name || r.ID == name {
			return &races[i].AttributeMod
		}
	}
	return nil
}

// raceAncestries são as metades de uma raça que se escolhe na criação — o
// suraggel é "aggelus" ou "sulfure".
func raceAncestries(name string) []filterOption {
	races, _, _ := book.CharacterCatalogs()
	for _, r := range races {
		if r.Name != name && r.ID != name {
			continue
		}
		outside := []filterOption{}
		for _, a := range r.Ancestries {
			outside = append(outside, filterOption{Value: a, Label: strings.ToUpper(a[:1]) + a[1:]})
		}
		return outside
	}
	return nil
}

// withVariantsRace é a entrada do `race-defs.json`, que é a que tem as
// variantes de habilidade.
func withVariantsRace(name string) *book.RaceForScreen {
	if race, found := book.RaceTraitsByKey()[name]; found {
		return &race
	}
	return nil
}

// attributeSavedChoices são os atributos que a pessoa já distribuiu.
func attributeSavedChoices(blob string) []string {
	var choice struct {
		FloatingPicks []string `json:"floatingPicks"`
	}
	if json.Unmarshal([]byte(blob), &choice) != nil {
		return nil
	}
	return choice.FloatingPicks
}

// savedAncestry é a metade escolhida, ou "".
func savedAncestry(blob string) string {
	var choice struct {
		Ancestry string `json:"ascendencia"`
	}
	if json.Unmarshal([]byte(blob), &choice) != nil {
		return ""
	}
	return choice.Ancestry
}

// sheet.OriginBenefitsOf são os benefícios MAIS o poder único.
//
// O catálogo guarda o poder único num campo à parte, e a ficha o trata como um
// dos dois que a pessoa leva (p85). Sem isso o poder da origem não aparece em
// lugar nenhum e não dá para escolhê-lo.
