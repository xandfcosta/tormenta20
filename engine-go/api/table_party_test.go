package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestOnlyTheGmMarksAGroup(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	rec := f.pede(t, f.jogador, http.MethodPost, f.tableUrl()+"/tabuleiro/marcar-area/0/0/9/9", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador marcou um grupo e recebeu %d, esperado 403", rec.Code)
	}
	// O CONTROLE: o mestre PODE. Sem ele, um 403 para todo mundo passaria igual.
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/marcar-area/0/0/9/9", ""); rec.Code != http.StatusOK {
		t.Errorf("o mestre não conseguiu marcar: %d", rec.Code)
	}
}

// TestMarkingDoesNotPatchTheScene — o irmão do guarda da régua.
//
// Marcar não muda a cena de ninguém, e a resposta tem de ser do tamanho disso.
// Uma marcação que devolvesse as regiões trocaria o mapa debaixo de quem está
// arrastando — que é exatamente o gesto que acabou de acontecer.
func TestMarkingDoesNotPatchTheScene(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	resposta := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/marcar-area/0/0/9/9", "{}")
	if !strings.Contains(resposta, "pecasmarcadas") {
		t.Fatalf("a marcação não voltou: %s", resposta)
	}
	if strings.Contains(resposta, "datastar-patch-elements") {
		t.Errorf("marcar remendou a cena:\n%s", resposta)
	}
}

// TestAGroupWithNoMarkedTokenRefusesWithASentence: o gesto que não tem sobre o que agir
// diz isso, em vez de gravar uma versão nova sem mudar nada.
func TestAGroupWithNoMarkedTokenRefusesWithASentence(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/grupo/mover/1/1", `{"pecasmarcadas":""}`)
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
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, _ := sceneIds(t, f)
	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(b.Tokens) == 0 {
		t.Fatal("a peça não entrou no mapa — o guarda mediria o vazio")
	}
	antes := b.Tokens[0]
	corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/grupo/mover/3/-2",
		`{"pecasmarcadas":"`+antes.ID+`"}`)

	b = f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if b.Tokens[0].X != antes.X+3 || b.Tokens[0].Y != antes.Y-2 {
		t.Errorf("a peça de %s foi para (%d,%d), esperado (%d,%d)",
			ficha, b.Tokens[0].X, b.Tokens[0].Y, antes.X+3, antes.Y-2)
	}
	if strings.Contains(corpo, `id="mesa-acervo"`) {
		t.Error("mover o grupo devolveu a Mesa inteira no meio de um arrasto")
	}
}

// TestTheRestingLayerServesBothGestures.
//
// UMA camada e não duas, e isto é conserto de um defeito medido: as duas se
// mostravam com `$ferramenta === ”`, e a que vem DEPOIS no DOM cobria a outra —
// o dedo nunca chegava ao laço, e o gesto simplesmente não acontecia.
//
// O `engoleoclique` entra na lista pela mesma razão: o navegador dispara `click`
// depois de um `pointerdown` + `pointerup` no mesmo elemento INCLUSIVE quando o
// dedo andou, e sem ele terminar um laço também moveria a peça da vez para onde
// o laço terminou.
func TestTheRestingLayerServesBothGestures(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	// COM PEÇA no mapa: a marca do grupo é vestida pela peça, e num tabuleiro
	// vazio a classe não aparece — o guarda acusaria a ausência dela sobre uma
	// cena que só não tem peça nenhuma.
	ficha, _ := sceneIds(t, f)
	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O valor é CONSTANTE no `.templ`, então ele sai LITERAL no HTML — só o
	// dinâmico é escapado. A primeira versão deste guarda procurava a forma
	// escapada, achava zero, e acusava "0 camadas" sobre uma cena correta.
	if quantas := strings.Count(tela, `data-show="$ferramenta === ''"`); quantas != 1 {
		t.Errorf("há %d camadas de repouso; com mais de uma a de baixo nunca recebe o dedo", quantas)
	}
	for _, pedaco := range []string{"marcar-area", "engoleoclique", "tabuleiro-peca-marcada"} {
		if !strings.Contains(tela, pedaco) {
			t.Errorf("a cena não tem %q: a seleção em área não acontece", pedaco)
		}
	}
}
