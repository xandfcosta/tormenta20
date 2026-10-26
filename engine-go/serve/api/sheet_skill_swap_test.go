package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// TROCAR A PERÍCIA E ROLAR DOIS d20 NO TESTE, DO DIÁLOGO ATÉ A FAIXA (ALE-500).

func swapperAtTheTable(t *testing.T, class, origin, originChoices string) (sceneFixture, int64) {
	t.Helper()
	f := umaMesaAoVivo(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Trocadora", Origin: origin, Level: 5, Charisma: 3, Wisdom: 1,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: originChoices, ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a ficha: %v", err)
	}
	seedClasse(t, f.s, id, class, 5)
	seedMember(t, f.s, f.campaignID, id)
	seedPericia(t, f.s, id, "Diplomacia", "charisma", false)
	// TREINADA em Enganação e Investigação: as trocas têm de mudar o número,
	// senão um teste que não trocasse nada passaria verde.
	seedPericia(t, f.s, id, "Enganação", "charisma", true)
	seedPericia(t, f.s, id, "Investigação", "intelligence", true)
	seedPericia(t, f.s, id, "Percepção", "wisdom", false)
	f.charID = id
	return f, id
}

func totalOf(t *testing.T, f sceneFixture) int {
	t.Helper()
	got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if got == nil {
		t.Fatal("o teste não chegou à mesa")
	}
	return got.Total
}

// p88 Alpinista Social: Diplomacia vira Enganação, de graça. Com o mesmo d20,
// o total é o de Enganação, e a faixa diz a troca.
func TestTheSocialClimberRollsDiplomacyWithDeception(t *testing.T) {
	f, id := swapperAtTheTable(t, "Bucaneiro", "charlatao", `["origin-charlatao-unique"]`)
	if refusal := rollsWith(t, f, id, "Enganação", `{"expertise_d20":10}`); refusal != "" {
		t.Fatalf("Enganação: %q", refusal)
	}
	deception := totalOf(t, f)
	before := pm(t, f, id)
	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":10,"expertise_spends":{"alpinista_social":1}}`); refusal != "" {
		t.Fatalf("Diplomacia com Alpinista Social: %q", refusal)
	}
	got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if got.Total != deception || got.UsedSkill != "Enganação" {
		t.Errorf("Diplomacia com a troca: total %d e perícia usada %q, want %d (o de Enganação) e Enganação",
			got.Total, got.UsedSkill, deception)
	}
	if spent := before - pm(t, f, id); spent != 0 {
		t.Errorf("o Alpinista Social cobrou %d PM, e ele é grátis", spent)
	}
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Diplomacia com Enganação") {
		t.Error("a faixa não diz que a Diplomacia foi rolada com Enganação")
	}
}

// p91 Detetive: o 1 PM é do GESTO de cena; sem ele, Percepção não troca.
func TestTheDetectiveSwapsOnlyAfterTheSceneGesture(t *testing.T) {
	f, id := swapperAtTheTable(t, "Guerreiro", "guarda", `["origin-guarda-unique"]`)
	body := `{"expertise_d20":10,"expertise_spends":{"detetive":1}}`
	if refusal := rollsWith(t, f, id, "Percepção", body); !strings.Contains(refusal, "não vale neste teste") {
		t.Fatalf("Detetive sem o gesto de cena tinha de ser recusado, e veio %q", refusal)
	}
	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/origin.guarda.detetive", ""); refusal != "" {
		t.Fatalf("usar o Detetive: %q", refusal)
	}
	if spent := before - pm(t, f, id); spent != 1 {
		t.Errorf("o gesto do Detetive cobrou %d PM, e a p91 diz 1", spent)
	}
	if refusal := rollsWith(t, f, id, "Investigação", `{"expertise_d20":10}`); refusal != "" {
		t.Fatalf("Investigação: %q", refusal)
	}
	investigation := totalOf(t, f)
	if refusal := rollsWith(t, f, id, "Percepção", body); refusal != "" {
		t.Fatalf("Percepção com a cena paga: %q", refusal)
	}
	if got := totalOf(t, f); got != investigation {
		t.Errorf("Percepção com o Detetive deu %d, want %d (o de Investigação)", got, investigation)
	}
}

// p45 Mestre dos Sussurros: dois d20, vale o maior — e a faixa mostra os dois.
func TestTheMasterOfWhispersKeepsTheBetterOfTwoDice(t *testing.T) {
	f, id := swapperAtTheTable(t, "Bardo", "soldado", "[]")
	if refusal := powerCommand(t, f, id, "escolhe/class.bardo.mestre-dos-sussurros", ""); refusal != "" {
		t.Fatalf("escolher o Mestre dos Sussurros: %q", refusal)
	}
	if refusal := rollsWith(t, f, id, "Enganação", `{"expertise_d20":0,"expertise_spends":{"mestre_dos_sussurros":1}}`); refusal != "" {
		t.Fatalf("Enganação com dois d20: %q", refusal)
	}
	got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if len(got.Rolls) != 2 || got.Roll != max(got.Rolls[0], got.Rolls[1]) {
		t.Errorf("dois d20: rolagens %v e d20 usado %d, want dois e o maior", got.Rolls, got.Roll)
	}
}
