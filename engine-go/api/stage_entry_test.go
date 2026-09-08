package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Os guardas da ENTRADA DO PALCO (ALE-235).
//
// A animação em si é do navegador e só o e2e a vê — linha do tempo, `animationstart`
// e duração não existem fora dele. O que se prende AQUI é o que o servidor
// escreve: as classes que substituem o mount e o gesto que diz o sentido. É a
// divisão de sempre — a regra na camada mais barata que a segura.

// aCenaComTresHerois monta a cena CHEIA, e três não é número redondo escolhido
// por gosto: com um herói só não há vizinho nenhum (as duas pontas viram
// espaçador), e é justamente o clique no retrato vizinho que o guarda precisa
// visitar. Com três, o do meio tem os dois.
func aCenaComTresHerois(t *testing.T) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	seedCharacterAtLevel(t, f.s, f.jogador, "Anã Clériga", 4, 20, 20, 6, 6)
	seedCharacterAtLevel(t, f.s, f.jogador, "Elfo Ladino", 2, 14, 14, 0, 0)
	return f, f.jogador
}

// TODO GESTO QUE MOVE O CURSOR DIZ O SENTIDO — a varredura da convenção.
//
// São CINCO gestos nesta cena (o quadro do filme no clique e no foco, os dois
// retratos vizinhos e a vaga de criar), e cada um precisa escrever o sentido e o
// índice junto do cursor. Escrito à mão cinco vezes, o sexto é o que esquece: o
// palco entraria pelo lado errado, sem erro em lugar nenhum, e só quem conhece a
// animação notaria.
//
// O guarda falha com o TRECHO ofensor, que é a diferença entre "conserte isto" e
// "procure".
func TestEveryGestureThatMovesTheCursorSaysTheDirection(t *testing.T) {
	f, quem := aCenaComTresHerois(t)
	tela := f.pede(t, quem, http.MethodGet, "/personagens", "").Body.String()

	// O CONTROLE vem primeiro: sem ele, "não achei escritor solto" é
	// indistinguível de "não achei escritor nenhum" — e as duas passam verde.
	// Cinco gestos, e a cena tem mais de um herói no seed.
	escritores := regexp.MustCompile(`\$cursor = \d+`).FindAllString(tela, -1)
	if len(escritores) < 5 {
		t.Fatalf("a cena só tem %d gestos que movem o cursor: o canal não está aberto, e a ausência abaixo não seria evidência", len(escritores))
	}

	// Todo `$cursor =` tem de vir precedido da guarda que escreve o sentido. A
	// expressão inteira é `if ($indice != N) { … } $cursor = ID`, então basta
	// olhar o que vem ANTES na mesma expressão.
	for _, atributo := range regexp.MustCompile(`data-on:(?:click|focusin)="([^"]*\$cursor = \d+[^"]*)"`).FindAllStringSubmatch(tela, -1) {
		gesto := atributo[1]
		if !strings.Contains(gesto, "$sentido") || !strings.Contains(gesto, "$indice") {
			t.Errorf("um gesto move o cursor sem dizer o sentido: %q — o palco entraria pelo lado errado, em silêncio", gesto)
		}
	}
}

func TestTheStageHasTheTwoPartsThatAnimate(t *testing.T) {
	f, quem := aCenaComTresHerois(t)
	tela := f.pede(t, quem, http.MethodGet, "/personagens", "").Body.String()

	for _, parte := range []string{"palco-retrato", "palco-placa"} {
		if !strings.Contains(tela, parte) {
			t.Errorf("a cena não tem %q: a animação de entrada ficaria sem alvo", parte)
		}
	}
	// E a classe que ENTRA é escrita por `data-class`, não pelo `class` fixo: no
	// `class` ela nasceria em todos os palcos ao mesmo tempo, e a animação
	// tocaria uma vez só, na carga.
	if !strings.Contains(tela, "palco-entra-adiante") || !strings.Contains(tela, "palco-entra-atras") {
		t.Error("a cena não escreve as duas direções da entrada")
	}
	if strings.Contains(tela, `class="relative flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-2 palco-entra`) {
		t.Error("a classe de entrada foi para o `class` fixo: ela tocaria na carga e nunca mais")
	}
}
