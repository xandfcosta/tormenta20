package api

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"t20engine/catalog"
)

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

type habilidadeDaRaca struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type racaDoLivro struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Tier            string         `json:"tier"`
	Tamanho         string         `json:"tamanho"`
	Deslocamento    int            `json:"deslocamento"`
	VisaoNoEscuro   bool           `json:"visaoNoEscuro"`
	VisaoNaPenumbra bool           `json:"visaoNaPenumbra"`
	AtributoMod     atributoDeRaca `json:"atributoMod"`
	// Ascendencias são as metades de uma raça que se escolhe na criação — o
	// suraggel é "aggelus" ou "sulfure". Elas importam aqui porque os DEUSES
	// citam a ascendência e não a raça ("Devotos: Aggelus"), e sem isto o elo
	// desses dois não existiria.
	Ascendencias []string           `json:"ascendencias"`
	Abilities    []habilidadeDaRaca `json:"abilities"`
	BookPage     int                `json:"bookPage"`
}

// atributoDeRaca são as DUAS formas do livro, e elas não se reduzem a uma: o
// humano escolhe três +1 onde quiser (`floating`), o elfo recebe +2 Int, +1 Des
// e −1 Con (`fixed`). Guardar as duas num mapa só faria a escolha do humano
// virar três atributos inventados.
type atributoDeRaca struct {
	Kind  string         `json:"kind"`
	Count int            `json:"count"`
	Value int            `json:"value"`
	Mods  map[string]int `json:"mods"`
}

// ordemDosAtributos é a do livro, e ela existe por DUAS razões — a segunda é a
// que morde: a ordem de um `map` em Go é ALEATÓRIA por projeto, então imprimir
// os modificadores direto do mapa daria uma ordem diferente a cada render. A
// página mudaria sozinha entre dois pedidos iguais, e qualquer teste sobre o
// texto seria intermitente.
var ordemDosAtributos = []struct{ Chave, Sigla string }{
	{"strength", "For"}, {"dexterity", "Des"}, {"constitution", "Con"},
	{"intelligence", "Int"}, {"wisdom", "Sab"}, {"charisma", "Car"},
}

// Escrito diz os modificadores como o livro os escreve.
//
//	"+2 Int, +1 Des, −1 Con"       // elfo
//	"+1 em três atributos"          // humano
func (a atributoDeRaca) Escrito() string {
	if a.Kind == "floating" {
		return fmt.Sprintf("%s em %s", comSinalInt(a.Value), porExtenso(a.Count))
	}
	var partes []string
	for _, atributo := range ordemDosAtributos {
		if mod, tem := a.Mods[atributo.Chave]; tem && mod != 0 {
			partes = append(partes, comSinalInt(mod)+" "+atributo.Sigla)
		}
	}
	return strings.Join(partes, ", ")
}

// porExtenso escreve a contagem da escolha livre. Vai até seis porque são seis
// atributos — não há sétima escolha possível.
func porExtenso(n int) string {
	nomes := []string{"nenhum atributo", "um atributo", "dois atributos", "três atributos",
		"quatro atributos", "cinco atributos", "seis atributos"}
	if n < 0 || n >= len(nomes) {
		return fmt.Sprintf("%d atributos", n)
	}
	return nomes[n]
}

// ── classe ───────────────────────────────────────────────────────────────────

type classeDoLivro struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BookPage int    `json:"bookPage"`
	// Derivados do que já existe — ver o cabeçalho do arquivo.
	Pericias []string `json:"-"`
	Escolhe  int      `json:"-"`
	Poderes  int      `json:"-"`
}

