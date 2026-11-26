package api

import (
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// A MARCA NA FILA, de ponta a ponta (ver GLOSSARY, **marca na fila**): a conjuração deixa a marca para o mestre pôr,
// ele marca quem falhou, e na vez dessa linha ele rola o dado, na linha, ou diz o total da mão. A faixa da magia
// informa.

func entryOf(t *testing.T, state *live.SessionRuntimeState, entryID string) live.InitiativeEntry {
	t.Helper()
	for _, e := range state.Initiative {
		if e.ID == entryID {
			return e
		}
	}
	t.Fatalf("a linha %q não está na fila", entryID)
	return live.InitiativeEntry{}
}

func TestTheGMMarksTheCreatureHitAndRollsOnItsTurn(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	goblin := f.actionSceneWithTheGoblinFirst(t, "Conjurador da Vez", f.charID, 2)
	// O aprimoramento 2 da Flecha Ácida aumenta o dano inicial e o por rodada em +1d6 (p193).
	if refusal, _ := castWith(t, f, "flecha-acida", `{"augment2":1}`); refusal != "" {
		t.Fatalf("conjurar a Flecha Ácida foi recusado: %q", refusal)
	}
	mark := stateOf(t, f.s.sessions, f.sessionID).Markable
	if mark == nil || mark.Source != "Flecha Ácida" || mark.Who != "Conjurador da Vez" || !mark.Counted || mark.TurnsLeft != 2 ||
		len(mark.Parts) != 1 || mark.Parts[0].Dice != "3d6" {
		t.Fatalf("a conjuração deixou a marca %+v; want o muco de 3d6 da Flecha, por dois turnos", mark)
	}

	markURL := f.tableUrl() + "/iniciativa/" + goblin + "/marcar"
	f.requests(t, f.player, http.MethodPost, markURL, "")
	if marks := entryOf(t, stateOf(t, f.s.sessions, f.sessionID), goblin).Marks; len(marks) != 0 {
		t.Fatalf("o jogador marcou o Goblin: %+v; quem diz quem falhou é o mestre", marks)
	}
	f.requests(t, f.gm, http.MethodPost, markURL, "")
	if marks := entryOf(t, stateOf(t, f.s.sessions, f.sessionID), goblin).Marks; len(marks) != 1 {
		t.Fatalf("o mestre marcou o Goblin, e ele tem %d marcas", len(marks))
	}

	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", "")
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, `aria-label="Rolar Flecha Ácida na vez de Goblin"`) {
		t.Error("na vez do Goblin, a mesa do mestre não oferece o Rolar do muco")
	}

	roll := f.tableUrl() + "/iniciativa/" + goblin + "/na-vez/0/rolar"
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, roll, `{"due_hand_0":40}`).Body.String()); refusal == "" {
		t.Error("os 40 da mesa passaram num 3d6")
	}
	f.requests(t, f.gm, http.MethodPost, roll, `{"due_hand_0":9}`)
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil || cast.Kind != live.RollOfTheMark || cast.Head() != "Conjurador da Vez · Flecha Ácida · início do turno de Goblin" {
		t.Fatalf("a faixa da magia diz %+v; want o muco da Flecha na vez do Goblin", cast)
	}
	if part := cast.Parts[0]; part.Total != 9 || !part.ByHand || part.Dice != "3d6" || part.Kind != "ácido" {
		t.Errorf("o muco rolou %+v; want os 9 da mesa em 3d6 de ácido", part)
	}
}

// AS CHAMAS DA CONDIÇÃO (p394): o Goblin Em Chamas rola 1d6 de fogo na vez dele, pelo mesmo caminho.
func TestTheBurningGoblinRollsItsFireOnItsTurn(t *testing.T) {
	f := newSceneFixture(t)
	goblin := f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	if _, err := f.s.sessions.UpdateInitiativeEntry(t.Context(), f.sessionID, goblin,
		live.EntryPatch{Conditions: &[]string{live.ConditionBurning}}); err != nil {
		t.Fatalf("pôr o Goblin em chamas: %v", err)
	}
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, `aria-label="Rolar Em Chamas na vez de Goblin"`) {
		t.Error("na vez do Goblin em chamas, a mesa do mestre não oferece o Rolar das chamas")
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/na-vez/0/rolar", `{}`).Body.String()); refusal != "" {
		t.Fatalf("rolar as chamas do Goblin foi recusado: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	// A CONDIÇÃO NÃO É MAGIA: ela não veste os campos de magia nem de quem conjurou.
	if cast == nil || cast.Kind != live.RollOfTheCondition || cast.Condition != "Em Chamas" || cast.Target != "Goblin" ||
		cast.Spell != "" || cast.Who != "" || len(cast.Parts) != 1 {
		t.Fatalf("a faixa diz %+v; want as chamas do Goblin", cast)
	}
	if part := cast.Parts[0]; part.Dice != "1d6" || part.Kind != "fogo" || part.Total < 1 || part.Total > 6 || part.ByHand {
		t.Errorf("as chamas rolaram %+v; want 1d6 de fogo rolado no app", part)
	}
}

// O CONFUSO (p394) rola o 1d6 na vez dele pelo mesmo caminho, e a faixa diz o que o resultado manda.
func TestTheConfusedGoblinRollsItsD6OnItsTurn(t *testing.T) {
	f := newSceneFixture(t)
	goblin := f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	if _, err := f.s.sessions.UpdateInitiativeEntry(t.Context(), f.sessionID, goblin,
		live.EntryPatch{Conditions: &[]string{"confuso"}}); err != nil {
		t.Fatalf("deixar o Goblin confuso: %v", err)
	}
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, `aria-label="Rolar Confuso na vez de Goblin"`) {
		t.Error("na vez do Goblin confuso, a mesa do mestre não oferece o Rolar do 1d6")
	}
	roll := f.tableUrl() + "/iniciativa/" + goblin + "/na-vez/0/rolar"
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, roll, `{"due_hand_0":3}`).Body.String()); refusal != "" {
		t.Fatalf("rolar o 1d6 do Goblin confuso foi recusado: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil || cast.Kind != live.RollOfTheCondition || cast.Condition != "Confuso" || len(cast.Parts) != 1 {
		t.Fatalf("a faixa diz %+v; want o 1d6 do Goblin confuso", cast)
	}
	if part := cast.Parts[0]; part.Total != 3 || !part.ByHand || !strings.Contains(part.Note, "não age") {
		t.Errorf("o 1d6 rolou %+v; want os 3 da mesa, que mandam não agir", part)
	}
	if page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, "não age") {
		t.Error("a faixa do jogador não diz o que o 3 do confuso manda")
	}
}
