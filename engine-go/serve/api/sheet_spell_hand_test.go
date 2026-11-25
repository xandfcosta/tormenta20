package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// A MÃO AGARRA E EMPURRA PELA MANOBRA DA MESA (p198, p234; ver GLOSSARY, **ação da magia**): o Agarrar sai da aba
// Ações contra o alvo da Mesa, rola o d20 com o Misticismo de quem conjurou mais 10, fica para o veredito do mestre
// com a mão no lugar da arma, e cobra a ação de movimento.

// mysticismOf é o total de Misticismo da ficha, computado.
func mysticismOf(t *testing.T, f sceneFixture) int {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	dto, err := f.s.sheetRules().LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar a ficha: %v", err)
	}
	computed, err := sheet.Compute(dto)
	if err != nil {
		t.Fatalf("computar a ficha: %v", err)
	}
	for _, e := range computed.Expertises {
		if e.Name == "Misticismo" {
			return e.Total
		}
	}
	t.Fatal("a ficha não tem Misticismo")
	return 0
}

func TestTheMightyHandGrabsWithMysticismAndTheMovement(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	// A ficha da bancada nasce sem perícias gravadas, e a de verdade tem todas: o Misticismo treinado é o do
	// Arcanista (p37).
	seedPericia(t, f.s, f.charID, "Misticismo", "intelligence", true)
	if refusal, _ := castWith(t, f, "mao-poderosa-de-talude", `{}`); refusal != "" {
		t.Fatalf("conjurar a Mão Poderosa foi recusado: %q", refusal)
	}
	goblin := f.actionSceneWithTheGoblinFirst(t, "Conjurador da Vez", f.charID, 2)
	mysticism := mysticismOf(t, f)

	// A mesa desenha as Ações da PRIMEIRA ficha do jogador na campanha: o conjurador fica sendo a única.
	if _, err := f.s.db.Exec(`DELETE FROM campaign_members WHERE campaignId = ? AND characterId != ?`, f.campaignID, f.charID); err != nil {
		t.Fatalf("tirar a outra ficha do jogador da mesa: %v", err)
	}
	actions := entre(t, superficieDeAcoes(t, acoesDaMesa(t, f)), "Ação de movimento", `</section>`)
	for _, maneuver := range []string{"Agarrar", "Empurrar"} {
		start := strings.Index(actions, maneuver+" · Mão Poderosa de Talude")
		if start < 0 {
			t.Fatalf("a aba Ações não desenhou o %s da Mão Poderosa no grupo do movimento:\n%s", maneuver, recorte(actions))
		}
		line := actions[start:min(len(actions), start+600)]
		if want := fmt.Sprintf("Misticismo +%d contra o Luta do alvo", mysticism+10); !strings.Contains(line, want) {
			t.Errorf("a linha do %s na aba Ações não diz %q:\n%s", maneuver, want, recorte(line))
		}
	}

	if refusal := spellAct(t, f, "mao-poderosa-de-talude", "Agarrar", `{}`); refusal == "" {
		t.Error("o Agarrar sem alvo na barra da Mesa passou")
	}
	if refusal := spellAct(t, f, "mao-poderosa-de-talude", "Agarrar", fmt.Sprintf(`{"turn_target":%q}`, goblin)); refusal != "" {
		t.Fatalf("o Agarrar foi recusado: %q", refusal)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	pending := state.PendingAttack
	if pending == nil || pending.Maneuver == nil || pending.Maneuver.Kind != "agarrar" || pending.TargetEntryID != goblin {
		t.Fatalf("a mesa tem %+v; want a manobra agarrar contra o Goblin", pending)
	}
	if pending.Weapon != "Mão Poderosa de Talude" {
		t.Errorf("a manobra é com %q; want a Mão Poderosa de Talude no lugar da arma", pending.Weapon)
	}
	if want := mysticism + 10; pending.Total-pending.Roll != want {
		t.Errorf("o Agarrar soma %d ao d20; want o Misticismo mais 10 = %d", pending.Total-pending.Roll, want)
	}
	if state.Scene.MovementLeft || !state.Scene.StandardLeft {
		t.Errorf("a vez ficou com movimento = %v e padrão = %v; want o movimento gasto e a padrão inteira",
			state.Scene.MovementLeft, state.Scene.StandardLeft)
	}

	// A CARTA DO MESTRE diz que é a mão, e não quem conjurou, que agarra: sem isso, ela leria como o Luta dele.
	gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	head := regexp.MustCompile(`(?s)<section id="table-attack".*?<div class="ds-strike-head"[^>]*>(.*?)</div>`).FindStringSubmatch(gm)
	if head == nil || !strings.Contains(head[1], "· Mão Poderosa de Talude") {
		t.Errorf("a cabeça da carta do mestre não diz que a manobra é da Mão Poderosa: %q", head)
	}

	t.Run("o cancelamento do mestre devolve a ação de movimento", func(t *testing.T) {
		if _, err := f.s.sessions.CancelAttack(t.Context(), f.sessionID, live.Attacker{UserID: f.gm, Role: "gm"}); err != nil {
			t.Fatalf("cancelar a manobra da mão: %v", err)
		}
		if scene := stateOf(t, f.s.sessions, f.sessionID).Scene; !scene.MovementLeft || !scene.StandardLeft {
			t.Errorf("depois do cancelamento, movimento = %v e padrão = %v; want as duas de volta", scene.MovementLeft, scene.StandardLeft)
		}
	})
}
