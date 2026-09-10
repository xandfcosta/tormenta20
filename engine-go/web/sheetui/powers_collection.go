package sheetui

import (
	"encoding/json"
	"strings"

	"t20engine/book"
	"t20engine/engine"
	"t20engine/sheet"
)

// O ACERVO DE PODERES de um personagem (ALE-272, fatia 8).
//
// Cinco procedências, e a ficha antiga já as tratava como uma lista só: as
// habilidades da RAÇA, os benefícios de ORIGEM escolhidos, as habilidades
// AUTOMÁTICAS de classe (as que o nível concede), e os poderes ESCOLHIDOS —
// de classe, gerais e da Tormenta.
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
	fora := []ownedPower{}
	fora = append(fora, raceAbilities(dto)...)
	fora = append(fora, originBenefits(dto)...)
	fora = append(fora, automaticAbilities(dto)...)
	fora = append(fora, chosenPowers(dto)...)
	return fora
}

// raceAbilities são as habilidades de cada raça da ficha.
//
// Quem lê o catálogo é o `RaceTraitsByKey` do dossiê — o MESMO `race-defs.json`, já
// indexado por id E por nome, porque o personagem guarda a raça por um dos dois.
// Um segundo leitor aqui seria uma terceira cópia da mesma decisão.
func raceAbilities(dto sheet.CharacterDTO) []ownedPower {
	fora := []ownedPower{}
	for _, r := range dto.Races {
		raca, tem := book.RaceTraitsByKey()[r.Race]
		if !tem {
			continue
		}
		for _, hab := range raca.Abilities {
			fora = append(fora, ownedPower{
				ID: hab.ID, Name: hab.Name, Detail: hab.Description,
				Source: "Raça · " + raca.Name,
			})
		}
	}
	return fora
}

// originBenefits são só os ESCOLHIDOS.
//
// A origem oferece mais benefícios do que o personagem leva (duas perícias e um
// poder, de uma lista maior), então listar todos mostraria como possuído o que
// ninguém escolheu.
func originBenefits(dto sheet.CharacterDTO) []ownedPower {
	origem, tem := book.Origins()[dto.Origin]
	if !tem {
		return nil
	}
	escolhidos := map[string]bool{}
	var ids []string
	if json.Unmarshal([]byte(dto.OriginChoices), &ids) == nil {
		for _, id := range ids {
			escolhidos[id] = true
		}
	}
	fora := []ownedPower{}
	for _, b := range sheet.OriginBenefitsOf(origem) {
		if !escolhidos[b.ID] {
			continue
		}
		fora = append(fora, ownedPower{
			ID: b.ID, Name: b.Name, Detail: b.Description,
			Source: "Origem · " + origem.Name, Page: origem.BookPage,
		})
	}
	return fora
}

// automaticAbilities são as que o NÍVEL concede, sem escolha.
//
// Quem decide a posse é o MOTOR (`engine.OwnedClassPowerIDs`), com a mesma
// regra que a derivação usa para somar os modificadores: nível alcançado, id
// escolhido, ou concedido por uma escolha de classe (o caminho do arcanista, o
// deus do clérigo). Uma segunda leitura aqui daria uma tela que mostra um poder
// que a ficha não soma — ou o contrário.
func automaticAbilities(dto sheet.CharacterDTO) []ownedPower {
	escolhas := sheet.ClassChoiceSelections(dto)
	fora := []ownedPower{}
	for _, classe := range dto.Classes {
		for _, poder := range book.ClassPowers() {
			if poder.ClassName != classe.ClassName || !automaticOwnership(poder, classe, escolhas) {
				continue
			}
			fora = append(fora, ownedPower{
				ID: poder.ID, Name: poder.Name, Detail: poder.Description,
				Source: "Classe · " + classe.ClassName, Page: poder.BookPage,
			})
		}
	}
	book.SortByName(fora, func(p ownedPower) string { return p.Name })
	return fora
}

