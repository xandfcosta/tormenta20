package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/board"
)

func TestOnlyTheGmMarksAGroup(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	rec := f.pede(t, f.player, http.MethodPost, f.tableUrl()+"/tabuleiro/marcar-area", `{"from":{"X":0,"Y":0},"to":{"X":9,"Y":9}}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador marcou um grupo e recebeu %d, esperado 403", rec.Code)
	}
	// O CONTROLE: o mestre PODE. Sem ele, um 403 para todo mundo passaria igual.
	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/tabuleiro/marcar-area", `{"from":{"X":0,"Y":0},"to":{"X":9,"Y":9}}`); rec.Code != http.StatusOK {
		t.Errorf("o mestre não conseguiu marcar: %d", rec.Code)
	}
}

// O PREDICADO, que é a regra inteira deste gesto.
//
// O laço decide QUAIS peças o arrasto do grupo vai mover. Os casos que olham
// 403-vs-200 e o TAMANHO da resposta ficam verdes com o handler sabotado para
// ignorar os dois cantos e marcar TODAS as peças: arranjar o resultado por ordem
// de chamada diz o que acontece *com* as linhas achadas e nada sobre *quais*
// linhas são essas.
//
// As três peças são o mínimo que distingue: uma DENTRO, uma FORA pelo eixo x e
// uma FORA pelo eixo y. Com uma fora só, um predicado que testasse um eixo e
// esquecesse o outro passaria em metade dos casos.
func TestTheLassoMarksOnlyWhatIsInsideIt(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	inside := f.seedToken(t, "Goblin de dentro", 3, 3)
	outsideX := f.seedToken(t, "Goblin à direita", 9, 3)
	outsideY := f.seedToken(t, "Goblin abaixo", 3, 9)

	// O laço vai de (2,2) a (5,5): pega o (3,3) e deixa os dois vizinhos fora.
	response := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/marcar-area",
		`{"from":{"X":2,"Y":2},"to":{"X":5,"Y":5}}`)

	if !strings.Contains(response, inside) {
		t.Errorf("o laço de (2,2) a (5,5) não pegou a peça em (3,3):\n%s", response)
	}
	for name, fromOutside := range map[string]string{"a de (9,3)": outsideX, "a de (3,9)": outsideY} {
		if strings.Contains(response, fromOutside) {
			t.Errorf("o laço de (2,2) a (5,5) marcou %s, que está fora dele:\n%s", name, response)
		}
	}
}

// seedToken põe uma peça no tabuleiro e devolve o ID que o SERVIDOR deu a ela.
//
// O id não é escolha de quem semeia: o `AddToken` atribui um UUID e ignora o
// campo. Um caso que comparasse o `marked_tokens` com o id que ele mesmo passou
// afirmaria sobre uma peça que não existe — e, como `strings.Contains` de um
// nome inventado é sempre falso, ele acusaria "o laço não pegou" para SEMPRE.
func (f sceneFixture) seedToken(t *testing.T, label string, x, y int) string {
	t.Helper()
	state, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: label, X: x, Y: y})
	if err != nil {
		t.Fatalf("pôr a peça %q: %v", label, err)
	}
	for _, token := range state.Tokens {
		if token.Label == label {
			return token.ID
		}
	}
	t.Fatalf("a peça %q não entrou no tabuleiro", label)
	return ""
}

// Arrastar da direita para a esquerda é o mesmo laço.
//
// O canto onde o dedo DESCEU vira `from`, e o de cima-à-esquerda não é sempre
// ele. Um predicado escrito como `de.X <= p.X && p.X <= ate.X` marca ZERO peças
// no arrasto invertido — e não estoura, não recusa, não diz nada: a barra
// simplesmente não aparece.
func TestTheLassoReadsTheCornersInAnyOrder(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	inside := f.seedToken(t, "Goblin de dentro", 3, 3)

	inverted := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/marcar-area",
		`{"from":{"X":5,"Y":5},"to":{"X":2,"Y":2}}`)
	if !strings.Contains(inverted, inside) {
		t.Errorf("o laço arrastado de (5,5) para (2,2) não pegou a peça em (3,3):\n%s", inverted)
	}
}

// Marcar não muda a cena de ninguém, e a resposta tem de ser do tamanho disso.
// Uma marcação que devolvesse as regiões trocaria o mapa debaixo de quem está
// arrastando — que é exatamente o gesto que acabou de acontecer.
func TestMarkingDoesNotPatchTheScene(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	response := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/marcar-area", "{}")
	if !strings.Contains(response, "marked_tokens") {
		t.Fatalf("a marcação não voltou: %s", response)
	}
	if strings.Contains(response, "datastar-patch-elements") {
		t.Errorf("marcar remendou a cena:\n%s", response)
	}
}

// O gesto que não tem sobre o que agir diz isso, em vez de gravar uma versão
// nova sem mudar nada.
func TestAGroupWithNoMarkedTokenRefusesWithASentence(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	body := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/grupo/mover", `{"delta":{"X":1,"Y":1},"marked_tokens":""}`)
	if !strings.Contains(body, "não há peça marcada") {
		t.Errorf("mover um grupo vazio não foi recusado com frase: %q", body[max(0, len(body)-200):])
	}
}

// Duas afirmações: as peças andam pelo delta, e a resposta é a do gesto contínuo
// (só o mapa). A segunda importa porque mover um grupo é um arrasto, e devolver
// a Mesa inteira no meio dele troca o elemento debaixo do dedo.
func TestTheGroupMovesThemAllInOneResponse(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	sheet, _ := sceneIds(t, f)
	f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+sheet+`"}`)

	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if len(b.Tokens) == 0 {
		t.Fatal("a peça não entrou no mapa — o guarda mediria o vazio")
	}
	before := b.Tokens[0]
	body := f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/grupo/mover",
		`{"delta":{"X":3,"Y":-2},"marked_tokens":"`+before.ID+`"}`)

	b = boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if b.Tokens[0].X != before.X+3 || b.Tokens[0].Y != before.Y-2 {
		t.Errorf("a peça de %s foi para (%d,%d), esperado (%d,%d)",
			sheet, b.Tokens[0].X, b.Tokens[0].Y, before.X+3, before.Y-2)
	}
	if strings.Contains(body, `id="table-archive"`) {
		t.Error("mover o grupo devolveu a Mesa inteira no meio de um arrasto")
	}
}

