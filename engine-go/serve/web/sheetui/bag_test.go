package sheetui

import (
	"strings"
)

// Os guardas da MOCHILA (ALE-272, fatia 7).
//
// O que eles prendem é a REGRA e a DECISÃO de tela: os dois tetos da p141, o
// eixo de equipar, a conta do dinheiro, e o que a tira faz com um item que já
// está fora do teto. Os números da carga não são reafirmados aqui — eles vêm do
// motor e têm guarda de regra própria em `engine/load_rules_test.go`.

// screenSaved é a aba desenhada.
// screenSaved recorta só a GRADE, e o recorte não é preciosismo: o nome de
// um item aparece também nos crachás da tira e em CADA diálogo de ficha de
// item, que são desenhados logo depois do painel. Procurar na tela inteira
// acharia justamente o que o filtro escondeu, e o teste passaria dizendo o
// contrário do que mede.
//
// O corte vai da grade até o fim da SEÇÃO, que é onde o painel acaba e os
// diálogos começam.
func screenSaved(screen string) string {
	start := strings.Index(screen, "grid-cols-3")
	if start < 0 {
		return ""
	}
	end := strings.Index(screen[start:], "</section>")
	if end < 0 {
		return screen[start:]
	}
	return screen[start : start+end]
}
