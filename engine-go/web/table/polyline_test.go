package table

import (
	"strings"
	"testing"

	"t20engine/engine"
)

// Os guardas da RÉGUA DE VÁRIAS PARADAS (ALE-203, item 5 do dono).
//
// "A régua não permite calcular distâncias com mais de uma parada. Ao montar uma
// régua, não tem como apagá-la sem trocar de ferramenta."
//
// A aritmética de UMA perna não é medida aqui: `engine.Measure` tem guarda de
// regra próprio, escrito contra a p224. O que se prende deste lado é a
// composição — o total é a soma das pernas —, o formato do rótulo, e a família de
// armadilha que esta superfície descobriu no Datastar.

// TestTheTotalIsTheSumOfTheLegs.
//
// A polilinha mede um CAMINHO, e o caminho é a soma dos trechos. Com DUAS
// paradas ela é uma reta e o total é a régua de sempre, que é o caso que não
// pode ter mudado.
func TestTheTotalIsTheSumOfTheLegs(t *testing.T) {
	reta := polylineReading([]engine.Square{{}, {X: 3}})
	if reta["reguatexto"] != rulerReading(engine.Measure(engine.Square{}, engine.Square{X: 3})) {
		t.Errorf("a régua de duas paradas deixou de ser a régua de sempre: %q", reta["reguatexto"])
	}

	// Três paradas: 3 + 4 = 7 quadrados de caminho.
	caminho := polylineReading([]engine.Square{{}, {X: 3}, {X: 3, Y: 4}})
	if !strings.HasPrefix(caminho["reguatexto"].(string), "7 quadrados") {
		t.Errorf("o total de 3+4 pernas saiu %q, esperado 7 quadrados", caminho["reguatexto"])
	}
	if rotulos := caminho["reguarotulos"].([]string); len(rotulos) != 2 {
		t.Errorf("três paradas deram %d rótulos, esperado 2 pernas: %v", len(rotulos), rotulos)
	}
}

// TestTheLegLabelComesInMetres — pedido do dono, e ele é sobre a UNIDADE.
//
// Sobre a linha cabe uma unidade só, e a que a mesa fala em voz alta é o metro:
// "ele está a nove metros" é a frase do turno, e "seis quadrados" obriga a
// converter de cabeça. A frase do TOTAL continua trazendo as duas, porque lá cabe.
func TestTheLegLabelComesInMetres(t *testing.T) {
	leitura := polylineReading([]engine.Square{{}, {X: 6}})
	rotulos := leitura["reguarotulos"].([]string)
	if len(rotulos) != 1 || rotulos[0] != "9,0m" {
		t.Errorf("a perna de 6 quadrados saiu %v, esperado [\"9,0m\"] — 6 × 1,5m (p236)", rotulos)
	}
	// E o TOTAL continua trazendo as duas unidades e a faixa.
	total := leitura["reguatexto"].(string)
	for _, pedaco := range []string{"6 quadrados", "9,0m", "alcance"} {
		if !strings.Contains(total, pedaco) {
			t.Errorf("a frase do total perdeu %q: %q", pedaco, total)
		}
	}
}

// TestAZeroLengthLegHasNoLabel.
//
// O instante logo depois de um clique tem a MIRA em cima da parada que acabou de
// nascer. Um "0,0m" piscando sob o dedo é ruído sobre o gesto que a pessoa está
// fazendo, e o vazio é o que apaga o nó (ver `existsLabel`).
func TestAZeroLengthLegHasNoLabel(t *testing.T) {
	leitura := polylineReading([]engine.Square{{X: 4, Y: 4}, {X: 4, Y: 4}})
	if rotulos := leitura["reguarotulos"].([]string); len(rotulos) != 1 || rotulos[0] != "" {
		t.Errorf("a perna de zero saiu %v, esperado um rótulo VAZIO", rotulos)
	}
}
