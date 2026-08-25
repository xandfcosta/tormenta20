package api

import (
	"cmp"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// O BUSCADOR DO LIVRO (ALE-264): ⌃K abre uma caixa que procura nas 1.072
// entradas do livro de uma vez — criaturas, condições, magias, poderes e itens
// —, sem tirar a mão do teclado e sem saber em qual ferramenta a coisa mora.
//
// A busca não é nova: o servidor já varre os quatro catálogos em
// `carregaCatalogos` e o bestiário em `carregaBestiarioDe`. O que este arquivo
// acrescenta é RANQUEAR (ver `pontuaBusca`) e dar um DESTINO a cada achado —
// porque aqui a lista é curta e cada linha tem de saber para onde levar.
//
// Decisão do dono, com mockup na mão: o Enter NAVEGA para a cena com a entrada
// aberta, em vez de desenhar o verbete dentro do próprio diálogo. Reusa as duas
// cenas que já existem em vez de manter um terceiro desenho da mesma regra.
//
// "Com a entrada ABERTA" virou `?entrada=<id>` na segunda passada: o endereço
// era uma BUSCA pelo nome, e cair numa lista de oito grupos para achar o que se
// escolheu é o oposto de escolher.

// achadosPorGrupo corta cada grupo, e o corte é DITO na tela ("+12").
//
// Seis porque a lista inteira tem de caber sem rolagem numa janela de laptop
// com cinco grupos possíveis. Corte silencioso seria pior que corte nenhum:
// quem não vê o que ficou de fora conclui que não existe.
const achadosPorGrupo = 6

// achadoDoBuscador é uma linha do resultado.
type achadoDoBuscador struct {
	Nome    string
	Detalhe string
	Destino string
	// Pagina é a do livro, e ZERO significa "o catálogo não sabe" — a linha sai
	// sem número em vez de sair com "p0". Desde a derivação das páginas
	// (`scripts/paginas-do-livro.py`) os cinco catálogos sabem a sua; o que
	// continua zerado são as 81 entradas que o Índice Remissivo não resolve.
	Pagina int
	// ponto não vai para a tela: ele é a ORDEM, e mostrá-lo convidaria a
	// discutir a nota em vez do resultado.
	ponto int
}

// grupoDoBuscador é um catálogo com o que sobrou, já cortado.
type grupoDoBuscador struct {
	Rotulo  string
	Achados []achadoDoBuscador
	// Total é antes do corte — é ele que escreve o "+12".
	Total int
	// Mais é para onde o "+12" leva: a cena da ferramenta com a MESMA busca.
	// Corte com saída, e não corte que informa e abandona.
	Mais string
}

func (g grupoDoBuscador) Cortados() int { return g.Total - len(g.Achados) }

type buscadorView struct {
	Busca  string
	Grupos []grupoDoBuscador
	// Achados é o total ANTES dos cortes de grupo.
	Achados int
	// PeloTexto diz que estes achados vieram da segunda passada — nenhum NOME
	// casou, e o que está na lista apenas MENCIONA o termo. A tela avisa: sem
	// isso, uma lista que não contém o que se pediu parece a lista errada.
	PeloTexto bool
}

func (v buscadorView) Buscando() bool { return strings.TrimSpace(v.Busca) != "" }

// buscaNoLivro monta o resultado do ⌃K.
//
// A ORDEM DOS GRUPOS é a do acervo e tem razão registrada lá: condição primeiro
// porque é a consulta mais frequente no meio do combate, criatura em seguida
// porque é a segunda, e os três do PERSONAGEM no fim — raça, classe e deus são
// consulta de criação de ficha, não de mesa com o combate em curso.
func buscaNoLivro(busca string) buscadorView {
	v := montaAchados(busca, pelosNomes)
	if v.Achados > 0 || !v.Buscando() {
		return v
	}
	// SEGUNDA PASSADA, e ela é o conserto de uma medição na tela: com o corpo
	// das regras valendo sempre, "abal" devolvia 295 entradas — 142 poderes cujo
	// texto diz "Abalado" — e a condição "Abalado", que era o que se procurava,
	// saía espremida num grupo de seis ao lado de "Naja" e "Jiboia".
	//
	// Então o corpo da regra só entra quando NOME nenhum casou. Quem digita
	// "abal" procura o verbete; quem digita "chance de falha" não sabe o nome, e
	// é para essa pessoa que a passada existe.
	v = montaAchados(busca, tambemPeloTexto)
	v.PeloTexto = v.Achados > 0
	return v
}

// pelosNomes e tambemPeloTexto nomeiam as duas passadas. Um booleano cru na
// chamada (`grupoBuscado(..., true)`) não diz nada de dentro do `grupoBuscado`.
const (
	pelosNomes      = false
	tambemPeloTexto = true
)

func montaAchados(busca string, peloTexto bool) buscadorView {
	v := buscadorView{Busca: busca}
	if !v.Buscando() {
		return v
	}
	a := catalogosDoLivro()
	racas, classes, deuses := catalogosDoPersonagem()
	for _, g := range []grupoDoBuscador{
		grupoBuscado("Condições", a.Condicoes, busca, peloTexto, destinoNoAcervo("condicoes", busca), camposDaCondicao, achadoDaCondicao),
		grupoBuscado("Criaturas", criaturasDoLivro(), busca, peloTexto, destinoNoBestiario(busca), camposDoVerbete, achadoDoVerbete),
		grupoBuscado("Magias", a.Magias, busca, peloTexto, destinoNoAcervo("magias", busca), camposDaMagia, achadoDaMagia),
		grupoBuscado("Poderes", a.Poderes, busca, peloTexto, destinoNoAcervo("poderes", busca), camposDoPoder, achadoDoPoder),
		grupoBuscado("Itens", a.Itens, busca, peloTexto, destinoNoAcervo("itens", busca), camposDoItem, achadoDoItem),
		grupoBuscado("Efeitos", tiposDeEfeito(), busca, peloTexto, destinoNoAcervo("efeitos", busca), camposDoEfeito, achadoDoEfeito),
		grupoBuscado("Escolas", escolasDeMagia(), busca, peloTexto, destinoNoAcervo("escolas", busca), camposDaEscola, achadoDaEscola),
		grupoBuscado("Raças", racas, busca, peloTexto, destinoNoAcervo("racas", busca), camposDaRaca, achadoDaRaca),
		grupoBuscado("Classes", classes, busca, peloTexto, destinoNoAcervo("classes", busca), camposDaClasse, achadoDaClasse),
		grupoBuscado("Deuses", deuses, busca, peloTexto, destinoNoAcervo("deuses", busca), camposDoDeus, achadoDoDeus),
	} {
		if g.Total == 0 {
			continue
		}
		v.Achados += g.Total
		v.Grupos = append(v.Grupos, g)
	}
	ordenaPorRelevancia(v.Grupos)
	return v
}

// ordenaPorRelevancia põe na frente o grupo que tem o MELHOR achado.
//
// O defeito foi visto pelo dono na tela: digitando "medo", o verbete "Medo" —
// nome inteiro, a pontuação máxima — aparecia no SEXTO grupo, abaixo de
// criaturas que só têm a palavra no nome ("Devorador de Medos"). A ordem dos
// grupos era a da fileira de abas, fixa, e ela é a certa para NAVEGAR (condição
// primeiro porque é a consulta do combate) e a errada para BUSCAR.
//
// Estável, então a ordem da fileira continua valendo no EMPATE: dois grupos com
// achados igualmente bons saem na ordem de sempre.
func ordenaPorRelevancia(grupos []grupoDoBuscador) {
	slices.SortStableFunc(grupos, func(a, b grupoDoBuscador) int {
		return cmp.Compare(b.melhorPonto(), a.melhorPonto())
	})
}

// melhorPonto é a nota do primeiro achado — a lista já vem ordenada pelo
// `melhorPrimeiro`, então o primeiro é o melhor.
func (g grupoDoBuscador) melhorPonto() int {
	if len(g.Achados) == 0 {
		return 0
	}
	return g.Achados[0].ponto
}

// grupoBuscado pontua, ordena e corta um catálogo.
//
// `campos` é a MESMA função que a cena dos catálogos usa para filtrar, e reusá-la
// é o que faz as duas superfícies concordarem sobre o que é buscável — uma
// segunda lista de campos aqui divergiria no dia em que alguém acrescentasse um.
func grupoBuscado[T any](
	rotulo string, lista []T, busca string, peloTexto bool, mais string,
	campos func(T) []string, comoAchado func(T) achadoDoBuscador,
) grupoDoBuscador {
	g := grupoDoBuscador{Rotulo: rotulo, Mais: mais}
	for _, e := range lista {
		// A linha é montada ANTES de saber se ela passa, e é deliberado: pegar o
		// nome de `campos(e)[0]` seria depender de uma ordem que nada garante, e
		// no dia em que um `camposDaX` mudasse de ordem a pontuação passaria a
		// medir a descrição — em silêncio, com a lista continuando a sair. São
		// 1.072 structs por tecla digitada, com 200ms de debounce na frente.
		a := comoAchado(e)
		a.ponto = pontuaBusca(a.Nome, busca)
		if a.ponto == 0 && peloTexto {
			a.ponto = pontuaTexto(campos(e), busca)
		}
		if a.ponto == 0 {
			continue
		}
		g.Achados = append(g.Achados, a)
	}
	g.Total = len(g.Achados)
	slices.SortStableFunc(g.Achados, melhorPrimeiro)
	if len(g.Achados) > achadosPorGrupo {
		g.Achados = g.Achados[:achadosPorGrupo]
	}
	return g
}

// melhorPrimeiro: pontuação alta na frente e, empatados, o nome mais CURTO.
//
// O desempate por tamanho não é estético: com "fogo", "Bola de Fogo" e
// "Explosão de Fogo Congelante" pontuam igual, e o nome curto é o que a pessoa
// tem mais chance de estar procurando. O terceiro critério é o nome, para a
// ordem não depender da ordem de leitura do catálogo.
func melhorPrimeiro(a, b achadoDoBuscador) int {
	if d := cmp.Compare(b.ponto, a.ponto); d != 0 {
		return d
	}
	if d := cmp.Compare(len(a.Nome), len(b.Nome)); d != 0 {
		return d
	}
	return cmp.Compare(a.Nome, b.Nome)
}

// ── de cada catálogo para uma linha ──────────────────────────────────────────

// camposDoVerbete é o irmão dos `camposDaX` do acervo, e o bestiário não tinha
// um: a cena dele filtra por nome e tipo em `filtraCriaturas`. Aqui ele precisa
// existir para o verbete passar pelo mesmo `grupoBuscado` que os outros quatro.
func camposDoVerbete(m verbete) []string {
	return append([]string{m.Name, nomeDoTipo(m.Tipo)}, m.SpecialAbilities...)
}

func achadoDaCondicao(c condicaoDoLivro) achadoDoBuscador {
	detalhe := "Condição"
	if c.UpgradesTo != "" {
		detalhe = "Condição · agrava para " + nomeDaCondicao(c.UpgradesTo)
	}
	return achadoDoBuscador{Nome: c.Name, Detalhe: detalhe, Destino: destinoDaEntrada("condicoes", c.ID), Pagina: c.BookPage}
}

func achadoDaMagia(m magiaDoLivro) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    m.Name,
		Detalhe: fmt.Sprintf("%dº círculo · %s", m.Circle, nomeDaExecucao(m.Execution)),
		Destino: destinoDaEntrada("magias", m.ID),
		Pagina:  m.BookPage,
	}
}

