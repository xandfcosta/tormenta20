package api

import (
	"context"
	"reflect"
	"testing"

	"t20engine/domain/live"
)

// O DANO NÃO LETAL ATRAVESSA ATÉ A FICHA (ALE-423, p236).
//
//	"Dano não letal conta para determinar quando você cai inconsciente, mas não
//	 para determinar quando você começa a sangrar ou morre. Efeitos de cura
//	 recuperam primeiro pontos de vida perdidos por dano não letal."
//
// O caso é o trajeto inteiro: a pancada entra pelo mesmo gesto do mestre, a
// parcela é gravada, as condições saem da regra e a cura paga a parcela antes
// do resto.
func TestNonLethalDamageKnocksOutWithoutBleeding(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 10, 4)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.State(ctx, sid); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.AddInitiativeEntry(ctx, sid, sheetCombatant("A", 12, charID)); err != nil {
		t.Fatalf("pôr na fila: %v", err)
	}
	entryID := stateOf(t, store, sid).Initiative[0].ID
	cheio := poolsOf(t, s, charID).HpCurrent
	if cheio <= 0 {
		t.Fatalf("o controle falhou: o personagem começou com %d PV", cheio)
	}
	bate := func(perda int64, naoLetal int64) int64 {
		t.Helper()
		if _, err := store.DeltaVitals(ctx, sid, entryID, live.PtrInt64(-perda), nil, naoLetal); err != nil {
			t.Fatalf("bater %d (não letal %d): %v", perda, naoLetal, err)
		}
		return poolsOf(t, s, charID).HpCurrent
	}

	// A QUEDA A ZERO, toda não letal: ele apaga e NÃO sangra.
	if hp := bate(cheio, cheio); hp != 0 {
		t.Fatalf("a pancada levou a %d PV, e ela era de exatamente os PV cheios", hp)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente"}) {
		t.Errorf("a 0 PV por dano NÃO LETAL as condições são %v, e o livro só derruba — "+
			"quem apagou assim não está morrendo", got)
	}
}

// O CONTROLE da mesma queda sendo LETAL: sem ele, "não sangrou" se explicaria
// igualmente bem por "o motor parou de fazer alguém sangrar".
func TestTheSameFallWithLethalDamageDoesBleed(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 10, 4)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.State(ctx, sid); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.AddInitiativeEntry(ctx, sid, sheetCombatant("A", 12, charID)); err != nil {
		t.Fatalf("pôr na fila: %v", err)
	}
	entryID := stateOf(t, store, sid).Initiative[0].ID
	cheio := poolsOf(t, s, charID).HpCurrent

	if _, err := store.DeltaVitals(ctx, sid, entryID, live.PtrInt64(-cheio), nil, 0); err != nil {
		t.Fatalf("bater: %v", err)
	}
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente", "sangrando"}) {
		t.Errorf("a 0 PV por dano LETAL as condições são %v, e o livro liga as duas", got)
	}
}

// E A CURA PAGA O NÃO LETAL PRIMEIRO, o que se mede pelo SANGRAMENTO: com 4 de
// dano letal e 6 de não letal num herói de 10 PV, curar 6 paga a parcela e
// deixa os 4 letais. Ele acorda, e não sangra em nenhum momento.
func TestHealingPaysTheNonLethalBeforeTheLethal(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 10, 4)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.State(ctx, sid); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.AddInitiativeEntry(ctx, sid, sheetCombatant("A", 12, charID)); err != nil {
		t.Fatalf("pôr na fila: %v", err)
	}
	entryID := stateOf(t, store, sid).Initiative[0].ID
	cheio := poolsOf(t, s, charID).HpCurrent

	letal := cheio - 6
	if letal <= 0 {
		t.Skipf("o guerreiro semeado tem %d PV, e o caso precisa de mais que 6", cheio)
	}
	bate := func(perda, naoLetal int64) {
		t.Helper()
		if _, err := store.DeltaVitals(ctx, sid, entryID, live.PtrInt64(-perda), nil, naoLetal); err != nil {
			t.Fatalf("bater: %v", err)
		}
	}
	bate(letal, 0)
	bate(6, 6)
	if hp := poolsOf(t, s, charID).HpCurrent; hp != 0 {
		t.Fatalf("as duas pancadas deixaram %d PV, e elas somavam os PV cheios", hp)
	}

	// CURAR 6 paga a parcela inteira. O PV vai a 6, e o dano que resta é LETAL.
	if _, err := store.DeltaVitals(ctx, sid, entryID, live.PtrInt64(6), nil, 0); err != nil {
		t.Fatalf("curar: %v", err)
	}
	if hp := poolsOf(t, s, charID).HpCurrent; hp != 6 {
		t.Errorf("curar 6 de 0 PV deu %d", hp)
	}
	// E AGORA a prova: outra pancada LETAL de 6 o leva a 0 de novo, e desta vez
	// ele SANGRA — porque a parcela não letal foi paga e não sobrou nada dela.
	bate(6, 0)
	if got := conditionsOf(t, s, charID); !reflect.DeepEqual(got, []string{"inconsciente", "sangrando"}) {
		t.Errorf("depois de a cura pagar o não letal, a queda seguinte deu %v — ela era "+
			"toda letal e tinha de fazer sangrar", got)
	}
}
