package sheet

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"t20engine/domain/book"
)

// AS DUAS REGRAS DE CONJURAÇÃO que a ficha e a rota JSON leem juntas (ALE-278).
//
// Elas moravam junto da conjuração, no `serve/api`, e a cena da ficha as lia de
// lá enquanto tudo era um pacote só. Vieram para cá pela dependência: uma é uma
// tabela sem dependência nenhuma, a outra pergunta sobre `[]ClassDTO` — nenhuma
// das duas toca HTTP, catálogo ou banco. Quem as chama hoje é o
// `app/character/cast.go`.
//
// É a mesma família do `equip.go` e do `temp_pools.go`, que saíram na fatia
// anterior: regra da ficha hospedada no `api` por acidente de história.

// SpellBasePmCost é a Tabela 4-1, "Custo de Magias" (livro p170).
var SpellBasePmCost = map[int]int{0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15}

// alwaysPrepareClasses são as classes que preparam magia sempre.
var alwaysPrepareClasses = map[string]bool{"Clérigo": true, "Druida": true}

// RequiresPreparation diz se esta ficha precisa PREPARAR a magia antes de
// conjurar: Clérigo e Druida sempre, e o Arcanista quando o caminho é "mago".
func RequiresPreparation(classes []ClassDTO, classChoicesRaw string) bool {
	hasArcanist := false
	for _, c := range classes {
		if alwaysPrepareClasses[c.ClassName] {
			return true
		}
		if c.ClassName == "Arcanista" {
			hasArcanist = true
		}
	}
	if !hasArcanist {
		return false
	}
	var choices map[string]struct {
		Path string `json:"caminho"`
	}
	_ = json.Unmarshal([]byte(classChoicesRaw), &choices)
	return choices["Arcanista"].Path == "mago"
}

// HighestCastableCircle é o maior círculo que o personagem alcança.
//
// O piso é o círculo da PRÓPRIA magia, e essa é a regra que não se adivinha: uma
// magia concedida por poder (Totem Espiritual, p42) é conjurável por quem não
// tem classe nenhuma de conjurador — no círculo dela, e só nele. Sem o piso, um
// bárbaro com Totem não conseguiria conjurar a magia que o poder lhe deu.
func HighestCastableCircle(classes []ClassDTO, spellCircle int) int {
	best := spellCircle
	progressions := book.SpellProgressions()
	for _, entry := range classes {
		prog, casts := progressions[entry.ClassName]
		if !casts {
			continue
		}
		for circle := 1; circle <= prog.MaxCircle; circle++ {
			level := prog.UnlockLevel[strconv.Itoa(circle)]
			if level == nil || entry.Level < int64(*level) {
				continue
			}
			if circle > best {
				best = circle
			}
		}
	}
	return best
}

// SpellCircleReach é o maior círculo que a ficha alcança PARA ESTA MAGIA, que é o que aprender e conjurar
// conferem e o que libera o aprimoramento de círculo alto (p171). Em ordem:
//
//   - o das classes que lançam a magia, e não o maior entre todas: o Arcanista 9 / Clérigo 1 alcança o 3º
//     círculo nas arcanas e o 1º nas divinas;
//   - sem nenhuma delas, o das classes que conjuram: um poder ensinou a magia de fora da lista (o Aspecto da
//     Primavera do druida, p61, aprende encantamento "arcana ou divina");
//   - sem classe que conjure, o da própria magia, que veio de fora das classes — o mesmo piso do
//     `HighestCastableCircle`.
//
// Magia desconhecida alcança o círculo 0.
//
//	sheet.SpellCircleReach(arcanista9clerigo1, "curar-ferimentos") // 1
func SpellCircleReach(dto CharacterDTO, spellID string) int {
	spell, found := book.SpellByID(spellID)
	if !found {
		return 0
	}
	casting := CastingClasses(dto, dto.Classes, spell.School, spell.Classes)
	classes := classesNamed(dto.Classes, casting)
	if len(classes) == 0 {
		classes = slices.DeleteFunc(slices.Clone(dto.Classes), func(c ClassDTO) bool { return !IsCasterClass(c.ClassName) })
	}
	if len(classes) == 0 {
		return spell.Circle
	}
	return HighestCastableCircle(classes, 0)
}

