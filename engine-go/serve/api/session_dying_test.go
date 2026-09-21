package api

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// conditionsOf lê as condições ligadas na ficha, em ordem, para comparar.
func conditionsOf(t *testing.T, s *Server, charID int64) []string {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), charID)
	if err != nil {
		t.Fatalf("ler a ficha %d: %v", charID, err)
	}
	got := sheet.UnmarshalStrings(row.Activeconditions)
	sort.Strings(got)
	if got == nil {
		got = []string{}
	}
	return got
}

// CAIR, ESTABILIZAR, ACORDAR E MORRER pelo caminho de verdade: o mestre fere e
// cura pela fila, e é a ficha que grava (T20 p236).
//
// É caso de INTEGRAÇÃO porque o que ele prende é a COSTURA: a regra mora no
// `engine.DyingConditionChange`, o PV e as condições moram na ficha, e quem os
// junta é o funil dos vitais — o único caminho de escrita, então nenhum gesto
// escapa das condições.
func TestFallingStabilizingWakingAndDyingFollowTheBook(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 10, 4)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := store.AddInitiativeEntry(sid, sheetCombatant("A", 12, charID)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entryID := store.GetState(sid).Initiative[0].ID
	pool := poolsOf(t, s, charID)
	// O CONTROLE: o personagem começa de pé e sem condição, senão "caiu" mediria
	// o estado de partida.
	if pool.HpCurrent <= 0 || len(conditionsOf(t, s, charID)) != 0 {
		t.Fatalf("o controle falhou: começou com %d PV e as condições %v", pool.HpCurrent, conditionsOf(t, s, charID))
	}
	threshold := min(int64(-10), -(pool.HpMax / 2))
	hit := func(delta int64) int64 {
		t.Helper()
		if _, err := store.DeltaVitals(sid, entryID, live.PtrInt64(delta), nil); err != nil {
			t.Fatalf("mexer %d no PV: %v", delta, err)
		}
		return poolsOf(t, s, charID).HpCurrent
	}

	// "Se ficar com 0 PV ou menos, você cai inconsciente e fica sangrando."
	if hp := hit(-(pool.HpCurrent + 3)); hp != -3 {
		t.Fatalf("o PV parou em %d, e a pancada o levava a -3: o piso não é mais zero", hp)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente", "sangrando"}) {
		t.Errorf("a -3 PV as condições são %v, e o livro liga inconsciente e sangrando", got)
	}
	// E a FILA espelha o negativo: os dois números da tela são um só.
	if mirror := live.DerefOr(store.GetState(sid).Initiative[0].HpCurrent, 0); mirror != -3 {
		t.Errorf("a fila mostra %d PV e a ficha tem -3", mirror)
	}

	// "…estabilizado… com qualquer efeito que cure pelo menos 1 PV."
	if hp := hit(+2); hp != -1 {
		t.Fatalf("curar 2 a partir de -3 deu %d", hp)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente"}) {
		t.Errorf("curado sem chegar a 1, está estável: sem sangrar e ainda inconsciente — e as condições são %v", got)
	}

	// "…recupere PV até um valor positivo (1 ou mais)… recobra a consciência."
	if hp := hit(+2); hp != 1 {
		t.Fatalf("curar 2 a partir de -1 deu %d", hp)
	}
	if got := conditionsOf(t, s, charID); len(got) != 0 {
		t.Errorf("a 1 PV acordou, e as condições continuam %v", got)
	}

	// Morrer: o PV para no limiar do livro, e quem morreu não sangra mais.
	if hp := hit(-500); hp != threshold {
		t.Fatalf("uma pancada enorme deixou %d PV, e o limiar da morte com %d totais é %d", hp, pool.HpMax, threshold)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente"}) {
		t.Errorf("morto não rola o teste de Constituição: as condições são %v", got)
	}
}

