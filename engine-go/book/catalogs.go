package book

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"t20engine/catalog"
	"t20engine/engine"
	"t20engine/search"

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
	// (`scripts/paginas-do-livro.py`). ZERO significa "o catálogo não sabe", e a
	// tela não desenha selo nenhum — mentir a página é pior que não mostrá-la.
	BookPage int `json:"bookPage"`
}

// SpellAugment é o que o livro imprime abaixo da magia: quanto custa a
// mais e o que muda. Eram `[]any` — a cena só contava quantos havia —, e o
// dono pediu para poder LER cada um.
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
	Fonte       string
	Description string
	BookPage    int
}

// Item é a entrada do catálogo de itens.
//
// Ela nasceu com os seis campos que a vitrine do mestre mostra e cresceu na
// ALE-272 (fatia 7): a Mochila do jogador precisa do EIXO de equipar, das
// estatísticas de arma/armadura/escudo, do consumível e da família a que uma
// melhoria se aplica. Um segundo leitor do mesmo `items.json` daria duas
// verdades sobre o mesmo arquivo, então quem cresce é este.
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
		porNome := func(a, b string) int { return col.CompareString(a, b) }

		acervo.Condicoes = MapOf[Condition]("conditions")
		slices.SortStableFunc(acervo.Condicoes, func(a, b Condition) int {
			return porNome(a.Name, b.Name)
		})

		acervo.Magias = MapOf[Spell]("spells")
		// Magia ordena por CÍRCULO e depois por nome, como a SPA: o mestre
		// procura "o que existe de 3º círculo", e alfabético puro embaralharia
		// os círculos.
		slices.SortStableFunc(acervo.Magias, func(a, b Spell) int {
			if a.Circle != b.Circle {
				return a.Circle - b.Circle
			}
			return porNome(a.Name, b.Name)
		})

		acervo.Poderes = FlattenedPowers()
		slices.SortStableFunc(acervo.Poderes, func(a, b Power) int {
			return porNome(a.Name, b.Name)
		})

		acervo.Itens = ListOf[Item]("items")
		slices.SortStableFunc(acervo.Itens, func(a, b Item) int {
			return porNome(a.Name, b.Name)
		})
	})
	return acervo
}

// MapOf lê um recurso guardado como OBJETO por id e devolve os valores.
//
// Catálogo ausente ou malformado devolve lista vazia em vez de derrubar a Mesa:
// a ferramenta abre sem aquela aba, e as outras três continuam servindo.
func MapOf[T any](nome string) []T {
	bruto, ok := catalog.Resource(nome)
	if !ok {
		return nil
	}
	var porID map[string]T
	if err := json.Unmarshal(bruto, &porID); err != nil {
		return nil
	}
	fora := make([]T, 0, len(porID))
	for _, v := range porID {
		fora = append(fora, v)
	}
	return fora
}

// ListOf lê um recurso guardado como ARRAY.
func ListOf[T any](nome string) []T {
	bruto, ok := catalog.Resource(nome)
	if !ok {
		return nil
	}
	var lista []T
	if err := json.Unmarshal(bruto, &lista); err != nil {
		return nil
	}
	return lista
}

