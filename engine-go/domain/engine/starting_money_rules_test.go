package engine

import (
	"strings"
	"testing"
)

// TestTheStartingPurseFollowsTable31 — Tabela 3-1: Dinheiro Inicial (p140).
//
// Os números são escritos na mão, colhidos do livro: as três pontas da tabela
// mais uma linha do meio de cada coluna impressa.
func TestTheStartingPurseFollowsTable31(t *testing.T) {
	cases := []struct {
		level int
		tibar int
	}{
		{2, 300},      // a primeira linha com valor
		{5, 2_000},    // meio da coluna da esquerda
		{10, 13_000},  // fim da coluna da esquerda
		{11, 19_000},  // começo da coluna da direita
		{20, 260_000}, // a última linha
	}
	for _, tc := range cases {
		tibar, err := StartingMoneyForLevel(tc.level)
		if err != nil {
			t.Errorf("nível %d: %v", tc.level, err)
			continue
		}
		if tibar != tc.tibar {
			t.Errorf("nível %d: T$ %d, esperado T$ %d", tc.level, tibar, tc.tibar)
		}
	}
}

// TestTheFirstLevelHasNoTableValue: a linha "1º 4d6" é a única sem número,
// e pedi-la em vez de rolar tem de doer.
func TestTheFirstLevelHasNoTableValue(t *testing.T) {
	_, err := StartingMoneyForLevel(1)
	if err == nil {
		t.Fatal("o 1º nível devolveu valor de tabela; ele rola 4d6")
	}
	if !strings.Contains(err.Error(), "4d6") {
		t.Errorf("a mensagem não diz o que fazer no lugar: %q", err)
	}
}

// TestALevelOutsideTheTableNamesTheOffendingValue: 0 e 21 não existem no livro, e a
// mensagem carrega o número que chegou.
func TestALevelOutsideTheTableNamesTheOffendingValue(t *testing.T) {
	for _, level := range []int{0, 21, -3} {
		_, err := StartingMoneyForLevel(level)
		if err == nil {
			t.Errorf("nível %d passou pela Tabela 3-1", level)
			continue
		}
		if !strings.Contains(err.Error(), "1 a 20") {
			t.Errorf("nível %d: a mensagem não diz a faixa esperada: %q", level, err)
		}
	}
}

// TestThePurseRollFitsInFourD6: 4d6 vai de 4 a 24, e cada ponta é
// alcançável. Roda o bastante para que uma soma de três dados (mínimo 3) ou de
// cinco (máximo 30) apareça.
func TestThePurseRollFitsInFourD6(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 400; i++ {
		tibar, err := RollStartingMoney()
		if err != nil {
			t.Fatalf("rolar: %v", err)
		}
		if tibar < 4 || tibar > 24 {
			t.Fatalf("T$ %d fora de 4d6 (4 a 24)", tibar)
		}
		seen[tibar] = true
	}
	// O controle: 400 rolagens de 4d6 sem variação nenhuma seria um dado preso,
	// e um dado preso passaria na asserção de faixa acima sem reclamar.
	if len(seen) < 5 {
		t.Fatalf("400 rolagens deram só %d resultados distintos — o dado está preso", len(seen))
	}
}