func achadoDoPoder(p poderDoLivro) achadoDoBuscador {
	return achadoDoBuscador{Nome: p.Name, Detalhe: p.Fonte, Destino: destinoDaEntrada("poderes", p.ID), Pagina: p.BookPage}
}

func achadoDoItem(i itemDoLivro) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    i.Name,
		Detalhe: nomeDaCategoria(i.Category),
		Destino: destinoDaEntrada("itens", i.ID),
		Pagina:  i.BookPage,
	}
}

func achadoDoEfeito(e efeitoDoLivro) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    e.Name,
		Detalhe: "Tipo de efeito",
		Destino: destinoDaEntrada("efeitos", e.ID),
		Pagina:  e.BookPage,
	}
}

func achadoDaEscola(e escolaDeMagia) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    e.Name,
		Detalhe: "Escola de magia",
		Destino: destinoDaEntrada("escolas", e.ID),
		Pagina:  e.BookPage,
	}
}

func achadoDaRaca(r racaDoLivro) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    r.Name,
		Detalhe: nomeDoTier(r.Tier) + " · " + r.AtributoMod.Escrito(),
		Destino: destinoDaEntrada("racas", r.ID),
		Pagina:  r.BookPage,
	}
}

func achadoDaClasse(c classeDoLivro) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    c.Name,
		Detalhe: fmt.Sprintf("Classe · %d poderes", c.Poderes),
		Destino: destinoDaEntrada("classes", c.ID),
		Pagina:  c.BookPage,
	}
}

