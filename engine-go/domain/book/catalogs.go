package book

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/search"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

type Condition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	// UpgradesTo é a condição em que esta AGRAVA — "Abalado" vira "Apavorado".
	UpgradesTo string `json:"upgradesTo,omitempty"`
	// BookPage é a página IMPRESSA do verbete, derivada do Índice Remissivo do
	// próprio livro e conferida contra o texto da página
	// (`scripts/book-pages.py`). ZERO significa "o catálogo não sabe", e a
	// tela não desenha selo nenhum — mentir a página é pior que não mostrá-la.
	BookPage int `json:"bookPage"`
}

// SpellAugment é o que o livro imprime abaixo da magia: quanto custa a mais e o
// que muda.
type SpellAugment struct {
	PmCost      int    `json:"pmCost"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// AugmentKindName: o livro separa o que AUMENTA um efeito do que o
// MUDA, e a diferença importa na hora de gastar mana.
func AugmentKindName(kind string) string {
	if kind == "aumenta" {
		return "aumenta"
	}
	return "muda"
}

type Spell struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Circle     int            `json:"circle"`
	School     string         `json:"school"`
	Execution  string         `json:"execution"`
	Range      string         `json:"range"`
	Duration   string         `json:"duration"`
	Resistance string         `json:"resistance,omitempty"`
	BaseEffect string         `json:"baseEffect"`
	Augments   []SpellAugment `json:"augments"`
	Classes    []string       `json:"classes"`
	BookPage   int            `json:"bookPage"`
}

// Power é o poder ACHATADO. O livro espalha poder por três catálogos —
// habilidade de classe, poder geral/de combate e poder concedido —, e o mestre
// quer UMA lista buscável. A `Fonte` diz de onde veio, que é o que o
// achatamento não pode perder.
type Power struct {
	ID          string
	Name        string
	Source      string
	Description string
	BookPage    int
}

// Item é a entrada do catálogo de itens — o que a vitrine do mestre mostra E o
// que a Mochila do jogador precisa. Um segundo leitor do mesmo `items.json`
// daria duas verdades sobre o mesmo arquivo, então quem cresce é este.
type Item struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Slots    float64 `json:"slots"`
	BookPage int     `json:"bookPage"`
	// Equip é o eixo do livro: `vested`, `wielded` ou `either`.
	Equip string `json:"equip"`
	Hands int    `json:"hands"`
	// AppliesTo é a família que uma MELHORIA ou um MATERIAL aceita — arma,
	// armadura, escudo, vestuário. Vazio em tudo que não é sobreposição.
	AppliesTo  []string          `json:"appliesTo"`
	Weapon     *Weapon           `json:"weapon"`
	Armor      *Armor            `json:"armor"`
	Shield     *Armor            `json:"shield"`
	Consumable *Consumable       `json:"consumable"`
	Modifiers  []engine.Modifier `json:"modifiers"`
}

type Weapon struct {
	Damage    string   `json:"damage"`
	CritRange int      `json:"critRange"`
	CritMult  int      `json:"critMult"`
	Type      string   `json:"type"`
	Purpose   string   `json:"purpose"`
	Traits    []string `json:"traits"`
}

// Armor serve armadura E escudo: os dois trazem os mesmos três
// números, e o livro os apresenta na mesma tabela (p154).
type Armor struct {
	Defense int  `json:"defense"`
	Penalty int  `json:"penalty"`
	Heavy   bool `json:"heavy"`
}

type Consumable struct {
	Scope   string         `json:"scope"`
	Instant *ImmediateGain `json:"instant"`
}

// ImmediateGain é o PV/PM que um consumível devolve na hora. O `Dice` é a
// rolagem que a MESA faz — a ficha não rola por ninguém.
type ImmediateGain struct {
	HP *GainRoll `json:"hp"`
	MP *GainRoll `json:"mp"`
}

type GainRoll struct {
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus"`
}

// ── a leitura, uma vez só ────────────────────────────────────────────────────

