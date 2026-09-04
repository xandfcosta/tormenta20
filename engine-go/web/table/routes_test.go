package table

import (
	"regexp"
	"testing"
)

// NENHUM SINAL DA MESA É DECLARADO DUAS VEZES (ALE-291).
//
// A cena declara ~40 sinais numa expressão só, e eles vivem todos no MESMO
// documento. Dois gestos com o mesmo nome não dão erro em lugar nenhum: o
// segundo `data-signals` simplesmente vence, e o que se vê é um gesto escrevendo
// no alvo do outro — o de criar peça apagando o alvo do de salvar peça.
//
// Este guarda nasceu de um quase-acidente medido: a peça avulsa ia usar
// `pecanome` e `pecatamanho`, que JÁ eram do diálogo de editar peça (ALE-206) e
// estão quatro linhas acima na mesma função. O GLOSSARIO já registra a mesma
// forma na linha do `buscador`, que se chama assim para não colidir com o
// `busca` das cenas.
func TestNoTableSignalIsDeclaredTwice(t *testing.T) {
	nomes := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9]*)\s*:`)
	vistos := map[string]bool{}
	medidos := 0
	for _, achado := range nomes.FindAllStringSubmatch(tableSignalsExpr(), -1) {
		nome := achado[1]
		medidos++
		if vistos[nome] {
			t.Errorf("o sinal %q é declarado duas vezes — o segundo vence, e um gesto passa a escrever no alvo do outro", nome)
		}
		vistos[nome] = true
	}
	// O DENOMINADOR. Sem ele, "nenhum repetido" e "o regex não casou com nada"
	// são a mesma linha verde. Eram 40 em setembro de 2026.
	if medidos < 25 {
		t.Fatalf("só %d sinais lidos — o guarda ficou cego", medidos)
	}
}
