package api

import (
	"slices"
	"strconv"
	"strings"
)

// OS FILTROS de cada catálogo (ALE-264).
//
// O bestiário sempre teve os dele — ND e tipo —, e os outros oito nasceram com
// busca e mais nada. O dono pediu filtro específico por catálogo, e a forma é a
// MESMA do bestiário: crachás que ligam e desligam, porque o mestre já aprendeu
// esse gesto lá.
//
// A regra de combinação é a do bestiário também: OU dentro de um filtro (2º ou
// 3º círculo), E entre filtros (2º círculo E da escola de evocação). É o que se
// espera de uma linha de crachás, e é o que a lista de tipos já fazia.
//
// Os filtros SOMEM durante a busca. Com termo digitado a cena responde outra
// pergunta — os oito catálogos, não este —, e um crachá de círculo aceso sobre
// uma lista que tem itens e condições diria que ele está filtrando o que não
// filtra.

// opcaoDeFiltro é um crachá: o valor que vai para o endereço e o rótulo que se lê.
type opcaoDeFiltro struct {
	Valor  string
	Rotulo string
}

// filtroDoAcervo é uma LINHA de crachás.
type filtroDoAcervo struct {
	// Chave é o nome no endereço e no sinal (`?circulo=3`). Minúscula e uma
	// palavra: ela vira nome de sinal do Datastar, e o analisador de HTML
	// minuscula nome de atributo.
	Chave  string
	Rotulo string
	Opcoes []opcaoDeFiltro
}

// filtrosDaAba diz quais linhas de crachá aquele catálogo mostra.
//
// Classes (14) e Efeitos (18) não têm nenhuma, e é decisão e não esquecimento:
// numa lista que cabe na tela, filtro é cromo que não poupa rolagem.
func filtrosDaAba(aba string) []filtroDoAcervo {
	switch aba {
	case "condicoes":
		return []filtroDoAcervo{{Chave: "efeito", Rotulo: "Tipo de efeito", Opcoes: opcoesDeEfeito()}}
	case "magias":
		return []filtroDoAcervo{
			{Chave: "circulo", Rotulo: "Círculo", Opcoes: opcoesDeCirculo()},
			{Chave: "escola", Rotulo: "Escola", Opcoes: opcoesDeEscola()},
			{Chave: "classe", Rotulo: "Classe", Opcoes: opcoesDeClasseDeMagia()},
		}
	case "pericias":
		return []filtroDoAcervo{
			{Chave: "atributo", Rotulo: "Atributo", Opcoes: opcoesDeAtributo()},
			{Chave: "treino", Rotulo: "Treino", Opcoes: []opcaoDeFiltro{
				{"so-treinada", "Só treinada"}, {"armadura", "Penalidade de armadura"},
			}},
		}
	case "poderes":
		return []filtroDoAcervo{{Chave: "fonte", Rotulo: "Fonte", Opcoes: opcoesDeFonteDePoder()}}
	case "itens":
		return []filtroDoAcervo{{Chave: "familia", Rotulo: "Família", Opcoes: familiasDeItem}}
	case "deuses":
		return []filtroDoAcervo{{Chave: "energia", Rotulo: "Energia", Opcoes: []opcaoDeFiltro{
			{"positiva", "Positiva"}, {"negativa", "Negativa"}, {"qualquer", "Qualquer"},
		}}}
	case "racas":
		return []filtroDoAcervo{{Chave: "linhagem", Rotulo: "Linhagem", Opcoes: []opcaoDeFiltro{
			{"comum", "Comum"}, {"extra", "Exótica"},
		}}}
	}
	return nil
}

// ── as opções que saem do próprio catálogo ───────────────────────────────────
//
// Lidas do dado e não escritas à mão: escola nova no livro aparece sozinha, e
// uma lista fixa aqui seria a que fica para trás em silêncio.

func opcoesDeEfeito() []opcaoDeFiltro {
	usados := map[string]bool{}
	for _, c := range catalogosDoLivro().Condicoes {
		for _, t := range c.Tags {
			usados[t] = true
		}
	}
	var fora []opcaoDeFiltro
	for _, e := range tiposDeEfeito() {
		if usados[e.ID] {
			fora = append(fora, opcaoDeFiltro{e.ID, e.Name})
		}
	}
	return fora
}

func opcoesDeCirculo() []opcaoDeFiltro {
	var fora []opcaoDeFiltro
	for _, circulo := range valoresDistintos(catalogosDoLivro().Magias, func(m magiaDoLivro) string {
		return strconv.Itoa(m.Circle)
	}) {
		fora = append(fora, opcaoDeFiltro{circulo, circulo + "º"})
	}
	return fora
}

func opcoesDeEscola() []opcaoDeFiltro {
	var fora []opcaoDeFiltro
	for _, escola := range valoresDistintos(catalogosDoLivro().Magias, func(m magiaDoLivro) string {
		return m.School
	}) {
		fora = append(fora, opcaoDeFiltro{escola, nomeDaEscola(escola)})
	}
	return fora
}

func opcoesDeClasseDeMagia() []opcaoDeFiltro {
	vistos := map[string]bool{}
	var fora []opcaoDeFiltro
	for _, m := range catalogosDoLivro().Magias {
		for _, c := range m.Classes {
			if !vistos[c] {
				vistos[c] = true
				fora = append(fora, opcaoDeFiltro{c, c})
			}
		}
	}
	slices.SortFunc(fora, func(a, b opcaoDeFiltro) int { return strings.Compare(a.Rotulo, b.Rotulo) })
	return fora
}

