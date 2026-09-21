package book

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"t20engine/domain/catalog"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

type RaceTrait struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type Race struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Tier           string        `json:"tier"`
	Size           string        `json:"tamanho"`
	Speed          int           `json:"deslocamento"`
	Darkvision     bool          `json:"visaoNoEscuro"`
	LowLightVision bool          `json:"visaoNaPenumbra"`
	AttributeMod   RaceAttribute `json:"atributoMod"`
	// Ancestries são as metades de uma raça que se escolhe na criação — o
	// suraggel é "aggelus" ou "sulfure". Elas importam aqui porque os DEUSES
	// citam a ascendência e não a raça ("Devotos: Aggelus"), e sem isto o elo
	// desses dois não existiria.
	Ancestries []string    `json:"ascendencias"`
	Abilities  []RaceTrait `json:"abilities"`
	BookPage   int         `json:"bookPage"`
}

// RaceAttribute são as DUAS formas do livro, e elas não se reduzem a uma: o
// humano escolhe três +1 onde quiser (`floating`), o elfo recebe +2 Int, +1 Des
// e −1 Con (`fixed`). Guardar as duas num mapa só faria a escolha do humano
// virar três atributos inventados.
type RaceAttribute struct {
	Kind  string         `json:"kind"`
	Count int            `json:"count"`
	Value int            `json:"value"`
	Mods  map[string]int `json:"mods"`
	// Exclude é o atributo PROIBIDO na distribuição — o lefou não põe o +1 em
	// Carisma, o osteon não põe em Constituição, e os dois ainda levam um −1
	// nesse mesmo atributo. Sem ele a tela ofereceria uma escolha que o motor
	// recusa.
	Exclude string `json:"exclude"`
}

// Escrito diz os modificadores como o livro os escreve.
//
//	"+2 Int, +1 Des, −1 Con"       // elfo
//	"+1 em três atributos"          // humano
func (a RaceAttribute) Escrito() string {
	if a.Kind == "floating" {
		return fmt.Sprintf("%s em %s", WithSign(a.Value), InWords(a.Count))
	}
	var parts []string
	for _, attribute := range AttributeOrder {
		if mod, found := a.Mods[attribute.Key]; found && mod != 0 {
			parts = append(parts, WithSign(mod)+" "+attribute.Abbreviation)
		}
	}
	return strings.Join(parts, ", ")
}

// InWords escreve a contagem da escolha livre. Vai até seis porque são seis
// atributos — não há sétima escolha possível.
func InWords(n int) string {
	names := []string{"nenhum atributo", "um atributo", "dois atributos", "três atributos",
		"quatro atributos", "cinco atributos", "seis atributos"}
	if n < 0 || n >= len(names) {
		return fmt.Sprintf("%d atributos", n)
	}
	return names[n]
}

// ── classe ───────────────────────────────────────────────────────────────────

type Class struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BookPage int    `json:"bookPage"`
	// Proficiencies é a linha "Proficiências." do bloco da classe (p36–83),
	// transcrita — ver `web/sheetui/proficiencies.go`.
	Proficiencies []string `json:"proficiencies"`
	// Spellcasting é a tabela de progressão de círculo, e ela é NULA para as
	// classes que não conjuram — ver `spellcasting.go`.
	Spellcasting *SpellProgression `json:"spellcasting"`
	// Derivados do que já existe — ver o cabeçalho do arquivo.
	Expertises []string `json:"-"`
	Chooses    int      `json:"-"`
	Powers     int      `json:"-"`
}

// ClassExpertises é o que `class-expertises` guarda: as treinadas de saída mais
// quantas o jogador escolhe.
type ClassExpertises struct {
	Fixed       []string `json:"fixed"`
	ChooseCount int      `json:"chooseCount"`
}

// ── perícia ──────────────────────────────────────────────────────────────────