// UMA camada de repouso e não duas: com as duas se mostrando pela mesma
// condição, a que vem DEPOIS no DOM cobre a outra — o dedo nunca chega ao laço,
// e o gesto simplesmente não acontece.
//
// O `engoleoclique` entra na lista pela mesma razão: o navegador dispara `click`
// depois de um `pointerdown` + `pointerup` no mesmo elemento INCLUSIVE quando o
// dedo andou, e sem ele terminar um laço também moveria a peça da vez para onde
// o laço terminou.
func TestTheRestingLayerServesBothGestures(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	// COM PEÇA no mapa: a marca do grupo é vestida pela peça, e num tabuleiro
	// vazio a classe não aparece — o guarda acusaria a ausência dela sobre uma
	// cena que só não tem peça nenhuma.
	sheet, _ := sceneIds(t, f)
	f.posta(t, f.gm, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+sheet+`"}`)
	screen := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O valor é CONSTANTE no `.templ`, então ele sai LITERAL no HTML — só o
	// dinâmico é escapado. Procurar a forma escapada acha zero e acusa
	// "0 camadas" sobre uma cena correta.
	if howMany := strings.Count(screen, `data-show="$tool === ''"`); howMany != 1 {
		t.Errorf("há %d camadas de repouso; com mais de uma a de baixo nunca recebe o dedo", howMany)
	}
	for _, chunk := range []string{"marcar-area", "swallow_click", "board-token-marked"} {
		if !strings.Contains(screen, chunk) {
			t.Errorf("a cena não tem %q: a seleção em área não acontece", chunk)
		}
	}
}
