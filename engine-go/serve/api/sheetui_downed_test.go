package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// downFixture derruba o personagem do jogador a -3 PV, numa cena de ação com a
// vez girada para ele — o teste de sangramento fica esperando o d20.
func downFixture(t *testing.T) sceneFixture {
	t.Helper()
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	var entryID string
	for _, e := range f.s.sessions.GetState(f.sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == f.charID {
			entryID = e.ID
		}
	}
	standing := poolsOf(t, f.s, f.charID).HpCurrent
	if _, err := f.s.sessions.DeltaVitals(f.sessionID, entryID, live.PtrInt64(-(standing + 3)), nil); err != nil {
		t.Fatalf("derrubar: %v", err)
	}
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil {
		t.Fatalf("girar para quem sangra: %v", err)
	}
	if f.s.sessions.GetState(f.sessionID).Scene.PendingBleeding(false) == nil {
		t.Fatal("o controle falhou: a vez chegou a quem sangra e o teste não abriu")
	}
	return f
}

// A FICHA DE QUEM CAIU DIZ O ESTADO, e a fila da Mesa também: sem a palavra, o
// morto e o que está a -3 são a mesma barra vazia (p236, ALE-366).
func TestTheSheetAndTheTableSayWhoIsDown(t *testing.T) {
	f := downFixture(t)
	sheetPage := f.pede(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", f.charID), "").Body.String()
	if !strings.Contains(sheetPage, "morrendo") {
		t.Error("a ficha a -3 PV e sangrando não diz \"morrendo\"")
	}
	if !strings.Contains(sheetPage, "−3/") {
		t.Error("o PV negativo da ficha não sai com o sinal de menos tipográfico, igual ao dos botões")
	}
	table := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(table, "morrendo") {
		t.Error("a fila da Mesa não diz \"morrendo\" ao lado do PV de quem sangra")
	}
	if !strings.Contains(table, "sangra — teste de Constituição (CD 15)") {
		t.Error("a faixa do mestre não pergunta o teste de quem sangra")
	}
}

// QUEM RESPONDE O TESTE: o dono da ficha e o mestre. Outro jogador da mesa lê a
// pergunta, e o envio dele é recusado dizendo de quem é o teste.
func TestOnlyTheOwnerOrTheGMAnswersTheBleedingCheck(t *testing.T) {
	f := downFixture(t)
	other := seedUser(t, f.s, "outro@t.com")
	otherChar := seedCharacterAtLevel(t, f.s, other, "Outro", "Guerreiro", 1, 10, 4)
	seedMember(t, f.s, f.campaignID, otherChar)
	send := func(user int64, die string, value int) string {
		return f.posta(t, user, f.tableUrl()+"/sangramento/"+die, fmt.Sprintf(`{"bleeding_roll": %d}`, value))
	}

	refused := send(other, "d20", 20)
	if !strings.Contains(refused, "só o dono da ficha ou o mestre") {
		t.Errorf("outro jogador mandou o d20 do teste e a resposta foi %q", refused)
	}
	if f.s.sessions.GetState(f.sessionID).Scene.PendingBleeding(false) == nil {
		t.Fatal("a recusa mexeu no teste mesmo assim")
	}
	// O dono manda um 1 (falha com qualquer Constituição plausível), e o
	// mestre manda o d6 — os dois papéis respondem.
	if body := send(f.player, "d20", 1); strings.Contains(body, "só o dono") {
		t.Fatalf("o dono foi recusado: %q", body)
	}
	if f.s.sessions.GetState(f.sessionID).Scene.PendingBleeding(true) == nil {
		t.Fatal("o d20 do dono não chegou: o teste não está esperando o d6")
	}
	if body := send(f.gm, "d6", 2); strings.Contains(body, "só o dono") {
		t.Fatalf("o mestre foi recusado: %q", body)
	}
	if hp := poolsOf(t, f.s, f.charID).HpCurrent; hp != -5 {
		t.Errorf("-3 menos o d6 de 2 é -5, e o PV foi para %d", hp)
	}
}
