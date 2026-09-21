package finder

import (
	"cmp"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/search"
	"t20engine/serve/web/routes"
)

// O BUSCADOR DO LIVRO: ⌃K abre uma caixa que procura em todas as entradas do
// livro de uma vez — criaturas, condições, magias, poderes e itens —, sem tirar
// a mão do teclado e sem saber em qual ferramenta a coisa mora.
//
// A busca não é nova: o servidor já varre os catálogos em `book.Catalogs` e o
// bestiário em `book.Creatures`. O que este arquivo acrescenta é RANQUEAR e dar
// um DESTINO a cada achado — aqui a lista é curta e cada linha tem de saber
// para onde levar.
//
// Decisão do dono: o Enter NAVEGA para a cena com a entrada aberta, em vez de
// desenhar o verbete dentro do próprio diálogo — reusa as cenas que já existem
// em vez de manter um terceiro desenho da mesma regra.
//
// "Com a entrada ABERTA" é `?entrada=<id>` e não uma busca pelo nome: cair numa
// lista de oito grupos para achar o que se escolheu é o oposto de escolher.

// hitsByGroup corta cada grupo, e o corte é DITO na tela ("+12").
//
// Seis porque a lista inteira tem de caber sem rolagem numa janela de laptop
// com cinco grupos possíveis. Corte silencioso seria pior que corte nenhum:
// quem não vê o que ficou de fora conclui que não existe.
const hitsByGroup = 6

// finderHit é uma linha do resultado.
type finderHit struct {
	Name        string
	Detail      string
	Destination string
	// Page é a do livro, e ZERO significa "o catálogo não sabe" — a linha sai
	// sem número em vez de sair com "p0".
	Page int
	// point não vai para a tela: ele é a ORDEM, e mostrá-lo convidaria a
	// discutir a nota em vez do resultado.
	point int
}

// finderGroup é um catálogo com o que sobrou, já cortado.
type finderGroup struct {
	Label    string
	Findings []finderHit
	// Total é antes do corte — é ele que escreve o "+12".
	Total int
	// More é para onde o "+12" leva: a cena da ferramenta com a MESMA busca.
	// Corte com saída, e não corte que informa e abandona.
	More string
}

func (g finderGroup) Cortados() int { return g.Total - len(g.Findings) }

type finderView struct {
	Search string
	Groups []finderGroup
	// Findings é o total ANTES dos cortes de grupo.
	Findings int
	// ByText diz que estes achados vieram da segunda passada — nenhum NOME
	// casou, e o que está na lista apenas MENCIONA o termo. A tela avisa: sem
	// isso, uma lista que não contém o que se pediu parece a lista errada.
	ByText bool
}

func (v finderView) Buscando() bool { return strings.TrimSpace(v.Search) != "" }

// searchTheBook monta o resultado do ⌃K.
func searchTheBook(search string) finderView {
	v := buildHits(search, pelosNomes)
	if v.Findings > 0 || !v.Buscando() {
		return v
	}
	// SEGUNDA PASSADA: o corpo da regra só entra quando NOME nenhum casou. Com
	// ele valendo sempre, "abal" afogava a condição "Abalado" entre as centenas
	// de poderes cujo TEXTO a menciona.
	//
	// Quem digita "abal" procura o verbete; quem digita "chance de falha" não
	// sabe o nome, e é para essa pessoa que a passada existe.
	v = buildHits(search, tambemPeloTexto)
	v.ByText = v.Findings > 0
	return v
}

// pelosNomes e tambemPeloTexto nomeiam as duas passadas. Um booleano cru na
// chamada (`foundGroup(..., true)`) não diz nada de dentro do `foundGroup`.
const (
	pelosNomes      = false
	tambemPeloTexto = true
)

