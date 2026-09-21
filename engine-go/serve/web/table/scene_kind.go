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
func sceneKindOf(requested string) (live.SceneKind, error) {
	switch live.SceneKind(requested) {
	case live.SceneAction, live.SceneExploration, live.SceneRoleplay:
		return live.SceneKind(requested), nil
	}
	return "", fmt.Errorf(
		"a cena %q não é uma das três do livro (p252): ação, exploração ou interpretação", requested)
}

// sceneView é a cena em curso, como a tela a lê.
type sceneView struct {
	Name   string
	Number int
}

func sceneOf(st *live.SessionRuntimeState) *sceneView {
	if st == nil || st.Scene == nil {
		return nil
	}
	return &sceneView{Name: st.Scene.Kind.Name(), Number: st.Scene.Number}
}
