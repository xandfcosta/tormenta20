package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// CELEBRAR RITUAL: "dobra seu limite de PM, mas muda a execução para 1 hora [...] e exige um gasto de
// T$ 10 por PM gasto" (p131). A conta aparece, e o app não cobra (decisão do dono).

// ritualCaster é a arcanista da bancada com o poder e o interruptor ligado.
func ritualCaster(t *testing.T, powers string) sceneFixture {
	t.Helper()
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID, powers, "{}")
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", f.charID)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{"conditional":"flag:ritual"}`).Body.String()); refusal != "" {
		t.Fatalf("ligar o ritual: %q", refusal)
	}
	return f
}

func ritualIsOn(t *testing.T, f sceneFixture) bool {
	t.Helper()
	var count int
	if err := f.s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM character_conditionals WHERE characterId = ? AND conditionalId = 'flag:ritual'", f.charID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count > 0
}

// O ritual vale para UMA conjuração: ligado para sempre, a próxima magia em combate sairia com o
// limite dobrado.
func TestTheRitualEndsWithTheCast(t *testing.T) {
	f := ritualCaster(t, `["celebrar-ritual"]`)
	refusal, spent := castWith(t, f, "luz", `{}`)
	if refusal != "" {
		t.Fatalf("o ritual fora de cena foi recusado: %q", refusal)
	}
	if spent != 1 {
		t.Errorf("gastou %d PM, e a Luz custa 1", spent)
	}
	if ritualIsOn(t, f) {
		t.Error("o interruptor ficou ligado depois do ritual")
	}
}

func TestARefusedRitualKeepsTheSwitchOn(t *testing.T) {
	f := ritualCaster(t, `["celebrar-ritual"]`)
	seedsNoMana(t, f)
	if refusal, _ := castWith(t, f, "bola-de-fogo", `{}`); refusal == "" {
		t.Fatal("o controle falhou: sem PM, a magia devia ser recusada")
	}
	if !ritualIsOn(t, f) {
		t.Error("a recusa desligou o interruptor, e o ritual nem saiu")
	}
}

// Uma hora não cabe num turno, e por isso a Bola de Fogo de 15 PM não sai no meio da luta.
func TestTheRitualDoesNotFitAnActionScene(t *testing.T) {
	f := ritualCaster(t, `["celebrar-ritual"]`)
	store := f.s.sessions
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code >= 300 {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	if _, err := store.State(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartScene(t.Context(), f.sessionID, live.SceneAction); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddInitiativeEntry(t.Context(), f.sessionID, sheetCombatant("Arcanista", 5, f.charID)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	refusal, spent := castWith(t, f, "luz", `{}`)
	if !strings.Contains(refusal, "1 hora") {
		t.Errorf("em cena de ação, o ritual devia ser recusado dizendo a hora, e veio %q", refusal)
	}
	if spent != 0 {
		t.Errorf("a recusa cobrou %d PM", spent)
	}
}

// "Mudanças na mesma característica da magia nunca se acumulam" (p171).
func TestTheRitualAndTheQuickenedSpellDoNotStack(t *testing.T) {
	f := ritualCaster(t, `["celebrar-ritual","magia-acelerada"]`)
	refusal, spent := castWith(t, f, "luz", `{"augment_quickened":1}`)
	if !strings.Contains(refusal, "ritual") {
		t.Errorf("o ritual acelerado devia ser recusado, e veio %q", refusal)
	}
	if spent != 0 {
		t.Errorf("a recusa cobrou %d PM", spent)
	}
}

// A conta dos T$ 10 por PM aparece no diálogo, e só com o interruptor ligado.
func TestTheRitualBillShowsUpOnlyWhileItIsOn(t *testing.T) {
	const bill = "Como ritual, T$ "
	f := newSceneFixture(t)
	seedPowerChoices(t, f.s, f.charID, `["celebrar-ritual"]`, "{}")
	learnSpell(t, f, "luz")
	page := func() string {
		return f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=spells", f.charID), "").Body.String()
	}
	if !strings.Contains(page(), "Custo") {
		t.Fatal("o controle falhou: a aba não desenhou o custo da magia")
	}
	if strings.Contains(page(), bill) {
		t.Error("a conta apareceu com o interruptor desligado")
	}
	f = ritualCaster(t, `["celebrar-ritual"]`)
	learnSpell(t, f, "luz")
	if !strings.Contains(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=spells", f.charID), "").Body.String(), bill) {
		t.Error("com o ritual ligado, o diálogo não mostra a conta")
	}
}

func seedsNoMana(t *testing.T, f sceneFixture) {
	t.Helper()
	if _, err := f.s.db.ExecContext(t.Context(),
		"INSERT OR REPLACE INTO character_damage (characterId, hpDamage, mpSpent) VALUES (?, 0, 999)", f.charID); err != nil {
		t.Fatalf("zerar o PM: %v", err)
	}
}
