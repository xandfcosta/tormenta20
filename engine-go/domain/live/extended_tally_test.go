package live

import (
	"strings"
	"testing"
)

// O TESTE ESTENDIDO (ver GLOSSARY, **teste estendido**): o contador que o mestre abre e alimenta. O app soma o que
// ele marca e diz o que o livro prevê (p222); não compara rolagem com CD nem decide sozinho.

func markAll(t *testing.T, st *SessionRuntimeState, entryID string, kinds ...string) {
	t.Helper()
	for _, k := range kinds {
		if err := MarkExtendedTest(st, entryID, k); err != nil {
			t.Fatalf("marcar %s em %q: %v", k, entryID, err)
		}
	}
}

func TestTheGroupTallyCountsTheDoubleAndSaysWhatTheBookPredicts(t *testing.T) {
	st := EmptyRuntimeState()
	if err := OpenExtendedTest(st, "  Atravessar o Pântano dos Juncos ", 5, false); err != nil {
		t.Fatal(err)
	}
	markAll(t, st, "", ExtendedSuccess, ExtendedDouble, ExtendedFailure)
	tally := st.ExtendedTest.TallyOf("")
	if tally.Successes != 3 || tally.Failures != 1 || tally.Outcome != "" {
		t.Fatalf("sucesso + sucesso que conta dois + falha deram %+v; want 3 sucessos e 1 falha, sem desfecho", tally)
	}
	if st.ExtendedTest.Name != "Atravessar o Pântano dos Juncos" {
		t.Errorf("o nome ficou %q, sem aparar", st.ExtendedTest.Name)
	}

	markAll(t, st, "", ExtendedDouble)
	if got := st.ExtendedTest.TallyOf(""); got.Successes != 5 || got.Outcome != ExtendedReached {
		t.Fatalf("aos 5 sucessos o contador diz %+v; want o exigido alcançado", got)
	}
	if err := MarkExtendedTest(st, "", ExtendedFailure); err == nil {
		t.Error("o contador que já chegou aceitou outra marca")
	}

	// DESFAZER tira a última marca, e o desfecho some com ela.
	if err := UndoExtendedMark(st); err != nil {
		t.Fatal(err)
	}
	if got := st.ExtendedTest.TallyOf(""); got.Successes != 3 || got.Outcome != "" {
		t.Errorf("depois de desfazer, o contador diz %+v; want 3 sucessos de novo", got)
	}
}

func TestThreeFailuresBeforeTheSuccessesAreTheTotalFailure(t *testing.T) {
	st := EmptyRuntimeState()
	if err := OpenExtendedTest(st, "Escalar um paredão", 3, false); err != nil {
		t.Fatal(err)
	}
	markAll(t, st, "", ExtendedSuccess, ExtendedFailure, ExtendedFailure, ExtendedFailure)
	if got := st.ExtendedTest.TallyOf(""); got.Failures != 3 || got.Outcome != ExtendedTotalFailure {
		t.Fatalf("três falhas deram %+v; want a falha total", got)
	}
}

func TestEachLineOfTheQueueHasItsOwnTally(t *testing.T) {
	st := EmptyRuntimeState()
	st.Initiative = []InitiativeEntry{{ID: "g", Label: "Guerreiro"}, {ID: "o", Label: "Ogro"}}
	if err := OpenExtendedTest(st, "Avalanche · Correr", 5, true); err != nil {
		t.Fatal(err)
	}
	// A FILA DO MOMENTO de abrir: quem entra depois não ganha contagem, e quem sai não leva a dele embora.
	st.Initiative = append(st.Initiative, InitiativeEntry{ID: "n", Label: "Novato"})
	if who := st.ExtendedTest.Who; len(who) != 2 || who[0].Label != "Guerreiro" || who[1].Label != "Ogro" {
		t.Fatalf("a contagem é de %+v; want o Guerreiro e o Ogro, a fila de quando abriu", who)
	}
	markAll(t, st, "g", ExtendedSuccess, ExtendedDouble)
	markAll(t, st, "o", ExtendedFailure)
	if g, o := st.ExtendedTest.TallyOf("g"), st.ExtendedTest.TallyOf("o"); g.Successes != 3 || g.Failures != 0 || o.Failures != 1 || o.Successes != 0 {
		t.Errorf("o Guerreiro ficou %+v e o Ogro %+v; want 3 sucessos para um e 1 falha para o outro", g, o)
	}
	for _, refused := range []string{"", "n"} {
		if err := MarkExtendedTest(st, refused, ExtendedSuccess); err == nil {
			t.Errorf("a marca em %q passou num teste de cada um da fila", refused)
		}
	}
}