// periciasDaClasse é o que `class-expertises` guarda: as treinadas de saída mais
// quantas o jogador escolhe.
type periciasDaClasse struct {
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
type periciaDoLivro struct {
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

var (
	periciasUmaVez  sync.Once
	periciasDoLivro []periciaDoLivro
)

func periciasDoAcervo() []periciaDoLivro {
	periciasUmaVez.Do(func() {
		periciasDoLivro = listaDoCatalogo[periciaDoLivro]("pericias")
		treinadaPor := map[string][]string{}
		var pericias map[string]periciasDaClasse
		if bruto, ok := catalog.Resource("class-expertises"); ok {
			_ = json.Unmarshal(bruto, &pericias)
		}
		// Só as FIXAS: a piscina de escolha tem quase tudo em quase toda classe,
		// e dizer que Acrobacia é "treinada por" doze classes porque ela está em
		// doze piscinas seria informação que não separa nada.
		for _, classe := range ordemDasClasses(pericias) {
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

// ordemDasClasses devolve os nomes em ordem estável: a de um `map` é aleatória,
// e sem isto a lista de classes de cada perícia mudaria a cada render.
func ordemDasClasses(pericias map[string]periciasDaClasse) []string {
	nomes := make([]string, 0, len(pericias))
	for nome := range pericias {
		nomes = append(nomes, nome)
	}
	slices.Sort(nomes)
	return nomes
}

// siglaDoAtributo escreve o atributo como a ficha e a tabela do livro escrevem.
func siglaDoAtributo(chave string) string {
	for _, a := range ordemDosAtributos {
		if a.Chave == chave {
			return a.Sigla
		}
	}
	return chave
}

func camposDaPericia(p periciaDoLivro) []string {
	return append([]string{p.Name, siglaDoAtributo(p.Attribute)}, p.Classes...)
}

// ── divindade ────────────────────────────────────────────────────────────────

// deusDoLivro: o nome do tipo é `deus` porque é a palavra do livro e do dado
// (`deuses.json`, `paladinoEligible`), e não "divindade" — uma palavra por
// conceito.
type deusDoLivro struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Major             bool     `json:"major"`
	Portfolio         string   `json:"portfolio"`
	Energia           string   `json:"energia"`
	Simbolo           string   `json:"simbolo"`
	ArmaPreferida     string   `json:"armaPreferida"`
	PoderesConcedidos []string `json:"poderesConcedidos"`
	Devotos           []string `json:"devotos"`
	BookPage          int      `json:"bookPage"`
}

// ── a leitura, uma vez só ────────────────────────────────────────────────────

var (
	personagemUmaVez sync.Once
	racasDoAcervo    []racaDoLivro
	classesDoAcervo  []classeDoLivro
	deusesDoAcervo   []deusDoLivro
)

func catalogosDoPersonagem() ([]racaDoLivro, []classeDoLivro, []deusDoLivro) {
	personagemUmaVez.Do(func() {
		// `races.json` é MAPA por id e `deuses.json` é LISTA — errar a forma
		// devolve lista vazia em silêncio, que é a degradação normal deste
		// carregador e seria um catálogo sumindo da tela sem aviso.
		racasDoAcervo = mapaDoCatalogo[racaDoLivro]("races")
		deusesDoAcervo = listaDoCatalogo[deusDoLivro]("deuses")
		classesDoAcervo = classesComOQueSeSabe()
		ordenaPorNome(racasDoAcervo, func(r racaDoLivro) string { return r.Name })
		ordenaPorNome(classesDoAcervo, func(c classeDoLivro) string { return c.Name })
		ordenaPorNome(deusesDoAcervo, func(d deusDoLivro) string { return d.Name })
	})
	return racasDoAcervo, classesDoAcervo, deusesDoAcervo
}

// ordenaPorNome usa o MESMO colador pt-BR do resto do acervo: sem ele "Ártico"
// cai depois de "Zumbi", porque a comparação de bytes põe todo acento no fim.
func ordenaPorNome[T any](lista []T, nome func(T) string) {
	col := collate.New(language.BrazilianPortuguese)
	slices.SortStableFunc(lista, func(a, b T) int {
		return col.CompareString(nome(a), nome(b))
	})
}

// classesComOQueSeSabe junta o catálogo mínimo com o que se DERIVA do resto.
func classesComOQueSeSabe() []classeDoLivro {
	classes := listaDoCatalogo[classeDoLivro]("classes")

	var pericias map[string]periciasDaClasse
	if bruto, ok := catalog.Resource("class-expertises"); ok {
		_ = json.Unmarshal(bruto, &pericias)
	}
	poderes := map[string]int{}
	for _, p := range poderesAchatados() {
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

func camposDaRaca(r racaDoLivro) []string {
	campos := []string{r.Name, r.Tier, r.Tamanho}
	for _, h := range r.Abilities {
		campos = append(campos, h.Name, h.Summary)
	}
	return campos
}

func camposDaClasse(c classeDoLivro) []string {
	return append([]string{c.Name}, c.Pericias...)
}

func camposDoDeus(d deusDoLivro) []string {
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
func nomeDoTier(tier string) string {
	switch tier {
	case "extra":
		return "Exótica"
	case "comum":
		return "Comum"
	}
	return tier
}
