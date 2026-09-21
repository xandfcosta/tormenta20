package table

import (
	"regexp"
	"strings"
	"testing"
)

// O GESTO DE UMA PEÇA NÃO PODE RESPONDER POR OUTRA.
//
// Cada peça pendura o próprio `pointermove__window` e `pointerup__window`, e a
// janela entrega o evento a TODOS eles. A única coisa que separa um gesto do
// outro é o valor de `$dragging` — e ele guardava o literal `'peca'`, igual
// para todas. No rascunho com duas peças, os dois `pointerup` passavam na mesma
// guarda: o PRIMEIRO DO DOM vencia, zerava o sinal, e o segundo achava o gesto
// já encerrado. Pegar o Beta movia o Alfa, que corria atrás do dedo desde o
// primeiro quadro.
//
// Com UMA peça o primeiro do DOM É o arrastado, e o defeito não aparece: um
// guarda com `toHaveCount(1)` mede a metade em que ele é invisível por
// construção. Percorrer a navegação não é cobertura quando a tela ramifica pelo
// NÚMERO de itens.
//
// ESTE GUARDA VARRE OS QUATRO PEDAÇOS DO GESTO — pegar, seguir, soltar e
// deslocar na tela — porque eles têm de concordar, e já divergiram: o
// `tokenStyling` perguntava `v.ArrastaAPeca == p.ID` enquanto o `takeToken`
// perguntava `v.Rascunho || v.ArrastaAPeca == id`. Um pedaço novo que volte a
// escrever um literal compartilhado nasce vermelho aqui.
var dragComparison = regexp.MustCompile(`\$dragging\s*(?:===|!==|=)\s*'([^']*)'`)

func tokenGesturePieces(v BoardView, p boardToken) map[string]string {
	return map[string]string{
		"pointerdown":            takeToken(v, p.ID),
		"pointermove__window":    followToken(v, p),
		"pointerup__window":      dropToken(v, p),
		"data-class (o deslize)": tokenStyling(p.ID, dragsItself(v, p.ID)),
	}
}

func TestNoTokenGestureAnswersForAnotherToken(t *testing.T) {
	tokens := []boardToken{
		{ID: "alfa-1111", Label: "Alfa", X: 3, Y: 3, Where: "3, 3"},
		{ID: "beta-2222", Label: "Beta", X: 8, Y: 3, Where: "8, 3"},
		{ID: "gama-3333", Label: "Gama", X: 8, Y: 9, Where: "8, 9"},
	}
	// AS DUAS SUPERFÍCIES, e uma por vez não basta: o rascunho dá o gesto a TODAS
	// as peças, e a mesa dá a uma só e o do grupo às outras. O defeito morava no
	// primeiro e o segundo é onde ele seria mais caro.
	//
	// Na mesa o alvo é a peça do MEIO de propósito. Com o alvo em primeiro lugar,
	// a ordem do DOM esconde a mistura sozinha — que é exatamente por que a mesa
	// nunca acusou nada.
	boards := map[string]BoardView{
		"rascunho":                 {Tokens: tokens, GM: true, Draft: true, Base: "/campanhas/1/lugares/2/tabuleiro"},
		"mesa, alvo no meio":       {Tokens: tokens, GM: true, DragsToken: "beta-2222", Base: "/campanhas/1/sessoes/2/tabuleiro"},
		"mesa, ninguém pode mover": {Tokens: tokens, GM: true, Base: "/campanhas/1/sessoes/2/tabuleiro"},
	}

	measured := 0
	for scene, v := range boards {
		for _, p := range v.Tokens {
			for where, expression := range tokenGesturePieces(v, p) {
				measured++
				for _, m := range dragComparison.FindAllStringSubmatch(expression, -1) {
					value := m[1]
					// Vazio é o FIM do gesto ("$dragging = ''"), e vale para todo mundo.
					if value == "" || value == p.ID || value == dragsTheParty {
						continue
					}
					t.Errorf("[%s] o %s da peça %s reconhece `$dragging === %q`, que não é o id dela nem %q.\n"+
						"Um valor compartilhado faz o gesto de uma peça responder pelas outras: a janela entrega o "+
						"evento a todas, e quem vence é a ORDEM DO DOM.\n  expressão: %s",
						scene, where, p.ID, value, dragsTheParty, expression)
				}
			}
		}
	}

	// O DENOMINADOR, porque uma lista de falhas vazia e um laço que não rodou se
	// parecem no terminal: três cenas × três peças × quatro pedaços.
	if measured != 36 {
		t.Fatalf("o guarda mediu %d expressões, e não 36 — a montagem das cenas é o primeiro suspeito", measured)
	}
}

// A metade POSITIVA, e ela não é decoração: o guarda de cima passa verde sobre
// uma expressão que não mencione `$dragging` nenhuma vez — um gesto morto e um
// gesto correto se parecem numa lista de violações vazia.
func TestEveryTokenThatDragsItselfIsGuardedByItsOwnId(t *testing.T) {
	tokens := []boardToken{
		{ID: "alfa-1111", Label: "Alfa", X: 3, Y: 3, Where: "3, 3"},
		{ID: "beta-2222", Label: "Beta", X: 8, Y: 3, Where: "8, 3"},
	}
	v := BoardView{Tokens: tokens, GM: true, Draft: true, Base: "/campanhas/1/lugares/2/tabuleiro"}
	for _, p := range v.Tokens {
		for where, expression := range tokenGesturePieces(v, p) {
			if !strings.Contains(expression, "'"+p.ID+"'") {
				t.Errorf("o %s da peça %s não cita o id dela em lugar nenhum: %s", where, p.ID, expression)
			}
		}
	}
}
