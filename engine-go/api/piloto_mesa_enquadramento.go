package api

import (
	"fmt"
	"strconv"
)

// O ENQUADRAMENTO do tabuleiro: o zoom e o centralizar (ALE-264 item 6, ALE-269
// item 9).
//
// Arquivo próprio porque é uma responsabilidade fechada e ela tem um DONO: o
// NAVEGADOR. Nada aqui vai ao servidor, e é essa escolha que tirou a ida à rede
// de cada passo de arrasto — a janela é um par de sinais, o `--quadrado` é o
// zoom, e o servidor nunca sabe onde cada pessoa está olhando. Por isso o
// remendo do stream redesenha as peças sem que ninguém perca o enquadramento.
//
// O que o servidor DÁ é o alvo: ele sabe onde as peças estão, e é ele quem
// escreve o número que o centralizar persegue.

// ── O ZOOM do plano (ALE-264, item 6) ────────────────────────────────────────
//
// `--quadrado` É o zoom, e isso já estava escrito no CSS antes de haver gesto: a
// grade, as peças, os marcadores e o terreno derivam todos do mesmo número.
// Mudar UM valor reenquadra a cena inteira.
//
// E a conta do clique acompanha de graça: ela já dividia por `$quadrado`, que é
// o mesmo número. Era isso que o comentário da camada de casas prometia com "o
// `$quadrado` acompanha o zoom quando ele chegar".

// Os LIMITES são os da SPA, com as razões dela (`board-viewport.ts`): abaixo de
// 20 a peça vira um ponto e o rótulo some; acima de 96 uma tela de 1024 mostra
// 10 quadrados, menos que dois deslocamentos padrão (9m = 6 quadrados, p106), e
// o mestre deixa de ver para onde dá para andar. Portar os números em vez de
// inventá-los é o que faz as duas telas enquadrarem igual.
const (
	quadradoMinimo = 20
	quadradoMaximo = 96
	quadradoPadrao = 44
	passoDoZoom    = 8
)

// ampliaOPlano soma um passo ao zoom, preso aos limites.
//
// O passo é EXPRESSÃO e não número desde a ALE-203: a roda decide o sinal dele
// já no navegador (`deltaY < 0 ? ...`), e ela precisava dos mesmos limites que os
// botões. Escritos à mão lá, seriam a segunda cópia dos tetos — e a que
// divergiria no dia em que o zoom máximo mudasse.
func ampliaOPlano(passo string) string {
	return fmt.Sprintf("$quadrado = Math.min(%d, Math.max(%d, $quadrado + (%s)))",
		quadradoMaximo, quadradoMinimo, passo)
}

// zoomNoLimite é a pergunta que desabilita o botão que não faria nada.
func zoomNoLimite(delta int) string {
	if delta < 0 {
		return fmt.Sprintf("$quadrado <= %d", quadradoMinimo)
	}
	return fmt.Sprintf("$quadrado >= %d", quadradoMaximo)
}

// ampliaAncorado muda o zoom SEM tirar de baixo do ponto o quadrado que estava
// lá (ALE-203).
//
// Sem âncora o zoom acontece a partir da QUINA da janela, e num plano infinito
// isso arrasta a cena inteira debaixo do dedo: aproximar para olhar o ogro do
// meio da tela empurra o ogro para fora dela. Com a moldura o defeito era menor
// porque o palco rolava dentro de uma caixa com fim; sem ela, três passos de
// zoom bastam para perder o grupo.
//
// A conta é a de sempre nesta família de ferramentas: guarde o ponto do plano
// que está sob a âncora, mude a escala, e reescreva a janela para que aquele
// mesmo ponto do plano volte para a mesma âncora. Os dois `const` vêm ANTES do
// `ampliaOPlano` porque leem `$quadrado`.
func ampliaAncorado(passo, pixelX, pixelY string) string {
	x, y := oPontoNoPlano(pixelX, pixelY)
	return fmt.Sprintf("const ancorax = %s, ancoray = %s; %s; $%s = ancorax * $quadrado - (%s); $%s = ancoray * $quadrado - (%s)",
		x, y, ampliaOPlano(passo), sinalDaVistaX, pixelX, sinalDaVistaY, pixelY)
}

// ampliaPeloMeioDaCena é o zoom SEM ponteiro: os botões e as teclas.
//
// A âncora é o MEIO da janela porque é ali que está o que a pessoa escolheu
// olhar. A quina seria o mesmo defeito do parágrafo acima, só que sem ninguém
// para culpar pelo lugar do dedo.
func ampliaPeloMeioDaCena(passo string) string {
	return fmt.Sprintf("const janela = document.getElementById(%q).getBoundingClientRect(); %s",
		idDaCena, ampliaAncorado(passo, "janela.width / 2", "janela.height / 2"))
}

