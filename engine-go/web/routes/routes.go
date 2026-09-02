// Package routes são os endereços que uma cena cita de OUTRA.
//
// Ele nasceu na ALE-278, quando o buscador foi sair: ele monta o link do
// resultado com `/mestre/bestiario`, que era uma constante do trilho do mestre.
// Depois de virar pacote, a cena não alcança mais o `api` — e a alternativa
// óbvia, escrever a string de novo do lado de lá, é a que quebra em silêncio.
//
// # O critério de entrada é estreito, e é de propósito
//
// **Só entra endereço que uma cena cita de outra.** O `/buscador` é escrito
// pelos arquivos do buscador e por mais ninguém; o `/livro/ler`, pelos do livro.
// Esses ficam com o dono, porque trazê-los para cá não compra nada e transforma
// este arquivo no lugar onde tudo cabe — que é o que o `plataforma` tem escrito
// na porta dele para não virar.
//
// Medido na entrada: das cinco constantes de rota que o `api` tinha, DUAS são
// citadas de fora da própria família. São estas.
//
// # Por que não a `Deps` de cada cena
//
// Porque isto é constante, não comportamento. Uma porta existe para a cena
// declarar o que ela precisa que ALGUÉM FAÇA por ela; um endereço não é feito
// por ninguém — ele é o mesmo em todo processo, em todo teste, para sempre.
// Passá-lo por interface seria cerimônia repetida em cada cena que linkar para
// outra, e são sete extrações pela frente.
//
// # O que este pacote NÃO é
//
// Não é o mapa de rotas do app: quem registra rota é cada cena, no `Routes` dela
// (ALE-278). Aqui ficam só os endereços CITADOS, e a diferença importa — uma
// cena pode atender vinte rotas e não aparecer aqui nenhuma vez.
//
// E ele não substitui o `legacy_addresses.go`, que é outra coisa: aquele guarda
// os endereços ANTIGOS, publicados pela SPA, e o desvio de cada um.
package routes

import "net/url"

// MasterBestiary é a base da cena do bestiário no trilho do mestre.
//
// O painel do bestiário na Mesa tem endereço próprio e as duas superfícies
// dividem o mesmo desenho — ver `bestiarioView.Base`. Quem linka para o verbete
// de uma criatura de fora da Mesa vem para cá.
const MasterBestiary = "/mestre/bestiario"

// Book é a cena que abre o PDF do livro.
//
// Ela é citada pela Mesa, que monta o botão de sair para o livro. Desde a
// ALE-280 o endereço é a raiz — não há prefixo a somar, e somar um produziria
// `//livro`, que o navegador lê como o HOST `livro`.
const Book = "/livro"

// masterRail é o prefixo do trilho do mestre, e ele existe para as três funções
// abaixo não o escreverem à mão.
//
// Elas escreviam: duas montavam `"/mestre/" + aba + …` enquanto a terceira usava
// a constante do bestiário. Três grafias do mesmo prefixo em quatro linhas é
// exatamente o que este pacote veio impedir, e ninguém tinha visto porque as
// três moravam juntas num arquivo de cena.
const masterRail = "/mestre/"

// MasterEntry é o endereço de UM verbete: a aba do trilho mostrando só ele.
//
//	routes.MasterEntry("condicoes", "abalado")  // "/mestre/condicoes?entrada=abalado"
//
// Não confundir com `MasterSearch`: `?entrada=` é o verbete, `?busca=` procura o
// termo e mostra os grupos. A diferença nasceu de um defeito de UX — o elo
// endereçava por busca, e clicar num conceito caía numa lista onde ele era o
// quinto grupo (ver GLOSSARIO, verbete **entrada**).
func MasterEntry(tab, id string) string {
	return masterRail + tab + "?entrada=" + url.QueryEscape(id)
}

// MasterSearch procura o termo dentro de UMA aba do trilho.
func MasterSearch(tab, term string) string {
	return masterRail + tab + "?busca=" + url.QueryEscape(term)
}

// MasterBestiarySearch procura no bestiário do mestre.
func MasterBestiarySearch(term string) string {
	return MasterBestiary + "?busca=" + url.QueryEscape(term)
}
