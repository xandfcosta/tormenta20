package api

import (
	"fmt"
)

// O ENQUADRAMENTO do tabuleiro: o zoom e o centralizar (ALE-264 item 6, ALE-269
// item 9).
//
// Arquivo próprio porque é uma responsabilidade fechada e ela tem um DONO: o
// NAVEGADOR. Nada aqui vai ao servidor, e é essa escolha que tirou a ida à rede
// de cada passo de arrasto — o palco rola nativamente, o `--quadrado` é o zoom,
// e o servidor nunca sabe onde cada pessoa está olhando. Por isso o remendo do
// stream redesenha as peças sem que ninguém perca o enquadramento.
//
// O que o servidor DÁ é o alvo: ele sabe onde as peças estão, e é ele quem
// escreve o número que o centralizar persegue.

// ── O ZOOM do plano (ALE-264, item 6) ────────────────────────────────────────
//
// `--quadrado` É o zoom, e isso já estava escrito no CSS antes de haver gesto: o
// plano tem `width: calc(var(--colunas) * var(--quadrado))`, e a grade, as
// peças, os marcadores e o terreno derivam do mesmo número. Mudar UM valor
// reenquadra a cena inteira, e a rolagem nativa continua valendo porque o plano
// muda de tamanho DE VERDADE — não é um `transform` que mente sobre o leiaute.
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
func ampliaOPlano(delta int) string {
	return fmt.Sprintf("$quadrado = Math.min(%d, Math.max(%d, $quadrado + (%d)))",
		quadradoMaximo, quadradoMinimo, delta)
}

// zoomNoLimite é a pergunta que desabilita o botão que não faria nada.
func zoomNoLimite(delta int) string {
	if delta < 0 {
		return fmt.Sprintf("$quadrado <= %d", quadradoMinimo)
	}
	return fmt.Sprintf("$quadrado >= %d", quadradoMaximo)
}

// estiloDoPalco é o que leva o zoom do sinal para o CSS.
//
// Vai no PALCO e não no plano porque é lá que a variável nasce, e porque o palco
// é um nó que o remendo não substitui — o zoom sobrevive a cada mudança na cena,
// que é a mesma razão de o enquadramento não estar no HTML.
const estiloDoPalco = "'--quadrado: ' + $quadrado + 'px'"

// zoomPelaRoda: só com CTRL, e a decisão é de não tirar nada de ninguém.
//
// A roda SOZINHA continua rolando o plano, que é como se percorre o mapa hoje —
// a SPA pôde tomar a roda para o zoom porque lá não há rolagem nativa para
// perder. `Ctrl+roda` é a convenção de mapa e o gesto que o navegador já ensina.
const zoomPelaRoda = "evt.ctrlKey && (evt.preventDefault(), " +
	"$quadrado = Math.min(96, Math.max(20, $quadrado + (evt.deltaY < 0 ? 8 : -8))))"

// zoomPeloTeclado: `+` e `-`, as mesmas teclas da SPA.
//
// A guarda de alvo de digitação é a mesma do atalho da barra: sem ela, digitar
// um "-" no nome de um combatente reenquadraria o tabuleiro atrás do formulário.
const zoomPeloTeclado = `!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && ` +
	`(evt.key === '+' || evt.key === '=' ? ` +
	`$quadrado = Math.min(96, $quadrado + 8) : ` +
	`evt.key === '-' ? $quadrado = Math.max(20, $quadrado - 8) : null)`

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
// quadrados cabem no palco é coisa que só o dedo sabe. O que vem do servidor é o
// ALVO em quadrados, e ele vem porque só o servidor sabe onde as peças estão.

// idDoPalco é como o botão do cabeçalho encontra a janela que rola.
//
// Um id e não um `querySelector` de classe: o botão mora FORA do palco (ele é
// ação e vive ao lado do zoom), e classe é o que alguém troca ao mexer no
// estilo — um seletor que deixa de casar não dá erro, só faz o botão parar de
// funcionar em silêncio.
const idDoPalco = "tabuleiro-palco"

// centralizaNasPecas rola o palco até o centro do que existe na cena.
//
// O alvo é o CENTRO da casa (o `+ 0.5`) menos metade da janela. Não há conta de
// limite aqui: o navegador já prende a rolagem às bordas do que existe, e pedir
// mais do que dá simplesmente para no fim.
//
// ATRIBUIÇÃO DIRETA e SEM `smooth`, e isso foi MEDIDO e não escolhido: com
// `scrollTo({behavior:'smooth'})` — e também com `scroll-behavior: smooth` no
// CSS — a rolagem não acontecia, o `scrollTop` ficava em ZERO e não havia erro em
// lugar nenhum. A mesma expressão sem a animação chegava ao lugar certo. O
// `prefers-reduced-motion` do navegador da medição dizia `no-preference`, então
// a preferência do usuário não era a causa.
//
// Um botão que anima e não chega é pior que um botão que salta.
func centralizaNasPecas(v tabuleiroView) string {
	col, lin := oCentroDaCena(v)
	return fmt.Sprintf(
		"const palco = document.getElementById(%q); "+
			"palco.scrollLeft = (%d + 0.5) * $quadrado - palco.clientWidth / 2; "+
			"palco.scrollTop = (%d + 0.5) * $quadrado - palco.clientHeight / 2",
		idDoPalco, col, lin,
	)
}

// oCentroDaCena é a caixa que contém todas as peças, em quadrados da moldura.
//
// O corpo da peça entra na conta e não só a âncora dela: uma Colossal ocupa 6×6
// (p107), e centralizar pela quina deixaria metade do dragão fora da janela.
//
// SEM PEÇA o alvo é o centro da moldura, e não a origem do plano: a moldura é o
// que está desenhado, e mandar a rolagem para um canto vazio pareceria um botão
// que não funciona.
func oCentroDaCena(v tabuleiroView) (col, lin int) {
	if len(v.Pecas) == 0 {
		return v.Colunas / 2, v.Linhas / 2
	}
	menorCol, maiorCol := v.Pecas[0].Col, v.Pecas[0].Col
	menorLin, maiorLin := v.Pecas[0].Lin, v.Pecas[0].Lin
	for _, p := range v.Pecas {
		menorCol = min(menorCol, p.Col)
		menorLin = min(menorLin, p.Lin)
		maiorCol = max(maiorCol, p.Col+p.Pegada-1)
		maiorLin = max(maiorLin, p.Lin+p.Pegada-1)
	}
	return (menorCol + maiorCol) / 2, (menorLin + maiorLin) / 2
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
