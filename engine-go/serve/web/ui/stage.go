package ui

import (
	"fmt"
	"strconv"
)

// A GRAMÁTICA DA CENA DE SELEÇÃO — palco, vizinhos, cursor e entrada.
//
// Duas cenas fazem a mesma pergunta ("escolha um da lista e veja o que
// escolheu"): o elenco (`web/characters`) e as campanhas (`web/campaigns`). O
// que está aqui é o que as duas têm de dizer IGUAL, porque quando elas divergem
// quem paga é o mestre, que reaprende a tela ao trocar de cena.

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
// último item.
func NeighborAt(neighbors []Neighbor, i int) *Neighbor {
	if i < 0 || i >= len(neighbors) {
		return nil
	}
	return &neighbors[i]
}

// StageWash é o facho de luz no matiz do próprio item — o que faz o palco
// parecer iluminado em vez de listado. Decorativo, e por isso o elemento que o
// usa é `aria-hidden`.
//
// @example StageWash("A Queda de Tauron") // "radial-gradient(ellipse 60% 50% …)"
func StageWash(name string) string {
	return "radial-gradient(ellipse 60% 50% at 50% 42%, oklch(0.55 0.15 " +
		strconv.Itoa(NameHue(name)) + " / 0.14), transparent 70%)"
}

// A ENTRADA DO PALCO: a CLASSE é o que substitui o mount.
//
// A cena inteira é desenhada e o cursor só alterna `data-show` — **nada nunca
// monta**, e uma animação presa ao mount não tocaria nunca. O que dispara aqui é
// a classe entrando num nó que não a tinha: o palco que sai perde a classe e o
// que entra ganha, e como são elementos DIFERENTES não existe o caso que não
// replica ("a mesma animação, já concluída, no mesmo nó"). Isso dispensa o
// morph, o reflow forçado e o id que muda a cada troca.

// EnteringStage escreve o `data-class` de um palco.
//
// Duas classes e não uma com direção por variável: `translateX(calc(var(--dir) *
// …))` dentro de `@keyframes` não é interpolado de forma confiável entre
// navegadores, e uma animação que não anima é o defeito mudo desta família.
func EnteringStage(id int64) string {
	return fmt.Sprintf(
		"{'stage-enters-forward': $cursor == %d && $direction == 1, 'stage-enters-back': $cursor == %d && $direction == -1}",
		id, id)
}

// CursorGesture é o ÚNICO escritor de `$cursor` numa cena de seleção, e é por
// isso que ele é uma função.
//
// São cinco gestos por cena que movem o cursor — o quadro no clique e no foco,
// os dois retratos vizinhos e a vaga de criar. Escrito à mão, o próximo é o que
// esquece a direção: o palco entra pelo lado errado, sem erro em lugar nenhum. O
// `TestEveryGestureThatMovesTheCursorSaysTheDirection` varre AS DUAS CENAS e
// recusa um `$cursor =` que não venha daqui.
//
// ELE É IDEMPOTENTE, e a guarda é a razão: um clique num quadro dispara
// `focusin` E `click`, os dois com este mesmo gesto. Sem ela a primeira passagem
// calcula o sentido certo e a SEGUNDA recalcula com o índice já atualizado — `N
// >= N` é sempre verdade —, e o palco entra "adiante" mesmo andando para trás.
//
// A guarda é um `if` de statements e não um ternário: sequência de comandos
// dentro de um ternário é erro de sintaxe, o Datastar engole o parse e o gesto
// INTEIRO vira nada.
//
// @example CursorGesture(2, 41) // "if ($last_index != 2) { … } $cursor = 41"
func CursorGesture(index int, id int64) string {
	return fmt.Sprintf(
		"if ($last_index != %d) { $direction = %d >= $last_index ? 1 : -1; $last_index = %d } $cursor = %d",
		index, index, index, id)
}

// StageSignals são os quatro sinais que toda cena de seleção declara.
//
// Ela existe porque `direction` e `last_index` são exigência do `CursorGesture` e não
// escolha da cena: uma cena que declarasse só `cursor` teria o gesto escrevendo
// em sinais que nascem indefinidos, e o primeiro passo do cursor entraria pelo
// lado errado — em silêncio, porque `undefined >= undefined` é `false` e o
// palco simplesmente escolhe "atrás".
//
// @example StageSignals(41) // "cursor: 41, direction: 1, last_index: 0"
func StageSignals(cursorID int64) string {
	return fmt.Sprintf("cursor: %d, direction: 1, last_index: 0", cursorID)
}

// theEnterThatOpens é o ⏎ de um marcador do trilho.
//
// Ele é função e não string escrita à mão em cada cena: o sítio que esquecer o
// `preventDefault` deixa o navegador disparar o `click` do botão JUNTO, e o
// gesto vira "escolher e abrir" quando a pessoa só queria escolher.
//
// @example theEnterThatOpens("/personagens/41")
func theEnterThatOpens(destination string) string {
	return fmt.Sprintf("evt.key === 'Enter' && (evt.preventDefault(), location.href = %q)", destination)
}