func buildHits(search string, byText bool) finderView {
	v := finderView{Search: search}
	if !v.Buscando() {
		return v
	}
	a := book.Catalogs()
	races, classes, gods := book.CharacterCatalogs()
	for _, g := range []finderGroup{
		foundGroup("Condições", a.Conditions, search, byText, routes.MasterSearch("condicoes", search), book.ConditionFields, conditionHit),
		foundGroup("Criaturas", book.Creatures(), search, byText, routes.MasterBestiarySearch(search), entryFields, entryHit),
		foundGroup("Magias", a.Spells, search, byText, routes.MasterSearch("magias", search), book.SpellFields, spellHit),
		foundGroup("Poderes", a.Powers, search, byText, routes.MasterSearch("poderes", search), book.PowerFields, powerHit),
		foundGroup("Itens", a.Items, search, byText, routes.MasterSearch("itens", search), book.ItemFields, itemHit),
		foundGroup("Efeitos", book.EffectKinds(), search, byText, routes.MasterSearch("efeitos", search), book.EffectFields, effectHit),
		foundGroup("Escolas", book.SpellSchools(), search, byText, routes.MasterSearch("escolas", search), book.SchoolFields, schoolHit),
		foundGroup("Perícias", book.Expertises(), search, byText, routes.MasterSearch("pericias", search), book.ExpertiseFields, expertiseHit),
		foundGroup("Raças", races, search, byText, routes.MasterSearch("racas", search), book.RaceFields, raceHit),
		foundGroup("Classes", classes, search, byText, routes.MasterSearch("classes", search), book.ClassFields, classHit),
		foundGroup("Deuses", gods, search, byText, routes.MasterSearch("deuses", search), book.GodFields, deityHit),
	} {
		if g.Total == 0 {
			continue
		}
		v.Findings += g.Total
		v.Groups = append(v.Groups, g)
	}
	sortByRelevance(v.Groups)
	return v
}

// sortByRelevance põe na frente o grupo que tem o MELHOR achado.
//
// A ordem da fileira de abas é a certa para NAVEGAR — condição primeiro porque
// é a consulta do combate, os três do personagem no fim — e a errada para
// BUSCAR: com ela fixa, o verbete "Medo" (nome inteiro, pontuação máxima) sai
// abaixo de criaturas que só têm a palavra no nome.
//
// Estável, então a ordem da fileira continua valendo no EMPATE: dois grupos com
// achados igualmente bons saem na ordem de sempre.
func sortByRelevance(groups []finderGroup) {
	slices.SortStableFunc(groups, func(a, b finderGroup) int {
		return cmp.Compare(b.melhorPonto(), a.melhorPonto())
	})
}

// melhorPonto é a nota do primeiro achado — a lista já vem ordenada pelo
// `bestFirst`, então o primeiro é o melhor.
func (g finderGroup) melhorPonto() int {
	if len(g.Findings) == 0 {
		return 0
	}
	return g.Findings[0].point
}

// foundGroup pontua, ordena e corta um catálogo.
//
// `fields` é a MESMA função que a cena dos catálogos usa para filtrar, e reusá-la
// é o que faz as duas superfícies concordarem sobre o que é buscável — uma
// segunda lista de campos aqui divergiria no dia em que alguém acrescentasse um.
func foundGroup[T any](
	label string, list []T, query string, byText bool, more string,
	fields func(T) []string, asFound func(T) finderHit,
) finderGroup {
	g := finderGroup{Label: label, More: more}
	for _, e := range list {
		// A linha é montada ANTES de saber se ela passa, e é deliberado: pegar o
		// nome de `campos(e)[0]` seria depender de uma ordem que nada garante, e
		// no dia em que um `entryFields` mudasse de ordem a pontuação passaria a
		// medir a descrição — em silêncio, com a lista continuando a sair.
		a := asFound(e)
		a.point = search.Score(a.Name, query)
		if a.point == 0 && byText {
			a.point = search.ScoreText(fields(e), query)
		}
		if a.point == 0 {
			continue
		}
		g.Findings = append(g.Findings, a)
	}
	g.Total = len(g.Findings)
	slices.SortStableFunc(g.Findings, bestFirst)
	if len(g.Findings) > hitsByGroup {
		g.Findings = g.Findings[:hitsByGroup]
	}
	return g
}

// bestFirst: pontuação alta na frente e, empatados, o nome mais CURTO.
//
// O desempate por tamanho não é estético: com "fogo", "Bola de Fogo" e
// "Explosão de Fogo Congelante" pontuam igual, e o nome curto é o que a pessoa
// tem mais chance de estar procurando. O terceiro critério é o nome, para a
// ordem não depender da ordem de leitura do catálogo.
func bestFirst(a, b finderHit) int {
	if d := cmp.Compare(b.point, a.point); d != 0 {
		return d
	}
	if d := cmp.Compare(len(a.Name), len(b.Name)); d != 0 {
		return d
	}
	return cmp.Compare(a.Name, b.Name)
}

