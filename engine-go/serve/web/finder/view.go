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
	Nome    string
	Detalhe string
	Destino string
	// Pagina é a do livro, e ZERO significa "o catálogo não sabe" — a linha sai
	// sem número em vez de sair com "p0".
	Pagina int
	// ponto não vai para a tela: ele é a ORDEM, e mostrá-lo convidaria a
	// discutir a nota em vez do resultado.
	ponto int
}

// finderGroup é um catálogo com o que sobrou, já cortado.
type finderGroup struct {
	Rotulo  string
	Achados []finderHit
	// Total é antes do corte — é ele que escreve o "+12".
	Total int
	// Mais é para onde o "+12" leva: a cena da ferramenta com a MESMA busca.
	// Corte com saída, e não corte que informa e abandona.
	Mais string
}

func (g finderGroup) Cortados() int { return g.Total - len(g.Achados) }

type finderView struct {
	Busca  string
	Grupos []finderGroup
	// Achados é o total ANTES dos cortes de grupo.
	Achados int
	// PeloTexto diz que estes achados vieram da segunda passada — nenhum NOME
	// casou, e o que está na lista apenas MENCIONA o termo. A tela avisa: sem
	// isso, uma lista que não contém o que se pediu parece a lista errada.
	PeloTexto bool
}

func (v finderView) Buscando() bool { return strings.TrimSpace(v.Busca) != "" }

// searchTheBook monta o resultado do ⌃K.
func searchTheBook(busca string) finderView {
	v := buildHits(busca, pelosNomes)
	if v.Achados > 0 || !v.Buscando() {
		return v
	}
	// SEGUNDA PASSADA: o corpo da regra só entra quando NOME nenhum casou. Com
	// ele valendo sempre, "abal" afogava a condição "Abalado" entre as centenas
	// de poderes cujo TEXTO a menciona.
	//
	// Quem digita "abal" procura o verbete; quem digita "chance de falha" não
	// sabe o nome, e é para essa pessoa que a passada existe.
	v = buildHits(busca, tambemPeloTexto)
	v.PeloTexto = v.Achados > 0
	return v
}

// pelosNomes e tambemPeloTexto nomeiam as duas passadas. Um booleano cru na
// chamada (`foundGroup(..., true)`) não diz nada de dentro do `foundGroup`.
const (
	pelosNomes      = false
	tambemPeloTexto = true
)

