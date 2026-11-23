package sheet

import (
	"encoding/json"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O CONJURADOR é a ficha vista por quem pergunta sobre magia (ver o GLOSSARY, **conjurador**): aprender,
// conjurar, a aba Magias, o Foco em Magia e a saída de classe fazem a MESMA pergunta — quem lança esta magia,
// por qual tradição, até que círculo —, e a resposta mora aqui para não ter seis versões.
//
// Ele se monta uma vez por ficha porque o que não depende da magia é caro de refazer: o Aprender pergunta
// sobre as 198 do livro, e as escolas do bardo saem de uma varredura nos poderes de classe.
//
// # As classes que lançam pelas escolas que escolheram (p44, p61)
//
// O bardo lança "magias arcanas […] que pertençam a essas escolas", e o druida as divinas. A escolha mora no
// poder de classe que escolhe um NÚMERO de escolas (`Kind: "school"` com `Count`), e é ele que diz quem lança
// assim — não há lista de classes aqui. O `Count` é o que separa: o Especialista em Escola do arcanista também
// escolhe escola, mas para somar na CD dela (p38), e o arcanista lança toda arcana.
//
// A TRADIÇÃO da magia sai das classes que o verbete lista, pela `list` de cada uma: a Explosão de Chamas lista
// só o Arcanista, que é arcano, e por isso o bardo de evocação a lança. Pela lista inteira, não: ela diz
// "Bardo" só em parte das arcanas.

// Caster é o conjurador da ficha. O valor zero não conjura nada; monte-o com `CasterOf`.
type Caster struct {
	// classes são as classes da ficha que conjuram, na ordem da ficha: o empate da tradição resolve pela
	// primeira.
	classes []casterClass
	// spells é o grimório, que a conta da memorização lê.
	spells []SpellDTO
}

type casterClass struct {
	name  string
	level int64
	// list é a tradição da classe, `arcana` ou `divina`.
	list string
	// reach é o maior círculo que a classe alcança no nível dela.
	reach int
	// schools são as escolas que a classe escolheu lançar, e picksSchools diz se ela as escolhe: sem isso, a
	// classe segue a lista do verbete.
	schools      []string
	picksSchools bool
	// memorizes é o arcanista do caminho mago, o único que memoriza (p37).
	memorizes bool
}

// CasterOf monta o conjurador da ficha. A saída de uma classe pergunta ao conjurador das classes que SOBRAM,
// e por isso ele lê `dto.Classes` em vez de guardar a ficha.
//
//	sheet.CasterOf(dto).Spell("bola-de-fogo").CircleReach // 3, no Arcanista 9
func CasterOf(dto CharacterDTO) Caster {
	progressions := book.SpellProgressions()
	picks := PowerPicks(dto.PowerChoices)
	var paths map[string]struct {
		Path string `json:"caminho"`
	}
	_ = json.Unmarshal([]byte(dto.ClassChoices), &paths)
	caster := Caster{spells: dto.Spells}
	for _, entry := range dto.Classes {
		prog, casts := progressions[entry.ClassName]
		if !casts {
			continue
		}
		class := casterClass{
			name: entry.ClassName, level: entry.Level, list: prog.List,
			reach:     HighestCastableCircle([]ClassDTO{entry}),
			memorizes: entry.ClassName == "Arcanista" && paths["Arcanista"].Path == "mago",
		}
		if power := schoolChoicePower(entry.ClassName); power != "" {
			class.picksSchools, class.schools = true, picks[power]
		}
		caster.classes = append(caster.classes, class)
	}
	return caster
}

// SpellOnTheSheet é o que uma magia é para esta ficha.
type SpellOnTheSheet struct {
	// Known diz se a magia existe no livro. A desconhecida não tem recusa nem aviso: quem a recusa é quem
	// procura no livro.
	Known bool
	Spell book.Spell
	// Casting são as classes da ficha que lançam a magia, na ordem da ficha — e não a lista do verbete: o bardo
	// de evocação lança a Bola de Fogo, que lista só o Arcanista (p44).
	Casting []string
	// Tradition é a lista de onde a magia vem para esta ficha, vazia quando nenhuma classe a lança. Ela não é
	// da magia: a Luz (p197) é arcana na mão do arcanista e divina na do clérigo. EMPATE resolve pelo maior
	// nível, a mesma regra do `engine.SpellPmLimit` para "a classe que fornece a habilidade" (p224): o app
	// não guarda de qual lista a magia foi aprendida, e uma segunda regra daria duas respostas na mesma ficha.
	Tradition string
	// Class é a classe com que a ficha lança a magia, a de maior nível entre as de `Casting`, vazia quando
	// nenhuma a lança: é de onde sai a `Tradition`, e o que o aprimoramento "Apenas Druidas" pergunta.
	Class string
	// CircleReach é o maior círculo que a ficha alcança PARA ESTA MAGIA (p170-171), em ordem:
	//
	//   - o das classes que lançam a magia, e não o maior entre todas: o Arcanista 9 / Clérigo 1 alcança o 3º
	//     círculo nas arcanas e o 1º nas divinas;
	//   - sem nenhuma delas, o das classes que conjuram: um poder ensinou a magia de fora da lista (o Aspecto
	//     da Primavera do druida, p61, aprende encantamento "arcana ou divina");
	//   - sem classe que conjure, o da própria magia, que veio de fora das classes: a magia concedida por poder
	//     (Totem Espiritual, p42) se conjura no círculo dela, e sem esse piso o bárbaro com Totem não
	//     conjuraria a magia que o poder lhe deu.
	CircleReach int
	// AboveTheCircle é a frase da magia acima do `CircleReach`, ou vazia: a recusa de aprender e de conjurar,
	// e o que a aba Magias escreve na linha trancada.
	AboveTheCircle string
	// OffTheList é o aviso da magia que nenhuma classe da ficha lança pela lista (p170), ou vazio. Não recusa
	// nada: um poder pode ensiná-la, e quem confere de onde ela veio é o mestre.
	OffTheList string
	// NeedsMemorizing diz que a ficha só lança a magia memorizada: todas as classes que a lançam são o mago
	// (p37). No multiclasse, a Luz que o clérigo também lança se conjura sem memorizar.
	NeedsMemorizing bool
}

// Spell é o que a magia `spellID` é para esta ficha.
//
//	sheet.CasterOf(arcanista4).Spell("bola-de-fogo").AboveTheCircle // "Bola de Fogo é do 2º círculo, e esta ficha alcança até o 1º (p170)"
func (c Caster) Spell(spellID string) SpellOnTheSheet {
	spell, found := book.SpellByID(spellID)
	if !found {
		return SpellOnTheSheet{Spell: book.Spell{ID: spellID, Name: spellID}}
	}
	out := SpellOnTheSheet{Known: true, Spell: spell, Casting: []string{}}
	var casting []casterClass
	for _, class := range c.classes {
		if class.casts(spell) {
			casting = append(casting, class)
			out.Casting = append(out.Casting, class.name)
		}
	}
	out.Class, out.Tradition = castingClassOf(casting)
	out.NeedsMemorizing = len(casting) > 0 && !slices.ContainsFunc(casting, func(class casterClass) bool { return !class.memorizes })
	if len(casting) == 0 {
		out.OffTheList = "fora da lista das suas classes: só vale se um poder a ensinou"
	}
	switch {
	case len(casting) > 0:
		out.CircleReach = highestReach(casting)
	case len(c.classes) > 0:
		out.CircleReach = highestReach(c.classes)
	default:
		out.CircleReach = spell.Circle
	}
	if spell.Circle > out.CircleReach {
		out.AboveTheCircle = fmt.Sprintf("%s é do %dº círculo, e esta ficha alcança até o %dº (p170)",
			spell.Name, spell.Circle, out.CircleReach)
	}
	return out
}

// Memorizes diz se a ficha tem o mago, o único que memoriza (p37).
func (c Caster) Memorizes() bool {
	return slices.ContainsFunc(c.classes, func(class casterClass) bool { return class.memorizes })
}

// Memorization é quantas magias o mago memorizou e quantas ele pode: metade das que conhece, arredondando
// para baixo — "se conhece 7 magias, escolhe 3" (p37). Contam só as magias que a ficha lança apenas pelo mago
// (`NeedsMemorizing`).
//
//	memorized, limit := sheet.CasterOf(dto).Memorization() // 2, 3 no mago com 7 magias e 2 memorizadas
func (c Caster) Memorization() (memorized, limit int) {
	known := 0
	for _, learned := range c.spells {
		if !c.Spell(learned.CatalogSpellID).NeedsMemorizing {
			continue
		}
		known++
		if learned.Prepared {
			memorized++
		}
	}
	return memorized, known / 2
}

// MemorizeRefusal é a frase que recusa memorizar mais esta magia, ou vazia. Desmemorizar não passa por aqui:
// sempre se pode. A memorizada acima do limite, gravada antes, fica, e só a próxima é recusada.
func (c Caster) MemorizeRefusal(spellID string) string {
	spell := c.Spell(spellID)
	if !spell.NeedsMemorizing {
		return fmt.Sprintf("só o mago memoriza magia (p37), e esta ficha conjura %s sem memorizar", spell.Spell.Name)
	}
	if memorized, limit := c.Memorization(); memorized >= limit {
		return fmt.Sprintf("o mago memoriza até metade das magias que conhece: já são %d de %d (p37)", memorized, limit)
	}
	return ""
}

// ReachesTheCircle diz se a ficha alcança o círculo da magia — "uma magia que possa lançar" (Foco em Magia,
// p131). Magia desconhecida não alcança.
func (s SpellOnTheSheet) ReachesTheCircle() bool {
	return s.Known && s.AboveTheCircle == ""
}

// At é a magia como as três contas do motor a recebem (a CD, o limite e o custo de PM, `ForSpell`).
func (s SpellOnTheSheet) At() engine.SpellAt {
	return engine.SpellAt{
		Tradition: s.Tradition, School: s.Spell.School, Range: s.Spell.Range, Spell: s.Spell.ID,
		Classes: s.Casting,
	}
}

// MainTradition é a tradição da classe conjuradora de maior nível, vazia sem classe que conjure: a que as
// caixas de CD e de limite de PM do Combate resumem.
func (c Caster) MainTradition() string {
	_, tradition := castingClassOf(c.classes)
	return tradition
}

// classOnlyCasts é o que cada marca "Apenas …" do livro pergunta sobre a magia lançada.
var classOnlyCasts = map[string]func(SpellOnTheSheet) bool{
	"arcanos": func(s SpellOnTheSheet) bool { return s.Tradition == "arcana" },
	"divinos": func(s SpellOnTheSheet) bool { return s.Tradition == "divina" },
	"druidas": func(s SpellOnTheSheet) bool { return s.Class == "Druida" },
}

// CastsAs diz se a ficha lança a magia como a marca "Apenas Arcanos / Divinos / Druidas" de um aprimoramento
// pede (`catalog.Augment.ClassOnly`, p187 e p197). Sem marca, sim; com uma marca que o mapa não conhece, não —
// e o `TestEveryClassOnlyMarkIsUnderstood` reprova a marca nova antes de ela chegar aqui.
//
//	sheet.CasterOf(clerigo).Spell("luz").CastsAs("arcanos") // false: a Luz do clérigo é divina
func (s SpellOnTheSheet) CastsAs(classOnly string) bool {
	if classOnly == "" {
		return true
	}
	casts, known := classOnlyCasts[classOnly]
	return known && casts(s)
}

// casts diz se a classe lança a magia. A que não escolhe escola segue a lista do verbete; a que escolhe
// lança a da escola escolhida, se o verbete a lista numa classe da mesma tradição.
func (class casterClass) casts(spell book.Spell) bool {
	if !class.picksSchools {
		return slices.Contains(spell.Classes, class.name)
	}
	progressions := book.SpellProgressions()
	sameTradition := slices.ContainsFunc(spell.Classes, func(listed string) bool {
		prog, casts := progressions[listed]
		return casts && prog.List == class.list
	})
	return sameTradition && slices.Contains(class.schools, spell.School)
}

// castingClassOf é a classe de maior nível entre estas e a lista dela, vazias sem nenhuma.
func castingClassOf(classes []casterClass) (name, tradition string) {
	best := int64(0)
	for _, class := range classes {
		if class.level > best {
			best, name, tradition = class.level, class.name, class.list
		}
	}
	return name, tradition
}

func highestReach(classes []casterClass) int {
	best := 0
	for _, class := range classes {
		best = max(best, class.reach)
	}
	return best
}

// schoolChoicePower é o poder em que a classe escolhe as escolas que lança, ou vazio quando ela não escolhe.
func schoolChoicePower(class string) string {
	for id, power := range book.ClassPowers() {
		if power.ClassName == class && power.Choice != nil && power.Choice.Kind == "school" && power.Choice.Count > 0 {
			return id
		}
	}
	return ""
}
