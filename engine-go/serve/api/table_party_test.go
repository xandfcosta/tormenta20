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

	rec := f.pede(t, f.jogador, http.MethodPost, f.tableUrl()+"/tabuleiro/marcar-area", `{"from":{"X":0,"Y":0},"to":{"X":9,"Y":9}}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador marcou um grupo e recebeu %d, esperado 403", rec.Code)
	}
	// O CONTROLE: o mestre PODE. Sem ele, um 403 para todo mundo passaria igual.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/marcar-area", `{"from":{"X":0,"Y":0},"to":{"X":9,"Y":9}}`); rec.Code != http.StatusOK {
		t.Errorf("o mestre não conseguiu marcar: %d", rec.Code)
	}
}

// TestTheLassoMarksOnlyWhatIsInsideIt — o PREDICADO, que é a regra inteira deste
// gesto (ALE-311).
//
// O laço decide QUAIS peças o arrasto do grupo vai mover, e até aqui nenhum caso
// afirmava isso. Os dois que cobriam a rota olhavam 403-vs-200 e o TAMANHO da
// resposta, e um deles posta `{}` — sem coordenada nenhuma.
//
// Medido: sabotado o handler para ignorar os dois cantos e marcar TODAS as
// peças, a suíte ficava verde. É literalmente o caso que o `CLAUDE.md` descreve
// em "quando o PREDICADO decide quem é afetado, prenda o predicado" — arranjar o
// resultado por ordem de chamada diz o que acontece *com* as linhas achadas e
// nada sobre *quais* linhas são essas.
//
// As três peças são o mínimo que distingue: uma DENTRO, uma FORA pelo eixo x e
// uma FORA pelo eixo y. Com uma fora só, um predicado que testasse um eixo e
// esquecesse o outro passaria em metade dos casos.
func TestTheLassoMarksOnlyWhatIsInsideIt(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	dentro := f.seedToken(t, "Goblin de dentro", 3, 3)
	foraNoX := f.seedToken(t, "Goblin à direita", 9, 3)
	foraNoY := f.seedToken(t, "Goblin abaixo", 3, 9)

	// O laço vai de (2,2) a (5,5): pega o (3,3) e deixa os dois vizinhos fora.
	resposta := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/marcar-area",
		`{"from":{"X":2,"Y":2},"to":{"X":5,"Y":5}}`)

	if !strings.Contains(resposta, dentro) {
		t.Errorf("o laço de (2,2) a (5,5) não pegou a peça em (3,3):\n%s", resposta)
	}
	for nome, deFora := range map[string]string{"a de (9,3)": foraNoX, "a de (3,9)": foraNoY} {
		if strings.Contains(resposta, deFora) {
			t.Errorf("o laço de (2,2) a (5,5) marcou %s, que está fora dele:\n%s", nome, resposta)
		}
	}
}

// seedToken põe uma peça no tabuleiro e devolve o ID que o SERVIDOR deu a ela.
//
// O id não é escolha de quem semeia: o `AddToken` atribui um UUID e ignora o
// campo. Um caso que comparasse o `marked_tokens` com o id que ele mesmo passou
// afirmaria sobre uma peça que não existe — e, como `strings.Contains` de um
// nome inventado é sempre falso, ele acusaria "o laço não pegou" para SEMPRE.
func (f sceneFixture) seedToken(t *testing.T, rotulo string, x, y int) string {
	t.Helper()
	estado, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab,
		board.BoardToken{Label: rotulo, X: x, Y: y})
	if err != nil {
		t.Fatalf("pôr a peça %q: %v", rotulo, err)
	}
	for _, peca := range estado.Tokens {
		if peca.Label == rotulo {
			return peca.ID
		}
	}
	t.Fatalf("a peça %q não entrou no tabuleiro", rotulo)
	return ""
}

// TestTheLassoReadsTheCornersInAnyOrder: arrastar da direita para a esquerda é o
// mesmo laço.
//
// O canto onde o dedo DESCEU vira `from`, e o de cima-à-esquerda não é sempre
// ele. Um predicado escrito como `de.X <= p.X && p.X <= ate.X` marca ZERO peças
// no arrasto invertido — e não estoura, não recusa, não diz nada: a barra
// simplesmente não aparece.
func TestTheLassoReadsTheCornersInAnyOrder(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	dentro := f.seedToken(t, "Goblin de dentro", 3, 3)

	invertido := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/marcar-area",
		`{"from":{"X":5,"Y":5},"to":{"X":2,"Y":2}}`)
	if !strings.Contains(invertido, dentro) {
		t.Errorf("o laço arrastado de (5,5) para (2,2) não pegou a peça em (3,3):\n%s", invertido)
	}
}

// TestMarkingDoesNotPatchTheScene — o irmão do guarda da régua.
//
// Marcar não muda a cena de ninguém, e a resposta tem de ser do tamanho disso.
// Uma marcação que devolvesse as regiões trocaria o mapa debaixo de quem está
// arrastando — que é exatamente o gesto que acabou de acontecer.
func TestMarkingDoesNotPatchTheScene(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	resposta := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/marcar-area", "{}")
	if !strings.Contains(resposta, "marked_tokens") {
		t.Fatalf("a marcação não voltou: %s", resposta)
	}
	if strings.Contains(resposta, "datastar-patch-elements") {
		t.Errorf("marcar remendou a cena:\n%s", resposta)
	}
}

// TestAGroupWithNoMarkedTokenRefusesWithASentence: o gesto que não tem sobre o que agir
// diz isso, em vez de gravar uma versão nova sem mudar nada.
func TestAGroupWithNoMarkedTokenRefusesWithASentence(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/grupo/mover", `{"delta":{"X":1,"Y":1},"marked_tokens":""}`)
	if !strings.Contains(corpo, "não há peça marcada") {
		t.Errorf("mover um grupo vazio não foi recusado com frase: %q", corpo[max(0, len(corpo)-200):])
	}
}

// TestTheGroupMovesThemAllInOneResponse.
//
// Duas afirmações: as peças andam pelo delta, e a resposta é a do gesto contínuo
// (só o mapa). A segunda importa porque mover um grupo é um arrasto, e devolver
// a Mesa inteira no meio dele é o defeito de 353 KB que a fatia 3 mediu.
func TestTheGroupMovesThemAllInOneResponse(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	f.seedOpenBoard(t, "stone")
	ficha, _ := sceneIds(t, f)
	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+ficha+`"}`)

	b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(b.Tokens) == 0 {
		t.Fatal("a peça não entrou no mapa — o guarda mediria o vazio")
	}
	antes := b.Tokens[0]
	corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/grupo/mover",
		`{"delta":{"X":3,"Y":-2},"marked_tokens":"`+antes.ID+`"}`)

	b = f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b.Tokens[0].X != antes.X+3 || b.Tokens[0].Y != antes.Y-2 {
		t.Errorf("a peça de %s foi para (%d,%d), esperado (%d,%d)",
			ficha, b.Tokens[0].X, b.Tokens[0].Y, antes.X+3, antes.Y-2)
	}
	if strings.Contains(corpo, `id="table-archive"`) {
		t.Error("mover o grupo devolveu a Mesa inteira no meio de um arrasto")
	}
}

// TestTheRestingLayerServesBothGestures.
//
// UMA camada e não duas, e isto é conserto de um defeito medido: as duas se
// mostravam com `$tool === ”`, e a que vem DEPOIS no DOM cobria a outra —
// o dedo nunca chegava ao laço, e o gesto simplesmente não acontecia.
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
	ficha, _ := sceneIds(t, f)
	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"map_selection":"`+ficha+`"}`)
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O valor é CONSTANTE no `.templ`, então ele sai LITERAL no HTML — só o
	// dinâmico é escapado. A primeira versão deste guarda procurava a forma
	// escapada, achava zero, e acusava "0 camadas" sobre uma cena correta.
	if quantas := strings.Count(tela, `data-show="$tool === ''"`); quantas != 1 {
		t.Errorf("há %d camadas de repouso; com mais de uma a de baixo nunca recebe o dedo", quantas)
	}
	for _, pedaco := range []string{"marcar-area", "swallow_click", "board-token-marked"} {
		if !strings.Contains(tela, pedaco) {
			t.Errorf("a cena não tem %q: a seleção em área não acontece", pedaco)
		}
	}
}
