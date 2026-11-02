package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// AS PENDÊNCIAS da ficha: o que ainda falta escolher.
//
// Pendência NÃO é erro — a ficha existe para ser preenchida aos poucos, e a
// forja promete por escrito "dá para criar assim e terminar na ficha". O que
// este arquivo faz é olhar a ficha e dizer, com a palavra que a pessoa lê, o que
// está faltando: o bônus de atributo da raça, os dois benefícios da origem, as
// vagas de poder abertas, o caminho, a devoção.
//
// # As REGRAS não estão aqui, e é isso que faz a recusa valer
//
// Quantas vagas o nível abre, quantos benefícios a origem dá, quais caminhos e
// quais deuses cada classe aceita: isso é `sheet/choices.go`. Elas leem o LIVRO
// e a FICHA, e a rota JSON as roda também — uma segunda cópia aqui divergiria na
// primeira regra nova, e a esquecida aceitaria o que a outra recusa.
//
// A pendência é a MESMA conta vista pelo outro lado: ela conta o que cabe e
// ainda não foi escolhido, onde a validação conta o que foi escolhido e não
// cabe.

// ── AS PENDÊNCIAS ────────────────────────────────────────────────────────────

// pendencia é uma escolha que ainda falta fazer.
type pendencia struct {
	// Source é `raca`, `origin` ou `class` — a aba do diálogo que a resolve.
	Source string
	Label  string
	// PowerID é o poder geral cuja escolha falta, quando a pendência é dessas.
	PowerID string
}

// sheetPendings são as escolhas que faltam, na ordem das abas.
func (s Scene) sheetPendings(dto sheet.CharacterDTO) []pendencia {
	outside := []pendencia{}
	outside = append(outside, s.attributeRacePending(dto)...)
	outside = append(outside, raceAbilityPendings(dto)...)
	outside = append(outside, originPending(dto)...)
	outside = append(outside, classPendings(dto)...)
	outside = append(outside, generalChoicePendings(dto)...)
	return outside
}

// generalChoicePendings é o poder geral que pede escolha e não tem nenhuma —
// o Foco em Perícia que veio da origem (Membro de Guilda) sem a perícia
// (ALE-510). O que se escolhe na lista de poderes já nasce com a escolha.
func generalChoicePendings(dto sheet.CharacterDTO) []pendencia {
	picks := sheet.PowerPicks(dto.PowerChoices)
	outside := []pendencia{}
	seen := map[string]bool{}
	for _, owned := range sheet.OwnedPowers(dto) {
		id := generalPowerIDOf(owned)
		power, chooses := book.PowerWithChoice(id)
		if id == "" || !chooses || seen[id] || len(picks[id]) > 0 {
			continue
		}
		seen[id] = true
		outside = append(outside, pendencia{
			Source: "classe", Label: power.Name + ": escolher " + strings.ToLower(power.Choice.Label),
			PowerID: id,
		})
	}
	return outside
}

// generalPowerIDOf é o id do poder geral que a linha possuída é — pelo id, ou
// pelo nome quando ela veio da origem com o id do benefício.
func generalPowerIDOf(owned sheet.OwnedPower) string {
	if _, found := book.GeneralPowers()[owned.ID]; found {
		return owned.ID
	}
	for id, power := range book.GeneralPowers() {
		if power.Name == owned.Name {
			return id
		}
	}
	return ""
}

// attributeRacePending é o `+1 ×3` do humano e a ascendência do suraggel.
//
// Ela PERGUNTA ao motor em vez de repetir a condição dele: o `resolveAtributoMod`
// já sabe quantas escolhas cada raça pede, que elas têm de ser distintas e qual
// atributo é proibido. Repetir as três regras aqui seria a asserção que se
// re-deriva da implementação, com a garantia de divergir no dia em que uma raça
// nova tiver uma quarta condição.
func (s Scene) attributeRacePending(dto sheet.CharacterDTO) []pendencia {
	if s.deps.Catalogs() == nil {
		return nil
	}
	outside := []pendencia{}
	for _, r := range dto.Races {
		if s.deps.Catalogs().RaceAttributeChoiceIsComplete(r.Race, dto.RaceAttributeChoices) {
			continue
		}
		outside = append(outside, pendencia{
			Source: "raca", Label: "Raça: distribuir o bônus de atributo de " + book.RaceLabel(r.Race),
		})
	}
	return outside
}

