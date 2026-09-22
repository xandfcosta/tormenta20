package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O ATAQUE NO TURNO: agredir é ação PADRÃO (p233), e quem não pode agir não
// ataca. Estes casos prendem a ligação entre o gesto de atacar e a economia de
// ação que a ALE-365 pôs no turno — a regra da troca mora no
// `engine.TurnBudget` e a do instante no `engine.MomentFor`.

// attackOnTurn monta a mesa com um Goblin do bestiário na frente e o
// personagem, de espada na mão, com a vez. Devolve a linha do Goblin, que é o
// alvo — do BESTIÁRIO porque é ele que traz a Defesa que o ataque mede.
func attackOnTurn(t *testing.T) (sceneFixture, string) {
	t.Helper()
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
	store := f.s.sessions
	if _, err := store.State(t.Context(), f.sessionID); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.StartScene(f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("começar a cena de ação: %v", err)
	}
	goblin := "goblin-salteador"
	if _, err := store.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		Label: "Goblin", Initiative: 20, Type: "npc", MonsterID: &goblin,
	}); err != nil {
		t.Fatalf("pôr o Goblin na fila: %v", err)
	}
	if _, err := store.AddInitiativeEntry(f.sessionID, sheetCombatant("Arcanista", 5, f.charID)); err != nil {
		t.Fatalf("pôr o personagem na fila: %v", err)
	}
	for range 2 { // a primeira vez é do Goblin, a segunda do personagem
		if _, err := store.NextTurn(f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	state := stateOf(t, store, f.sessionID)
	if state.Initiative[state.TurnIndex].CharacterID == nil {
		t.Fatalf("a vez tinha de ser do personagem, e é de %q", state.Initiative[state.TurnIndex].Label)
	}
	for _, e := range state.Initiative {
		if e.Label == "Goblin" {
			return f, e.ID
		}
	}
	t.Fatal("o Goblin não está na fila")
	return f, ""
}

var commandErrorSignal = regexp.MustCompile(`"command_error":("(?:[^"\\]|\\.)*")`)

// tableRefusal lê a frase que a mesa acendeu no rodapé. Falha alto quando o
// sinal não veio: sem ele, "não recusou" e "o canal não existe" são a mesma
// string vazia.
func tableRefusal(t *testing.T, body string) string {
	t.Helper()
	found := commandErrorSignal.FindStringSubmatch(body)
	if found == nil {
		t.Fatalf("a resposta não trouxe o sinal command_error: %q", body[:min(220, len(body))])
	}
	var sentence string
	if err := json.Unmarshal([]byte(found[1]), &sentence); err != nil {
		t.Fatalf("o command_error não é uma string JSON: %v", err)
	}
	return sentence
}

func TestAttackingSpendsTheStandardAction(t *testing.T) {
	f, goblin := attackOnTurn(t)
	attack := f.tableUrl() + "/iniciativa/" + goblin + "/atacar"

	rec := f.pede(t, f.player, http.MethodPost, attack, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	// O CONTROLE: de pé, na vez e armado, o provisório NASCE. Sem isto, um
	// ataque recusado por qualquer outro motivo passaria pelos casos abaixo.
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack == nil {
		t.Fatalf("o ataque permitido não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	// PROPOR NÃO COBRA: o provisório ainda pode ser cancelado, e um ataque
	// cancelado não aconteceu.
	if scene := stateOf(t, f.s.sessions, f.sessionID).Scene; !scene.StandardLeft {
		t.Fatal("propor o ataque já gastou a ação padrão, antes de o mestre confirmar")
	}
	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar o ataque deu %d", rec.Code)
	}
	scene := stateOf(t, f.s.sessions, f.sessionID).Scene
	if scene.StandardLeft {
		t.Error("o ataque confirmado não gastou a ação padrão")
	}
	if !scene.MovementLeft {
		t.Error("atacar não toca na ação de movimento")
	}

	// O SEGUNDO ATAQUE NÃO SAI: a padrão acabou e a troca da p233 é de mão
	// única — movimento não vira padrão. O que se prende é o PROVISÓRIO, que
	// não pode nascer.
	rec = f.pede(t, f.player, http.MethodPost, attack, "")
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack != nil {
		t.Error("sem ação padrão no turno, um segundo ataque foi rolado")
	}
	if refusal := tableRefusal(t, rec.Body.String()); !strings.Contains(refusal, "não sobrou ação neste turno") {
		t.Errorf("o segundo ataque tinha de ser recusado por falta de ação, e veio %q", refusal)
	}
}

func TestAStunnedCharacterDoesNotAttack(t *testing.T) {
	f, goblin := attackOnTurn(t)
	var pc string
	for _, e := range stateOf(t, f.s.sessions, f.sessionID).Initiative {
		if e.CharacterID != nil {
			pc = e.ID
		}
	}
	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+pc+"/condicao/atordoado", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar Atordoado deu %d", rec.Code)
	}
	rec := f.pede(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack != nil {
		t.Error("atordoado, o personagem rolou um ataque")
	}
	if refusal := tableRefusal(t, rec.Body.String()); !strings.Contains(refusal, "não dá para agir") {
		t.Errorf("a recusa tinha de dizer que não dá para agir, e veio %q", refusal)
	}
}
