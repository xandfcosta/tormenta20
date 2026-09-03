package master

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"t20engine/search"
	"t20engine/web/bookui"

	"t20engine/book"
)

// OS CATÁLOGOS do mestre (ALE-258): condições, magias, poderes e itens numa
// busca só. Segunda das quatro ferramentas da Mesa do Mestre.
//
// São 992 entradas — 35 condições, 198 magias, 566 poderes e 193 itens (contadas
// pelo próprio carregador, não estimadas) —, e
// isso é outra ordem de grandeza que o bestiário. A SPA aguenta com
// `VirtualList`; a cena do servidor manda TUDO, por decisão do dono, e o
// raciocínio fica registrado porque ele não é óbvio: hoje o navegador BAIXA
// 156 KB de JSON de catálogo para poder filtrar, e depois monta a lista aos
// poucos. Servindo HTML pronto a rede quase empata e o JSON some do bundle; o
// que sobra de custo é DOM no telefone, que é o preço aceito para o mestre
// poder rolar a lista inteira e usar o Ctrl+F do navegador — duas coisas que
// lista virtualizada tira.

// ── a regra da busca, e ela NÃO é a das outras listas ────────────────────────

// matchesAllTerms: TODO termo separado por espaço precisa aparecer em algum
// dos campos. "luz cur" casa com o que carrega as duas coisas.
//
// Isto NÃO é o `search.Matches` das outras cenas, e a diferença é deliberada — o
// comentário do `catalog-model.ts` a explica e ela sobrevive ao porte. O
// `search.Matches` é tolerante a typo por subsequência, o que serve para escolher UM
// item de uma lista; aqui o mestre está estreitando uma REFERÊNCIA por palavras
// que ele sabe, e subsequência arrastaria quase-acertos que fazem uma consulta
// de regra parecer errada no meio da sessão.
//
// O que as duas compartilham é a `search.Fold`: acento não separa "ilusão" de
// "ilusao", porque ninguém digita til numa busca apressada.
func matchesAllTerms(campos []string, busca string) bool {
	alvo := search.Fold(strings.TrimSpace(busca))
	if alvo == "" {
		return true
	}
	palheiro := search.Fold(strings.Join(campos, " "))
	for _, termo := range strings.Fields(alvo) {
		if !strings.Contains(palheiro, termo) {
			return false
		}
	}
	return true
}

// ── o que se busca ───────────────────────────────────────────────────────────

// collectionTab é uma parada da fileira de abas.
type collectionTab struct {
	ID     string
	Rotulo string
}

// A ordem é a da SPA: condição primeiro porque é a consulta mais frequente no
// meio do combate, e item por último porque é a de entre-cenas.
// As três últimas entraram na ALE-264 e vão no FIM pela mesma razão que decidiu
// a ordem original: raça, classe e deus são consulta de CRIAÇÃO de personagem, e
// as quatro primeiras são consulta de mesa com o combate em curso.
var collectionTabs = []collectionTab{
	{"condicoes", "Condições"},
	{"magias", "Magias"},
	{"pericias", "Perícias"},
	{"poderes", "Poderes"},
	{"itens", "Itens"},
	{"efeitos", "Efeitos"},
	{"escolas", "Escolas"},
	{"racas", "Raças"},
	{"classes", "Classes"},
	{"deuses", "Deuses"},
}

// tabLabel devolve o nome que a fileira mostra, para a frase de volta não
// dizer "condicoes" com a cara de chave.
func tabLabel(id string) string {
	for _, a := range collectionTabs {
		if a.ID == id {
			return a.Rotulo
		}
	}
	return id
}

func knownTab(id string) string {
	for _, a := range collectionTabs {
		if a.ID == id {
			return id
		}
	}
	// Aba desconhecida cai na primeira em vez de mostrar tela vazia: o `?aba=`
	// é endereço e alguém o digita errado.
	return collectionTabs[0].ID
}

