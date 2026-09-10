package ui

import (
	"fmt"
	"strconv"
)

// A GRAMÁTICA DA CENA DE SELEÇÃO — palco, vizinhos, cursor e entrada (ALE-297).
//
// Duas cenas fazem a mesma pergunta ("escolha um da lista e veja o que
// escolheu"): o elenco (`web/characters`) e as campanhas (`web/campaigns`). O
// que está aqui é o que as duas têm de dizer IGUAL, porque quando elas divergem
// quem paga é o mestre, que reaprende a tela ao trocar de cena.
//
// Isto morava dentro de `characters/view.go`, privado, e mudou de casa quando a
// segunda cena precisou. É a lição escrita no `CLAUDE.md` com todas as letras:
// **instrumento que mora dentro de um chamador tem exatamente um chamador** — e
// isso não aparece em revisão de diff nenhuma, porque não há linha errada, só
// uma linha que não pôde ser escrita.

// Neighbor é o retrato APAGADO que ladeia o palco — quem vem antes e quem vem
// depois no trilho.
//
// Ele carrega só o que o vizinho desenha, e não o cartão inteiro da cena: o
// vizinho não mostra vitais, nem sinopse, nem ações, e passar a estrutura cheia
// convidaria a próxima pessoa a mostrar.
type Neighbor struct {
	ID       int64
	Name     string
	Monogram string
	Gradient string
	// Index é a posição dele no trilho, e ela existe para o gesto saber o
	// SENTIDO do movimento: clicar no vizinho da esquerda anda para trás, e é
	// isso que faz o palco entrar pelo lado certo. Sem o índice, o clique no
	// retrato vizinho seria o único gesto da cena sem direção.
	Index int
}

// NeighborAt é o vizinho na posição `i` do trilho, ou nil quando `i` cai fora.
//
// Nil NÃO significa "não desenhe": significa ESPAÇADOR. Sem a caixa vazia nas
// pontas da lista o palco escorrega para o lado ao chegar no primeiro ou no
// último item, que é a família de defeitos da ALE-99 — a mesma razão do
// `min-h-[2lh]` no nome e do travessão na Defesa.
func NeighborAt(vizinhos []Neighbor, i int) *Neighbor {
	if i < 0 || i >= len(vizinhos) {
		return nil
	}
	return &vizinhos[i]
}

// StageWash é o facho de luz no matiz do próprio item — o que faz o palco
// parecer iluminado em vez de listado. Decorativo, e por isso o elemento que o
// usa é `aria-hidden`.
//
// @example StageWash("A Queda de Tauron") // "radial-gradient(ellipse 60% 50% …)"
func StageWash(nome string) string {
	return "radial-gradient(ellipse 60% 50% at 50% 42%, oklch(0.55 0.15 " +
		strconv.Itoa(NameHue(nome)) + " / 0.14), transparent 70%)"
}

// A ENTRADA DO PALCO (ALE-235, entregue na ALE-239): a classe que substitui o
// mount.
//
// Na SPA a animação era `animate-in`, que dispara no mount — e o `<Show keyed>`
// reconstruía o nó a cada troca justamente para ela disparar (ALE-97). Aqui a
// cena inteira é desenhada e o cursor só alterna `data-show`: **nada nunca
// monta**, e uma animação presa ao mount não tocaria nunca.
//
// O que substitui o mount é a CLASSE entrando num nó que não a tinha. O palco
// que sai perde a classe e o que entra ganha — são elementos DIFERENTES, então
// não existe o caso que não replica ("a mesma animação, já concluída, no mesmo
// nó"). Isso dispensa o morph, o reflow forçado e o id que muda a cada troca,
// que eram as saídas que a issue previa.
//
// Que ele é GERAL e não um remendo do elenco é o que a ALE-297 provou: a cena de
// campanhas herdou a entrada inteira sem uma linha nova de CSS.

// EnteringStage escreve o `data-class` de um palco.
//
// Duas classes e não uma com direção por variável: `translateX(calc(var(--dir) *
// …))` dentro de `@keyframes` não é interpolado de forma confiável entre
// navegadores, e uma animação que não anima é o defeito mudo desta família.
func EnteringStage(id int64) string {
	return fmt.Sprintf(
		"{'palco-entra-adiante': $cursor == %d && $direction == 1, 'palco-entra-atras': $cursor == %d && $direction == -1}",
		id, id)
}

// CursorGesture é o ÚNICO escritor de `$cursor` numa cena de seleção, e é por
// isso que ele é uma função (ALE-235).
//
// São CINCO gestos que movem o cursor no elenco — o quadro do filme no clique e
// no foco, os dois retratos vizinhos e a vaga de criar — e o mesmo tanto em
// campanhas. Escrito à mão cinco vezes, o sexto é o que esquece: o palco
// entraria pelo lado errado, sem erro em lugar nenhum, e só quem conhece a
// animação notaria. O `TestEveryGestureThatMovesTheCursorSaysTheDirection`
// varre AS DUAS CENAS e recusa um `$cursor =` que não venha daqui.
//
// # Ele é IDEMPOTENTE, e isso foi medido no navegador
//
// Um clique num quadro do filme dispara `focusin` E `click`, os dois com este
// mesmo gesto. Sem a guarda, a primeira passagem calcula o sentido certo e
// escreve o índice; a SEGUNDA recalcula com o índice já atualizado — `N >= N` é
// sempre verdade — e o palco entra "adiante" mesmo andando para trás. O sintoma
// é uma animação na direção errada, que ninguém lê como defeito de lógica.
//
// A guarda é um `if` de statements e não um ternário: ver a armadilha do
// Datastar no guia do pacote — sequência de comandos dentro de um ternário é
// erro de sintaxe, o framework engole o parse e o gesto INTEIRO vira nada.
//
// @example CursorGesture(2, 41) // "if ($last_index != 2) { … } $cursor = 41"
func CursorGesture(index int, id int64) string {
	return fmt.Sprintf(
		"if ($last_index != %d) { $direction = %d >= $last_index ? 1 : -1; $last_index = %d } $cursor = %d",
		index, index, index, id)
}

// StageSignals são os quatro sinais que toda cena de seleção declara.
//
// Ela existe porque `sentido` e `indice` são exigência do `CursorGesture` e não
// escolha da cena: uma cena que declarasse só `cursor` teria o gesto escrevendo
// em sinais que nascem indefinidos, e o primeiro passo do cursor entraria pelo
// lado errado — em silêncio, porque `undefined >= undefined` é `false` e o
// palco simplesmente escolhe "atrás".
//
// @example StageSignals(41) // "cursor: 41, sentido: 1, indice: 0"
func StageSignals(cursorID int64) string {
	return fmt.Sprintf("cursor: %d, direction: 1, last_index: 0", cursorID)
}

// theEnterThatOpens é o ⏎ de um marcador do trilho.
//
// Ele é função e não string escrita à mão em cada cena pelo motivo de sempre
// nesta família: são quatro sítios hoje (dois por cena de seleção), e o quinto é
// o que esquece o `preventDefault` — sem ele o navegador dispara o `click` do
// botão JUNTO, e o gesto vira "escolher e abrir" quando a pessoa só queria
// escolher.
//
// @example theEnterThatOpens("/personagens/41")
func theEnterThatOpens(destino string) string {
	return fmt.Sprintf("evt.key === 'Enter' && (evt.preventDefault(), location.href = %q)", destino)
}