// ── de cada catálogo para uma linha ──────────────────────────────────────────

// entryFields existe para o verbete do bestiário passar pelo mesmo `foundGroup`
// que os outros catálogos: a cena dele filtra por outro caminho
// (`book.FilterCreatures`), que não serve aqui.
func entryFields(m book.Entry) []string {
	return append([]string{m.Name, book.TypeName(m.Kind)}, m.SpecialAbilities...)
}

func conditionHit(c book.Condition) finderHit {
	detail := "Condição"
	if c.UpgradesTo != "" {
		detail = "Condição · agrava para " + book.ConditionName(c.UpgradesTo)
	}
	return finderHit{Name: c.Name, Detail: detail, Destination: routes.MasterEntry("condicoes", c.ID), Page: c.BookPage}
}

func spellHit(m book.Spell) finderHit {
	return finderHit{
		Name:        m.Name,
		Detail:      fmt.Sprintf("%dº círculo · %s", m.Circle, book.CastingName(m.Execution)),
		Destination: routes.MasterEntry("magias", m.ID),
		Page:        m.BookPage,
	}
}

func powerHit(p book.Power) finderHit {
	return finderHit{Name: p.Name, Detail: p.Source, Destination: routes.MasterEntry("poderes", p.ID), Page: p.BookPage}
}

func itemHit(i book.Item) finderHit {
	return finderHit{
		Name:        i.Name,
		Detail:      book.CategoryName(i.Category),
		Destination: routes.MasterEntry("itens", i.ID),
		Page:        i.BookPage,
	}
}

func effectHit(e book.EffectKind) finderHit {
	return finderHit{
		Name:        e.Name,
		Detail:      "Tipo de efeito",
		Destination: routes.MasterEntry("efeitos", e.ID),
		Page:        e.BookPage,
	}
}

func schoolHit(e book.SpellSchool) finderHit {
	return finderHit{
		Name:        e.Name,
		Detail:      "Escola de magia",
		Destination: routes.MasterEntry("escolas", e.ID),
		Page:        e.BookPage,
	}
}

func expertiseHit(p book.Expertise) finderHit {
	detail := "Perícia · " + book.AttributeAbbrev(p.Attribute)
	if p.TrainedOnly {
		detail += " · só treinada"
	}
	return finderHit{
		Name:        p.Name,
		Detail:      detail,
		Destination: routes.MasterEntry("pericias", p.ID),
		Page:        p.BookPage,
	}
}

func raceHit(r book.Race) finderHit {
	return finderHit{
		Name:        r.Name,
		Detail:      book.TierName(r.Tier) + " · " + r.AttributeMod.Escrito(),
		Destination: routes.MasterEntry("racas", r.ID),
		Page:        r.BookPage,
	}
}

func classHit(c book.Class) finderHit {
	return finderHit{
		Name:        c.Name,
		Detail:      fmt.Sprintf("Classe · %d poderes", c.Powers),
		Destination: routes.MasterEntry("classes", c.ID),
		Page:        c.BookPage,
	}
}

func deityHit(d book.God) finderHit {
	return finderHit{
		Name:        d.Name,
		Detail:      d.Portfolio,
		Destination: routes.MasterEntry("deuses", d.ID),
		Page:        d.BookPage,
	}
}

func entryHit(m book.Entry) finderHit {
	return finderHit{
		Name:        m.Name,
		Detail:      fmt.Sprintf("ND %s · %s", book.CRWritten(m.ND), book.TypeName(m.Kind)),
		Destination: routes.MasterBestiary + "?criatura=" + url.QueryEscape(m.ID),
		Page:        m.BookPage,
	}
}

// Os três destinos que as linhas acima usam, e por que são três:
//
// `routes.MasterSearch` leva à cena dos catálogos buscando pelo NOME, e não por
// um id: a cena não tem endereço para uma entrada só — ela tem `?aba=` e
// `?busca=`, que já são recarregáveis. Inventar um terceiro estado de cena
// seria mais uma coisa a manter.
//
// `routes.MasterEntry` é o endereço de UM verbete: a aba dele, mostrando só
// ele. Quem clica num conceito pediu o conceito, e não uma busca pelo termo nos
// oito catálogos.
//
// `routes.MasterBestiarySearch` leva à cena do bestiário filtrada, e serve
// tanto ao "+12" (com o termo digitado) quanto ao que o `entryHit` faz com o
// id.