func buildHits(busca string, peloTexto bool) finderView {
	v := finderView{Busca: busca}
	if !v.Buscando() {
		return v
	}
	a := book.Catalogs()
	racas, classes, deuses := book.CharacterCatalogs()
	for _, g := range []finderGroup{
		foundGroup("Condições", a.Condicoes, busca, peloTexto, routes.MasterSearch("condicoes", busca), book.ConditionFields, conditionHit),
		foundGroup("Criaturas", book.Creatures(), busca, peloTexto, routes.MasterBestiarySearch(busca), entryFields, entryHit),
		foundGroup("Magias", a.Magias, busca, peloTexto, routes.MasterSearch("magias", busca), book.SpellFields, spellHit),
		foundGroup("Poderes", a.Poderes, busca, peloTexto, routes.MasterSearch("poderes", busca), book.PowerFields, powerHit),
		foundGroup("Itens", a.Itens, busca, peloTexto, routes.MasterSearch("itens", busca), book.ItemFields, itemHit),
		foundGroup("Efeitos", book.EffectKinds(), busca, peloTexto, routes.MasterSearch("efeitos", busca), book.EffectFields, effectHit),
		foundGroup("Escolas", book.SpellSchools(), busca, peloTexto, routes.MasterSearch("escolas", busca), book.SchoolFields, schoolHit),
		foundGroup("Perícias", book.Expertises(), busca, peloTexto, routes.MasterSearch("pericias", busca), book.ExpertiseFields, expertiseHit),
		foundGroup("Raças", racas, busca, peloTexto, routes.MasterSearch("racas", busca), book.RaceFields, raceHit),
		foundGroup("Classes", classes, busca, peloTexto, routes.MasterSearch("classes", busca), book.ClassFields, classHit),
		foundGroup("Deuses", deuses, busca, peloTexto, routes.MasterSearch("deuses", busca), book.GodFields, deityHit),
	} {
		if g.Total == 0 {
			continue
		}
		v.Achados += g.Total
		v.Grupos = append(v.Grupos, g)
	}
	sortByRelevance(v.Grupos)
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
func sortByRelevance(grupos []finderGroup) {
	slices.SortStableFunc(grupos, func(a, b finderGroup) int {
		return cmp.Compare(b.melhorPonto(), a.melhorPonto())
	})
}

// melhorPonto é a nota do primeiro achado — a lista já vem ordenada pelo
// `bestFirst`, então o primeiro é o melhor.
func (g finderGroup) melhorPonto() int {
	if len(g.Achados) == 0 {
		return 0
	}
	return g.Achados[0].ponto
}

// foundGroup pontua, ordena e corta um catálogo.
//
// `campos` é a MESMA função que a cena dos catálogos usa para filtrar, e reusá-la
// é o que faz as duas superfícies concordarem sobre o que é buscável — uma
// segunda lista de campos aqui divergiria no dia em que alguém acrescentasse um.
func foundGroup[T any](
	rotulo string, lista []T, busca string, peloTexto bool, mais string,
	campos func(T) []string, comoAchado func(T) finderHit,
) finderGroup {
	g := finderGroup{Rotulo: rotulo, Mais: mais}
	for _, e := range lista {
		// A linha é montada ANTES de saber se ela passa, e é deliberado: pegar o
		// nome de `campos(e)[0]` seria depender de uma ordem que nada garante, e
		// no dia em que um `entryFields` mudasse de ordem a pontuação passaria a
		// medir a descrição — em silêncio, com a lista continuando a sair.
		a := comoAchado(e)
		a.ponto = search.Score(a.Nome, busca)
		if a.ponto == 0 && peloTexto {
			a.ponto = search.ScoreText(campos(e), busca)
		}
		if a.ponto == 0 {
			continue
		}
		g.Achados = append(g.Achados, a)
	}
	g.Total = len(g.Achados)
	slices.SortStableFunc(g.Achados, bestFirst)
	if len(g.Achados) > hitsByGroup {
		g.Achados = g.Achados[:hitsByGroup]
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
	if d := cmp.Compare(b.ponto, a.ponto); d != 0 {
		return d
	}
	if d := cmp.Compare(len(a.Nome), len(b.Nome)); d != 0 {
		return d
	}
	return cmp.Compare(a.Nome, b.Nome)
}

// ── de cada catálogo para uma linha ──────────────────────────────────────────

// entryFields existe para o verbete do bestiário passar pelo mesmo `foundGroup`
// que os outros catálogos: a cena dele filtra por outro caminho
// (`book.FilterCreatures`), que não serve aqui.
func entryFields(m book.Entry) []string {
	return append([]string{m.Name, book.TypeName(m.Tipo)}, m.SpecialAbilities...)
}

func conditionHit(c book.Condition) finderHit {
	detalhe := "Condição"
	if c.UpgradesTo != "" {
		detalhe = "Condição · agrava para " + book.ConditionName(c.UpgradesTo)
	}
	return finderHit{Nome: c.Name, Detalhe: detalhe, Destino: routes.MasterEntry("condicoes", c.ID), Pagina: c.BookPage}
}

func spellHit(m book.Spell) finderHit {
	return finderHit{
		Nome:    m.Name,
		Detalhe: fmt.Sprintf("%dº círculo · %s", m.Circle, book.CastingName(m.Execution)),
		Destino: routes.MasterEntry("magias", m.ID),
		Pagina:  m.BookPage,
	}
}

func powerHit(p book.Power) finderHit {
	return finderHit{Nome: p.Name, Detalhe: p.Fonte, Destino: routes.MasterEntry("poderes", p.ID), Pagina: p.BookPage}
}

func itemHit(i book.Item) finderHit {
	return finderHit{
		Nome:    i.Name,
		Detalhe: book.CategoryName(i.Category),
		Destino: routes.MasterEntry("itens", i.ID),
		Pagina:  i.BookPage,
	}
}

func effectHit(e book.EffectKind) finderHit {
	return finderHit{
		Nome:    e.Name,
		Detalhe: "Tipo de efeito",
		Destino: routes.MasterEntry("efeitos", e.ID),
		Pagina:  e.BookPage,
	}
}

func schoolHit(e book.SpellSchool) finderHit {
	return finderHit{
		Nome:    e.Name,
		Detalhe: "Escola de magia",
		Destino: routes.MasterEntry("escolas", e.ID),
		Pagina:  e.BookPage,
	}
}

func expertiseHit(p book.Expertise) finderHit {
	detalhe := "Perícia · " + book.AttributeAbbrev(p.Attribute)
	if p.SoTreinada {
		detalhe += " · só treinada"
	}
	return finderHit{
		Nome:    p.Name,
		Detalhe: detalhe,
		Destino: routes.MasterEntry("pericias", p.ID),
		Pagina:  p.BookPage,
	}
}

func raceHit(r book.Race) finderHit {
	return finderHit{
		Nome:    r.Name,
		Detalhe: book.TierName(r.Tier) + " · " + r.AttributeMod.Escrito(),
		Destino: routes.MasterEntry("racas", r.ID),
		Pagina:  r.BookPage,
	}
}

func classHit(c book.Class) finderHit {
	return finderHit{
		Nome:    c.Name,
		Detalhe: fmt.Sprintf("Classe · %d poderes", c.Poderes),
		Destino: routes.MasterEntry("classes", c.ID),
		Pagina:  c.BookPage,
	}
}

func deityHit(d book.God) finderHit {
	return finderHit{
		Nome:    d.Name,
		Detalhe: d.Portfolio,
		Destino: routes.MasterEntry("deuses", d.ID),
		Pagina:  d.BookPage,
	}
}

func entryHit(m book.Entry) finderHit {
	return finderHit{
		Nome:    m.Name,
		Detalhe: fmt.Sprintf("ND %s · %s", book.CRWritten(m.ND), book.TypeName(m.Tipo)),
		Destino: routes.MasterBestiary + "?criatura=" + url.QueryEscape(m.ID),
		Pagina:  m.BookPage,
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