// FlattenedPowers junta os três catálogos de poder numa lista só.
//
// Os poderes DIVINOS ficam de fora, e a razão é da SPA: o dado deles carrega
// página do livro e nenhum texto de regra, então não há o que consultar.
func FlattenedPowers() []Power {
	var fora []Power

	for _, p := range ListOf[struct {
		ID          string `json:"id"`
		ClassName   string `json:"className"`
		Name        string `json:"name"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}]("class-powers") {
		fora = append(fora, Power{
			ID: p.ID, Name: p.Name, Fonte: p.ClassName, Description: p.Description, BookPage: p.BookPage,
		})
	}

	for _, p := range ListOf[struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}]("general-powers") {
		fora = append(fora, Power{
			ID: "general." + p.ID, Name: p.Name, Fonte: "Geral · " + p.Kind,
			Description: p.Description, BookPage: p.BookPage,
		})
	}

	return append(fora, DivinePowers()...)
}

// DivinePowers são os que os DEUSES concedem, e este bloco é conserto de uma
// lacuna que o dono viu na tela: nos cartões de Valkaria, Wynna e Thwor a maior
// parte dos poderes concedidos não virava elo.
//
// A causa era um comentário desatualizado. Ele dizia que os poderes divinos
// "carregam página do livro e nenhum texto de regra, então não há o que
// consultar" — e o dado DESMENTE: os 80 têm descrição completa. Por causa dessa
// frase o acervo lia só o `granted-powers`, que são 36 dos 72 nomes.
//
// Lidos do `divine-powers`, que é o catálogo completo. Ele guarda uma linha por
// (poder, DEUS) — "Coragem Total" aparece quatro vezes, uma para Arsenal,
// Khalmyr, Lin-Wu e Valkaria, com a mesma descrição —, então aqui eles são
// juntados por NOME e os deuses viram a fonte. Sem juntar, a lista teria o mesmo
// poder quatro vezes e o elo não saberia para qual apontar.
func DivinePowers() []Power {
	type divino struct {
		DeusID      string `json:"deusId"`
		Name        string `json:"name"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}

	porNome := map[string]*Power{}
	var ordem []string
	for _, p := range ListOf[divino]("divine-powers") {
		if achado, tem := porNome[p.Name]; tem {
			achado.Fonte += ", " + GodName(p.DeusID)
			continue
		}
		porNome[p.Name] = &Power{
			// O id é o NOME em forma de chave: o `divine-powers` não traz `id`,
			// e o elo endereça por id. Prefixado para não colidir com um poder
			// de classe de mesmo nome.
			ID:          "divino." + KeyOfName(p.Name),
			Name:        p.Name,
			Fonte:       "Divino · " + GodName(p.DeusID),
			Description: p.Description,
			BookPage:    p.BookPage,
		}
		ordem = append(ordem, p.Name)
	}

	fora := make([]Power, 0, len(ordem))
	for _, nome := range ordem {
		fora = append(fora, *porNome[nome])
	}
	return fora
}

// KeyOfName transforma um nome em chave de endereço: sem acento, minúsculo,
// espaços viram hífen. É a mesma forma dos ids que os catálogos já usam.
func KeyOfName(nome string) string {
	return strings.ReplaceAll(search.Fold(nome), " ", "-")
}

// ── o que cada catálogo busca ────────────────────────────────────────────────
//
// Os campos ficam aqui e não espalhados na cena, para a aba e a busca unificada
// concordarem POR CONSTRUÇÃO em vez de por cópia.

func ConditionFields(c Condition) []string {
	return append([]string{c.Name, c.Description}, c.Tags...)
}
func SpellFields(m Spell) []string { return []string{m.Name, m.BaseEffect} }
func PowerFields(p Power) []string { return []string{p.Name, p.Fonte, p.Description} }
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

// ConditionName resolve o id de uma condição no nome que se lê.
//
// DIVERGÊNCIA DELIBERADA do original, e por isso escrita: a SPA imprime o
// `upgradesTo` cru, então a linha sai "Agrava para apavorado" em caixa baixa. O
// dado do agravamento é um id, e o nome existe no mesmo catálogo — resolver é
// olhar a tabela ao lado, não inventar. Se alguém preferir o cru, muda aqui e
// nas duas telas.
func ConditionName(id string) string {
	for _, c := range Catalogs().Condicoes {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

// Escrito diz o custo como o livro: "+2 PM".
func (a SpellAugment) Escrito() string { return fmt.Sprintf("+%d PM", a.PmCost) }

type GMCatalogs struct {
	Condicoes []Condition
	Magias    []Spell
	Poderes   []Power
	Itens     []Item
}

var (
	acervoUmaVez sync.Once
	acervo       GMCatalogs
)

// GodName resolve o id que o poder divino guarda ("lin-wu") no nome que se lê.
//
// Lê o catálogo DIRETO e não pelo `CharacterCatalogs`, e isto é conserto de
// um DEADLOCK que pendurou a suíte inteira sem erro nenhum: aquele carregador
// tem um `sync.Once` que chama o `FlattenedPowers` para contar os poderes de
// cada classe, e o `FlattenedPowers` chamava de volta o `CharacterCatalogs`
// daqui. `Once` reentrante trava para sempre — não é pânico, não é teste
// vermelho: é o processo parado.
//
// Quem apontou o dedo foi o `go test -timeout 25s`, que despeja a pilha de todas
// as goroutines. Sem o timeout, o sintoma era "a suíte demora".
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
		}]("deuses") {
			nomePorDeus[d.ID] = d.Name
		}
	})
	if nome, tem := nomePorDeus[id]; tem {
		return nome
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

// OS ELOS entre entradas do acervo (ALE-264).
//
// O livro é uma rede: a condição Abalado termina em "Medo.", que é um TIPO DE
// EFEITO definido na p228; ela agrava para Apavorado, que é outra condição; o
// deus concede poderes que têm verbete próprio. Na tela isso era tudo texto
// morto — o mestre lia "Medo" e tinha de ir procurar o que era.
//
// O elo leva para a MESMA cena com a entrada filtrada
// (`routes.MasterSearch`), que é o endereço que o buscador já usa. Nenhuma
// superfície nova: a entrada aparece sozinha na aba dela, e o botão do livro
// está ao lado se a pessoa quiser o texto completo.
//
// O que NÃO virou elo, e é decisão: nome de entrada citado no meio de descrição
// de MAGIA ou de PODER. Medido no catálogo — são 3 citações em 668 entradas,
// porque as descrições da casa são resumos e não o texto do livro. Varrer 992
// descrições atrás de 35 nomes para achar três acertos é custo por tela sem
// retorno. Nas CONDIÇÕES o número é outro (11 em 35) e por isso elas têm.

// trecho é um pedaço de descrição. `Aba` vazia é texto puro; preenchida, o
// pedaço é um ELO para aquela aba do acervo.
//
// Um tipo e não HTML montado em string: texto do catálogo passa pelo escape do
// templ como qualquer outro, e montar `<a>` aqui seria abrir mão disso para
// sempre — a primeira descrição com um `<` viraria tela quebrada ou pior.
type Chunk struct {
	Texto string
	Aba   string
	// ID é a chave do verbete de destino. O elo endereça por ID e não pelo
	// texto: nome é tela e muda com revisão do livro, id é como os catálogos já
	// se referem uns aos outros.
	ID string
	// Pagina, quando maior que zero, faz o pedaço virar um elo para o LIVRO em
	// vez de para o acervo: é uma referência escrita no texto ("veja a página
	// 230"), e ela merece o mesmo clique que o botão de página do cartão.
	Pagina int
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
func WithConditionLinks(texto, exceto string) []Chunk {
	return WithPageLinks(splitOnNames(texto, conditionNamesBySize(), exceto, "condicoes"))
}

// pageRef é como o livro cita a si mesmo: "veja a página 230",
// "pág. 172". Medido no catálogo — são cinco ocorrências, duas nos tipos de
// efeito e três nos dragões —, e cada uma era texto morto: o número estava lá e
// não levava a lugar nenhum.
var pageRef = regexp.MustCompile(`(?i)p[áa]g(?:ina)?\.?\s*(\d{1,3})`)

// WithPageLinks parte os pedaços de TEXTO PURO nas referências de página.
//
// Roda DEPOIS da varredura de nomes e só sobre o que sobrou como texto: um
// pedaço que já virou elo para um verbete não pode virar elo para o livro
// também — dois destinos na mesma palavra é uma escolha que ninguém pediu.
func WithPageLinks(pedacos []Chunk) []Chunk {
	var fora []Chunk
	for _, pedaco := range pedacos {
		if pedaco.Aba != "" {
			fora = append(fora, pedaco)
			continue
		}
		fora = append(fora, splitOnPages(pedaco.Texto)...)
	}
	return fora
}

func splitOnPages(texto string) []Chunk {
	marcas := pageRef.FindAllStringSubmatchIndex(texto, -1)
	if marcas == nil {
		return []Chunk{{Texto: texto}}
	}
	var fora []Chunk
	fim := 0
	for _, m := range marcas {
		pagina, err := strconv.Atoi(texto[m[2]:m[3]])
		if err != nil || pagina <= 0 {
			continue
		}
		if antes := texto[fim:m[0]]; antes != "" {
			fora = append(fora, Chunk{Texto: antes})
		}
		fora = append(fora, Chunk{Texto: texto[m[0]:m[1]], Pagina: pagina})
		fim = m[1]
	}
	if resto := texto[fim:]; resto != "" {
		fora = append(fora, Chunk{Texto: resto})
	}
	return fora
}

// WithLinks é a varredura para os catálogos que NÃO citam condições — só as
// referências de página. Ver o cabeçalho: em magia e poder as citações de
// condição são 3 em 668, e varrer 992 descrições atrás delas é custo sem retorno.
func WithLinks(texto string) []Chunk {
	return splitOnPages(texto)
}

// conditionID resolve o nome no id com que o catálogo a guarda.
func conditionID(nome string) string {
	for _, c := range Catalogs().Condicoes {
		if c.Name == nome {
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
		for _, c := range Catalogs().Condicoes {
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
func splitOnNames(texto string, nomes []string, exceto, aba string) []Chunk {
	for _, nome := range nomes {
		if nome == exceto {
			continue
		}
		onde := wholeWordIndex(texto, nome)
		if onde < 0 {
			continue
		}
		var fora []Chunk
		if antes := texto[:onde]; antes != "" {
			fora = append(fora, splitOnNames(antes, nomes, exceto, aba)...)
		}
		fora = append(fora, Chunk{Texto: nome, Aba: aba, ID: conditionID(nome)})
		if depois := texto[onde+len(nome):]; depois != "" {
			fora = append(fora, splitOnNames(depois, nomes, exceto, aba)...)
		}
		return fora
	}
	return []Chunk{{Texto: texto}}
}

// wholeWordIndex acha o nome com fronteira dos dois lados, ou -1.
func wholeWordIndex(texto, nome string) int {
	de := 0
	for {
		onde := strings.Index(texto[de:], nome)
		if onde < 0 {
			return -1
		}
		onde += de
		if isBoundary(texto, onde-1) && isBoundary(texto, onde+len(nome)) {
			return onde
		}
		de = onde + 1
	}
}

// fronteira: fora do texto conta como fronteira, e letra não conta.
func isBoundary(texto string, i int) bool {
	if i < 0 || i >= len(texto) {
		return true
	}
	r := rune(texto[i])
	return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r >= 0x80)
}

// ── os elos que vêm de CAMPO e não de texto ──────────────────────────────────

// DevoteeLink acha a aba e o id de um devoto do deus ("Elfos", "Bárbaros").
//
// O dado vem no PLURAL e as entradas são singulares. A primeira versão tentava
// só tirar "s" e "es", e o dono viu os buracos: MEDIDOS, faltavam elo em
// "Anões", "Golens" e "Sereias/Tritões" — plurais que o português não faz
// acrescentando letra, e um nome composto por barra em que as DUAS metades vão
// para o plural.
//
// E em "Aggelus" e "Sulfure", que não são plural de nada: são as ASCENDÊNCIAS do
// suraggel, e o catálogo já as guarda no campo `ascendencias`. O elo leva à raça
// que as contém — resolver por dado e não por uma tabela de exceções escrita à
// mão, que envelheceria na primeira raça nova.
//
// Não achou, não vira elo: "Quaisquer" e "Aventureiros (todas as classes)" não
// são verbete de nada.
func DevoteeLink(nome string) (aba, id string) {
	racas, classes, _ := CharacterCatalogs()
	for _, candidato := range singular(nome) {
		for _, r := range racas {
			if r.Name == candidato {
				return "racas", r.ID
			}
		}
		for _, c := range classes {
			if c.Name == candidato {
				return "classes", c.ID
			}
		}
	}
	for _, r := range racas {
		if slices.Contains(r.Ascendencias, fold(nome)) {
			return "racas", r.ID
		}
	}
	return "", ""
}

// singular devolve as formas a tentar, do nome como veio ao singular provável.
//
// As regras são as do português, e cada uma nasceu de um caso do catálogo:
//
//	"Elfos"            → "Elfo"      (s)
//	"Caçadores"        → "Caçador"   (es)
//	"Anões"            → "Anão"      (ões → ão)
//	"Golens"           → "Golem"     (ns → m)
//	"Sereias/Tritões"  → "Sereia/Tritão"  (as duas metades)
func singular(nome string) []string {
	singular := func(palavra string) []string {
		fora := []string{palavra}
		for de, para := range map[string]string{"ões": "ão", "ãos": "ão", "ns": "m", "es": "", "s": ""} {
			if strings.HasSuffix(palavra, de) {
				fora = append(fora, strings.TrimSuffix(palavra, de)+para)
			}
		}
		return fora
	}
	if !strings.Contains(nome, "/") {
		return singular(nome)
	}
	// Nome composto: cada metade vai para o singular, e só a combinação de todas
	// as metades no singular casa "Sereia/Tritão".
	var partes [][]string
	for _, parte := range strings.Split(nome, "/") {
		partes = append(partes, singular(parte))
	}
	fora := []string{nome}
	for _, esquerda := range partes[0] {
		for _, direita := range partes[len(partes)-1] {
			fora = append(fora, esquerda+"/"+direita)
		}
	}
	return fora
}

// fold é minúsculas sem acento, para casar a ascendência que o catálogo
// guarda em caixa baixa ("aggelus") com o nome que o deus escreve ("Aggelus").
func fold(s string) string {
	return strings.ToLower(s)
}
