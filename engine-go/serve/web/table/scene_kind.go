package table

import (
	"fmt"

	"t20engine/domain/live"
)

// sceneKindOf lê o tipo da cena que o caminho pede.
//
// TIPO DESCONHECIDO RECUSA em vez de cair na cena de ação. Cair no padrão
// pareceria funcionar — o mestre clicaria "interpretação" e ganharia um combate
// —, e o único sinal seria a rodada aparecendo numa conversa.
func sceneKindOf(pedido string) (live.SceneKind, error) {
	switch live.SceneKind(pedido) {
	case live.SceneAction, live.SceneExploration, live.SceneRoleplay:
		return live.SceneKind(pedido), nil
	}
	return "", fmt.Errorf(
		"a cena %q não é uma das três do livro (p252): ação, exploração ou interpretação", pedido)
}

// sceneView é a cena em curso, como a tela a lê.
type sceneView struct {
	Tipo   live.SceneKind
	Nome   string
	Numero int
	// ContaRodadas diz se esta cena mede tempo em rodadas — só a de ação (p252).
	// A tela usa para não desenhar rodada e vez numa conversa.
	ContaRodadas bool
}

func sceneOf(st *live.SessionRuntimeState) *sceneView {
	if st == nil || st.Scene == nil {
		return nil
	}
	return &sceneView{
		Tipo: st.Scene.Kind, Nome: st.Scene.Kind.Name(),
		Numero: st.Scene.Number, ContaRodadas: st.Scene.CountsRounds(),
	}
}