// zoomPeloTeclado: `+` e `-`, as mesmas teclas da SPA.
//
// A guarda de alvo de digitação é a mesma do atalho da barra: sem ela, digitar
// um "-" no nome de um combatente reenquadraria o tabuleiro atrás do formulário.
var zoomPeloTeclado = semAlvoDeDigitacao +
	fmt.Sprintf("(evt.key === '+' || evt.key === '=' ? (() => { %s })() : "+
		"evt.key === '-' ? (() => { %s })() : null)",
		ampliaPeloMeioDaCena(oPasso(passoDoZoom)), ampliaPeloMeioDaCena(oPasso(-passoDoZoom)))

// ── CENTRALIZAR NAS PEÇAS (ALE-269, item 9) ──────────────────────────────────
//
// Num plano sem bordas "voltar ao começo" não significa nada: o que o mestre
// quer é ACHAR O GRUPO. Por isso o alvo é a caixa que contém as peças, e não a
// origem do plano — a mesma decisão que a SPA tomou no `fit`.
//
// Ele CENTRALIZA e não aproxima, também como a SPA: mexer no zoom junto tiraria
// da pessoa a escala que ela acabou de escolher, e achar o grupo é a pergunta
// que ela fez.
//
// A conta é do navegador porque ela é sobre PIXELS e sobre a janela — quantos
// quadrados cabem na tela é coisa que só o dedo sabe. O que vem do servidor é o
// ALVO em quadrados, e ele vem porque só o servidor sabe onde as peças estão.

// centralizaNasPecas põe o grupo no meio da tela.
//
// Ela ROLAVA o palco, e o palco deixou de rolar: sem moldura não há caixa com fim
// para o navegador prender a rolagem. O que se move agora é a JANELA, que é um
// par de sinais — e por isso a conta de limite também sumiu, porque num plano
// infinito não existe "pedir mais do que dá".
//
// O gesto SALTA e não desliza, e isso continua sendo medição e não gosto: com
// `smooth` (no `scrollTo` ou no CSS) a rolagem não acontecia, o `scrollTop`
// ficava em ZERO e não havia erro em lugar nenhum. Um botão que anima e não
// chega é pior que um botão que salta.
func centralizaNasPecas(v tabuleiroView) string {
	x, y := oCentroDaCena(v)
	return centralizaAJanelaEm(x, y)
}

// oCentroDaCena é a caixa que contém todas as peças, em quadrados do PLANO.
//
// O corpo da peça entra na conta e não só a âncora dela: uma Colossal ocupa 6×6
// (p107), e centralizar pela quina deixaria metade do dragão fora da janela.
//
// SEM PEÇA o alvo é a ORIGEM do plano, e não mais "o meio da moldura" — a
// moldura era o que estava desenhado, e agora não há nada desenhado além do que
// existe. Num plano infinito e vazio, o (0,0) é o único lugar sobre o qual duas
// pessoas concordam.
func oCentroDaCena(v tabuleiroView) (x, y int) {
	if len(v.Pecas) == 0 {
		return 0, 0
	}
	menorX, maiorX := v.Pecas[0].X, v.Pecas[0].X
	menorY, maiorY := v.Pecas[0].Y, v.Pecas[0].Y
	for _, p := range v.Pecas {
		menorX = min(menorX, p.X)
		menorY = min(menorY, p.Y)
		maiorX = max(maiorX, p.X+p.Pegada-1)
		maiorY = max(maiorY, p.Y+p.Pegada-1)
	}
	return (menorX + maiorX) / 2, (menorY + maiorY) / 2
}

// oAlvoDoCentralizar é o que o botão PROMETE, e ele muda com a cena.
//
// Duas frases porque são dois alvos: com peças na mesa o gesto acha o grupo, e
// numa cena vazia ele devolve a janela ao meio do mapa. Uma frase só mentiria
// numa das duas — e "Centralizar nas peças" numa cena sem peça nenhuma é
// exatamente o tipo de rótulo que ensina que o botão está quebrado.
func oAlvoDoCentralizar(v tabuleiroView) string {
	if len(v.Pecas) == 0 {
		return "Centralizar o mapa"
	}
	return "Centralizar nas peças"
}

// oPasso escreve um passo de zoom como número em JavaScript.
func oPasso(delta int) string { return strconv.Itoa(delta) }
