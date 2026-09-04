package sheetui

import (
	"strconv"
	"strings"

	"t20engine/book"
	"t20engine/sheet"
)

// AS PENDÊNCIAS da ficha: o que ainda falta escolher (ALE-272, fatia 8).
//
// Pendência NÃO é erro — a ficha existe para ser preenchida aos poucos, e a
// forja promete por escrito "dá para criar assim e terminar na ficha"
// (ALE-169). O que este arquivo faz é olhar a ficha e dizer, com a palavra que a
// pessoa lê, o que está faltando: o bônus de atributo da raça, os dois
// benefícios da origem, as vagas de poder abertas, o caminho, o devoto.
//
// # As REGRAS não estão aqui, e é isso que faz a recusa valer
//
// Quantas vagas o nível abre, quantos benefícios a origem dá, quais caminhos e
// quais deuses cada classe aceita: isso é `sheet/choices.go` desde a ALE-278.
// Elas leem o LIVRO e a FICHA, e a rota JSON as roda também — uma segunda cópia
// aqui divergiria na primeira regra nova, e a esquecida aceitaria o que a outra
// recusa. Até a ALE-272 elas eram 363 linhas de `shared/rules/abilities-*.ts` e
// o `handleUpdateAbilities` gravava os cinco blobs sem conferir NADA: um pedido
// montado à mão punha vinte poderes num personagem de nível 1, e o motor somava
// os modificadores de todos.
//
// A pendência é a MESMA conta vista pelo outro lado — ela conta o que cabe e
// ainda não foi escolhido, onde a validação conta o que foi escolhido e não
// cabe.

// ── AS PENDÊNCIAS ────────────────────────────────────────────────────────────

// pendencia é uma escolha que ainda falta fazer.
type pendencia struct {
	// Fonte é `raca`, `origem` ou `classe` — a aba do diálogo que a resolve.
	Fonte  string
	Rotulo string
}

// asPendenciasDaFicha são as escolhas que faltam, na ordem das abas.
func (s Scene) sheetPendings(dto sheet.CharacterDTO) []pendencia {
	fora := []pendencia{}
	fora = append(fora, s.attributeRacePending(dto)...)
	fora = append(fora, originPending(dto)...)
	fora = append(fora, classPendings(dto)...)
	return fora
}

// aPendenciaDoAtributoDeRaca é o `+1 ×3` do humano e a ascendência do suraggel.
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
	fora := []pendencia{}
	for _, r := range dto.Races {
		if s.deps.Catalogs().RaceAttributeChoiceIsComplete(r.Race, dto.RaceAttributeChoices) {
			continue
		}
		fora = append(fora, pendencia{
			Fonte: "raca", Rotulo: "Raça: distribuir o bônus de atributo de " + r.Race,
		})
	}
	return fora
}

func originPending(dto sheet.CharacterDTO) []pendencia {
	origem, tem := book.Origins()[dto.Origin]
	if !tem {
		return nil
	}
	// A COBRANÇA É PELO QUE A ORIGEM OFERECE, e não pelo teto de dois. O
	// Amnésico é a exceção que ensinou: ele tem ZERO benefícios na lista, porque
	// "em vez de dois benefícios, recebe uma perícia e um poder escolhidos pelo
	// mestre" (p88) — cobrar dois dele daria uma pendência que a pessoa não tem
	// como resolver, para sempre.
	oferece := len(sheet.OriginBenefitsOf(origem))
	teto := sheet.BenefitsOriginLimit
	if oferece < teto {
		teto = oferece
	}
	faltam := teto - len(sheet.UnmarshalStrings(dto.OriginChoices))
	if faltam <= 0 {
		return nil
	}
	palavra := "benefícios"
	if faltam == 1 {
		palavra = "benefício"
	}
	return []pendencia{{
		Fonte: "origem", Rotulo: "Origem: " + strconv.Itoa(faltam) + " " + palavra + " por escolher",
	}}
}

func classPendings(dto sheet.CharacterDTO) []pendencia {
	escolhas := sheet.ClassChoiceSelections(dto)
	usadas := len(sheet.UnmarshalStrings(dto.ClassPowers))
	fora := []pendencia{}
	for _, classe := range dto.Classes {
		if faltam := sheet.PowerSlots(classe.Level) - usadas; faltam > 0 {
			palavra := "poderes"
			if faltam == 1 {
				palavra = "poder"
			}
			fora = append(fora, pendencia{
				Fonte:  "classe",
				Rotulo: classe.ClassName + ": " + strconv.Itoa(faltam) + " " + palavra + " por escolher",
			})
		}
		blob := escolhas[classe.ClassName]
		if len(sheet.ClassDevotees(classe.ClassName)) > 0 && blob.Devoto == "" {
			fora = append(fora, pendencia{
				Fonte: "classe", Rotulo: classe.ClassName + ": escolher devoto",
			})
		}
		if len(sheet.LevelPaths(classe.ClassName, classe.Level)) > 0 && blob.Caminho == "" {
			fora = append(fora, pendencia{
				Fonte: "classe", Rotulo: classe.ClassName + ": escolher caminho",
			})
		}
	}
	return fora
}

// asPendenciasEscritas é "3 escolhas pendentes", com o singular certo.
func writtenPendings(total int) string {
	if total == 1 {
		return "1 escolha pendente"
	}
	return strconv.Itoa(total) + " escolhas pendentes"
}

// aFonteEscrita é o rótulo da aba do diálogo.
var writtenSources = map[string]string{"raca": "Raça", "origem": "Origem", "classe": "Classe"}

func writtenSource(fonte string) string {
	if nome, tem := writtenSources[fonte]; tem {
		return nome
	}
	return strings.ToUpper(fonte)
}