// automaticOwnership pergunta ao MOTOR se a classe concede este poder.
//
// A lista de ESCOLHIDOS entra vazia de propósito: quem escolheu já aparece em
// `chosenPowers`, e passá-la aqui listaria o mesmo poder duas vezes.
func automaticOwnership(
	poder book.ClassPower, classe sheet.ClassDTO, escolhas map[string]engine.ClassChoiceSelections,
) bool {
	doMotor := &engine.ClassPower{
		ID: poder.ID, ClassName: poder.ClassName, Name: poder.Name,
		GrantedAtLevel: poder.GrantedAtLevel,
	}
	if poder.GrantedByChoice != nil {
		doMotor.GrantedByChoice = &engine.GrantedByChoice{
			Field: poder.GrantedByChoice.Field, Value: poder.GrantedByChoice.Value,
		}
	}
	return engine.OwnsClassPower(doMotor, int(classe.Level), nil, escolhas[classe.ClassName])
}

// chosenPowers lê o blob `classChoices` — o caminho do arcanista, o deus
// do clérigo. Blob torto vira mapa vazio: a aba não pode deixar de abrir porque
// uma linha do banco está errada.
// chosenPowers são os ids da coluna `classPowers` — poder de classe,
// poder geral ou poder da Tormenta, nessa ordem de busca.
func chosenPowers(dto sheet.CharacterDTO) []ownedPower {
	var ids []string
	if json.Unmarshal([]byte(dto.ClassPowers), &ids) != nil {
		return nil
	}
	fora := []ownedPower{}
	for _, id := range ids {
		if poder, tem := book.ClassPowers()[id]; tem {
			fora = append(fora, ownedPower{
				ID: poder.ID, Name: poder.Name, Detail: poder.Description,
				Source: "Classe · " + poder.ClassName, Page: poder.BookPage,
			})
			continue
		}
		if poder, tem := book.GeneralPowers()[id]; tem {
			fora = append(fora, ownedPower{
				ID: poder.ID, Name: poder.Name, Detail: poder.Description,
				Source: powerGeneralSource(poder), Page: poder.BookPage,
			})
		}
	}
	return fora
}

// powerGeneralSource separa o poder da TORMENTA do poder geral comum: eles
// moram no mesmo catálogo e a mesa os trata como coisas diferentes.
func powerGeneralSource(poder book.GeneralPower) string {
	if poder.Kind == "tormenta" {
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
func attributeRaceMod(nome string) *book.RaceAttribute {
	racas, _, _ := book.CharacterCatalogs()
	for i, r := range racas {
		if r.Name == nome || r.ID == nome {
			return &racas[i].AttributeMod
		}
	}
	return nil
}

// raceAncestries são as metades de uma raça que se escolhe na criação — o
// suraggel é "aggelus" ou "sulfure".
func raceAncestries(nome string) []filterOption {
	racas, _, _ := book.CharacterCatalogs()
	for _, r := range racas {
		if r.Name != nome && r.ID != nome {
			continue
		}
		fora := []filterOption{}
		for _, a := range r.Ascendencias {
			fora = append(fora, filterOption{Valor: a, Rotulo: strings.ToUpper(a[:1]) + a[1:]})
		}
		return fora
	}
	return nil
}

// withVariantsRace é a entrada do `race-defs.json`, que é a que tem as
// variantes de habilidade.
func withVariantsRace(nome string) *book.RaceForScreen {
	if raca, tem := book.RaceTraitsByKey()[nome]; tem {
		return &raca
	}
	return nil
}

// attributeSavedChoices são os atributos que a pessoa já distribuiu.
func attributeSavedChoices(blob string) []string {
	var escolha struct {
		FloatingPicks []string `json:"floatingPicks"`
	}
	if json.Unmarshal([]byte(blob), &escolha) != nil {
		return nil
	}
	return escolha.FloatingPicks
}

// savedAncestry é a metade escolhida, ou "".
func savedAncestry(blob string) string {
	var escolha struct {
		Ascendencia string `json:"ascendencia"`
	}
	if json.Unmarshal([]byte(blob), &escolha) != nil {
		return ""
	}
	return escolha.Ascendencia
}

// sheet.OriginBenefitsOf são os benefícios MAIS o poder único.
//
// O catálogo guarda o poder único num campo à parte, e a ficha o trata como um
// dos dois que a pessoa leva (p85) — a SPA já os juntava assim. Sem isso o poder
// da origem não aparece em lugar nenhum e não dá para escolhê-lo.
