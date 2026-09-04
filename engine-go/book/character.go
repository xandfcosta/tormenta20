package book

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"t20engine/catalog"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

type RaceTrait struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type Race struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Tier            string        `json:"tier"`
	Tamanho         string        `json:"tamanho"`
	Deslocamento    int           `json:"deslocamento"`
	VisaoNoEscuro   bool          `json:"visaoNoEscuro"`
	VisaoNaPenumbra bool          `json:"visaoNaPenumbra"`
	AttributeMod    RaceAttribute `json:"atributoMod"`
	// Ascendencias são as metades de uma raça que se escolhe na criação — o
	// suraggel é "aggelus" ou "sulfure". Elas importam aqui porque os DEUSES
	// citam a ascendência e não a raça ("Devotos: Aggelus"), e sem isto o elo
	// desses dois não existiria.
	Ascendencias []string    `json:"ascendencias"`
	Abilities    []RaceTrait `json:"abilities"`
	BookPage     int         `json:"bookPage"`
}

// atributoDeRaca são as DUAS formas do livro, e elas não se reduzem a uma: o
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
	// nesse mesmo atributo. Ele entrou na fatia 8, com o diálogo que oferece os
	// atributos: sem ele a tela ofereceria uma escolha que o motor recusa.
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
	var partes []string
	for _, atributo := range AttributeOrder {
		if mod, tem := a.Mods[atributo.Chave]; tem && mod != 0 {
			partes = append(partes, WithSign(mod)+" "+atributo.Sigla)
		}
	}
	return strings.Join(partes, ", ")
}

// porExtenso escreve a contagem da escolha livre. Vai até seis porque são seis
// atributos — não há sétima escolha possível.
func InWords(n int) string {
	nomes := []string{"nenhum atributo", "um atributo", "dois atributos", "três atributos",
		"quatro atributos", "cinco atributos", "seis atributos"}
	if n < 0 || n >= len(nomes) {
		return fmt.Sprintf("%d atributos", n)
	}
	return nomes[n]
}

// ── classe ───────────────────────────────────────────────────────────────────

type Class struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BookPage int    `json:"bookPage"`
	// Proficiencias é a linha "Proficiências." do bloco da classe (p36–83),
	// transcrita. Ela chegou na ALE-272 com o painel de Proficiências da ficha:
	// a tabela existia só em TypeScript, fora do alcance da validação de schema
	// — ver `web/sheetui/proficiencias.go`.
	Proficiencias []string `json:"proficiencies"`
	// Spellcasting é a tabela de progressão de círculo, e ela é NULA para as
	// classes que não conjuram — ver `spellcasting.go`.
	Spellcasting *SpellProgression `json:"spellcasting"`
	// Derivados do que já existe — ver o cabeçalho do arquivo.
	Pericias []string `json:"-"`
	Escolhe  int      `json:"-"`
	Poderes  int      `json:"-"`
}

// periciasDaClasse é o que `class-expertises` guarda: as treinadas de saída mais
// quantas o jogador escolhe.
type ClassExpertises struct {
	Fixed       []string `json:"fixed"`
	ChooseCount int      `json:"chooseCount"`
}

// ── perícia ──────────────────────────────────────────────────────────────────

// periciaDoLivro é uma das 29, com o que o livro imprime ao lado do nome.
//
// As duas regras vêm da Tabela 2-1 (p115) e não de uma lista no código: o motor
// tinha as três de penalidade de armadura escritas à mão em
// `engine/breakdowns.go`, e a tabela do livro concorda com elas — o que é uma
// boa notícia e não um motivo para manter duas fontes.
type Expertise struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
	// SoTreinada: sem treinamento, nem se rola.
	SoTreinada bool `json:"soTreinada"`
	// PenalidadeDeArmadura: armadura pesada atrapalha.
	PenalidadeDeArmadura bool `json:"penalidadeDeArmadura"`
	BookPage             int  `json:"bookPage"`
	// Classes são as que treinam a perícia de saída, DERIVADAS de
	// `class-expertises` — ver o cabeçalho do arquivo.
	Classes []string `json:"-"`
}

