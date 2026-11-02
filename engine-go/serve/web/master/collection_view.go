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

// OS CATÁLOGOS do mestre: condições, magias, poderes e itens numa busca só.
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

// collectionTab é uma parada da fileira de abas.
type collectionTab struct {
	ID    string
	Label string
}

// A ORDEM é por frequência na mesa: condição primeiro, porque é a consulta mais
// comum no meio do combate. Raça, classe e deus vão no FIM porque são consulta
// de CRIAÇÃO de personagem, e não de mesa com o combate em curso.
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
			return a.Label
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
	Label      string
	Conditions []book.Condition
	Spells     []book.Spell
	Powers     []book.Power
	Items      []book.Item
	Effects    []book.EffectKind
	Schools    []book.SpellSchool
	Expertises []book.Expertise
	Races      []book.Race
	Classes    []book.Class
	Gods       []book.God
}

// entryName é o nome do primeiro verbete do grupo, ou vazio. Quem o chama é a caixa
// do elo, em que o grupo tem um verbete só.
func (g collectionGroup) entryName() string {
	switch {
	case len(g.Conditions) > 0:
		return g.Conditions[0].Name
	case len(g.Spells) > 0:
		return g.Spells[0].Name
	case len(g.Powers) > 0:
		return g.Powers[0].Name
	case len(g.Items) > 0:
		return g.Items[0].Name
	case len(g.Effects) > 0:
		return g.Effects[0].Name
	case len(g.Schools) > 0:
		return g.Schools[0].Name
	case len(g.Expertises) > 0:
		return g.Expertises[0].Name
	case len(g.Races) > 0:
		return g.Races[0].Name
	case len(g.Classes) > 0:
		return g.Classes[0].Name
	case len(g.Gods) > 0:
		return g.Gods[0].Name
	}
	return ""
}

func (g collectionGroup) Count() int {
	return len(g.Conditions) + len(g.Spells) + len(g.Powers) + len(g.Items) +
		len(g.Effects) + len(g.Schools) + len(g.Expertises) + len(g.Races) + len(g.Classes) + len(g.Gods)
}

