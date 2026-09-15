package search

import "testing"

// A TOLERÂNCIA A ERRO DE DIGITAÇÃO, medida onde ela mora.
//
// O caso veio do teste do buscador junto com o pacote (ALE-278): ele chama a
// função direto e não sabe nada de tela, então esta é a camada dele. Os outros
// casos daquele arquivo pedem por HTTP e afirmam o que a pessoa vê — esses
// ficaram lá.

// TestTheNearlyEqualGapFitsInTwoLetters.
//
// PROVADO VERMELHO: a primeira versão do `noMatch` devolvia `len(letras)+1`
// quando o alvo não casava, e num resto de UMA letra isso é 2 — dentro da folga.
// O efeito medido: "abal" casava com "Naja" pelo último "a", e a busca por nome
// devolvia 282 entradas em vez de uma. Sentinela calculado a partir da entrada é
// sentinela que a entrada alcança.
func TestTheNearlyEqualGapFitsInTwoLetters(t *testing.T) {
	if !isNearlyEqual("necromante", "ncromante") {
		t.Error("uma letra pulada devia casar — é o typo que se comete de verdade")
	}
	if isNearlyEqual("naja", "abal") {
		t.Error("“abal” casou com “naja”: o buraco não está sendo contado")
	}
	if isNearlyEqual("dragao venerave", "dv") {
		t.Error("duas letras distantes casaram um nome inteiro")
	}
}