// Catalogs lê os quatro catálogos e os ORDENA uma vez.
//
// Ordenar aqui e não a cada pedido: a ordem não depende do filtro, e refazer
// quatro ordenações a cada tecla digitada seria trabalho por nada. O
// `sync.Once` é o mesmo padrão do bestiário.
func Catalogs() GMCatalogs {
	acervoUmaVez.Do(func() {
		col := collate.New(language.BrazilianPortuguese)
		byName := func(a, b string) int { return col.CompareString(a, b) }

		acervo.Conditions = MapOf[Condition]("conditions")
		slices.SortStableFunc(acervo.Conditions, func(a, b Condition) int {
			return byName(a.Name, b.Name)
		})

		acervo.Spells = MapOf[Spell]("spells")
		// Magia ordena por CÍRCULO e depois por nome: o mestre procura "o que
		// existe de 3º círculo", e alfabético puro embaralharia os círculos.
		slices.SortStableFunc(acervo.Spells, func(a, b Spell) int {
			if a.Circle != b.Circle {
				return a.Circle - b.Circle
			}
			return byName(a.Name, b.Name)
		})

		acervo.Powers = FlattenedPowers()
		slices.SortStableFunc(acervo.Powers, func(a, b Power) int {
			return byName(a.Name, b.Name)
		})

		acervo.Items = ListOf[Item]("items")
		slices.SortStableFunc(acervo.Items, func(a, b Item) int {
			return byName(a.Name, b.Name)
		})
	})
	return acervo
}

// MapOf lê um recurso guardado como OBJETO por id e devolve os valores.
//
// Catálogo ausente ou malformado devolve lista vazia em vez de derrubar a Mesa:
// a ferramenta abre sem aquela aba, e as outras três continuam servindo.
func MapOf[T any](name string) []T {
	raw, ok := catalog.Resource(name)
	if !ok {
		return nil
	}
	var byID map[string]T
	if err := json.Unmarshal(raw, &byID); err != nil {
		return nil
	}
	outside := make([]T, 0, len(byID))
	for _, v := range byID {
		outside = append(outside, v)
	}
	return outside
}

// ListOf lê um recurso guardado como ARRAY.
func ListOf[T any](name string) []T {
	raw, ok := catalog.Resource(name)
	if !ok {
		return nil
	}
	var list []T
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
}

