package sheetui

import (
	"regexp"
)

// Os guardas da FICHA em Datastar (ALE-272, fatia 1).
//
// O que eles prendem é o que a casca PROMETE: o endereço das abas, a posse, e os
// dois gestos que o crachá tem. Painel nenhum foi portado ainda — o que existe
// aqui é o envoltório, e é ele que precisa estar certo antes de sete painéis se
// pendurarem nele.

// sheetOf monta uma ficha alcançável, com classe (o degrau precisa de uma).
//
// Ela NÃO se chama `sheet`: o pacote da cena importa o `sheet` de verdade, e uma
// função com o nome do pacote o esconde do arquivo inteiro.
// O ENDEREÇO DAS ABAS é contrato, e ele veio da SPA inteiro.
//
// `?tab=` é link compartilhado e favorito: `abilities` continua sendo Poderes
// (o valor sobreviveu de propósito ao renome Habilidades→Poderes), e os dois
// nomes velhos da Mochila continuam chegando nela. Lixo cai na primeira aba, e
// não numa tela em branco.
var sceneAlert = regexp.MustCompile(`role="alert"[^>]*>([^<]*)</p>`)