// collectionGroup é um catálogo com o que sobrou do filtro. Grupo VAZIO não é
// montado: cabeçalho sobre nada é ruído numa consulta no meio do combate.
type collectionGroup struct {
	Rotulo    string
	Condicoes []book.Condition
	Magias    []book.Spell
	Poderes   []book.Power
	Itens     []book.Item
	Efeitos   []book.EffectKind
	Escolas   []book.SpellSchool
	Pericias  []book.Expertise
	Racas     []book.Race
	Classes   []book.Class
	Deuses    []book.God
}

func (g collectionGroup) Count() int {
	return len(g.Condicoes) + len(g.Magias) + len(g.Poderes) + len(g.Itens) +
		len(g.Efeitos) + len(g.Escolas) + len(g.Pericias) + len(g.Racas) + len(g.Classes) + len(g.Deuses)
}

// collectionCriteria é o que a URL (ou os sinais) pedem da cena.
type collectionCriteria struct {
	Term string
	Aba  string
	// Entrada é o ID de UM verbete, e ela ganha de tudo: com ela a cena mostra
	// aquele verbete sozinho. É o endereço que um ELO usa — quem clica em "Medo"
	// pediu o Medo, não uma busca por "medo" nos oito catálogos.
	Entrada string
	// Filtros são os crachás acesos, por chave (`{"circulo": ["2","3"]}`). Vêm
	// da URL na carga fria e dos sinais quando o Datastar chama, como a busca.
	Filtros map[string][]string
}

// collectionView é a cena inteira numa resposta.
type collectionView struct {
	Term string
	// Livro é o endereço do PDF servido (ALE-264). Zero valor = não há livro
	// configurado, e aí o cartão mostra a página em texto puro — que é o que o
	// mestre com o livro de papel na mesa usa.
	Book bookui.BookAddress
	// Aba só importa quando NÃO se está buscando: com termo digitado a cena
	// mostra os quatro catálogos agrupados, que é a decisão que a ALE-22
	// registrou — a versão em React filtrava só a aba ativa, e "bola de fogo"
	// digitado em Condições dizia "nada encontrado" com a magia existindo.
	Aba string
	// Entrada é o id do verbete que a cena está mostrando sozinho, ou vazio.
	Entrada string
	// Filtros é o que a cena tem para oferecer, e Acesos o que está ligado.
	Filtros []collectionFilter
	Acesos  map[string][]string
	Grupos  []collectionGroup
	Achados int
}

func (v collectionView) Searching() bool { return strings.TrimSpace(v.Term) != "" }

// loadCollection monta a cena: os quatro catálogos quando há busca, um só
// quando não há.
func loadCollection(c collectionCriteria, livro bookui.BookAddress) collectionView {
	v := collectionView{
		Term: c.Term, Aba: knownTab(c.Aba), Book: livro, Entrada: c.Entrada,
		Acesos: c.Filtros,
	}
	v.Filtros = filtersForTab(v.Aba)
	// A ENTRADA vem primeiro e encerra: ela é um endereço para um verbete só, e
	// misturá-la com busca daria uma tela que responde duas perguntas.
	if c.Entrada != "" {
		v.Grupos = []collectionGroup{groupForEntry(v.Aba, c.Entrada)}
		v.Achados = v.Grupos[0].Count()
		return v
	}
	busca := c.Term
	a := book.Catalogs()

	if !v.Searching() {
		v.Grupos = []collectionGroup{groupForTab(a, v.Aba, c.Filtros)}
		v.Achados = v.Grupos[0].Count()
		return v
	}

	racas, classes, deuses := book.CharacterCatalogs()
	for _, g := range []collectionGroup{
		{Rotulo: "Condições", Condicoes: filter(a.Condicoes, book.ConditionFields, busca)},
		{Rotulo: "Magias", Magias: filter(a.Magias, book.SpellFields, busca)},
		{Rotulo: "Poderes", Poderes: filter(a.Poderes, book.PowerFields, busca)},
		{Rotulo: "Itens", Itens: filter(a.Itens, book.ItemFields, busca)},
		{Rotulo: "Efeitos", Efeitos: filter(book.EffectKinds(), book.EffectFields, busca)},
		{Rotulo: "Escolas", Escolas: filter(book.SpellSchools(), book.SchoolFields, busca)},
		{Rotulo: "Perícias", Pericias: filter(book.Expertises(), book.ExpertiseFields, busca)},
		{Rotulo: "Raças", Racas: filter(racas, book.RaceFields, busca)},
		{Rotulo: "Classes", Classes: filter(classes, book.ClassFields, busca)},
		{Rotulo: "Deuses", Deuses: filter(deuses, book.GodFields, busca)},
	} {
		if g.Count() == 0 {
			continue
		}
		v.Achados += g.Count()
		v.Grupos = append(v.Grupos, g)
	}
	return v
}

