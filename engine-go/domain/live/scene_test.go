package live

import (
	"encoding/json"
	"testing"
)

// A CENA COMO O LIVRO A DEFINE (p252): um pedaço distinto da história, com um
// TIPO, e não um interruptor de combate.
//
// "Uma cena não é uma unidade de tempo fixa, mas um pedaço distinto da
// história… O mestre decide quando uma cena começa e termina." E só uma das
// três mede tempo em rodadas.

func TestOnlyTheActionSceneCountsRounds(t *testing.T) {
	st := EmptyRuntimeState()
	StartScene(st, SceneRoleplay)
	if !st.InScene() {
		t.Fatal("uma cena de interpretação É uma cena")
	}
	if st.CountsRounds() {
		t.Errorf("só a cena de AÇÃO mede tempo em rodadas (p252)")
	}
	if st.TurnIndex != -1 || st.Round != 0 {
		t.Errorf("fora da cena de ação não há vez nem rodada, e veio turno %d rodada %d",
			st.TurnIndex, st.Round)
	}

	StartScene(st, SceneAction)
	if !st.CountsRounds() {
		t.Errorf("a cena de AÇÃO mede tempo em rodadas")
	}
}

// A SESSÃO TEM UMA SEQUÊNCIA — "uma sessão típica tem em torno de três cenas"
// (p252) —, e é o número que permite dizer isso. Hoje a sessão tem um booleano,
// então ela tem no máximo UMA.
func TestEachSceneIsNumberedInSequence(t *testing.T) {
	st := EmptyRuntimeState()
	StartScene(st, SceneRoleplay)
	if st.Scene.Number != 1 {
		t.Errorf("a primeira cena da sessão é a 1, e veio %d", st.Scene.Number)
	}
	EndScene(st)
	StartScene(st, SceneAction)
	if st.Scene.Number != 2 {
		t.Errorf("a cena depois da primeira é a 2, e veio %d", st.Scene.Number)
	}
	EndScene(st)
	if st.InScene() {
		t.Error("encerrada, não há cena em curso")
	}
	if st.ScenesSoFar != 2 {
		t.Errorf("a sessão teve 2 cenas, e ela conta %d", st.ScenesSoFar)
	}
}

// "Uma cena pode ser INTERROMPIDA para dar lugar a outra… se os personagens
// estão discutindo na corte e de repente são atacados, a cena da discussão
// acaba e uma nova cena começa — um combate" (p252).
//
// Começar uma cena com outra em curso é isso, e não um erro: a anterior acaba.
func TestStartingASceneInterruptsTheOneInPlay(t *testing.T) {
	st := EmptyRuntimeState()
	StartScene(st, SceneRoleplay)
	StartScene(st, SceneAction)
	if st.Scene.Kind != SceneAction || st.Scene.Number != 2 {
		t.Errorf("a corte virou combate: cena 2, de ação — e veio %+v", st.Scene)
	}
}

// ENCERRAR DEVOLVE O COMBATE AO COMEÇO E GUARDA A FILA. É a regra que já
// existia, e ela continua: o mestre que encerra a briga do castelo não paga
// oito goblins digitados de novo.
func TestEndingASceneKeepsTheQueue(t *testing.T) {
	st := EmptyRuntimeState()
	st.Initiative = []InitiativeEntry{{ID: "a", Label: "Arwen"}}
	StartScene(st, SceneAction)
	st.Round, st.TurnIndex, st.TurnsTaken = 3, 0, 7
	EndScene(st)
	if len(st.Initiative) != 1 {
		t.Error("encerrar não esvazia a fila — quem esvazia é o ResetInitiative")
	}
	if st.Round != 0 || st.TurnIndex != -1 || st.TurnsTaken != 0 {
		t.Errorf("encerrada, a contagem volta ao começo, e veio %d/%d/%d",
			st.Round, st.TurnIndex, st.TurnsTaken)
	}
}

// O BLOB ANTIGO CONTINUA LEGÍVEL. O estado é JSON gravado numa coluna, e há
// sessões gravadas com `sceneActive: true` — ler isso como "fora de cena"
// apagaria o combate de quem estiver jogando no dia da migração.
func TestTheOldBlobStillOpensInAScene(t *testing.T) {
	var st SessionRuntimeState
	antigo := `{"initiative":[],"round":2,"turnIndex":0,"turnsTaken":3,"sceneActive":true}`
	if err := json.Unmarshal([]byte(antigo), &st); err != nil {
		t.Fatalf("ler o blob antigo: %v", err)
	}
	if !st.InScene() {
		t.Fatal("`sceneActive: true` é uma cena em curso")
	}
	if st.Scene.Kind != SceneAction {
		t.Errorf("a cena antiga era o combate: ela abre como cena de AÇÃO, e veio %q", st.Scene.Kind)
	}
	if st.Round != 2 {
		t.Errorf("a rodada gravada sobrevive, e veio %d", st.Round)
	}

	fora := `{"initiative":[],"round":0,"turnIndex":-1,"sceneActive":false}`
	if err := json.Unmarshal([]byte(fora), &st); err != nil {
		t.Fatalf("ler o blob antigo: %v", err)
	}
	if st.InScene() {
		t.Error("`sceneActive: false` é fora de cena")
	}
}