// collectionCriteria é o que a URL (ou os sinais) pedem da cena.
type collectionCriteria struct {
	Term string
	Aba  string
	// Entry é o ID de UM verbete, e ela ganha de tudo: com ela a cena mostra
	// aquele verbete sozinho. É o endereço que um ELO usa — quem clica em "Medo"
	// pediu o Medo, não uma busca por "medo" nos oito catálogos.
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

// loadCollection monta a cena: os quatro catálogos quando há busca, um só
// quando não há.
func loadCollection(c collectionCriteria, bookRef bookui.BookAddress) collectionView {
	v := collectionView{
		Term: c.Term, Aba: knownTab(c.Aba), Book: bookRef, Entry: c.Entry,
		Lit: c.Filters,
	}
	v.Filters = filtersForTab(v.Aba)
	v.Total = groupForTab(book.Catalogs(), v.Aba, nil).Count()
	// A ENTRADA vem primeiro e encerra: ela é um endereço para um verbete só, e
	// misturá-la com busca daria uma tela que responde duas perguntas.
	if c.Entry != "" {
		v.Groups = []collectionGroup{groupForEntry(v.Aba, c.Entry)}
		v.Findings = v.Groups[0].Count()
		return v
	}
	search := c.Term
	a := book.Catalogs()

	if !v.Searching() {
		v.Groups = []collectionGroup{groupForTab(a, v.Aba, c.Filters)}
		v.Findings = v.Groups[0].Count()
		return v
	}

	races, classes, gods := book.CharacterCatalogs()
	for _, g := range []collectionGroup{
		{Label: "Condições", Conditions: filter(a.Conditions, book.ConditionFields, search)},
		{Label: "Magias", Spells: filter(a.Spells, book.SpellFields, search)},
		{Label: "Poderes", Powers: filter(a.Powers, book.PowerFields, search)},
		{Label: "Itens", Items: filter(a.Items, book.ItemFields, search)},
		{Label: "Efeitos", Effects: filter(book.EffectKinds(), book.EffectFields, search)},
		{Label: "Escolas", Schools: filter(book.SpellSchools(), book.SchoolFields, search)},
		{Label: "Perícias", Expertises: filter(book.Expertises(), book.ExpertiseFields, search)},
		{Label: "Raças", Races: filter(races, book.RaceFields, search)},
		{Label: "Classes", Classes: filter(classes, book.ClassFields, search)},
		{Label: "Deuses", Gods: filter(gods, book.GodFields, search)},
	} {
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
	whole := groupForTab(book.Catalogs(), aba, nil)
	outside := collectionGroup{Label: whole.Label}
	for _, c := range whole.Conditions {
		if c.ID == id {
			outside.Conditions = append(outside.Conditions, c)
		}
	}
	for _, m := range whole.Spells {
		if m.ID == id {
			outside.Spells = append(outside.Spells, m)
		}
	}
	for _, p := range whole.Powers {
		if p.ID == id {
			outside.Powers = append(outside.Powers, p)
		}
	}
	for _, i := range whole.Items {
		if i.ID == id {
			outside.Items = append(outside.Items, i)
		}
	}
	for _, e := range whole.Effects {
		if e.ID == id {
			outside.Effects = append(outside.Effects, e)
		}
	}
	for _, e := range whole.Schools {
		if e.ID == id {
			outside.Schools = append(outside.Schools, e)
		}
	}
	for _, p := range whole.Expertises {
		if p.ID == id {
			outside.Expertises = append(outside.Expertises, p)
		}
	}
	for _, r := range whole.Races {
		if r.ID == id {
			outside.Races = append(outside.Races, r)
		}
	}
	for _, c := range whole.Classes {
		if c.ID == id {
			outside.Classes = append(outside.Classes, c)
		}
	}
	for _, d := range whole.Gods {
		if d.ID == id {
			outside.Gods = append(outside.Gods, d)
		}
	}
	return outside
}

// groupForTab monta o catálogo da cena, já com os crachás aplicados.
func groupForTab(a book.GMCatalogs, aba string, lit map[string][]string) collectionGroup {
	races, classes, gods := book.CharacterCatalogs()
	switch aba {
	case "magias":
		return collectionGroup{Label: "Magias", Spells: applyFilters(a.Spells, lit, spellMatches)}
	case "poderes":
		return collectionGroup{Label: "Poderes", Powers: applyFilters(a.Powers, lit, powerMatches)}
	case "itens":
		return collectionGroup{Label: "Itens", Items: applyFilters(a.Items, lit, itemMatches)}
	case "efeitos":
		return collectionGroup{Label: "Efeitos", Effects: book.EffectKinds()}
	case "escolas":
		return collectionGroup{Label: "Escolas", Schools: book.SpellSchools()}
	case "pericias":
		return collectionGroup{Label: "Perícias", Expertises: applyFilters(book.Expertises(), lit, expertiseMatches)}
	case "racas":
		return collectionGroup{Label: "Raças", Races: applyFilters(races, lit, raceMatches)}
	case "classes":
		return collectionGroup{Label: "Classes", Classes: classes}
	case "deuses":
		return collectionGroup{Label: "Deuses", Gods: applyFilters(gods, lit, godMatches)}
	default:
		return collectionGroup{Label: "Condições", Conditions: applyFilters(a.Conditions, lit, conditionMatches)}
	}
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
	if duration := book.DurationName(m.Duration); duration != "" {
		parts = append(parts, "Duração: "+duration)
	}
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

// energyWritten é a energia do deus com a inicial maiúscula: o dado a guarda em
// caixa baixa (`negativa`), que é como o filtro a casa.
func energyWritten(energy string) string {
	if energy == "" {
		return ""
	}
	return strings.ToUpper(energy[:1]) + energy[1:]
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