func TestWhatTheExtendedTestRefusesSaysWhy(t *testing.T) {
	st := EmptyRuntimeState()
	for _, c := range []struct {
		name     string
		required int
		each     bool
		says     string
	}{
		{"  ", 5, false, "o que é"},
		{"Ritual", 0, false, "sucessos exigidos"},
		{"Avalanche", 5, true, "fila"},
	} {
		if err := OpenExtendedTest(st, c.name, c.required, c.each); err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("abrir %q com %d (cada um: %v) devolveu %v; want a recusa que fala de %q", c.name, c.required, c.each, err, c.says)
		}
	}
	if err := MarkExtendedTest(st, "", ExtendedSuccess); err == nil {
		t.Error("marcou sem teste aberto")
	}
	if err := OpenExtendedTest(st, "Ritual", 7, false); err != nil {
		t.Fatal(err)
	}
	if err := OpenExtendedTest(st, "Outro", 3, false); err == nil || !strings.Contains(err.Error(), "Ritual") {
		t.Errorf("abrir com o Ritual aberto devolveu %v; want a recusa que nomeia o aberto", err)
	}
	if err := MarkExtendedTest(st, "", "talvez"); err == nil {
		t.Error("a marca \"talvez\" passou")
	}
	if err := UndoExtendedMark(st); err == nil {
		t.Error("desfez sem marca nenhuma")
	}
	CloseExtendedTest(st)
	if st.ExtendedTest != nil {
		t.Error("encerrar deixou o teste aberto")
	}
}

// O CONTADOR É DA MESA: o jogador o vê em cena e fora dela — a jornada acontece fora de cena.
func TestThePlayerSeesTheTallyInAndOutOfScene(t *testing.T) {
	for _, inScene := range []bool{true, false} {
		st := EmptyRuntimeState()
		if inScene {
			st.Scene = &Scene{Kind: SceneAction}
		}
		if err := OpenExtendedTest(st, "Jornada", 5, false); err != nil {
			t.Fatal(err)
		}
		if seen := RedactForPlayers(st).ExtendedTest; seen == nil || seen.Name != "Jornada" {
			t.Errorf("em cena %v, o jogador recebeu o contador %+v", inScene, seen)
		}
	}
}

// A MARCA NO RASCUNHO não alcança o estado de onde ele saiu: a gravação que falha não pode deixar rastro.
func TestMarkingADraftLeavesTheOriginalAsItWas(t *testing.T) {
	st := EmptyRuntimeState()
	if err := OpenExtendedTest(st, "Ritual", 7, false); err != nil {
		t.Fatal(err)
	}
	markAll(t, st, "", ExtendedSuccess)
	draft := CloneState(st)
	markAll(t, draft, "", ExtendedFailure)
	if err := UndoExtendedMark(draft); err != nil {
		t.Fatal(err)
	}
	if err := UndoExtendedMark(draft); err != nil {
		t.Fatal(err)
	}
	if got := st.ExtendedTest.TallyOf(""); got.Successes != 1 || got.Failures != 0 {
		t.Errorf("o estado de antes ficou %+v depois de mexer no rascunho; want o 1 sucesso de antes", got)
	}
}