// ordemDasClasses devolve os nomes em ordem estável: a de um `map` é aleatória,
// e sem isto a lista de classes de cada perícia mudaria a cada render.
func ClassOrder(pericias map[string]ClassExpertises) []string {
	nomes := make([]string, 0, len(pericias))
	for nome := range pericias {
		nomes = append(nomes, nome)
	}
	slices.Sort(nomes)
	return nomes
}

// siglaDoAtributo escreve o atributo como a ficha e a tabela do livro escrevem.
func AttributeAbbrev(chave string) string {
	for _, a := range AttributeOrder {
		if a.Chave == chave {
			return a.Sigla
		}
	}
	return chave
}

func ExpertiseFields(p Expertise) []string {
	return append([]string{p.Name, AttributeAbbrev(p.Attribute)}, p.Classes...)
}

// ── divindade ────────────────────────────────────────────────────────────────

// deusDoLivro: o nome do tipo é `deus` porque é a palavra do livro e do dado
// (`deuses.json`, `paladinoEligible`), e não "divindade" — uma palavra por
// conceito.
type God struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Major bool   `json:"major"`
	// Os dois elegíveis entraram na fatia 8: o Paladino e o Druida escolhem
	// devoto de listas próprias (p82 e p61), e é o catálogo que diz quem entra
	// em cada uma.
	PaladinoEligible  bool     `json:"paladinoEligible"`
	DruidaEligible    bool     `json:"druidaEligible"`
	Portfolio         string   `json:"portfolio"`
	Energia           string   `json:"energia"`
	Simbolo           string   `json:"simbolo"`
	ArmaPreferida     string   `json:"armaPreferida"`
	PoderesConcedidos []string `json:"poderesConcedidos"`
	Devotos           []string `json:"devotos"`
	BookPage          int      `json:"bookPage"`
}

// ── a leitura, uma vez só ────────────────────────────────────────────────────

// ordenaPorNome usa o MESMO colador pt-BR do resto do acervo: sem ele "Ártico"
// cai depois de "Zumbi", porque a comparação de bytes põe todo acento no fim.
func SortByName[T any](lista []T, nome func(T) string) {
	col := collate.New(language.BrazilianPortuguese)
	slices.SortStableFunc(lista, func(a, b T) int {
		return col.CompareString(nome(a), nome(b))
	})
}

// classesComOQueSeSabe junta o catálogo mínimo com o que se DERIVA do resto.
func ClassesWithKnownExpertises() []Class {
	classes := ListOf[Class]("classes")

	var pericias map[string]ClassExpertises
	if bruto, ok := catalog.Resource("class-expertises"); ok {
		_ = json.Unmarshal(bruto, &pericias)
	}
	poderes := map[string]int{}
	for _, p := range FlattenedPowers() {
		poderes[p.Fonte]++
	}

	for i := range classes {
		if p, tem := pericias[classes[i].Name]; tem {
			classes[i].Pericias = p.Fixed
			classes[i].Escolhe = p.ChooseCount
		}
		classes[i].Poderes = poderes[classes[i].Name]
	}
	return classes
}

// ── o que cada um busca ──────────────────────────────────────────────────────
//
// Como os quatro primeiros: os campos ficam aqui e não espalhados na cena, para
// a aba e a busca unificada concordarem por construção.

func RaceFields(r Race) []string {
	campos := []string{r.Name, r.Tier, r.Tamanho}
	for _, h := range r.Abilities {
		campos = append(campos, h.Name, h.Summary)
	}
	return campos
}

func ClassFields(c Class) []string {
	return append([]string{c.Name}, c.Pericias...)
}

func GodFields(d God) []string {
	campos := []string{d.Name, d.Portfolio, d.Simbolo, d.ArmaPreferida}
	campos = append(campos, d.PoderesConcedidos...)
	return append(campos, d.Devotos...)
}

