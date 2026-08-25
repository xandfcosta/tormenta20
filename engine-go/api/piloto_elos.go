package api

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// OS ELOS entre entradas do acervo (ALE-264).
//
// O livro é uma rede: a condição Abalado termina em "Medo.", que é um TIPO DE
// EFEITO definido na p228; ela agrava para Apavorado, que é outra condição; o
// deus concede poderes que têm verbete próprio. Na tela isso era tudo texto
// morto — o mestre lia "Medo" e tinha de ir procurar o que era.
//
// O elo leva para a MESMA cena com a entrada filtrada
// (`destinoNoAcervo`), que é o endereço que o buscador já usa. Nenhuma
// superfície nova: a entrada aparece sozinha na aba dela, e o botão do livro
// está ao lado se a pessoa quiser o texto completo.
//
// O que NÃO virou elo, e é decisão: nome de entrada citado no meio de descrição
// de MAGIA ou de PODER. Medido no catálogo — são 3 citações em 668 entradas,
// porque as descrições da casa são resumos e não o texto do livro. Varrer 992
// descrições atrás de 35 nomes para achar três acertos é custo por tela sem
// retorno. Nas CONDIÇÕES o número é outro (11 em 35) e por isso elas têm.

// efeitoDoLivro é um tipo de efeito — a família que a condição carrega no rodapé.
type efeitoDoLivro struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	BookPage    int    `json:"bookPage"`
}

var (
	efeitosUmaVez   sync.Once
	efeitosDoAcervo []efeitoDoLivro
)

func tiposDeEfeito() []efeitoDoLivro {
	efeitosUmaVez.Do(func() {
		efeitosDoAcervo = listaDoCatalogo[efeitoDoLivro]("tipos-de-efeito")
	})
	return efeitosDoAcervo
}

// nomeDoEfeito resolve o id que a condição guarda (`tags: ["medo"]`) no nome que
// se lê. Id desconhecido volta como veio: a tag é dado, e dado envelhece — some
// o elo, não o rótulo.
func nomeDoEfeito(id string) string {
	for _, e := range tiposDeEfeito() {
		if e.ID == id {
			return e.Name
		}
	}
	return id
}

func camposDoEfeito(e efeitoDoLivro) []string { return []string{e.Name, e.Description} }

// ── escola de magia ──────────────────────────────────────────────────────────

// escolaDeMagia é a família de uma magia (T20 p172). Ela mora aqui, ao lado do
// tipo de efeito, porque nasceu pela mesma razão: a magia a CITA, e citação sem
// destino é texto morto. E as duas coisas se tocam — o livro diz que "escolas de
// magia contam como tipos de efeitos".
type escolaDeMagia struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Abrev é a forma curta que o livro imprime nas tabelas ("Abjur"). Ilusão
	// não tem — o livro não deu —, e vazio aqui é ausência e não dado.
	Abrev    string `json:"abrev,omitempty"`
	BookPage int    `json:"bookPage"`
}

var (
	escolasUmaVez  sync.Once
	escolasDoLivro []escolaDeMagia
)

func escolasDeMagia() []escolaDeMagia {
	escolasUmaVez.Do(func() {
		escolasDoLivro = listaDoCatalogo[escolaDeMagia]("escolas-de-magia")
	})
	return escolasDoLivro
}

// nomeDaEscola resolve o id que a magia guarda ("evocacao") no nome que se lê.
//
// Sai do CATÁLOGO e não de uma tabela no código, e isso é o conserto de uma
// duplicação que eu mesmo criei duas horas antes: escrevi as oito à mão para
// rotular o filtro, e agora elas têm verbete. Duas listas dos mesmos oito nomes
// divergem na primeira correção de acento.
func nomeDaEscola(id string) string {
	for _, e := range escolasDeMagia() {
		if e.ID == id {
			return e.Name
		}
	}
	return id
}

func camposDaEscola(e escolaDeMagia) []string {
	return []string{e.Name, e.Abrev, e.Description}
}

// ── o texto com elos dentro ──────────────────────────────────────────────────

