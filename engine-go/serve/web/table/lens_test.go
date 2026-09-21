package table

import (
	"strings"
	"testing"

	"t20engine/domain/board"
)

// Os guardas de VER COMO JOGADOR (ALE-193, superfície 7 da ALE-269).
//
// A REDAÇÃO por papel tem guarda no `tabuleiro` (`BoardForRole`) e no
// `piloto_mesa_tabuleiro_test`. O que se prende aqui é o que só existe desde a
// lente: que ela usa a redação em vez de reescrevê-la, que a CONTAGEM responde a
// pergunta do mestre, que os controles continuam dele, e que ela não sobrevive
// ao fim da cena.

// onLens acende a lente do mestre e devolve a cena que ele passa a ver.
// TestTheLensCountComesFromTheDifference, e não de uma varredura por `Hidden`.
//
// A CORTINA esvazia a cena inteira sem marcar peça nenhuma como escondida: uma
// contagem por campo diria "nenhuma peça escondida" sobre um mapa que a mesa
// simplesmente não vê. Comparar os dois retratos cobre tudo o que a redação tira,
// inclusive o que ela vier a tirar depois.
func TestTheLensCountComesFromTheDifference(t *testing.T) {
	forGM := &board.BoardState{
		Curtained: true,
		Tokens: []board.BoardToken{
			{ID: "a", Label: "Taverneiro"}, {ID: "b", Label: "Ogro"},
		},
	}
	fromTable, hidden := seesTableHowScene(forGM)
	if hidden != 2 {
		t.Errorf("com a cortina fechada a lente contou %d escondidas, esperado 2", hidden)
	}
	if fromTable != nil && len(fromTable.Tokens) != 0 {
		t.Errorf("a cortina deixou %d peças na cena da mesa", len(fromTable.Tokens))
	}
}

// TestTheLensSentenceAgreesInNumber.
//
// "1 peças escondidas" é o defeito que passa por todo teste que compara com um
// `fmt.Sprintf` do mesmo jeito — o teste re-derivaria o erro. Os três casos são
// escritos por extenso.
func TestTheLensSentenceAgreesInNumber(t *testing.T) {
	cases := map[int]string{
		0: "Nenhuma peça escondida nesta cena.",
		1: "1 peça escondida não aparece.",
		3: "3 peças escondidas não aparecem.",
	}
	for howMany, want := range cases {
		if sentence := lensPhrase(howMany); !strings.HasSuffix(sentence, want) {
			t.Errorf("com %d escondidas a tira disse %q, esperado terminar em %q", howMany, sentence, want)
		}
	}
}