// opcoesDeFonteDePoder: as 14 classes mais "geral" e "divino".
//
// A fonte de um poder é a string que o cartão mostra ("Arcanista", "Geral ·
// combate", "Divino · Khalmyr"), e filtrar por ela inteira daria um crachá por
// deus. O filtro casa pelo COMEÇO — ver `poderCasa`.
func opcoesDeFonteDePoder() []opcaoDeFiltro {
	fora := []opcaoDeFiltro{{"Geral", "Geral"}, {"Divino", "Divino"}}
	_, classes, _ := catalogosDoPersonagem()
	for _, c := range classes {
		fora = append(fora, opcaoDeFiltro{c.Name, c.Name})
	}
	return fora
}

// familiasDeItem agrupa as quinze categorias em seis famílias.
//
// Quinze crachás numa linha é uma lista, não um filtro: o mestre procura "uma
// arma" e não "uma arma marcial de fogo". As categorias continuam escritas no
// cartão, que é onde a distinção fina importa.
var familiasDeItem = []opcaoDeFiltro{
	{"armas", "Armas"},
	{"protecao", "Armaduras e escudos"},
	{"vestuario", "Vestuário"},
	{"consumo", "Consumíveis"},
	{"montaria", "Montarias e veículos"},
	{"outros", "Outros"},
}

// familiaDaCategoria mapeia a categoria do dado para a família do crachá.
func familiaDaCategoria(categoria string) string {
	switch {
	case strings.HasPrefix(categoria, "weapon"):
		return "armas"
	case strings.HasPrefix(categoria, "armor"), categoria == "shield":
		return "protecao"
	case categoria == "apparel":
		return "vestuario"
	case categoria == "consumable", categoria == "meal", categoria == "catalyst":
		return "consumo"
	case categoria == "animal", categoria == "vehicle":
		return "montaria"
	}
	return "outros"
}

func valoresDistintos[T any](lista []T, de func(T) string) []string {
	vistos := map[string]bool{}
	var fora []string
	for _, e := range lista {
		if v := de(e); v != "" && !vistos[v] {
			vistos[v] = true
			fora = append(fora, v)
		}
	}
	slices.Sort(fora)
	return fora
}

// ── o casamento, catálogo a catálogo ─────────────────────────────────────────

// opcoesDeAtributo são os seis, na ordem da ficha e do livro — nunca alfabética.
func opcoesDeAtributo() []opcaoDeFiltro {
	usados := map[string]bool{}
	for _, p := range periciasDoAcervo() {
		usados[p.Attribute] = true
	}
	var fora []opcaoDeFiltro
	for _, a := range ordemDosAtributos {
		if usados[a.Chave] {
			fora = append(fora, opcaoDeFiltro{a.Chave, a.Sigla})
		}
	}
	return fora
}

func periciaCasa(p periciaDoLivro, chave, valor string) bool {
	switch chave {
	case "atributo":
		return p.Attribute == valor
	case "treino":
		// As duas marcas do livro num filtro só: são as duas coisas que mudam
		// COMO a perícia se usa, e separá-las em duas linhas de um crachá cada
		// gastaria duas linhas para dizer o que uma diz.
		return (valor == "so-treinada" && p.SoTreinada) ||
			(valor == "armadura" && p.PenalidadeDeArmadura)
	}
	return true
}

func condicaoCasa(c condicaoDoLivro, chave, valor string) bool {
	if chave == "efeito" {
		return slices.Contains(c.Tags, valor)
	}
	return true
}

func magiaCasa(m magiaDoLivro, chave, valor string) bool {
	switch chave {
	case "circulo":
		return strconv.Itoa(m.Circle) == valor
	case "escola":
		return m.School == valor
	case "classe":
		return slices.Contains(m.Classes, valor)
	}
	return true
}

// poderCasa pelo COMEÇO da fonte: ela é "Geral · combate" e "Divino · Khalmyr",
// e o crachá diz "Geral" e "Divino". Para classe a fonte é o nome puro.
func poderCasa(p poderDoLivro, chave, valor string) bool {
	if chave == "fonte" {
		return p.Fonte == valor || strings.HasPrefix(p.Fonte, valor+" ·")
	}
	return true
}

func itemCasa(i itemDoLivro, chave, valor string) bool {
	if chave == "familia" {
		return familiaDaCategoria(i.Category) == valor
	}
	return true
}

func deusCasa(d deusDoLivro, chave, valor string) bool {
	if chave == "energia" {
		return d.Energia == valor
	}
	return true
}

func racaCasa(r racaDoLivro, chave, valor string) bool {
	if chave == "linhagem" {
		return r.Tier == valor
	}
	return true
}

// aplicaFiltros é a regra de combinação: OU dentro de um filtro, E entre eles.
//
// A ordem em que as chaves saem do mapa não importa porque tudo entre filtros é
// E — mas vale dizer, porque um dia alguém acrescenta um "ou" e a ordem passa a
// decidir o resultado.
func aplicaFiltros[T any](lista []T, escolhidos map[string][]string, casa func(T, string, string) bool) []T {
	for chave, valores := range escolhidos {
		if len(valores) == 0 {
			continue
		}
		var restantes []T
		for _, entrada := range lista {
			for _, valor := range valores {
				if casa(entrada, chave, valor) {
					restantes = append(restantes, entrada)
					break
				}
			}
		}
		lista = restantes
	}
	return lista
}