// fallenCombatant monta uma cena de ação com o personagem a -3 PV, sangrando, e
// com a vez girando para ele — o ponto em que a p236 pede o teste.
func fallenCombatant(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 10, 4)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := store.StartScene(sid, live.SceneAction); err != nil {
		t.Fatalf("começar a cena: %v", err)
	}
	if _, err := store.AddInitiativeEntry(sid, live.InitiativeEntry{ID: "npc", Label: "Goblin", Initiative: 20, Type: "npc"}); err != nil {
		t.Fatalf("pôr o NPC: %v", err)
	}
	if _, err := store.AddInitiativeEntry(sid, sheetCombatant("A", 5, charID)); err != nil {
		t.Fatalf("pôr o personagem: %v", err)
	}
	var entryID string
	for _, e := range store.GetState(sid).Initiative {
		if e.CharacterID != nil {
			entryID = e.ID
		}
	}
	standing := poolsOf(t, s, charID).HpCurrent
	if _, err := store.DeltaVitals(sid, entryID, live.PtrInt64(-(standing + 3)), nil); err != nil {
		t.Fatalf("derrubar: %v", err)
	}
	if _, err := store.NextTurn(sid); err != nil { // vez do goblin
		t.Fatalf("girar: %v", err)
	}
	if st := store.GetState(sid); st.Scene.Bleeding != nil {
		t.Fatalf("o controle falhou: a vez é do goblin e já há teste de sangramento aberto")
	}
	if _, err := store.NextTurn(sid); err != nil { // vez de quem sangra
		t.Fatalf("girar para quem sangra: %v", err)
	}
	return s, sid, charID
}

// NO INÍCIO DA VEZ DE QUEM SANGRA, o teste abre; passar estabiliza (p236). O
// jogador rola e digita (decisão do dono, ALE-366), e o servidor soma a
// Constituição — um 20 natural passa com qualquer Constituição plausível.
func TestTheBleedingCheckOpensOnTheTurnAndPassingStabilizes(t *testing.T) {
	s, sid, charID := fallenCombatant(t)
	check := s.sessions.GetState(sid).Scene.Bleeding
	if check == nil || check.CharacterID != charID || check.AwaitingD6 {
		t.Fatalf("a vez chegou a quem sangra e o teste não abriu esperando o d20: %+v", check)
	}
	if _, err := s.sessions.RollBleedingD20(sid, 20); err != nil {
		t.Fatalf("mandar o d20: %v", err)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente"}) {
		t.Errorf("passou no teste e as condições são %v — estável é sem sangrar e ainda inconsciente", got)
	}
	if hp := poolsOf(t, s, charID).HpCurrent; hp != -3 {
		t.Errorf("passar não mexe no PV, e ele foi para %d", hp)
	}
	after := s.sessions.GetState(sid).Scene.Bleeding
	if after == nil || after.Outcome == "" || after.AwaitingD6 {
		t.Errorf("a faixa tem de dizer que estabilizou, e o teste ficou %+v", after)
	}
	if _, err := s.sessions.RollBleedingD20(sid, 20); err == nil {
		t.Error("o teste já foi resolvido, e um segundo d20 foi aceito")
	}
}

// FALHAR PEDE O d6, e o d6 sai do PV pelo caminho de toda pancada (p236).
func TestFailingTheBleedingCheckAsksForTheD6AndLosesIt(t *testing.T) {
	s, sid, charID := fallenCombatant(t)
	if _, err := s.sessions.RollBleedingD6(sid, 4); err == nil {
		t.Fatal("o d6 foi aceito antes do d20")
	}
	if _, err := s.sessions.RollBleedingD20(sid, 1); err != nil {
		t.Fatalf("mandar o d20: %v", err)
	}
	check := s.sessions.GetState(sid).Scene.Bleeding
	if check == nil || !check.AwaitingD6 {
		t.Fatalf("um 1 no d20 falha, e o teste tinha de esperar o d6: %+v", check)
	}
	for _, bad := range []int{0, 7} {
		if _, err := s.sessions.RollBleedingD6(sid, bad); err == nil {
			t.Errorf("o d6 %d foi aceito", bad)
		}
	}
	if _, err := s.sessions.RollBleedingD6(sid, 4); err != nil {
		t.Fatalf("mandar o d6: %v", err)
	}
	if hp := poolsOf(t, s, charID).HpCurrent; hp != -7 {
		t.Errorf("-3 menos 4 é -7, e o PV foi para %d", hp)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente", "sangrando"}) {
		t.Errorf("falhou e continua sangrando, e as condições são %v", got)
	}
	if after := s.sessions.GetState(sid).Scene.Bleeding; after == nil || after.Outcome == "" || after.AwaitingD6 {
		t.Errorf("a faixa tem de dizer quanto perdeu, e o teste ficou %+v", after)
	}
}

// A VEZ QUE GIRA LEVA O TESTE: o mestre conduz, e a mesa não trava esperando.
func TestTheNextTurnClearsAnUnansweredBleedingCheck(t *testing.T) {
	s, sid, _ := fallenCombatant(t)
	if _, err := s.sessions.NextTurn(sid); err != nil {
		t.Fatalf("girar: %v", err)
	}
	if check := s.sessions.GetState(sid).Scene.Bleeding; check != nil {
		t.Errorf("a vez saiu de quem sangra e o teste ficou aberto: %+v", check)
	}
}
