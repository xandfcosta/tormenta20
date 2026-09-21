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
