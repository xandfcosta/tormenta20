package api

import (
	"slices"
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
	return parteNosNomes(texto, nomesDeCondicaoPorTamanho(), exceto, "condicoes")
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

// eloDoDevoto acha a aba de um devoto do deus ("Elfos", "Bárbaros").
//
// O dado vem no PLURAL e as entradas são singulares, então a busca tenta o nome
// como veio e depois sem o "s" final. Não achou, não vira elo: "Quaisquer" é
// devoto de Aharadak e não é raça nem classe nenhuma.
func eloDoDevoto(nome string) (aba, id string) {
	candidatos := []string{nome, strings.TrimSuffix(nome, "s"), strings.TrimSuffix(nome, "es")}
	racas, classes, _ := catalogosDoPersonagem()
	for _, candidato := range candidatos {
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
	return "", ""
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