// nomeDoTier escreve o `tier` da raça. Fica com a palavra do DADO no
// identificador, como `nomeDaCategoria` e `nomeDoTipo`, em vez de inventar uma
// palavra nova em português para um conceito que a tela mostra só como
// "Comum"/"Exótica" — o livro tem a seção "Raças Exóticas" e nada além disso.
//
// O valor é `extra` e NÃO `exotica`, e isto é conserto de um defeito que só
// apareceu ao medir o dado para montar o filtro: eu tinha escrito `exotica`, e
// como o `else` devolve "Comum", as DEZESSETE raças diziam Comum — inclusive as
// nove exóticas. Um `switch` com valor desconhecido devolvendo o valor cru seria
// feio na tela e visível; um `else` com um dos dois rótulos é mentira silenciosa.
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
		periciasDoLivro = ListOf[Expertise]("pericias")
		treinadaPor := map[string][]string{}
		var pericias map[string]ClassExpertises
		if bruto, ok := catalog.Resource("class-expertises"); ok {
			_ = json.Unmarshal(bruto, &pericias)
		}
		// Só as FIXAS: a piscina de escolha tem quase tudo em quase toda classe,
		// e dizer que Acrobacia é "treinada por" doze classes porque ela está em
		// doze piscinas seria informação que não separa nada.
		for _, classe := range ClassOrder(pericias) {
			for _, nome := range pericias[classe].Fixed {
				treinadaPor[nome] = append(treinadaPor[nome], classe)
			}
		}
		for i := range periciasDoLivro {
			periciasDoLivro[i].Classes = treinadaPor[periciasDoLivro[i].Name]
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
		deusesDoAcervo = ListOf[God]("deuses")
		classesDoAcervo = ClassesWithKnownExpertises()
		SortByName(racasDoAcervo, func(r Race) string { return r.Name })
		SortByName(classesDoAcervo, func(c Class) string { return c.Name })
		SortByName(deusesDoAcervo, func(d God) string { return d.Name })
	})
	return racasDoAcervo, classesDoAcervo, deusesDoAcervo
}

// Os três catálogos DO PERSONAGEM no acervo do mestre (ALE-264): raça, classe e
// divindade.
//
// Eles chegam depois dos quatro primeiros porque a pergunta que respondem é
// outra. Condição, magia, poder e item são consulta de MESA — o mestre abre no
// meio do combate. Raça, classe e deus são consulta de CRIAÇÃO, e é por isso que
// entram no fim da fileira de abas: a ordem das abas segue a frequência da
// consulta, que é a razão registrada em `abasDoAcervo` desde a ALE-258.
//
// A CLASSE é o caso especial e vale dizer o que ela NÃO tem: o catálogo dela
// nasceu nesta issue com três campos — id, nome e página — porque ela existia só
// como uma lista de nomes dentro de `options.json`. PV, PM e proficiências são
// transcrição de tabela, e transcrever à mão é exatamente o que o
// `scripts/paginas-do-livro.py` existe para não fazer. O que a tela mostra além
// do nome ela DERIVA do que já está no repositório: as perícias treinadas saem
// de `class-expertises`, a conta de poderes sai de `class-powers`. Quem quiser o
// bloco inteiro tem o botão do livro ao lado do nome.

// ── raça ─────────────────────────────────────────────────────────────────────

// ordemDosAtributos é a do livro, e ela existe por DUAS razões — a segunda é a
// que morde: a ordem de um `map` em Go é ALEATÓRIA por projeto, então imprimir
// os modificadores direto do mapa daria uma ordem diferente a cada render. A
// página mudaria sozinha entre dois pedidos iguais, e qualquer teste sobre o
// texto seria intermitente.
var AttributeOrder = []struct{ Chave, Sigla string }{
	{"strength", "For"}, {"dexterity", "Des"}, {"constitution", "Con"},
	{"intelligence", "Int"}, {"wisdom", "Sab"}, {"charisma", "Car"},
}
