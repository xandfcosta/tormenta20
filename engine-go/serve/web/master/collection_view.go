package master

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"t20engine/domain/search"
	"t20engine/serve/web/bookui"

	"t20engine/domain/book"
)

// A CENA dos Catálogos do mestre: um catálogo por aba, e todos numa busca só.
// Quais são os catálogos, e o que cada um sabe fazer, é do `collectionCatalogs`.
//
// A cena manda TUDO, e não uma lista virtualizada: o custo é DOM no telefone, e
// o que se compra com ele é o mestre poder rolar a lista inteira e usar o Ctrl+F
// do navegador — duas coisas que lista virtualizada tira.

// ── a regra da busca, e ela NÃO é a das outras listas ────────────────────────

// matchesAllTerms: TODO termo separado por espaço precisa aparecer em algum dos
// campos. "luz cur" casa com o que carrega as duas coisas.
//
// Isto NÃO é o `search.Matches` das outras cenas, e a diferença é deliberada:
// aquele é tolerante a typo por subsequência, o que serve para escolher UM item
// de uma lista; aqui o mestre estreita uma REFERÊNCIA por palavras que ele sabe,
// e subsequência arrastaria quase-acertos que fazem uma consulta de regra
// parecer errada no meio da sessão.
//
// O que as duas compartilham é a `search.Fold`: acento não separa "ilusão" de
// "ilusao", porque ninguém digita til numa busca apressada.
func matchesAllTerms(fields []string, query string) bool {
	target := search.Fold(strings.TrimSpace(query))
	if target == "" {
		return true
	}
	haystack := search.Fold(strings.Join(fields, " "))
	for _, term := range strings.Fields(target) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

// ── o que se busca ───────────────────────────────────────────────────────────

// tabLabel devolve o nome que o trilho mostra, para a frase de volta não dizer
// "condicoes" com a cara de chave.
func tabLabel(aba string) string { return catalogNamed(aba).Stop.Label }

// collectionGroup é um catálogo com o que sobrou do filtro. Grupo VAZIO não é
// montado: cabeçalho sobre nada é ruído numa consulta no meio do combate.
type collectionGroup struct {
	Label   string
	Entries []collectionEntry
}

// entryName é o nome do primeiro verbete do grupo, ou vazio. Quem o chama é a caixa
// do elo, em que o grupo tem um verbete só.
func (g collectionGroup) entryName() string {
	if len(g.Entries) == 0 {
		return ""
	}
	return g.Entries[0].Name
}

func (g collectionGroup) Count() int { return len(g.Entries) }

// collectionCriteria é o que a URL (ou os sinais) pedem da cena.
type collectionCriteria struct {
	Term string
	Aba  string
	// Entry é o ID de UM verbete, e ela ganha de tudo: com ela a cena mostra
	// aquele verbete sozinho. É o endereço que um ELO usa — quem clica em "Medo"
	// pediu o Medo, não uma busca por "medo" em todos os catálogos.
	Entry string
	// Filters são os crachás acesos, por chave (`{"circulo": ["2","3"]}`). Vêm
	// da URL na carga fria e dos sinais quando o Datastar chama, como a busca.
	Filters map[string][]string
}

// collectionView é a cena inteira numa resposta.
type collectionView struct {
	Term string
	// Book é o endereço do PDF servido. Zero valor = não há livro configurado, e
	// aí o cartão mostra a página em texto puro — que é o que o mestre com o
	// livro de papel na mesa usa.
	Book bookui.BookAddress
	// Aba só importa quando NÃO se está buscando: com termo digitado a cena mostra
	// TODOS os catálogos agrupados. Filtrando só a aba ativa, "bola de fogo"
	// digitado em Condições diria "nada encontrado" com a magia existindo.
	Aba string
	// Entry é o id do verbete que a cena está mostrando sozinho, ou vazio.
	Entry string
	// Filters é o que a cena tem para oferecer, e Acesos o que está ligado.
	Filters  []collectionFilter
	Lit      map[string][]string
	Groups   []collectionGroup
	Findings int
	// Total é o tamanho do catálogo da cena sem filtro nenhum.
	Total int
}

func (v collectionView) Searching() bool { return strings.TrimSpace(v.Term) != "" }

// litCount é quantos crachás da cena estão ligados.
func (v collectionView) litCount() int {
	lit := 0
	for _, f := range v.Filters {
		lit += len(v.Lit[f.Key])
	}
	return lit
}

// filtersWritten é o rótulo do botão dos filtros. O número é o único sinal de relance
// de que a lista está filtrada, com o diálogo fechado.
func filtersWritten(lit int) string {
	if lit == 0 {
		return "Filtros"
	}
	return fmt.Sprintf("Filtros · %d", lit)
}

// offersFilters diz se a cena desenha os filtros: o catálogo tem de ter algum, e com
// termo digitado eles somem.
func (v collectionView) offersFilters() bool { return !v.Searching() && len(v.Filters) > 0 }

// groupCount é o que o cabeçalho do grupo diz: na busca, quantos achados; no
// catálogo, quanto dele sobrou dos filtros.
func (v collectionView) groupCount(g collectionGroup) string {
	if v.Searching() || g.Count() == v.Total {
		return strconv.Itoa(g.Count())
	}
	return fmt.Sprintf("%d de %d", g.Count(), v.Total)
}

// loadCollection monta a cena: todos os catálogos quando há busca, um só
// quando não há.
func loadCollection(c collectionCriteria, bookRef bookui.BookAddress) collectionView {
	v := collectionView{
		Term: c.Term, Aba: knownTab(c.Aba), Book: bookRef, Entry: c.Entry,
		Lit: c.Filters,
	}
	catalog := catalogNamed(v.Aba)
	v.Filters = catalog.Filters()
	v.Total = catalog.group(nil).Count()
	// A ENTRADA vem primeiro e encerra: ela é um endereço para um verbete só, e
	// misturá-la com busca daria uma tela que responde duas perguntas.
	if c.Entry != "" {
		v.Groups = []collectionGroup{groupForEntry(v.Aba, c.Entry)}
		v.Findings = v.Groups[0].Count()
		return v
	}

	if !v.Searching() {
		v.Groups = []collectionGroup{catalog.group(c.Filters)}
		v.Findings = v.Groups[0].Count()
		return v
	}

	for _, catalog := range collectionCatalogs {
		g := collectionGroup{Label: catalog.Stop.Label, Entries: catalog.matching(c.Term)}
		if g.Count() == 0 {
			continue
		}
		v.Findings += g.Count()
		v.Groups = append(v.Groups, g)
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
	whole := catalogNamed(aba).group(nil)
	outside := collectionGroup{Label: whole.Label}
	for _, e := range whole.Entries {
		if e.ID == id {
			outside.Entries = append(outside.Entries, e)
		}
	}
	return outside
}

func filter[T any](list []T, fields func(T) []string, search string) []T {
	var outside []T
	for _, e := range list {
		if matchesAllTerms(fields(e), search) {
			outside = append(outside, e)
		}
	}
	return outside
}

// ── como o livro escreve ─────────────────────────────────────────────────────

// collectionSignals: só a busca e a aba viajam. O que se vê chega desenhado.
func collectionSignals(v collectionView) string {
	search, _ := json.Marshal(v.Term)
	aba, _ := json.Marshal(v.Aba)
	parts := []string{fmt.Sprintf("search: %s", search), fmt.Sprintf("aba: %s", aba)}
	// UM sinal por filtro da cena, e só os DELA: um sinal de círculo declarado
	// na cena das condições viajaria em toda requisição dali para nada.
	for _, f := range v.Filters {
		values, _ := json.Marshal(v.Lit[f.Key])
		if v.Lit[f.Key] == nil {
			values = []byte("[]")
		}
		parts = append(parts, fmt.Sprintf("%s: %s", f.Key, values))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// badgeOn diz se aquele valor está ligado, para a cena não precisar de
// `slices`.
func (v collectionView) badgeOn(key, value string) bool {
	return slices.Contains(v.Lit[key], value)
}

// toggleBadge é a expressão que o clique roda: liga o que está desligado e
// desliga o que está ligado, no sinal daquele filtro.
//
// Escrita aqui e não no templ porque é a MESMA para os seis filtros, e uma
// linha de JavaScript copiada seis vezes é a que diverge na sétima.
func toggleBadge(aba, key, value string) string {
	signal := "$" + key
	return fmt.Sprintf(
		"%s = %s.includes(%q) ? %s.filter(v => v !== %q) : [...%s, %q]; @get('/mestre/%s')",
		signal, signal, value, signal, value, signal, value, aba,
	)
}

// augmentsWritten concorda em número, que é a razão de existir: "1
// aprimoramento" e "3 aprimoramentos".
func augmentsWritten(n int) string {
	if n == 1 {
		return "1 aprimoramento"
	}
	return fmt.Sprintf("%d aprimoramentos", n)
}

// spellMeta é a linha de meta da magia, com a resistência só quando há.
func spellMeta(m book.Spell) string {
	parts := []string{
		"Execução: " + book.SpellExecutionWritten(m),
		"Alcance: " + book.SpellRangeWritten(m),
		m.Targeting,
	}
	parts = append(parts, "Duração: "+book.SpellDurationWritten(m))
	if resistance := book.SpellResistanceWritten(m); resistance != "" {
		parts = append(parts, "Resistência: "+resistance)
	}
	return strings.Join(parts, " · ")
}

// raceMeta é a linha de meta da raça. Visão no escuro cobre a penumbra, e por isso
// sai uma só.
func raceMeta(r book.Race) string {
	parts := []string{book.TierName(r.Tier), r.Size, fmt.Sprintf("Deslocamento: %dm", r.Speed)}
	if r.Darkvision {
		parts = append(parts, "Visão no escuro")
	} else if r.LowLightVision {
		parts = append(parts, "Visão na penumbra")
	}
	return strings.Join(parts, " · ")
}

// expertiseMarks são os crachás da perícia, com o atributo na frente.
func expertiseMarks(p book.Expertise) []string {
	marks := []string{book.AttributeAbbrev(p.Attribute)}
	if p.TrainedOnly {
		marks = append(marks, "Só treinada")
	}
	if p.ArmorPenalty {
		marks = append(marks, "Penalidade de armadura")
	}
	return marks
}

// firstUpper é o dado em caixa baixa com a inicial maiúscula, para a tela: o dado
// guarda `negativa` e `contato` porque é assim que o filtro os casa.
func firstUpper(text string) string {
	if text == "" {
		return ""
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

// nonEmpty é o crachá que só sai quando o dado existe.
func nonEmpty(text string) []string {
	if text == "" {
		return nil
	}
	return []string{text}
}

func classTarget(name string) (string, string) { return "classes", book.KeyOfName(name) }

func expertiseTarget(name string) (string, string) { return "pericias", book.KeyOfName(name) }

func powerTarget(name string) (string, string) {
	if id := bookui.PowerID(name); id != "" {
		return "poderes", id
	}
	return "", ""
}

// priceWritten é o dinheiro do livro em pt-BR: vírgula decimal e no máximo duas
// casas.
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