// Expertise é uma das 29, com o que o livro imprime ao lado do nome.
//
// As duas regras vêm da Tabela 2-1 (p115) e não de uma lista escrita à mão no
// código: duas fontes para a mesma tabela divergem.
type Expertise struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
	// TrainedOnly: sem treinamento, nem se rola.
	TrainedOnly bool `json:"soTreinada"`
	// ArmorPenalty: armadura pesada atrapalha.
	ArmorPenalty bool `json:"penalidadeDeArmadura"`
	BookPage     int  `json:"bookPage"`
	// Classes são as que treinam a perícia de saída, DERIVADAS de
	// `class-expertises` — ver o cabeçalho do arquivo.
	Classes []string `json:"-"`
}

// ClassOrder devolve os nomes em ordem estável: a de um `map` é aleatória,
// e sem isto a lista de classes de cada perícia mudaria a cada render.
func ClassOrder(expertises map[string]ClassExpertises) []string {
	names := make([]string, 0, len(expertises))
	for name := range expertises {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// AttributeAbbrev escreve o atributo como a ficha e a tabela do livro escrevem.
func AttributeAbbrev(key string) string {
	for _, a := range AttributeOrder {
		if a.Key == key {
			return a.Abbreviation
		}
	}
	return key
}

func ExpertiseFields(p Expertise) []string {
	return append([]string{p.Name, AttributeAbbrev(p.Attribute)}, p.Classes...)
}

// ── divindade ────────────────────────────────────────────────────────────────

// God: o nome do tipo é `deus` porque é a palavra do livro e do dado
// (`deuses.json`, `paladinoEligible`), e não "divindade" — uma palavra por
// conceito.
type God struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Major bool   `json:"major"`
	// O Paladino e o Druida escolhem devoto de listas próprias (p82 e p61), e é
	// o catálogo que diz quem entra em cada uma.
	PaladinEligible bool     `json:"paladinoEligible"`
	DruidEligible   bool     `json:"druidaEligible"`
	Portfolio       string   `json:"portfolio"`
	Energy          string   `json:"energia"`
	Symbol          string   `json:"simbolo"`
	PreferredWeapon string   `json:"armaPreferida"`
	GrantedPowers   []string `json:"poderesConcedidos"`
	Devotees        []string `json:"devotos"`
	BookPage        int      `json:"bookPage"`
}

// ── a leitura, uma vez só ────────────────────────────────────────────────────

// SortByName usa o MESMO colador pt-BR do resto do acervo: sem ele "Ártico"
// cai depois de "Zumbi", porque a comparação de bytes põe todo acento no fim.
func SortByName[T any](list []T, name func(T) string) {
	col := collate.New(language.BrazilianPortuguese)
	slices.SortStableFunc(list, func(a, b T) int {
		return col.CompareString(name(a), name(b))
	})
}

// ClassesWithKnownExpertises junta o catálogo mínimo com o que se DERIVA do resto.
func ClassesWithKnownExpertises() []Class {
	classes := ListOf[Class]("classes")

	var expertises map[string]ClassExpertises
	if raw, ok := catalog.Resource("class-expertises"); ok {
		_ = json.Unmarshal(raw, &expertises)
	}
	powers := map[string]int{}
	for _, p := range FlattenedPowers() {
		powers[p.Source]++
	}

	for i := range classes {
		if p, found := expertises[classes[i].Name]; found {
			classes[i].Expertises = p.Fixed
			classes[i].Chooses = p.ChooseCount
		}
		classes[i].Powers = powers[classes[i].Name]
	}
	return classes
}

// ── o que cada um busca ──────────────────────────────────────────────────────
//
// Como os quatro primeiros: os campos ficam aqui e não espalhados na cena, para
// a aba e a busca unificada concordarem por construção.

func RaceFields(r Race) []string {
	fields := []string{r.Name, r.Tier, r.Size}
	for _, h := range r.Abilities {
		fields = append(fields, h.Name, h.Summary)
	}
	return fields
}

func ClassFields(c Class) []string {
	return append([]string{c.Name}, c.Expertises...)
}

func GodFields(d God) []string {
	fields := []string{d.Name, d.Portfolio, d.Symbol, d.PreferredWeapon}
	fields = append(fields, d.GrantedPowers...)
	return append(fields, d.Devotees...)
}

// TierName escreve o `tier` da raça. Fica com a palavra do DADO no
// identificador, como `TierName` e `TypeName`, em vez de inventar uma
// palavra nova em português para um conceito que a tela mostra só como
// "Comum"/"Exótica" — o livro tem a seção "Raças Exóticas" e nada além disso.
//
// O valor no dado é `extra` e NÃO `exotica`. Errá-lo não aparece: como o `else`
// devolve "Comum", as dezessete raças dizem Comum — inclusive as nove exóticas.
// Um `switch` com valor desconhecido devolvendo o valor cru seria feio na tela e
// visível; um `else` com um dos dois rótulos é mentira silenciosa.
func TierName(tier string) string {
	switch tier {
	case "extra":
		return "Exótica"
	case "comum":
		return "Comum"
	}
	return tier
}

var (
	periciasUmaVez  sync.Once
	periciasDoLivro []Expertise
)

func Expertises() []Expertise {
	periciasUmaVez.Do(func() {
		periciasDoLivro = ListOf[Expertise]("expertises")
		trainedBy := map[string][]string{}
		var expertises map[string]ClassExpertises
		if raw, ok := catalog.Resource("class-expertises"); ok {
			_ = json.Unmarshal(raw, &expertises)
		}
		// Só as FIXAS: a piscina de escolha tem quase tudo em quase toda classe,
		// e dizer que Acrobacia é "treinada por" doze classes porque ela está em
		// doze piscinas seria informação que não separa nada.
		for _, class := range ClassOrder(expertises) {
			for _, name := range expertises[class].Fixed {
				trainedBy[name] = append(trainedBy[name], class)
			}
		}
		for i := range periciasDoLivro {
			periciasDoLivro[i].Classes = trainedBy[periciasDoLivro[i].Name]
		}
	})
	return periciasDoLivro
}

var (
	personagemUmaVez sync.Once
	racasDoAcervo    []Race
	classesDoAcervo  []Class
	deusesDoAcervo   []God
)

func CharacterCatalogs() ([]Race, []Class, []God) {
	personagemUmaVez.Do(func() {
		// `races.json` é MAPA por id e `deuses.json` é LISTA — errar a forma
		// devolve lista vazia em silêncio, que é a degradação normal deste
		// carregador e seria um catálogo sumindo da tela sem aviso.
		racasDoAcervo = MapOf[Race]("races")
		deusesDoAcervo = ListOf[God]("gods")
		classesDoAcervo = ClassesWithKnownExpertises()
		SortByName(racasDoAcervo, func(r Race) string { return r.Name })
		SortByName(classesDoAcervo, func(c Class) string { return c.Name })
		SortByName(deusesDoAcervo, func(d God) string { return d.Name })
	})
	return racasDoAcervo, classesDoAcervo, deusesDoAcervo
}

// Os três catálogos DO PERSONAGEM no acervo do mestre: raça, classe e divindade.
//
// Eles ficam no FIM da fileira de abas porque a ordem segue a frequência da
// consulta (ver `collectionTabs`): condição, magia, poder e item são consulta de
// MESA, aberta no meio do combate; raça, classe e deus são consulta de CRIAÇÃO.
//
// A CLASSE é o caso especial e vale dizer o que ela NÃO tem: três campos apenas
// — id, nome e página. PV, PM e proficiências são transcrição de tabela, e
// transcrever à mão é o que o `scripts/book-pages.py` existe para não fazer. O
// que a tela mostra além do nome ela DERIVA do que já está no repositório: as
// perícias treinadas saem de `class-expertises`, a conta de poderes sai de
// `class-powers`. Quem quiser o bloco inteiro tem o botão do livro ao lado.

// ── raça ─────────────────────────────────────────────────────────────────────

// AttributeOrder é a do livro, e a razão que morde é a segunda: a ordem de um
// `map` em Go é ALEATÓRIA por projeto, então imprimir os modificadores direto do
// mapa daria uma ordem diferente a cada render — a página mudaria sozinha entre
// dois pedidos iguais, e qualquer teste sobre o texto seria intermitente.
var AttributeOrder = []struct{ Key, Abbreviation string }{
	{"strength", "For"}, {"dexterity", "Des"}, {"constitution", "Con"},
	{"intelligence", "Int"}, {"wisdom", "Sab"}, {"charisma", "Car"},
}
