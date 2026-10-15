package sheetui

import (
	"encoding/json"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// O ACERVO DE PODERES na tela: a lista que o `domain/sheet` monta, com a
// procedência escrita como a mesa a lê.
//
// # Por que a POSSE não mora mais aqui
//
// Ela morava — e, sendo função privada deste pacote, tinha exatamente um
// chamador. A mesma pergunta é do caso de uso (usar um poder que não é seu,
// receber a concessão de um poder que não é seu), e o `app/` não alcança o
// `serve/`. A lista desceu para o `domain/sheet` na ALE-426; aqui ficou a
// FRASE, que é o que esta camada tem a dizer.

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

// ownedPowersOf veste o acervo da ficha com a procedência escrita.
func ownedPowersOf(dto sheet.CharacterDTO) []ownedPower {
	owned := sheet.OwnedPowers(dto)
	outside := make([]ownedPower, 0, len(owned))
	for _, p := range owned {
		outside = append(outside, ownedPower{
			ID: p.ID, Name: p.Name, Detail: p.Detail, Page: p.Page,
			Source: writtenProvenance(p),
		})
	}
	return outside
}

// writtenProvenance é a procedência como a mesa a lê.
//
// O poder da Tormenta e o geral moram no mesmo catálogo e a mesa os trata como
// coisas diferentes; o divino já vem com a fonte escrita pelo catálogo, que
// junta os deuses que concedem o mesmo poder.
func writtenProvenance(p sheet.OwnedPower) string {
	switch p.Source {
	case sheet.PowerFromRace:
		return "Raça · " + p.Owner
	case sheet.PowerFromOrigin:
		return "Origem · " + p.Owner
	case sheet.PowerFromClass:
		return "Classe · " + p.Owner
	case sheet.PowerFromTormenta:
		return "Poder da Tormenta"
	case sheet.PowerFromGod:
		return p.Owner
	}
	return "Poder geral"
}

// powerGeneralSource separa o poder da TORMENTA do poder geral comum para o
// diálogo de ESCOLHER, que lista o catálogo inteiro e não passa pelo acervo.
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