func achadoDoDeus(d deusDoLivro) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    d.Name,
		Detalhe: d.Portfolio,
		Destino: destinoDaEntrada("deuses", d.ID),
		Pagina:  d.BookPage,
	}
}

func achadoDoVerbete(m verbete) achadoDoBuscador {
	return achadoDoBuscador{
		Nome:    m.Name,
		Detalhe: fmt.Sprintf("ND %s · %s", ndEscrito(m.ND), nomeDoTipo(m.Tipo)),
		Destino: rotaDoBestiarioDoMestre + "?criatura=" + url.QueryEscape(m.ID),
		Pagina:  m.BookPage,
	}
}

// destinoNoAcervo leva à cena dos catálogos com a entrada já filtrada.
//
// Pelo NOME e não por um id na URL, e é escolha e não preguiça: a cena não tem
// endereço para uma entrada só — ela tem `?aba=` e `?busca=`, que já são
// endereços recarregáveis. Buscar pelo nome exato deixa a entrada sozinha na
// lista, que é o que a pessoa pediu, sem inventar um terceiro estado de cena
// que precisaria ser mantido junto.
func destinoNoAcervo(aba, nome string) string {
	return "/piloto/mestre/" + aba + "?busca=" + url.QueryEscape(nome)
}

// destinoDaEntrada é o endereço de UM verbete: a aba dele, mostrando só ele.
//
// Diferente do `destinoNoAcervo`, que faz uma BUSCA. A diferença apareceu na
// tela: clicar no elo "Medo" caía numa busca por "medo" nos oito catálogos, com
// o verbete procurado espremido no quinto grupo. Quem clica num conceito pediu o
// conceito.
func destinoDaEntrada(aba, id string) string {
	return "/piloto/mestre/" + aba + "?entrada=" + url.QueryEscape(id)
}

// destinoNoBestiario leva à cena do bestiário filtrada. Serve tanto para o "+12"
// (com o termo digitado) quanto para o que o `achadoDoVerbete` faz com o id.
func destinoNoBestiario(busca string) string {
	return rotaDoBestiarioDoMestre + "?busca=" + url.QueryEscape(busca)
}