// FlattenedPowers junta os três catálogos de poder numa lista só: habilidade de
// classe, poder geral/de combate e poder concedido pelos deuses.
func FlattenedPowers() []Power {
	var outside []Power

	for _, p := range ListOf[struct {
		ID          string `json:"id"`
		ClassName   string `json:"className"`
		Name        string `json:"name"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}]("class-powers") {
		outside = append(outside, Power{
			ID: p.ID, Name: p.Name, Source: p.ClassName, Description: p.Description, BookPage: p.BookPage,
		})
	}

	for _, p := range ListOf[struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}]("general-powers") {
		outside = append(outside, Power{
			ID: "general." + p.ID, Name: p.Name, Source: "Geral · " + p.Kind,
			Description: p.Description, BookPage: p.BookPage,
		})
	}

	return append(outside, DivinePowers()...)
}

// DivinePowers são os que os DEUSES concedem.
//
// Lidos do `divine-powers` e NÃO do `granted-powers`, que tem metade dos nomes.
// O completo guarda uma linha por (poder, DEUS) — "Coragem Total" aparece quatro
// vezes, uma para Arsenal, Khalmyr, Lin-Wu e Valkaria, com a mesma descrição —,
// então aqui eles são juntados por NOME e os deuses viram a fonte. Sem juntar, a
// lista teria o mesmo poder quatro vezes e o elo não saberia para qual apontar.
func DivinePowers() []Power {
	type divino struct {
		DeusID      string `json:"deusId"`
		Name        string `json:"name"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}

	byName := map[string]*Power{}
	var order []string
	for _, p := range ListOf[divino]("divine-powers") {
		if found, present := byName[p.Name]; present {
			found.Source += ", " + GodName(p.DeusID)
			continue
		}
		byName[p.Name] = &Power{
			// O id é o NOME em forma de chave: o `divine-powers` não traz `id`,
			// e o elo endereça por id. Prefixado para não colidir com um poder
			// de classe de mesmo nome.
			ID:          "divino." + KeyOfName(p.Name),
			Name:        p.Name,
			Source:      "Divino · " + GodName(p.DeusID),
			Description: p.Description,
			BookPage:    p.BookPage,
		}
		order = append(order, p.Name)
	}

	outside := make([]Power, 0, len(order))
	for _, name := range order {
		outside = append(outside, *byName[name])
	}
	return outside
}

// KeyOfName transforma um nome em chave de endereço: sem acento, minúsculo,
// espaços viram hífen. É a mesma forma dos ids que os catálogos já usam.
func KeyOfName(name string) string {
	return strings.ReplaceAll(search.Fold(name), " ", "-")
}

// ── o que cada catálogo busca ────────────────────────────────────────────────
//
// Os campos ficam aqui e não espalhados na cena, para a aba e a busca unificada
// concordarem POR CONSTRUÇÃO em vez de por cópia.

func ConditionFields(c Condition) []string {
	return append([]string{c.Name, c.Description}, c.Tags...)
}
func SpellFields(m Spell) []string { return []string{m.Name, m.BaseEffect} }
func PowerFields(p Power) []string { return []string{p.Name, p.Source, p.Description} }
func ItemFields(i Item) []string   { return []string{i.Name, i.Category} }

// ── a busca unificada ────────────────────────────────────────────────────────

func CastingName(e string) string {
	if r, ok := executionLabel[e]; ok {
		return r
	}
	return e
}

func RangeName(a string) string {
	if r, ok := rangeLabel[a]; ok {
		return r
	}
	return a
}

func CategoryName(c string) string {
	if r, ok := categoryLabel[c]; ok {
		return r
	}
	return c
}

// ── o que a cena precisa escrever ────────────────────────────────────────────

// ConditionName resolve o id de uma condição no nome que se lê. Sem ela a linha
// sai "Agrava para apavorado" em caixa baixa: o dado do agravamento é um id, e o
// nome existe no mesmo catálogo.
func ConditionName(id string) string {
	for _, c := range Catalogs().Conditions {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

// Escrito diz o custo como o livro: "+2 PM".
func (a SpellAugment) Escrito() string { return fmt.Sprintf("+%d PM", a.PmCost) }

type GMCatalogs struct {
	Conditions []Condition
	Spells     []Spell
	Powers     []Power
	Items      []Item
}

var (
	acervoUmaVez sync.Once
	acervo       GMCatalogs
)

// GodName resolve o id que o poder divino guarda ("lin-wu") no nome que se lê.
//
// Lê o catálogo DIRETO e NÃO pelo `CharacterCatalogs`: aquele carregador tem um
// `sync.Once` que chama o `FlattenedPowers`, e chamá-lo de volta daqui fecha o
// ciclo. `Once` reentrante trava para sempre — não é pânico, não é teste
// vermelho, é o processo parado, e o sintoma na bancada é "a suíte demora". Quem
// aponta o dedo é o `go test -timeout`, que despeja a pilha das goroutines.
var (
	deusesUmaVez sync.Once
	nomePorDeus  map[string]string
)

func GodName(id string) string {
	deusesUmaVez.Do(func() {
		nomePorDeus = map[string]string{}
		for _, d := range ListOf[struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}]("gods") {
			nomePorDeus[d.ID] = d.Name
		}
	})
	if name, found := nomePorDeus[id]; found {
		return name
	}
	return id
}

var executionLabel = map[string]string{
	"padrao":    "Padrão",
	"movimento": "Movimento",
	"completa":  "Completa",
	"livre":     "Livre",
	"reacao":    "Reação",
}

var rangeLabel = map[string]string{
	"pessoal":   "Pessoal",
	"toque":     "Toque",
	"curto":     "Curto",
	"medio":     "Médio",
	"longo":     "Longo",
	"ilimitado": "Ilimitado",
}

var categoryLabel = map[string]string{
	"weapon-simple":  "Arma simples",
	"weapon-martial": "Arma marcial",
	"weapon-exotic":  "Arma exótica",
	"weapon-firearm": "Arma de fogo",
	"armor-light":    "Armadura leve",
	"armor-heavy":    "Armadura pesada",
	"shield":         "Escudo",
	"apparel":        "Vestuário",
	"consumable":     "Consumível",
	"meal":           "Refeição",
	"catalyst":       "Catalisador",
	"improvement":    "Melhoria",
	"material":       "Material",
	"animal":         "Animal",
	"vehicle":        "Veículo",
}

// WithSign escreve o modificador como o livro: "+2", "-1". O irmão que aceita
// ausência é o `WithSignPtr`, no `bestiary.go`.
func WithSign(n int) string {
	if n >= 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// OS ELOS entre entradas do acervo.
//
// O livro é uma rede: a condição Abalado termina em "Medo.", que é um TIPO DE
// EFEITO definido na p228; ela agrava para Apavorado, que é outra condição; o
// deus concede poderes que têm verbete próprio.
//
// O elo leva para a MESMA cena com a entrada filtrada (`routes.MasterSearch`),
// que é o endereço que o buscador já usa — nenhuma superfície nova.
//
// O que NÃO virou elo, e é decisão: nome de entrada citado no meio de descrição
// de MAGIA ou de PODER. São 3 citações em 668 entradas, porque as descrições da
// casa são resumos e não o texto do livro, e varrer 992 descrições atrás de 35
// nomes para achar três acertos é custo por tela sem retorno. Nas CONDIÇÕES são
// 11 em 35, e por isso elas têm.

// trecho é um pedaço de descrição. `Aba` vazia é texto puro; preenchida, o
// pedaço é um ELO para aquela aba do acervo.
//
// Um tipo e não HTML montado em string: texto do catálogo passa pelo escape do
// templ como qualquer outro, e montar `<a>` aqui seria abrir mão disso para
// sempre — a primeira descrição com um `<` viraria tela quebrada ou pior.
type Chunk struct {
	Text string
	Aba  string
	// ID é a chave do verbete de destino. O elo endereça por ID e não pelo
	// texto: nome é tela e muda com revisão do livro, id é como os catálogos já
	// se referem uns aos outros.
	ID string
	// Page, quando maior que zero, faz o pedaço virar um elo para o LIVRO em
	// vez de para o acervo: é uma referência escrita no texto ("veja a página
	// 230"), e ela merece o mesmo clique que o botão de página do cartão.
	Page int
}

// WithConditionLinks parte a descrição nos nomes de CONDIÇÃO que ela cita.
//
//	"Desprevenido e imóvel; -2 em ataques"
//	→ [{Desprevenido, condicoes}, {" e imóvel; -2 em ataques", ""}]
//
// Do MAIS LONGO para o mais curto, e isso é regra e não gosto: "Desprevenido"
// contém "Desprevenido" e nada mais, mas o dia em que existir "Cego" e "Cego de
// Nascença", casar o curto primeiro deixaria metade do nome longo solta na tela.
//
// A própria entrada é excluída: um elo que aponta para a página em que já se
// está é ruído com cara de saída.
func WithConditionLinks(text, except string) []Chunk {
	return WithPageLinks(splitOnNames(text, conditionNamesBySize(), except, "condicoes"))
}

// pageRef é como o livro cita a si mesmo: "veja a página 230", "pág. 172".
var pageRef = regexp.MustCompile(`(?i)p[áa]g(?:ina)?\.?\s*(\d{1,3})`)

// WithPageLinks parte os pedaços de TEXTO PURO nas referências de página.
//
// Roda DEPOIS da varredura de nomes e só sobre o que sobrou como texto: um
// pedaço que já virou elo para um verbete não pode virar elo para o livro
// também — dois destinos na mesma palavra é uma escolha que ninguém pediu.
func WithPageLinks(chunks []Chunk) []Chunk {
	var outside []Chunk
	for _, chunk := range chunks {
		if chunk.Aba != "" {
			outside = append(outside, chunk)
			continue
		}
		outside = append(outside, splitOnPages(chunk.Text)...)
	}
	return outside
}

func splitOnPages(text string) []Chunk {
	marks := pageRef.FindAllStringSubmatchIndex(text, -1)
	if marks == nil {
		return []Chunk{{Text: text}}
	}
	var outside []Chunk
	end := 0
	for _, m := range marks {
		page, err := strconv.Atoi(text[m[2]:m[3]])
		if err != nil || page <= 0 {
			continue
		}
		if before := text[end:m[0]]; before != "" {
			outside = append(outside, Chunk{Text: before})
		}
		outside = append(outside, Chunk{Text: text[m[0]:m[1]], Page: page})
		end = m[1]
	}
	if rest := text[end:]; rest != "" {
		outside = append(outside, Chunk{Text: rest})
	}
	return outside
}

// WithLinks é a varredura para os catálogos que NÃO citam condições — só as
// referências de página. Ver o cabeçalho para por que magia e poder ficam de
// fora da varredura de nomes.
func WithLinks(text string) []Chunk {
	return splitOnPages(text)
}

// conditionID resolve o nome no id com que o catálogo a guarda.
func conditionID(name string) string {
	for _, c := range Catalogs().Conditions {
		if c.Name == name {
			return c.ID
		}
	}
	return ""
}

var (
	nomesUmaVez sync.Once
	nomesLongos []string
)

func conditionNamesBySize() []string {
	nomesUmaVez.Do(func() {
		for _, c := range Catalogs().Conditions {
			nomesLongos = append(nomesLongos, c.Name)
		}
		slices.SortFunc(nomesLongos, func(a, b string) int { return len(b) - len(a) })
	})
	return nomesLongos
}

// splitOnNames é a varredura, e ela casa PALAVRA INTEIRA com a caixa do livro.
//
// Caixa exata porque no texto do livro a condição é escrita com maiúscula
// ("fica Abalado") e a palavra comum não ("um efeito de medo") — casar sem caixa
// encheria a tela de elos que não são citação nenhuma.
func splitOnNames(text string, names []string, except, aba string) []Chunk {
	for _, name := range names {
		if name == except {
			continue
		}
		where := wholeWordIndex(text, name)
		if where < 0 {
			continue
		}
		var outside []Chunk
		if before := text[:where]; before != "" {
			outside = append(outside, splitOnNames(before, names, except, aba)...)
		}
		outside = append(outside, Chunk{Text: name, Aba: aba, ID: conditionID(name)})
		if after := text[where+len(name):]; after != "" {
			outside = append(outside, splitOnNames(after, names, except, aba)...)
		}
		return outside
	}
	return []Chunk{{Text: text}}
}

// wholeWordIndex acha o nome com fronteira dos dois lados, ou -1.
func wholeWordIndex(text, name string) int {
	de := 0
	for {
		where := strings.Index(text[de:], name)
		if where < 0 {
			return -1
		}
		where += de
		if isBoundary(text, where-1) && isBoundary(text, where+len(name)) {
			return where
		}
		de = where + 1
	}
}

// fronteira: fora do texto conta como fronteira, e letra não conta.
func isBoundary(text string, i int) bool {
	if i < 0 || i >= len(text) {
		return true
	}
	r := rune(text[i])
	return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r >= 0x80)
}

// ── os elos que vêm de CAMPO e não de texto ──────────────────────────────────

// DevoteeLink acha a aba e o id de um devoto do deus ("Elfos", "Bárbaros").
//
// O dado vem no PLURAL e as entradas são singulares, e tirar "s"/"es" não basta:
// "Anões", "Golens" e "Sereias/Tritões" são plurais que o português não faz
// acrescentando letra, e o último é composto com as DUAS metades no plural.
//
// "Aggelus" e "Sulfure" não são plural de nada: são as ASCENDÊNCIAS do suraggel,
// e o catálogo já as guarda no campo `ascendencias`. O elo leva à raça que as
// contém — resolver por dado e não por uma tabela de exceções escrita à mão, que
// envelheceria na primeira raça nova.
//
// Não achou, não vira elo: "Quaisquer" e "Aventureiros (todas as classes)" não
// são verbete de nada.
func DevoteeLink(name string) (aba, id string) {
	races, classes, _ := CharacterCatalogs()
	for _, candidate := range singular(name) {
		for _, r := range races {
			if r.Name == candidate {
				return "racas", r.ID
			}
		}
		for _, c := range classes {
			if c.Name == candidate {
				return "classes", c.ID
			}
		}
	}
	for _, r := range races {
		if slices.Contains(r.Ancestries, fold(name)) {
			return "racas", r.ID
		}
	}
	return "", ""
}

// singular devolve as formas a tentar, do nome como veio ao singular provável.
// Cada regra é um caso do catálogo:
//
//	"Elfos"            → "Elfo"      (s)
//	"Caçadores"        → "Caçador"   (es)
//	"Anões"            → "Anão"      (ões → ão)
//	"Golens"           → "Golem"     (ns → m)
//	"Sereias/Tritões"  → "Sereia/Tritão"  (as duas metades)
func singular(name string) []string {
	singular := func(word string) []string {
		outside := []string{word}
		for de, to := range map[string]string{"ões": "ão", "ãos": "ão", "ns": "m", "es": "", "s": ""} {
			if strings.HasSuffix(word, de) {
				outside = append(outside, strings.TrimSuffix(word, de)+to)
			}
		}
		return outside
	}
	if !strings.Contains(name, "/") {
		return singular(name)
	}
	// Nome composto: cada metade vai para o singular, e só a combinação de todas
	// as metades no singular casa "Sereia/Tritão".
	var parts [][]string
	for _, part := range strings.Split(name, "/") {
		parts = append(parts, singular(part))
	}
	excluded := []string{name}
	for _, left := range parts[0] {
		for _, right := range parts[len(parts)-1] {
			excluded = append(excluded, left+"/"+right)
		}
	}
	return excluded
}

// fold é minúsculas sem acento, para casar a ascendência que o catálogo
// guarda em caixa baixa ("aggelus") com o nome que o deus escreve ("Aggelus").
func fold(s string) string {
	return strings.ToLower(s)
}

// DefenseLabel é a Defesa como a TELA a diz, e é UMA função porque a ficha e o
// diálogo do elenco mostram o mesmo herói: uma frase montada em dois lugares
// diverge no dia em que uma delas ganhar um caso.
//
// O Caído dá −5 na Defesa contra ataques corpo a corpo e +5 contra ataques à
// distância (T20 p394), então um personagem caído não tem "a Defesa": tem duas.
// O motor calcula as duas e mantém o `Total` intacto, que é a leitura certa do
// livro.
//
// O TOTAL SAI do rótulo quando as duas divergem (decisão do dono): na mesa a
// pergunta é "acerta?", e enquanto o alvo está caído nenhum ataque testa contra
// o total — mostrá-lo em destaque é mostrar com confiança o único número que não
// responde a pergunta feita. Ele continua no diálogo de decomposição, com as
// duas linhas que dizem de onde os ±5 vêm.
func DefenseLabel(d engine.DefenseBreakdown) string {
	if d.VsMelee == d.Total && d.VsRanged == d.Total {
		return strconv.Itoa(d.Total)
	}
	// As abreviaturas são as que a ficha já usa nos ataques ("Atq CaC", "Atq
	// Dist"): duas grafias para o mesmo par fariam a mesma tela chamar corpo a
	// corpo de duas coisas.
	return strconv.Itoa(d.VsMelee) + " CaC · " + strconv.Itoa(d.VsRanged) + " Dist"
}