// groupForEntry acha UM verbete pelo id, dentro da aba pedida.
//
// Pelo ID e não pelo nome: nome é texto de tela e muda com a revisão do livro;
// id é a chave com que os catálogos se referem uns aos outros ("upgradesTo":
// "apavorado"). Um elo é uma referência entre DADOS, e referência por texto de
// tela é a que quebra em silêncio no dia de uma correção de acento.
//
// Id desconhecido devolve grupo VAZIO, e a cena diz que não achou — endereço se
// digita à mão e catálogo muda; inventar um verbete seria pior.
func groupForEntry(aba, id string) collectionGroup {
	// Sem filtro: o elo pede UM verbete pelo id, e um crachá aceso na cena de
	// origem não pode esconder o destino do elo.
	inteiro := groupForTab(book.Catalogs(), aba, nil)
	fora := collectionGroup{Rotulo: inteiro.Rotulo}
	for _, c := range inteiro.Condicoes {
		if c.ID == id {
			fora.Condicoes = append(fora.Condicoes, c)
		}
	}
	for _, m := range inteiro.Magias {
		if m.ID == id {
			fora.Magias = append(fora.Magias, m)
		}
	}
	for _, p := range inteiro.Poderes {
		if p.ID == id {
			fora.Poderes = append(fora.Poderes, p)
		}
	}
	for _, i := range inteiro.Itens {
		if i.ID == id {
			fora.Itens = append(fora.Itens, i)
		}
	}
	for _, e := range inteiro.Efeitos {
		if e.ID == id {
			fora.Efeitos = append(fora.Efeitos, e)
		}
	}
	for _, e := range inteiro.Escolas {
		if e.ID == id {
			fora.Escolas = append(fora.Escolas, e)
		}
	}
	for _, p := range inteiro.Pericias {
		if p.ID == id {
			fora.Pericias = append(fora.Pericias, p)
		}
	}
	for _, r := range inteiro.Racas {
		if r.ID == id {
			fora.Racas = append(fora.Racas, r)
		}
	}
	for _, c := range inteiro.Classes {
		if c.ID == id {
			fora.Classes = append(fora.Classes, c)
		}
	}
	for _, d := range inteiro.Deuses {
		if d.ID == id {
			fora.Deuses = append(fora.Deuses, d)
		}
	}
	return fora
}

// groupForTab monta o catálogo da cena, já com os crachás aplicados.
func groupForTab(a book.GMCatalogs, aba string, acesos map[string][]string) collectionGroup {
	racas, classes, deuses := book.CharacterCatalogs()
	switch aba {
	case "magias":
		return collectionGroup{Rotulo: "Magias", Magias: applyFilters(a.Magias, acesos, spellMatches)}
	case "poderes":
		return collectionGroup{Rotulo: "Poderes", Poderes: applyFilters(a.Poderes, acesos, powerMatches)}
	case "itens":
		return collectionGroup{Rotulo: "Itens", Itens: applyFilters(a.Itens, acesos, itemMatches)}
	case "efeitos":
		return collectionGroup{Rotulo: "Efeitos", Efeitos: book.EffectKinds()}
	case "escolas":
		return collectionGroup{Rotulo: "Escolas", Escolas: book.SpellSchools()}
	case "pericias":
		return collectionGroup{Rotulo: "Perícias", Pericias: applyFilters(book.Expertises(), acesos, expertiseMatches)}
	case "racas":
		return collectionGroup{Rotulo: "Raças", Racas: applyFilters(racas, acesos, raceMatches)}
	case "classes":
		return collectionGroup{Rotulo: "Classes", Classes: classes}
	case "deuses":
		return collectionGroup{Rotulo: "Deuses", Deuses: applyFilters(deuses, acesos, godMatches)}
	default:
		return collectionGroup{Rotulo: "Condições", Condicoes: applyFilters(a.Condicoes, acesos, conditionMatches)}
	}
}

