package api

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"t20engine/catalog"
	"t20engine/engine"
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

// casaTodosOsTermos: TODO termo separado por espaço precisa aparecer em algum
// dos campos. "luz cur" casa com o que carrega as duas coisas.
//
// Isto NÃO é o `casaBusca` das outras cenas, e a diferença é deliberada — o
// comentário do `catalog-model.ts` a explica e ela sobrevive ao porte. O
// `casaBusca` é tolerante a typo por subsequência, o que serve para escolher UM
// item de uma lista; aqui o mestre está estreitando uma REFERÊNCIA por palavras
// que ele sabe, e subsequência arrastaria quase-acertos que fazem uma consulta
// de regra parecer errada no meio da sessão.
//
// O que as duas compartilham é a `dobra`: acento não separa "ilusão" de
// "ilusao", porque ninguém digita til numa busca apressada.
func casaTodosOsTermos(campos []string, busca string) bool {
	alvo := dobra(strings.TrimSpace(busca))
	if alvo == "" {
		return true
	}
	palheiro := dobra(strings.Join(campos, " "))
	for _, termo := range strings.Fields(alvo) {
		if !strings.Contains(palheiro, termo) {
			return false
		}
	}
	return true
}

// ── o que se busca ───────────────────────────────────────────────────────────

type condicaoDoLivro struct {
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

// aprimoramentoDaMagia é o que o livro imprime abaixo da magia: quanto custa a
// mais e o que muda. Eram `[]any` — a cena só contava quantos havia —, e o
// dono pediu para poder LER cada um.
type aprimoramentoDaMagia struct {
	PmCost      int    `json:"pmCost"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// Escrito diz o custo como o livro: "+2 PM".
func (a aprimoramentoDaMagia) Escrito() string { return fmt.Sprintf("+%d PM", a.PmCost) }

// nomeDoTipoDeAprimoramento: o livro separa o que AUMENTA um efeito do que o
// MUDA, e a diferença importa na hora de gastar mana.
func nomeDoTipoDeAprimoramento(kind string) string {
	if kind == "aumenta" {
		return "aumenta"
	}
	return "muda"
}

type magiaDoLivro struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Circle     int                    `json:"circle"`
	School     string                 `json:"school"`
	Execution  string                 `json:"execution"`
	Range      string                 `json:"range"`
	Duration   string                 `json:"duration"`
	Resistance string                 `json:"resistance,omitempty"`
	BaseEffect string                 `json:"baseEffect"`
	Augments   []aprimoramentoDaMagia `json:"augments"`
	Classes    []string               `json:"classes"`
	BookPage   int                    `json:"bookPage"`
}

// poderDoLivro é o poder ACHATADO. O livro espalha poder por três catálogos —
// habilidade de classe, poder geral/de combate e poder concedido —, e o mestre
// quer UMA lista buscável. A `Fonte` diz de onde veio, que é o que o
// achatamento não pode perder.
type poderDoLivro struct {
	ID          string
	Name        string
	Fonte       string
	Description string
	BookPage    int
}

// itemDoLivro é a entrada do catálogo de itens.
//
// Ela nasceu com os seis campos que a vitrine do mestre mostra e cresceu na
// ALE-272 (fatia 7): a Mochila do jogador precisa do EIXO de equipar, das
// estatísticas de arma/armadura/escudo, do consumível e da família a que uma
// melhoria se aplica. Um segundo leitor do mesmo `items.json` daria duas
// verdades sobre o mesmo arquivo, então quem cresce é este.
type itemDoLivro struct {
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
	AppliesTo  []string           `json:"appliesTo"`
	Weapon     *armaDoLivro       `json:"weapon"`
	Armor      *protecaoDoLivro   `json:"armor"`
	Shield     *protecaoDoLivro   `json:"shield"`
	Consumable *consumivelDoLivro `json:"consumable"`
	Modifiers  []engine.Modifier  `json:"modifiers"`
}

type armaDoLivro struct {
	Damage    string   `json:"damage"`
	CritRange int      `json:"critRange"`
	CritMult  int      `json:"critMult"`
	Type      string   `json:"type"`
	Purpose   string   `json:"purpose"`
	Traits    []string `json:"traits"`
}

// protecaoDoLivro serve armadura E escudo: os dois trazem os mesmos três
// números, e o livro os apresenta na mesma tabela (p154).
type protecaoDoLivro struct {
	Defense int  `json:"defense"`
	Penalty int  `json:"penalty"`
	Heavy   bool `json:"heavy"`
}

type consumivelDoLivro struct {
	Scope   string         `json:"scope"`
	Instant *ganhoImediato `json:"instant"`
}

// ganhoImediato é o PV/PM que um consumível devolve na hora. O `Dice` é a
// rolagem que a MESA faz — a ficha não rola por ninguém.
type ganhoImediato struct {
	HP *rolagemDoGanho `json:"hp"`
	MP *rolagemDoGanho `json:"mp"`
}

type rolagemDoGanho struct {
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus"`
}

// ── a leitura, uma vez só ────────────────────────────────────────────────────

type acervoDoMestre struct {
	Condicoes []condicaoDoLivro
	Magias    []magiaDoLivro
	Poderes   []poderDoLivro
	Itens     []itemDoLivro
}

var (
	acervoUmaVez sync.Once
	acervo       acervoDoMestre
)

// catalogosDoLivro lê os quatro catálogos e os ORDENA uma vez.
//
// Ordenar aqui e não a cada pedido: a ordem não depende do filtro, e refazer
// quatro ordenações a cada tecla digitada seria trabalho por nada. O
// `sync.Once` é o mesmo padrão do bestiário.
func catalogosDoLivro() acervoDoMestre {
	acervoUmaVez.Do(func() {
		col := collate.New(language.BrazilianPortuguese)
		porNome := func(a, b string) int { return col.CompareString(a, b) }

		acervo.Condicoes = mapaDoCatalogo[condicaoDoLivro]("conditions")
		slices.SortStableFunc(acervo.Condicoes, func(a, b condicaoDoLivro) int {
			return porNome(a.Name, b.Name)
		})

		acervo.Magias = mapaDoCatalogo[magiaDoLivro]("spells")
		// Magia ordena por CÍRCULO e depois por nome, como a SPA: o mestre
		// procura "o que existe de 3º círculo", e alfabético puro embaralharia
		// os círculos.
		slices.SortStableFunc(acervo.Magias, func(a, b magiaDoLivro) int {
			if a.Circle != b.Circle {
				return a.Circle - b.Circle
			}
			return porNome(a.Name, b.Name)
		})

		acervo.Poderes = poderesAchatados()
		slices.SortStableFunc(acervo.Poderes, func(a, b poderDoLivro) int {
			return porNome(a.Name, b.Name)
		})

		acervo.Itens = listaDoCatalogo[itemDoLivro]("items")
		slices.SortStableFunc(acervo.Itens, func(a, b itemDoLivro) int {
			return porNome(a.Name, b.Name)
		})
	})
	return acervo
}

// mapaDoCatalogo lê um recurso guardado como OBJETO por id e devolve os valores.
//
// Catálogo ausente ou malformado devolve lista vazia em vez de derrubar a Mesa:
// a ferramenta abre sem aquela aba, e as outras três continuam servindo.
func mapaDoCatalogo[T any](nome string) []T {
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

// listaDoCatalogo lê um recurso guardado como ARRAY.
func listaDoCatalogo[T any](nome string) []T {
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

// poderesAchatados junta os três catálogos de poder numa lista só.
//
// Os poderes DIVINOS ficam de fora, e a razão é da SPA: o dado deles carrega
// página do livro e nenhum texto de regra, então não há o que consultar.
func poderesAchatados() []poderDoLivro {
	var fora []poderDoLivro

	for _, p := range listaDoCatalogo[struct {
		ID          string `json:"id"`
		ClassName   string `json:"className"`
		Name        string `json:"name"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}]("class-powers") {
		fora = append(fora, poderDoLivro{
			ID: p.ID, Name: p.Name, Fonte: p.ClassName, Description: p.Description, BookPage: p.BookPage,
		})
	}

	for _, p := range listaDoCatalogo[struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}]("general-powers") {
		fora = append(fora, poderDoLivro{
			ID: "general." + p.ID, Name: p.Name, Fonte: "Geral · " + p.Kind,
			Description: p.Description, BookPage: p.BookPage,
		})
	}

	return append(fora, poderesDivinos()...)
}

// poderesDivinos são os que os DEUSES concedem, e este bloco é conserto de uma
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
func poderesDivinos() []poderDoLivro {
	type divino struct {
		DeusID      string `json:"deusId"`
		Name        string `json:"name"`
		Description string `json:"description"`
		BookPage    int    `json:"bookPage"`
	}

	porNome := map[string]*poderDoLivro{}
	var ordem []string
	for _, p := range listaDoCatalogo[divino]("divine-powers") {
		if achado, tem := porNome[p.Name]; tem {
			achado.Fonte += ", " + nomeDoDeus(p.DeusID)
			continue
		}
		porNome[p.Name] = &poderDoLivro{
			// O id é o NOME em forma de chave: o `divine-powers` não traz `id`,
			// e o elo endereça por id. Prefixado para não colidir com um poder
			// de classe de mesmo nome.
			ID:          "divino." + chaveDoNome(p.Name),
			Name:        p.Name,
			Fonte:       "Divino · " + nomeDoDeus(p.DeusID),
			Description: p.Description,
			BookPage:    p.BookPage,
		}
		ordem = append(ordem, p.Name)
	}

	fora := make([]poderDoLivro, 0, len(ordem))
	for _, nome := range ordem {
		fora = append(fora, *porNome[nome])
	}
	return fora
}

// nomeDoDeus resolve o id que o poder divino guarda ("lin-wu") no nome que se lê.
//
// Lê o catálogo DIRETO e não pelo `catalogosDoPersonagem`, e isto é conserto de
// um DEADLOCK que pendurou a suíte inteira sem erro nenhum: aquele carregador
// tem um `sync.Once` que chama o `poderesAchatados` para contar os poderes de
// cada classe, e o `poderesAchatados` chamava de volta o `catalogosDoPersonagem`
// daqui. `Once` reentrante trava para sempre — não é pânico, não é teste
// vermelho: é o processo parado.
//
// Quem apontou o dedo foi o `go test -timeout 25s`, que despeja a pilha de todas
// as goroutines. Sem o timeout, o sintoma era "a suíte demora".
var (
	deusesUmaVez sync.Once
	nomePorDeus  map[string]string
)

func nomeDoDeus(id string) string {
	deusesUmaVez.Do(func() {
		nomePorDeus = map[string]string{}
		for _, d := range listaDoCatalogo[struct {
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

// chaveDoNome transforma um nome em chave de endereço: sem acento, minúsculo,
// espaços viram hífen. É a mesma forma dos ids que os catálogos já usam.
func chaveDoNome(nome string) string {
	return strings.ReplaceAll(dobra(nome), " ", "-")
}

// ── o que cada catálogo busca ────────────────────────────────────────────────
//
// Os campos ficam aqui e não espalhados na cena, para a aba e a busca unificada
// concordarem POR CONSTRUÇÃO em vez de por cópia.

func camposDaCondicao(c condicaoDoLivro) []string {
	return append([]string{c.Name, c.Description}, c.Tags...)
}
func camposDaMagia(m magiaDoLivro) []string { return []string{m.Name, m.BaseEffect} }
func camposDoPoder(p poderDoLivro) []string { return []string{p.Name, p.Fonte, p.Description} }
func camposDoItem(i itemDoLivro) []string   { return []string{i.Name, i.Category} }

// ── a busca unificada ────────────────────────────────────────────────────────

// abaDoAcervo é uma parada da fileira de abas.
type abaDoAcervo struct {
	ID     string
	Rotulo string
}

// A ordem é a da SPA: condição primeiro porque é a consulta mais frequente no
// meio do combate, e item por último porque é a de entre-cenas.
// As três últimas entraram na ALE-264 e vão no FIM pela mesma razão que decidiu
// a ordem original: raça, classe e deus são consulta de CRIAÇÃO de personagem, e
// as quatro primeiras são consulta de mesa com o combate em curso.
var abasDoAcervo = []abaDoAcervo{
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

// rotuloDaAba devolve o nome que a fileira mostra, para a frase de volta não
// dizer "condicoes" com a cara de chave.
func rotuloDaAba(id string) string {
	for _, a := range abasDoAcervo {
		if a.ID == id {
			return a.Rotulo
		}
	}
	return id
}

func abaConhecida(id string) string {
	for _, a := range abasDoAcervo {
		if a.ID == id {
			return id
		}
	}
	// Aba desconhecida cai na primeira em vez de mostrar tela vazia: o `?aba=`
	// é endereço e alguém o digita errado.
	return abasDoAcervo[0].ID
}

// grupoDoAcervo é um catálogo com o que sobrou do filtro. Grupo VAZIO não é
// montado: cabeçalho sobre nada é ruído numa consulta no meio do combate.
type grupoDoAcervo struct {
	Rotulo    string
	Condicoes []condicaoDoLivro
	Magias    []magiaDoLivro
	Poderes   []poderDoLivro
	Itens     []itemDoLivro
	Efeitos   []efeitoDoLivro
	Escolas   []escolaDeMagia
	Pericias  []periciaDoLivro
	Racas     []racaDoLivro
	Classes   []classeDoLivro
	Deuses    []deusDoLivro
}

func (g grupoDoAcervo) Quantos() int {
	return len(g.Condicoes) + len(g.Magias) + len(g.Poderes) + len(g.Itens) +
		len(g.Efeitos) + len(g.Escolas) + len(g.Pericias) + len(g.Racas) + len(g.Classes) + len(g.Deuses)
}

// criteriosDoAcervo é o que a URL (ou os sinais) pedem da cena.
type criteriosDoAcervo struct {
	Busca string
	Aba   string
	// Entrada é o ID de UM verbete, e ela ganha de tudo: com ela a cena mostra
	// aquele verbete sozinho. É o endereço que um ELO usa — quem clica em "Medo"
	// pediu o Medo, não uma busca por "medo" nos oito catálogos.
	Entrada string
	// Filtros são os crachás acesos, por chave (`{"circulo": ["2","3"]}`). Vêm
	// da URL na carga fria e dos sinais quando o Datastar chama, como a busca.
	Filtros map[string][]string
}

// catalogosView é a cena inteira numa resposta.
type catalogosView struct {
	Busca string
	// Livro é o endereço do PDF servido (ALE-264). Zero valor = não há livro
	// configurado, e aí o cartão mostra a página em texto puro — que é o que o
	// mestre com o livro de papel na mesa usa.
	Livro enderecoDoLivro
	// Aba só importa quando NÃO se está buscando: com termo digitado a cena
	// mostra os quatro catálogos agrupados, que é a decisão que a ALE-22
	// registrou — a versão em React filtrava só a aba ativa, e "bola de fogo"
	// digitado em Condições dizia "nada encontrado" com a magia existindo.
	Aba string
	// Entrada é o id do verbete que a cena está mostrando sozinho, ou vazio.
	Entrada string
	// Filtros é o que a cena tem para oferecer, e Acesos o que está ligado.
	Filtros []filtroDoAcervo
	Acesos  map[string][]string
	Grupos  []grupoDoAcervo
	Achados int
}

func (v catalogosView) Buscando() bool { return strings.TrimSpace(v.Busca) != "" }

// carregaCatalogos monta a cena: os quatro catálogos quando há busca, um só
// quando não há.
func carregaCatalogos(c criteriosDoAcervo, livro enderecoDoLivro) catalogosView {
	v := catalogosView{
		Busca: c.Busca, Aba: abaConhecida(c.Aba), Livro: livro, Entrada: c.Entrada,
		Acesos: c.Filtros,
	}
	v.Filtros = filtrosDaAba(v.Aba)
	// A ENTRADA vem primeiro e encerra: ela é um endereço para um verbete só, e
	// misturá-la com busca daria uma tela que responde duas perguntas.
	if c.Entrada != "" {
		v.Grupos = []grupoDoAcervo{grupoDaEntrada(v.Aba, c.Entrada)}
		v.Achados = v.Grupos[0].Quantos()
		return v
	}
	busca := c.Busca
	a := catalogosDoLivro()

	if !v.Buscando() {
		v.Grupos = []grupoDoAcervo{grupoDaAba(a, v.Aba, c.Filtros)}
		v.Achados = v.Grupos[0].Quantos()
		return v
	}

	racas, classes, deuses := catalogosDoPersonagem()
	for _, g := range []grupoDoAcervo{
		{Rotulo: "Condições", Condicoes: filtra(a.Condicoes, camposDaCondicao, busca)},
		{Rotulo: "Magias", Magias: filtra(a.Magias, camposDaMagia, busca)},
		{Rotulo: "Poderes", Poderes: filtra(a.Poderes, camposDoPoder, busca)},
		{Rotulo: "Itens", Itens: filtra(a.Itens, camposDoItem, busca)},
		{Rotulo: "Efeitos", Efeitos: filtra(tiposDeEfeito(), camposDoEfeito, busca)},
		{Rotulo: "Escolas", Escolas: filtra(escolasDeMagia(), camposDaEscola, busca)},
		{Rotulo: "Perícias", Pericias: filtra(periciasDoAcervo(), camposDaPericia, busca)},
		{Rotulo: "Raças", Racas: filtra(racas, camposDaRaca, busca)},
		{Rotulo: "Classes", Classes: filtra(classes, camposDaClasse, busca)},
		{Rotulo: "Deuses", Deuses: filtra(deuses, camposDoDeus, busca)},
	} {
		if g.Quantos() == 0 {
			continue
		}
		v.Achados += g.Quantos()
		v.Grupos = append(v.Grupos, g)
	}
	return v
}

// grupoDaEntrada acha UM verbete pelo id, dentro da aba pedida.
//
// Pelo ID e não pelo nome: nome é texto de tela e muda com a revisão do livro;
// id é a chave com que os catálogos se referem uns aos outros ("upgradesTo":
// "apavorado"). Um elo é uma referência entre DADOS, e referência por texto de
// tela é a que quebra em silêncio no dia de uma correção de acento.
//
// Id desconhecido devolve grupo VAZIO, e a cena diz que não achou — endereço se
// digita à mão e catálogo muda; inventar um verbete seria pior.
func grupoDaEntrada(aba, id string) grupoDoAcervo {
	// Sem filtro: o elo pede UM verbete pelo id, e um crachá aceso na cena de
	// origem não pode esconder o destino do elo.
	inteiro := grupoDaAba(catalogosDoLivro(), aba, nil)
	fora := grupoDoAcervo{Rotulo: inteiro.Rotulo}
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

// grupoDaAba monta o catálogo da cena, já com os crachás aplicados.
func grupoDaAba(a acervoDoMestre, aba string, acesos map[string][]string) grupoDoAcervo {
	racas, classes, deuses := catalogosDoPersonagem()
	switch aba {
	case "magias":
		return grupoDoAcervo{Rotulo: "Magias", Magias: aplicaFiltros(a.Magias, acesos, magiaCasa)}
	case "poderes":
		return grupoDoAcervo{Rotulo: "Poderes", Poderes: aplicaFiltros(a.Poderes, acesos, poderCasa)}
	case "itens":
		return grupoDoAcervo{Rotulo: "Itens", Itens: aplicaFiltros(a.Itens, acesos, itemCasa)}
	case "efeitos":
		return grupoDoAcervo{Rotulo: "Efeitos", Efeitos: tiposDeEfeito()}
	case "escolas":
		return grupoDoAcervo{Rotulo: "Escolas", Escolas: escolasDeMagia()}
	case "pericias":
		return grupoDoAcervo{Rotulo: "Perícias", Pericias: aplicaFiltros(periciasDoAcervo(), acesos, periciaCasa)}
	case "racas":
		return grupoDoAcervo{Rotulo: "Raças", Racas: aplicaFiltros(racas, acesos, racaCasa)}
	case "classes":
		return grupoDoAcervo{Rotulo: "Classes", Classes: classes}
	case "deuses":
		return grupoDoAcervo{Rotulo: "Deuses", Deuses: aplicaFiltros(deuses, acesos, deusCasa)}
	default:
		return grupoDoAcervo{Rotulo: "Condições", Condicoes: aplicaFiltros(a.Condicoes, acesos, condicaoCasa)}
	}
}

func filtra[T any](lista []T, campos func(T) []string, busca string) []T {
	var fora []T
	for _, e := range lista {
		if casaTodosOsTermos(campos(e), busca) {
			fora = append(fora, e)
		}
	}
	return fora
}

// ── como o livro escreve ─────────────────────────────────────────────────────

var rotuloDaExecucao = map[string]string{
	"padrao":    "Padrão",
	"movimento": "Movimento",
	"completa":  "Completa",
	"livre":     "Livre",
	"reacao":    "Reação",
}

var rotuloDoAlcance = map[string]string{
	"pessoal":   "Pessoal",
	"toque":     "Toque",
	"curto":     "Curto",
	"medio":     "Médio",
	"longo":     "Longo",
	"ilimitado": "Ilimitado",
}

func nomeDaExecucao(e string) string {
	if r, ok := rotuloDaExecucao[e]; ok {
		return r
	}
	return e
}

func nomeDoAlcance(a string) string {
	if r, ok := rotuloDoAlcance[a]; ok {
		return r
	}
	return a
}

var rotuloDaCategoria = map[string]string{
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

func nomeDaCategoria(c string) string {
	if r, ok := rotuloDaCategoria[c]; ok {
		return r
	}
	return c
}

// ── o que a cena precisa escrever ────────────────────────────────────────────

// sinaisDosCatalogos: só a busca e a aba viajam. O que se vê chega desenhado.
func sinaisDosCatalogos(v catalogosView) string {
	busca, _ := json.Marshal(v.Busca)
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

// crachaAceso diz se aquele valor está ligado, para a cena não precisar de
// `slices`.
func (v catalogosView) crachaAceso(chave, valor string) bool {
	return slices.Contains(v.Acesos[chave], valor)
}

// alternaOCracha é a expressão que o clique roda: liga o que está desligado e
// desliga o que está ligado, no sinal daquele filtro.
//
// Escrita aqui e não no templ porque é a MESMA para os seis filtros, e uma
// linha de JavaScript copiada seis vezes é a que diverge na sétima.
func alternaOCracha(aba, chave, valor string) string {
	sinal := "$" + chave
	return fmt.Sprintf(
		"%s = %s.includes(%q) ? %s.filter(v => v !== %q) : [...%s, %q]; @get('/piloto/mestre/%s')",
		sinal, sinal, valor, sinal, valor, sinal, valor, aba,
	)
}

// nomeDaCondicao resolve o id de uma condição no nome que se lê.
//
// DIVERGÊNCIA DELIBERADA do original, e por isso escrita: a SPA imprime o
// `upgradesTo` cru, então a linha sai "Agrava para apavorado" em caixa baixa. O
// dado do agravamento é um id, e o nome existe no mesmo catálogo — resolver é
// olhar a tabela ao lado, não inventar. Se alguém preferir o cru, muda aqui e
// nas duas telas.
func nomeDaCondicao(id string) string {
	for _, c := range catalogosDoLivro().Condicoes {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

// aprimoramentosEscritos concorda em número, que é a razão de existir: "1
// aprimoramento" e "3 aprimoramentos".
func aprimoramentosEscritos(n int) string {
	if n == 1 {
		return "1 aprimoramento disponível."
	}
	return fmt.Sprintf("%d aprimoramentos disponíveis.", n)
}

// precoEscrito é o dinheiro do livro em pt-BR, no MESMO formato do
// `formatTibar` da SPA: vírgula decimal e no máximo duas casas.
//
// Duas casas e não zero porque o preço do livro é fracionário — uma vela custa
// T$ 0,1 (p143) —, e cortar a fração poria "T$ 0" numa linha de compra.
func precoEscrito(v float64) string {
	return "T$ " + numeroPtBR(v)
}

// espacosEscritos: espaço de mochila também é fracionário (item leve ocupa 0,5).
func espacosEscritos(v float64) string { return numeroPtBR(v) }

// numeroPtBR escreve com vírgula decimal e sem zero à toa: 2 sai "2", 0.5 sai
// "0,5", 12.25 sai "12,25".
func numeroPtBR(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if len(s) > 2 && strings.HasSuffix(s, ".0") {
		s = s[:len(s)-2]
	}
	return strings.Replace(s, ".", ",", 1)
}
