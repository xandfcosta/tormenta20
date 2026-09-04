package sheetui

import (
	"fmt"
	"math"

	"t20engine/sheet"
)

// O DINHEIRO da ficha (ALE-272, fatia 7).
//
// São três gestos diferentes na mesa e um campo só no banco: "achamos 350 no
// baú", "paguei 80 pela estalagem", e escrever o total — que é o gesto da forja
// (Tabela 3-1, p140) e o de consertar um erro de digitação (ALE-224).

// ── o DINHEIRO, e a conta que ele exige ──────────────────────────────────────

// afterGestureBalance resolve o saldo e a recusa dos três modos (ALE-224).
//
// # Arredondar em DUAS CASAS não é enfeite
//
// O dinheiro do livro é fracionário — uma vela custa T$ 0,1 — e soma binária de
// décimos não fecha: 1200,3 − 80,1 dá 1120,1999999999998 em ponto flutuante, e
// esse número iria para o banco e para a tela. Duas casas é a mesma precisão que
// a tela mostra, então o que se lê e o que se grava passam a ser o mesmo número.
//
// # O piso é ZERO, e não um aviso
//
// Dívida na ficha viraria carga de moeda NEGATIVA, que COMPRARIA espaço na
// mochila em vez de ocupar (ALE-215).
func afterGestureBalance(saldo float64, modo string, valor float64) (float64, string) {
	if valor < 0 {
		return 0, "informe um valor a partir de 0"
	}
	depois := saldo
	switch modo {
	case "receber":
		depois = emDuasCasas(saldo + valor)
	case "gastar":
		depois = emDuasCasas(saldo - valor)
	case "corrigir":
		depois = emDuasCasas(valor)
	default:
		return 0, fmt.Sprintf("%q não é um jeito de mexer no dinheiro", modo)
	}
	if depois < 0 {
		return 0, "não dá para gastar T$ " + virgulaCom(valor) + ": você tem T$ " + virgulaCom(saldo) + "."
	}
	if depois > sheet.MaxTibar {
		return 0, "T$ " + virgulaCom(depois) + " passa do limite de T$ " + virgulaCom(sheet.MaxTibar) + " da ficha."
	}
	return depois, ""
}

func emDuasCasas(valor float64) float64 {
	return math.Round(valor*100) / 100
}
