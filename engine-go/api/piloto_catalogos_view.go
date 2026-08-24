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
}

type magiaDoLivro struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Circle     int      `json:"circle"`
	School     string   `json:"school"`
	Execution  string   `json:"execution"`
	Range      string   `json:"range"`
	Duration   string   `json:"duration"`
	Resistance string   `json:"resistance,omitempty"`
	BaseEffect string   `json:"baseEffect"`
	Augments   []any    `json:"augments"`
	Classes    []string `json:"classes"`
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
}

type itemDoLivro struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Slots    float64 `json:"slots"`
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
	}]("class-powers") {
		fora = append(fora, poderDoLivro{p.ID, p.Name, p.ClassName, p.Description})
	}

	for _, p := range listaDoCatalogo[struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
	}]("general-powers") {
		fora = append(fora, poderDoLivro{"general." + p.ID, p.Name, "Geral · " + p.Kind, p.Description})
	}

	for _, p := range listaDoCatalogo[struct {
		ID     string   `json:"id"`
		Name   string   `json:"name"`
		Deuses []string `json:"deuses"`
		Effect string   `json:"effect"`
	}]("granted-powers") {
		fora = append(fora, poderDoLivro{
			"granted." + p.ID, p.Name, "Concedido · " + strings.Join(p.Deuses, ", "), p.Effect,
		})
	}
	return fora
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
var abasDoAcervo = []abaDoAcervo{
	{"condicoes", "Condições"},
	{"magias", "Magias"},
	{"poderes", "Poderes"},
	{"itens", "Itens"},
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
}

func (g grupoDoAcervo) Quantos() int {
	return len(g.Condicoes) + len(g.Magias) + len(g.Poderes) + len(g.Itens)
}

// catalogosView é a cena inteira numa resposta.
type catalogosView struct {
	Busca string
	// Aba só importa quando NÃO se está buscando: com termo digitado a cena
	// mostra os quatro catálogos agrupados, que é a decisão que a ALE-22
	// registrou — a versão em React filtrava só a aba ativa, e "bola de fogo"
	// digitado em Condições dizia "nada encontrado" com a magia existindo.
	Aba     string
	Grupos  []grupoDoAcervo
	Achados int
}

func (v catalogosView) Buscando() bool { return strings.TrimSpace(v.Busca) != "" }

// carregaCatalogos monta a cena: os quatro catálogos quando há busca, um só
// quando não há.
func carregaCatalogos(busca, aba string) catalogosView {
	v := catalogosView{Busca: busca, Aba: abaConhecida(aba)}
	a := catalogosDoLivro()

	if !v.Buscando() {
		v.Grupos = []grupoDoAcervo{grupoDaAba(a, v.Aba)}
		v.Achados = v.Grupos[0].Quantos()
		return v
	}

	for _, g := range []grupoDoAcervo{
		{Rotulo: "Condições", Condicoes: filtra(a.Condicoes, camposDaCondicao, busca)},
		{Rotulo: "Magias", Magias: filtra(a.Magias, camposDaMagia, busca)},
		{Rotulo: "Poderes", Poderes: filtra(a.Poderes, camposDoPoder, busca)},
		{Rotulo: "Itens", Itens: filtra(a.Itens, camposDoItem, busca)},
	} {
		if g.Quantos() == 0 {
			continue
		}
		v.Achados += g.Quantos()
		v.Grupos = append(v.Grupos, g)
	}
	return v
}

func grupoDaAba(a acervoDoMestre, aba string) grupoDoAcervo {
	switch aba {
	case "magias":
		return grupoDoAcervo{Rotulo: "Magias", Magias: a.Magias}
	case "poderes":
		return grupoDoAcervo{Rotulo: "Poderes", Poderes: a.Poderes}
	case "itens":
		return grupoDoAcervo{Rotulo: "Itens", Itens: a.Itens}
	default:
		return grupoDoAcervo{Rotulo: "Condições", Condicoes: a.Condicoes}
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
	return fmt.Sprintf(`{busca: %s, aba: %s}`, busca, aba)
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
