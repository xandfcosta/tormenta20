package table

import (
	"testing"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// Os guardas do GESTO CONTÍNUO do pincel (ALE-203, itens 8 e 9 do dono).
//
// A aritmética do traço não é medida aqui: `tabuleiro.StrokeSquares` tem guarda
// próprio, e ele prende a regra ("o traço não tem buraco") no lugar mais barato.
// O que se prende deste lado é o que só existe deste lado — que a rota pinta o
// SEGMENTO numa gravação só, que a resposta não devolve a Mesa inteira, e que a
// tela liga os gestos que fazem o traço acontecer.

// TestTheStrokePaintsTheWholeSegment.
//
// O caso mede o SEGMENTO, e não o número de gravações. Eu tinha escrito uma
// segunda asserção sobre a `Version` do tabuleiro — "uma versão a mais, senão a
// mesa recebe um quadro por casa" — e ela media a coisa errada: o `PaintTerrain`
// sobe a versão POR CASA, então um traço de dez casas sobe dez, dentro de um
// `apply` só. Quem garante a gravação única é a estrutura (`PaintStroke` chama
// `apply` uma vez, e o `boardCommand` publica uma vez), não um contador —
// e um teste que afirma o contrário fica vermelho sobre um app correto.
func contem(casas []engine.Square, alvo engine.Square) bool {
	for _, c := range casas {
		if c == alvo {
			return true
		}
	}
	return false
}

// TestEveryKindHasADrawing.
//
// O guarda que paga o preço de o desenho morar fora do domínio: `drawing`
// entra em pânico numa espécie sem entrada, e este caso faz o pânico acontecer
// na suíte em vez de na mesa. Sem ele, a quinta espécie nasceria com uma casa
// que não se distingue de nenhuma outra — indistinguível de "o pincel não
// funcionou", que é a família de defeito que esta issue inteira persegue.
//
// E ele afirma também que os CANTOS são distintos: duas espécies no mesmo canto
// desenham uma por cima da outra, e a casa com folhagens (difícil E camuflagem,
// p267) mostraria uma só.
func TestEveryKindHasADrawing(t *testing.T) {
	cantos := map[string]string{}
	for _, pincel := range tabuleiro.TerrainKinds {
		d := drawing(pincel.ID)
		if d.Icone == "" || d.Canto == "" {
			t.Errorf("a espécie %q tem desenho incompleto: %+v", pincel.ID, d)
		}
		if dono, tem := cantos[d.Canto]; tem {
			t.Errorf("o canto %q é de %q e de %q — uma desenha por cima da outra",
				d.Canto, dono, pincel.ID)
		}
		cantos[d.Canto] = string(pincel.ID)
	}
}

// primeirosAtributos encurta a tag para a mensagem caber na saída do teste.
func primeirosAtributos(tag string) string {
	if len(tag) > 160 {
		return tag[:160] + "…"
	}
	return tag
}