// classesNamed são as entradas de `classes` cujo nome está em `names`.
func classesNamed(classes []ClassDTO, names []string) []ClassDTO {
	out := []ClassDTO{}
	for _, c := range classes {
		if slices.Contains(names, c.ClassName) {
			out = append(out, c)
		}
	}
	return out
}

// CanCastCircleOf diz se a ficha alcança o círculo da magia — "uma magia que
// possa lançar" (Foco em Magia, p131). Magia desconhecida não alcança.
func CanCastCircleOf(dto CharacterDTO, spellID string) bool {
	spell, found := book.SpellByID(spellID)
	return found && spell.Circle <= SpellCircleReach(dto, spellID)
}

// AboveTheCircle é a frase da magia que passa do círculo que a ficha alcança para ela, ou vazia. É a
// recusa de aprender e de conjurar, e o que a aba Magias escreve na linha trancada. Magia desconhecida
// é vazia: quem a recusa é quem procura no livro.
//
//	sheet.AboveTheCircle(arcanista4, "bola-de-fogo") // "Bola de Fogo é do 2º círculo, e esta ficha alcança até o 1º (p170)"
func AboveTheCircle(dto CharacterDTO, spellID string) string {
	spell, found := book.SpellByID(spellID)
	if !found {
		return ""
	}
	reach := SpellCircleReach(dto, spellID)
	if spell.Circle <= reach {
		return ""
	}
	return fmt.Sprintf("%s é do %dº círculo, e esta ficha alcança até o %dº (p170)", spell.Name, spell.Circle, reach)
}

// OffTheList é o aviso da magia que nenhuma classe da ficha lança pela lista (p170), ou vazio. Não
// recusa nada: um poder pode ensiná-la, e quem confere de onde ela veio é o mestre.
func OffTheList(dto CharacterDTO, spellID string) string {
	spell, found := book.SpellByID(spellID)
	if !found || len(CastingClasses(dto, dto.Classes, spell.School, spell.Classes)) > 0 {
		return ""
	}
	return "fora da lista das suas classes: só vale se um poder a ensinou"
}

// IsCasterClass diz se a classe conjura.
func IsCasterClass(name string) bool {
	_, casts := book.SpellProgressions()[name]
	return casts
}

// SpellTradition é a LISTA de onde esta magia vem para ESTE personagem:
// `arcana`, `divina`, ou vazio quando nenhuma classe dele a concede.
//
// A tradição não é propriedade da magia — a Luz (p197) é arcana na mão do
// arcanista e divina na do clérigo, e o catálogo a declara com as quatro
// classes que a conhecem. Quem tem lista é a CLASSE (`spellcasting.list`), e
// por isso a pergunta precisa das duas pontas.
//
// EMPATE RESOLVE PELO MAIOR NÍVEL, que é a mesma regra que o
// `engine.SpellPmLimit` já usa para "a classe que fornece a habilidade" (p224):
// um Arcanista 20/Clérigo 10 que conjura uma magia das duas listas a conjura
// como arcanista. O app não guarda de qual lista a magia foi aprendida, e
// inventar uma segunda regra aqui faria a mesma pergunta ter duas respostas na
// mesma ficha.
//
// @example SpellTradition(classes, []string{"Arcanista"}) // "arcana"
func SpellTradition(classes []ClassDTO, spellClasses []string) string {
	grants := map[string]bool{}
	for _, name := range spellClasses {
		grants[name] = true
	}
	progressions := book.SpellProgressions()
	best, tradition := int64(0), ""
	for _, entry := range classes {
		prog, casts := progressions[entry.ClassName]
		if !casts || !grants[entry.ClassName] || entry.Level <= best {
			continue
		}
		best, tradition = entry.Level, prog.List
	}
	return tradition
}
