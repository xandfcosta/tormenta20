package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// A IMUNIDADE RECUSA A CONDIÇÃO (ALE-423, p228-229).
//
//	"A criatura é imune a um tipo de efeito ou outro elemento (como um tipo de
//	 dano, uma condição ou uma habilidade). Ela não sofre nenhuma consequência
//	 direta daquilo contra a qual ela é imune."
//
// INTEGRAÇÃO porque o que pode quebrar é a COMPOSIÇÃO: o gesto do mestre bate
// numa rota, a cena acha o verbete do bestiário por trás da linha, o motor cruza
// o TIPO e a MENTE da criatura com as TAGS da condição, e a recusa volta pelo
// fio em prosa. Um teste do `ImmuneToCondition` prova a regra e nada sobre o
// mestre conseguir aplicar a condição assim mesmo.

// umMonstroNaFila põe um verbete do bestiário na fila e devolve o id da linha.
func umMonstroNaFila(t *testing.T, f sceneFixture, monstro, rotulo string) string {
	t.Helper()
	_, err := f.s.sessions.AddInitiativeEntry(context.Background(), f.sessionID,
		live.InitiativeEntry{Label: rotulo, Initiative: 10, Type: "npc", MonsterID: &monstro})
	if err != nil {
		t.Fatalf("pôr %s na fila: %v", rotulo, err)
	}
	estado := stateOf(t, f.s.sessions, f.sessionID)
	for _, linha := range estado.Initiative {
		if linha.Label == rotulo {
			return linha.ID
		}
	}
	t.Fatalf("%s não entrou na fila", rotulo)
	return ""
}

// marcaCondicao bate no gesto do mestre e devolve a recusa, ou vazio.
func marcaCondicao(t *testing.T, f sceneFixture, entryID, condicao string) string {
	t.Helper()
	rec := f.requests(t, f.gm, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+entryID+"/condicao/"+condicao, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("marcar %q deu %d", condicao, rec.Code)
	}
	return tableRefusal(t, rec.Body.String())
}

func condicoesDaLinha(t *testing.T, f sceneFixture, entryID string) []string {
	t.Helper()
	estado := stateOf(t, f.s.sessions, f.sessionID)
	i := live.FindEntryIndex(estado, entryID)
	if i < 0 {
		t.Fatalf("a linha %q sumiu da fila", entryID)
	}
	return estado.Initiative[i].Conditions
}

// O ZUMBI É IMUNE AOS DOIS LADOS: a veneno pelo TIPO (morto-vivo) e a medo pela
// MENTE (Inteligência nula). E a recusa diz QUAL deles barrou.
func TestTheUndeadRefusesPoisonAndFearAndSaysWhich(t *testing.T) {
	f := newSceneFixture(t)
	zumbi := umMonstroNaFila(t, f, "zumbi", "Zumbi")

	veneno := marcaCondicao(t, f, zumbi, "envenenado")
	if !strings.Contains(veneno, "veneno") {
		t.Errorf("envenenar o Zumbi não foi recusado pelo tipo de efeito: %q", veneno)
	}
	medo := marcaCondicao(t, f, zumbi, "apavorado")
	if !strings.Contains(medo, "medo") {
		t.Errorf("apavorar o Zumbi não foi recusado pelo tipo de efeito: %q", medo)
	}

	// O CONTROLE: uma condição de tipo que ele NÃO é imune entra. Sem ele, uma
	// rota que recusasse tudo passaria nas duas asserções acima.
	if recusa := marcaCondicao(t, f, zumbi, "agarrado"); recusa != "" {
		t.Fatalf("o controle já estava errado: agarrar o Zumbi foi recusado com %q", recusa)
	}
	se := condicoesDaLinha(t, f, zumbi)
	if len(se) != 1 || se[0] != "agarrado" {
		t.Errorf("a linha do Zumbi ficou com %v — só o Agarrado podia ter entrado", se)
	}
}

// INTELIGÊNCIA ZERO NÃO É INTELIGÊNCIA NULA, e a Aparição é o caso que separa as
// duas metades da regra: morto-vivo com Int 0, ela é imune a veneno pelo TIPO e
// SENTE MEDO como qualquer um.
//
// É o caso que um código tratando 0 como "sem mente" erraria — e ele passaria no
// Zumbi, que é o exemplo óbvio.
func TestTheUndeadWithIntelligenceZeroStillFeelsFear(t *testing.T) {
	f := newSceneFixture(t)
	aparicao := umMonstroNaFila(t, f, "aparicao", "Aparição")

	if recusa := marcaCondicao(t, f, aparicao, "envenenado"); !strings.Contains(recusa, "veneno") {
		t.Fatalf("o controle já estava errado: envenenar a Aparição não foi recusado: %q", recusa)
	}
	if recusa := marcaCondicao(t, f, aparicao, "apavorado"); recusa != "" {
		t.Errorf("apavorar a Aparição foi recusado com %q — ela tem Inteligência 0, e a p228 "+
			"isenta quem tem Inteligência NULA", recusa)
	}
	if se := condicoesDaLinha(t, f, aparicao); len(se) != 1 || se[0] != "apavorado" {
		t.Errorf("a linha da Aparição ficou com %v, e o Apavorado tinha de ter entrado", se)
	}
}

// TIRAR UMA CONDIÇÃO DE UM IMUNE CONTINUA FUNCIONANDO.
//
// A imunidade barra o que ENTRA. Barrar os dois sentidos prenderia na linha para
// sempre a condição que alguém marcou antes desta regra existir — e o mestre não
// teria como limpar a mesa.
func TestRemovingAConditionFromAnImmuneCreatureStillWorks(t *testing.T) {
	f := newSceneFixture(t)
	zumbi := umMonstroNaFila(t, f, "zumbi", "Zumbi")

	// A condição entra POR BAIXO do gesto, como se tivesse sido marcada antes
	// desta regra: o caminho da tela agora a recusaria.
	presas := []string{"envenenado"}
	if _, err := f.s.sessions.UpdateInitiativeEntry(context.Background(), f.sessionID, zumbi,
		live.EntryPatch{Conditions: &presas}); err != nil {
		t.Fatalf("plantar a condição antiga: %v", err)
	}
	if se := condicoesDaLinha(t, f, zumbi); len(se) != 1 {
		t.Fatalf("o controle já estava errado: a linha ficou com %v", se)
	}

	if recusa := marcaCondicao(t, f, zumbi, "envenenado"); recusa != "" {
		t.Errorf("TIRAR o Envenenado do Zumbi foi recusado com %q — a imunidade barra o "+
			"que entra, e o que já está preso não teria como sair", recusa)
	}
	if se := condicoesDaLinha(t, f, zumbi); len(se) != 0 {
		t.Errorf("a linha do Zumbi ficou com %v depois de tirar a condição", se)
	}
}