// raceAbilityPendings são as escolhas de habilidade de raça que faltam: "Lefou:
// Deformidade, 1 escolha por fazer", "Kliren: Vanguardista: escolher o Ofício".
func raceAbilityPendings(dto sheet.CharacterDTO) []pendencia {
	outside := []pendencia{}
	for _, block := range sheet.RaceAbilityChoices(dto) {
		missing := block.Ability.Choice.Count - len(block.Picks)
		if missing <= 0 {
			continue
		}
		label := book.RaceLabel(block.Race) + ": " + block.Ability.Name + ": "
		switch {
		case block.Ability.Choice.Kind == "craft":
			label += "escolher o Ofício"
		case missing == 1:
			label += "1 escolha por fazer"
		default:
			label += strconv.Itoa(missing) + " escolhas por fazer"
		}
		outside = append(outside, pendencia{Source: "raca", Label: label})
	}
	return outside
}

// originBenefitCeiling é quantos benefícios a origem COBRA: o que ela oferece,
// até o teto de dois. O Amnésico é a exceção que ensinou: "em vez de dois
// benefícios, recebe uma perícia e um poder escolhidos pelo mestre" (p88), e
// cobrar dois de quem só tem o poder único daria uma pendência que a pessoa
// não tem como resolver, para sempre.
func originBenefitCeiling(origin book.Origin) int {
	return min(len(sheet.OriginBenefitsOf(origin)), sheet.BenefitsOriginLimit)
}

func originPending(dto sheet.CharacterDTO) []pendencia {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	missing := originBenefitCeiling(origin) - len(sheet.UnmarshalStrings(dto.OriginChoices))
	if missing <= 0 {
		return nil
	}
	word := "benefícios"
	if missing == 1 {
		word = "benefício"
	}
	return []pendencia{{
		Source: "origem", Label: "Origem: " + strconv.Itoa(missing) + " " + word + " por escolher",
	}}
}

func classPendings(dto sheet.CharacterDTO) []pendencia {
	choices := sheet.ClassChoiceSelections(dto)
	used := sheet.UsedPowerSlots(dto)
	outside := []pendencia{}
	for _, class := range dto.Classes {
		if missing := sheet.PowerSlots(class.Level) - used; missing > 0 {
			word := "poderes"
			if missing == 1 {
				word = "poder"
			}
			outside = append(outside, pendencia{
				Source: "classe",
				Label:  class.ClassName + ": " + strconv.Itoa(missing) + " " + word + " por escolher",
			})
		}
		blob := choices[class.ClassName]
		if len(sheet.LevelPaths(class.ClassName, class.Level)) > 0 && blob.Path == "" {
			outside = append(outside, pendencia{
				Source: "classe", Label: class.ClassName + ": escolher caminho",
			})
		}
	}
	if sheet.DevotionRequired(dto.Classes) && (dto.God == nil || *dto.God == "") {
		outside = append(outside, pendencia{Source: "classe", Label: "Devoção: escolher o deus"})
	}
	for _, counted := range sheet.CountedChoices(dto) {
		// O que falta é o que AINDA cabe: com menos treinadas que o limite, a
		// pendência seria impossível de resolver. A opção que se repete (o
		// terreno) sempre cabe.
		available := counted.Limit
		if !counted.Repeats {
			available = min(available, len(counted.Offered))
		}
		missing := available - len(counted.Picks)
		if missing <= 0 {
			continue
		}
		word := " " + counted.Noun + "s por escolher"
		if missing == 1 {
			word = " " + counted.Noun + " por escolher"
		}
		outside = append(outside, pendencia{
			Source: "classe", Label: counted.Power.Name + ": " + strconv.Itoa(missing) + word,
		})
	}
	return outside
}

// writtenPendings é "3 escolhas pendentes", com o singular certo.
func writtenPendings(total int) string {
	if total == 1 {
		return "1 escolha pendente"
	}
	return strconv.Itoa(total) + " escolhas pendentes"
}

// writtenSources é o rótulo da aba do diálogo.
var writtenSources = map[string]string{"raca": "Raça", "origem": "Origem", "classe": "Classe"}