func filter[T any](lista []T, campos func(T) []string, busca string) []T {
	var fora []T
	for _, e := range lista {
		if matchesAllTerms(campos(e), busca) {
			fora = append(fora, e)
		}
	}
	return fora
}

// ── como o livro escreve ─────────────────────────────────────────────────────

// collectionSignals: só a busca e a aba viajam. O que se vê chega desenhado.
func collectionSignals(v collectionView) string {
	busca, _ := json.Marshal(v.Term)
	aba, _ := json.Marshal(v.Aba)
	partes := []string{fmt.Sprintf("busca: %s", busca), fmt.Sprintf("aba: %s", aba)}
	// UM sinal por filtro da cena, e só os DELA: um sinal de círculo declarado
	// na cena das condições viajaria em toda requisição dali para nada.
	for _, f := range v.Filtros {
		valores, _ := json.Marshal(v.Acesos[f.Chave])
		if v.Acesos[f.Chave] == nil {
			valores = []byte("[]")
		}
		partes = append(partes, fmt.Sprintf("%s: %s", f.Chave, valores))
	}
	return "{" + strings.Join(partes, ", ") + "}"
}

// badgeOn diz se aquele valor está ligado, para a cena não precisar de
// `slices`.
func (v collectionView) badgeOn(chave, valor string) bool {
	return slices.Contains(v.Acesos[chave], valor)
}

// toggleBadge é a expressão que o clique roda: liga o que está desligado e
// desliga o que está ligado, no sinal daquele filtro.
//
// Escrita aqui e não no templ porque é a MESMA para os seis filtros, e uma
// linha de JavaScript copiada seis vezes é a que diverge na sétima.
func toggleBadge(aba, chave, valor string) string {
	sinal := "$" + chave
	return fmt.Sprintf(
		"%s = %s.includes(%q) ? %s.filter(v => v !== %q) : [...%s, %q]; @get('/mestre/%s')",
		sinal, sinal, valor, sinal, valor, sinal, valor, aba,
	)
}

// augmentsWritten concorda em número, que é a razão de existir: "1
// aprimoramento" e "3 aprimoramentos".
func augmentsWritten(n int) string {
	if n == 1 {
		return "1 aprimoramento disponível."
	}
	return fmt.Sprintf("%d aprimoramentos disponíveis.", n)
}

// priceWritten é o dinheiro do livro em pt-BR, no MESMO formato do
// `formatTibar` da SPA: vírgula decimal e no máximo duas casas.
//
// Duas casas e não zero porque o preço do livro é fracionário — uma vela custa
// T$ 0,1 (p143) —, e cortar a fração poria "T$ 0" numa linha de compra.
func priceWritten(v float64) string {
	return "T$ " + numberPtBR(v)
}

// slotsWritten: espaço de mochila também é fracionário (item leve ocupa 0,5).
func slotsWritten(v float64) string { return numberPtBR(v) }

// numberPtBR escreve com vírgula decimal e sem zero à toa: 2 sai "2", 0.5 sai
// "0,5", 12.25 sai "12,25".
func numberPtBR(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if len(s) > 2 && strings.HasSuffix(s, ".0") {
		s = s[:len(s)-2]
	}
	return strings.Replace(s, ".", ",", 1)
}