// trecho é um pedaço de descrição. `Aba` vazia é texto puro; preenchida, o
// pedaço é um ELO para aquela aba do acervo.
//
// Um tipo e não HTML montado em string: texto do catálogo passa pelo escape do
// templ como qualquer outro, e montar `<a>` aqui seria abrir mão disso para
// sempre — a primeira descrição com um `<` viraria tela quebrada ou pior.
type trecho struct {
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

// comElosParaCondicoes parte a descrição nos nomes de CONDIÇÃO que ela cita.
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
func comElosParaCondicoes(texto, exceto string) []trecho {
	return comElosDePagina(parteNosNomes(texto, nomesDeCondicaoPorTamanho(), exceto, "condicoes"))
}

// referenciaDePagina é como o livro cita a si mesmo: "veja a página 230",
// "pág. 172". Medido no catálogo — são cinco ocorrências, duas nos tipos de
// efeito e três nos dragões —, e cada uma era texto morto: o número estava lá e
// não levava a lugar nenhum.
var referenciaDePagina = regexp.MustCompile(`(?i)p[áa]g(?:ina)?\.?\s*(\d{1,3})`)

// comElosDePagina parte os pedaços de TEXTO PURO nas referências de página.
//
// Roda DEPOIS da varredura de nomes e só sobre o que sobrou como texto: um
// pedaço que já virou elo para um verbete não pode virar elo para o livro
// também — dois destinos na mesma palavra é uma escolha que ninguém pediu.
func comElosDePagina(pedacos []trecho) []trecho {
	var fora []trecho
	for _, pedaco := range pedacos {
		if pedaco.Aba != "" {
			fora = append(fora, pedaco)
			continue
		}
		fora = append(fora, parteNasPaginas(pedaco.Texto)...)
	}
	return fora
}

func parteNasPaginas(texto string) []trecho {
	marcas := referenciaDePagina.FindAllStringSubmatchIndex(texto, -1)
	if marcas == nil {
		return []trecho{{Texto: texto}}
	}
	var fora []trecho
	fim := 0
	for _, m := range marcas {
		pagina, err := strconv.Atoi(texto[m[2]:m[3]])
		if err != nil || pagina <= 0 {
			continue
		}
		if antes := texto[fim:m[0]]; antes != "" {
			fora = append(fora, trecho{Texto: antes})
		}
		fora = append(fora, trecho{Texto: texto[m[0]:m[1]], Pagina: pagina})
		fim = m[1]
	}
	if resto := texto[fim:]; resto != "" {
		fora = append(fora, trecho{Texto: resto})
	}
	return fora
}

// comElosDoTexto é a varredura para os catálogos que NÃO citam condições — só as
// referências de página. Ver o cabeçalho: em magia e poder as citações de
// condição são 3 em 668, e varrer 992 descrições atrás delas é custo sem retorno.
func comElosDoTexto(texto string) []trecho {
	return parteNasPaginas(texto)
}

// idDaCondicao resolve o nome no id com que o catálogo a guarda.
func idDaCondicao(nome string) string {
	for _, c := range catalogosDoLivro().Condicoes {
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

func nomesDeCondicaoPorTamanho() []string {
	nomesUmaVez.Do(func() {
		for _, c := range catalogosDoLivro().Condicoes {
			nomesLongos = append(nomesLongos, c.Name)
		}
		slices.SortFunc(nomesLongos, func(a, b string) int { return len(b) - len(a) })
	})
	return nomesLongos
}

// parteNosNomes é a varredura, e ela casa PALAVRA INTEIRA com a caixa do livro.
//
// Caixa exata porque no texto do livro a condição é escrita com maiúscula
// ("fica Abalado") e a palavra comum não ("um efeito de medo") — casar sem caixa
// encheria a tela de elos que não são citação nenhuma.
func parteNosNomes(texto string, nomes []string, exceto, aba string) []trecho {
	for _, nome := range nomes {
		if nome == exceto {
			continue
		}
		onde := indicePalavraInteira(texto, nome)
		if onde < 0 {
			continue
		}
		var fora []trecho
		if antes := texto[:onde]; antes != "" {
			fora = append(fora, parteNosNomes(antes, nomes, exceto, aba)...)
		}
		fora = append(fora, trecho{Texto: nome, Aba: aba, ID: idDaCondicao(nome)})
		if depois := texto[onde+len(nome):]; depois != "" {
			fora = append(fora, parteNosNomes(depois, nomes, exceto, aba)...)
		}
		return fora
	}
	return []trecho{{Texto: texto}}
}

// indicePalavraInteira acha o nome com fronteira dos dois lados, ou -1.
func indicePalavraInteira(texto, nome string) int {
	de := 0
	for {
		onde := strings.Index(texto[de:], nome)
		if onde < 0 {
			return -1
		}
		onde += de
		if fronteira(texto, onde-1) && fronteira(texto, onde+len(nome)) {
			return onde
		}
		de = onde + 1
	}
}

// fronteira: fora do texto conta como fronteira, e letra não conta.
func fronteira(texto string, i int) bool {
	if i < 0 || i >= len(texto) {
		return true
	}
	r := rune(texto[i])
	return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r >= 0x80)
}

// ── os elos que vêm de CAMPO e não de texto ──────────────────────────────────

// eloDoDevoto acha a aba e o id de um devoto do deus ("Elfos", "Bárbaros").
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
func eloDoDevoto(nome string) (aba, id string) {
	racas, classes, _ := catalogosDoPersonagem()
	for _, candidato := range noSingular(nome) {
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
		if slices.Contains(r.Ascendencias, dobraSimples(nome)) {
			return "racas", r.ID
		}
	}
	return "", ""
}

// noSingular devolve as formas a tentar, do nome como veio ao singular provável.
//
// As regras são as do português, e cada uma nasceu de um caso do catálogo:
//
//	"Elfos"            → "Elfo"      (s)
//	"Caçadores"        → "Caçador"   (es)
//	"Anões"            → "Anão"      (ões → ão)
//	"Golens"           → "Golem"     (ns → m)
//	"Sereias/Tritões"  → "Sereia/Tritão"  (as duas metades)
func noSingular(nome string) []string {
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

// dobraSimples é minúsculas sem acento, para casar a ascendência que o catálogo
// guarda em caixa baixa ("aggelus") com o nome que o deus escreve ("Aggelus").
func dobraSimples(s string) string {
	return strings.ToLower(s)
}

// idDoPoder devolve o id do poder concedido pelo deus, ou vazio se ele não tem
// verbete no acervo.
func idDoPoder(nome string) string {
	for _, p := range catalogosDoLivro().Poderes {
		if p.Name == nome {
			return p.ID
		}
	}
	return ""
}
