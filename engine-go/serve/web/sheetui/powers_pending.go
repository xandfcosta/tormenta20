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
// vagas de poder abertas, o caminho, o devoto.
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
}

// sheetPendings são as escolhas que faltam, na ordem das abas.
func (s Scene) sheetPendings(dto sheet.CharacterDTO) []pendencia {
	outside := []pendencia{}
	outside = append(outside, s.attributeRacePending(dto)...)
	outside = append(outside, originPending(dto)...)
	outside = append(outside, classPendings(dto)...)
	return outside
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
			Source: "raca", Label: "Raça: distribuir o bônus de atributo de " + r.Race,
		})
	}
	return outside
}

func originPending(dto sheet.CharacterDTO) []pendencia {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	// A COBRANÇA É PELO QUE A ORIGEM OFERECE, e não pelo teto de dois. O
	// Amnésico é a exceção que ensinou: ele tem ZERO benefícios na lista, porque
	// "em vez de dois benefícios, recebe uma perícia e um poder escolhidos pelo
	// mestre" (p88) — cobrar dois dele daria uma pendência que a pessoa não tem
	// como resolver, para sempre.
	offers := len(sheet.OriginBenefitsOf(origin))
	ceiling := sheet.BenefitsOriginLimit
	if offers < ceiling {
		ceiling = offers
	}
	missing := ceiling - len(sheet.UnmarshalStrings(dto.OriginChoices))
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
	used := len(sheet.UnmarshalStrings(dto.ClassPowers))
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
		if len(sheet.ClassDevotees(class.ClassName)) > 0 && blob.Devotee == "" {
			outside = append(outside, pendencia{
				Source: "classe", Label: class.ClassName + ": escolher devoto",
			})
		}
		if len(sheet.LevelPaths(class.ClassName, class.Level)) > 0 && blob.Path == "" {
			outside = append(outside, pendencia{
				Source: "classe", Label: class.ClassName + ": escolher caminho",
			})
		}
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

func writtenSource(source string) string {
	if name, found := writtenSources[source]; found {
		return name
	}
	return strings.ToUpper(source)
}
