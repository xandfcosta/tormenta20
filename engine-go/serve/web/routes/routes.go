// Package routes são os endereços que uma cena cita de OUTRA.
//
// Uma cena não alcança as constantes de rota de outra, e a alternativa óbvia —
// escrever a string de novo do lado de lá — é a que quebra em silêncio.
//
// # O critério de entrada é estreito, e é de propósito
//
// **Só entra endereço que uma cena cita de outra.** O `/buscador` é escrito
// pelos arquivos do buscador e por mais ninguém; o `/livro/ler`, pelos do livro.
// Esses ficam com o dono, porque trazê-los para cá não compra nada e transforma
// este arquivo no lugar onde tudo cabe.
//
// **A exceção é o endereço que a PRÓPRIA cena não consegue escrever uma vez
// só**, e ela tem um número: o da sessão estava copiado em dezenove sítios
// dentro da Mesa, todos montando o mesmo `Sprintf` (ALE-346). Um endereço com
// ids dentro não tem onde morar numa cena feita de funções soltas — cada uma
// recebe os ids e remonta o caminho —, e é aí que trazê-lo para cá compra o que
// o critério de cima diz que não compra.
//
// # Por que não a `Deps` de cada cena
//
// Porque isto é constante, não comportamento. Uma porta existe para a cena
// declarar o que ela precisa que ALGUÉM FAÇA por ela; um endereço não é feito
// por ninguém — ele é o mesmo em todo processo, em todo teste, para sempre.
// Passá-lo por interface seria cerimônia repetida em cada cena que linkar para
// outra.
//
// # O que este pacote NÃO é
//
// Não é o mapa de rotas do app: quem registra rota é cada cena, no `Routes`
// dela. Aqui ficam só os endereços CITADOS, e a diferença importa — uma cena
// pode atender vinte rotas e não aparecer aqui nenhuma vez.
package routes

import (
	"fmt"
	"net/url"
)

// MasterBestiary é a base da cena do bestiário no trilho do mestre.
//
// O painel do bestiário na Mesa tem endereço próprio e as duas superfícies
// dividem o mesmo desenho — ver `bestiarioView.Base`. Quem linka para o verbete
// de uma criatura de fora da Mesa vem para cá.
const MasterBestiary = "/mestre/bestiario"

// Book é a cena que abre o PDF do livro.
//
// Ela é citada pela Mesa, que monta o botão de sair para o livro. O endereço é a
// raiz: não há prefixo a somar, e somar um produziria `//livro`, que o navegador
// lê como o HOST `livro`.
const Book = "/livro"

// Reader é a cena que DESENHA o livro, uma página por vez, com o termo
// destacado.
//
// Não é o visualizador do navegador, que fica a um clique de distância: ver o
// verbete **leitor** no GLOSSARY.
const Reader = "/livro/ler"

// Entry é a rota que devolve UM verbete como fragmento, para a caixa que o
// mostra por cima da cena sem tirar a pessoa da regra que ela lia.
const Entry = "/verbete"

// masterRail é o prefixo do trilho do mestre, e ele existe para as três funções
// abaixo não o escreverem à mão: três grafias do mesmo prefixo em quatro linhas
// é exatamente o que este pacote veio impedir.
const masterRail = "/mestre/"

// MasterEntry é o endereço de UM verbete: a aba do trilho mostrando só ele.
//
//	routes.MasterEntry("condicoes", "abalado")  // "/mestre/condicoes?entrada=abalado"
//
// Não confundir com `MasterSearch`: `?entrada=` é o verbete, `?busca=` procura o
// termo e mostra os grupos. Endereçar um elo por busca faz clicar num conceito
// cair numa lista onde ele é o quinto grupo (ver GLOSSARY, verbete **entrada**).
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

// Session é PARA ONDE se entra numa sessão: a cena ao vivo dela.
//
// Ela é a ÚNICA grafia deste endereço no repositório, e quem cobra é o
// `TestNoHandwrittenSessionAddress`: fora daqui, nenhum literal de string
// escreve `/sessoes` — só o padrão do chi, que traz os parâmetros nomeados.
//
// Citam-na de fora o Hub, o cartão da campanha e duas linhas da campanha aberta;
// de DENTRO, a própria Mesa, pelo `View.SessionBase`, que a chama uma vez e
// pendura o verbo. Eram vinte e três grafias do mesmo caminho antes da ALE-346.
//
// O endereço é ANINHADO na campanha porque a sessão é dela: não existe sessão
// fora de uma campanha, e o pai já atende em `/campanhas/{id}`.
//
//	routes.Session(1, 4) // "/campanhas/1/sessoes/4"
func Session(campaignID, sessionID int64) string {
	return fmt.Sprintf("/campanhas/%d/sessoes/%d", campaignID, sessionID)
}

// PlaceDraft é PARA ONDE se entra num RASCUNHO DE LUGAR: a cena que o mestre
// monta no acervo da campanha, fora da sessão.
//
// O endereço é da CAMPANHA e não da mesa porque o lugar é do acervo e sobrevive
// a qualquer sessão — montar a cripta na quinta-feira não pode depender de haver
// uma partida rolando. É a diferença que o GLOSSARY desenha entre o rascunho e a
// cortina.
//
//	routes.PlaceDraft(12, 7) // "/campanhas/12/lugares/7"
func PlaceDraft(campaignID, placeID int64) string {
	return fmt.Sprintf("/campanhas/%d/lugares/%d", campaignID, placeID)
}

// CampaignTab é a crônica aberta numa seção: `/campanhas/12?tab=lugares`.
//
// A ABA é endereço e não estado do navegador — decisão que a cena da campanha já
// carrega —, e é por isso que ela cabe numa função em vez de num sinal: o link
// volta no histórico, abre em nova aba e é o que alguém cola no chat da mesa.
//
// Aba vazia devolve a crônica sem query, que é a visão geral.
//
//	routes.CampaignTab(12, "lugares") // "/campanhas/12?tab=lugares"
func CampaignTab(campaignID int64, aba string) string {
	if aba == "" {
		return fmt.Sprintf("/campanhas/%d", campaignID)
	}
	return fmt.Sprintf("/campanhas/%d?tab=%s", campaignID, url.QueryEscape(aba))
}
