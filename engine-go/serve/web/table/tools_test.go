package table

import (
	"strings"
	"testing"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// Os guardas do TRILHO DE FERRAMENTAS.
//
// A forma — vertical, sobreposto — é do CSS e não se afirma aqui: o que um teste
// de render pode dizer sobre leiaute é pouco, e o que importa dela está no
// navegador. O que se prende é o que sobrevive a um `Contains`: que a lista, os
// botões e o TECLADO concordam sobre quais ferramentas existem, e que a borracha
// deixou de apagar a coisa errada.

// Os dois saem do mesmo `rail`, e este guarda existe porque a alternativa —
// uma tabela de teclas escrita à mão — falharia em silêncio dos dois jeitos: uma
// tecla que liga uma ferramenta sem botão, e um botão que a tecla não alcança.
func TestTheKeyboardAndTheRailAgreeOnWhatExists(t *testing.T) {
	for _, role := range []struct {
		Name string
		GM   bool
	}{{"o mestre", true}, {"o jogador", false}} {
		keyboard := railKeyboard(role.GM)
		for _, f := range rail(role.GM) {
			if !strings.Contains(keyboard, `evt.key === "`+f.Shortcut+`"`) {
				t.Errorf("%s tem o botão %q e não tem a tecla %s", role.Name, f.Label, f.Shortcut)
			}
		}
	}
	// E o inverso: o jogador NÃO pode ter a tecla de uma ferramenta do mestre.
	// Uma tecla que liga o pincel na tela de quem não pinta deixaria o clique
	// mudo — a camada não existe lá, e o gesto simplesmente não faria nada.
	forPlayer := railKeyboard(false)
	for _, f := range MapTools() {
		if f.GMOnly && strings.Contains(forPlayer, `$tool = "`+f.ID+`"`) {
			t.Errorf("o jogador tem a tecla de %q, que é do mestre", f.Label)
		}
	}
}

// Numerar por POSIÇÃO faria a mesma ferramenta trocar de tecla entre os dois
// papéis — o trilho do jogador tem três entradas e o do mestre tem nove. Quem
// aprendeu `3 = gabarito` mestrando tem de continuar com `3 = gabarito` jogando.
func TestTheShortcutIsFixedPerTool(t *testing.T) {
	forGM := map[string]string{}
	for _, f := range rail(true) {
		forGM[f.ID] = f.Shortcut
	}
	for _, f := range rail(false) {
		if forGM[f.ID] != f.Shortcut {
			t.Errorf("%q é a tecla %s para o jogador e %s para o mestre",
				f.Label, f.Shortcut, forGM[f.ID])
		}
	}
	// E nenhuma tecla serve a duas ferramentas: a segunda ganharia a disputa no
	// `?:` encadeado e a primeira ficaria inalcançável, sem erro nenhum.
	seen := map[string]string{}
	for _, f := range MapTools() {
		if before, repeated := seen[f.Shortcut]; repeated {
			t.Errorf("a tecla %s serve a %q e a %q", f.Shortcut, before, f.Label)
		}
		seen[f.Shortcut] = f.Label
	}
}

// Sem a guarda, digitar "5" no PV de um combatente trocaria a ferramenta do mapa
// atrás do formulário. Já aconteceu com o `-` do zoom, e é por isso que a guarda
// é uma constante compartilhada em vez de três cópias.
func TestTheShortcutDoesNotStealTheKeyFromWhoIsTyping(t *testing.T) {
	keyboard := railKeyboard(true)
	for _, target := range []string{"INPUT", "TEXTAREA", "SELECT"} {
		if !strings.Contains(keyboard, target) {
			t.Errorf("o atalho não se protege de %s", target)
		}
	}
	if !strings.Contains(keyboard, "isContentEditable") {
		t.Error("o atalho não se protege de um campo `contenteditable`")
	}
	// E o ESC NÃO pode estar aqui. Ele tem dono — o `scene.js` mapeia Escape para
	// "voltar" e o para no `document`, então um `keydown__window` nunca o vê.
	// Provado no navegador com controle: `F2` no mesmo nó liga a ferramenta e
	// `Escape` não chega nem a um `addEventListener` cru na janela.
	//
	// O guarda é NEGATIVO de propósito: um ramo de Escape aqui não daria erro em
	// lugar nenhum — ele só ficaria prometendo uma saída que a tela não cumpre.
	if strings.Contains(keyboard, "Escape") {
		t.Error("o trilho promete o Esc, que o `scene.js` engole antes de chegar à janela")
	}
}

// A versão do tabuleiro é o que acorda a mesa inteira pelo stream. Subir por um
// clique em chão limpo mandaria um quadro para seis pessoas para dizer que nada
// mudou — e o `writeTable` compara o HTML depois, mas o trabalho de renderizar
// nove regiões já teria acontecido.
func TestClearingAnAlreadyCleanSquareReturnsFalse(t *testing.T) {
	b := &board.BoardState{}
	if board.ClearSquare(b, engine.Square{X: 1, Y: 1}) {
		t.Error("limpar chão limpo disse que mudou alguma coisa")
	}
	if b.Version != 0 {
		t.Errorf("a versão subiu para %d sem mudança nenhuma", b.Version)
	}
	b.Difficult = append(b.Difficult, engine.Square{X: 1, Y: 1})
	if !board.ClearSquare(b, engine.Square{X: 1, Y: 1}) {
		t.Error("limpar uma casa pintada disse que nada mudou")
	}
	if b.Version != 1 {
		t.Errorf("a versão ficou em %d depois de uma mudança", b.Version)
	}
}
