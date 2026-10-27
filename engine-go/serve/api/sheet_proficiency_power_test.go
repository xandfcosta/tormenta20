package api

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// O PODER PROFICIÊNCIA (p129, ALE-510):
//
//	"Escolha uma proficiência: armas marciais, armas de fogo, armaduras
//	pesadas ou escudos (se for proficiente em armas marciais, você também pode
//	escolher armas exóticas). Você recebe essa proficiência. Você pode escolher
//	este poder outras vezes para proficiências diferentes."

const proficiencyPowerPath = "escolha/proficiencia/"

// guerreiroWithTheClassDefault é o guerreiro com a coluna no padrão da classe.
func guerreiroWithTheClassDefault(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := guerreiro(t)
	saveHand(t, f, id, `["armas-simples","armas-marciais","armaduras-leves","armaduras-pesadas","escudos"]`)
	return f, id
}

func TestTheProficiencyPowerGrantsTheChosenProficiencyAndTakesASlot(t *testing.T) {
	f, id := guerreiroWithTheClassDefault(t)
	if refusal := powerCommand(t, f, id, proficiencyPowerPath+"armas-de-fogo", ""); refusal != "" {
		t.Fatalf("escolher armas de fogo na Proficiência foi recusado: %q", refusal)
	}
	if !saved(t, f, id)["armas-de-fogo"] {
		t.Error("armas de fogo, escolhida no poder Proficiência, não foi gravada na ficha")
	}
	if got := storedClassPowers(t, f, id); !slices.Contains(got, "proficiencia") {
		t.Errorf("o poder Proficiência não ocupou vaga: poderes escolhidos %v", got)
	}
	screen := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=proficiencies", id), "").Body.String()
	if !strings.Contains(screen, ">poder</span>") || !strings.Contains(screen, "Padrão: Proficiência") {
		t.Error("o painel não diz que armas de fogo vem do poder Proficiência")
	}
}

// "para proficiências diferentes", e exóticas só para quem sabe usar marciais.
func TestTheProficiencyPowerRefusesWhatTheSheetHasAndExoticWithoutMartial(t *testing.T) {
	f, id := guerreiroWithTheClassDefault(t)
	if refusal := powerCommand(t, f, id, proficiencyPowerPath+"armas-marciais", ""); refusal == "" {
		t.Error("a Proficiência em armas marciais passou num guerreiro, que já as tem")
	}
	if refusal := powerCommand(t, f, id, proficiencyPowerPath+"armas-exoticas", ""); refusal != "" {
		t.Errorf("a Proficiência em armas exóticas foi recusada a um guerreiro, que sabe usar marciais: %q", refusal)
	}

	g, other := guerreiro(t)
	saveHand(t, g, other, `["armas-simples"]`)
	if refusal := powerCommand(t, g, other, proficiencyPowerPath+"armas-exoticas", ""); !strings.Contains(refusal, "marciais") {
		t.Errorf("a Proficiência em armas exóticas sem marciais deu %q; want a recusa que cita as marciais", refusal)
	}
}

// Tirar a escolha tira a proficiência, e restaurar o padrão a mantém.
func TestTheProficiencyPowerChoiceLeavesWithTheChoiceAndStaysOnRestore(t *testing.T) {
	f, id := guerreiroWithTheClassDefault(t)
	if refusal := powerCommand(t, f, id, proficiencyPowerPath+"armas-de-fogo", ""); refusal != "" {
		t.Fatalf("escolher armas de fogo foi recusado: %q", refusal)
	}
	rec := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/proficiencias/padrao", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("restaurar deu %d: %s", rec.Code, rec.Body.String())
	}
	if !saved(t, f, id)["armas-de-fogo"] {
		t.Error("restaurar o padrão tirou armas de fogo, que o poder Proficiência concede")
	}
	if refusal := powerCommand(t, f, id, proficiencyPowerPath+"armas-de-fogo/tira", ""); refusal != "" {
		t.Fatalf("tirar a escolha foi recusado: %q", refusal)
	}
	if saved(t, f, id)["armas-de-fogo"] {
		t.Error("tirar a escolha da Proficiência deixou armas de fogo na ficha")
	}
	if !saved(t, f, id)["armas-marciais"] {
		t.Error("tirar a escolha levou junto uma proficiência da classe")
	}
}

// A lista oferece só o que a ficha ainda não tem.
func TestTheProficiencyPowerOffersOnlyWhatTheSheetLacks(t *testing.T) {
	f, id := guerreiroWithTheClassDefault(t)
	page := powerScreen(t, f, id)
	if !strings.Contains(page, "/escolha/proficiencia/armas-de-fogo") {
		t.Error("a lista de poderes não oferece armas de fogo ao guerreiro, que não as tem")
	}
	if strings.Contains(page, "/escolha/proficiencia/armas-marciais") {
		t.Error("a lista oferece armas marciais ao guerreiro, que já as tem")
	}
}
