package api

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"t20engine/infra/db/sqlcgen"
)

// A ESCOLHA DE HABILIDADE DE RAÇA (ALE-493): a Deformidade do lefou (p24) e o
// Vanguardista do kliren (p28), gravadas no blob da raça pelo id da habilidade.

func raceAbilityCommand(t *testing.T, f sceneFixture, id int64, ability, option string) string {
	t.Helper()
	return powerCommand(t, f, id, "raca/"+ability+"/"+url.PathEscape(option), "")
}

func storedRaceAttributeChoices(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	return row.Raceattributechoices
}

func lefouWithSpread(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := barbaro(t, 3)
	seedRaca(t, f.s, id, "lefou")
	return f, id
}

// "+2 em duas perícias a sua escolha […] Você pode trocar um desses bônus por
// um poder da Tormenta" (p24): duas escolhas, no máximo um poder.
func TestTheDeformityTakesTwoSkillsOrASkillAndATormentaPower(t *testing.T) {
	f, id := lefouWithSpread(t)

	for _, pick := range []string{"Furtividade", "olhos-vermelhos"} {
		if refusal := raceAbilityCommand(t, f, id, "lefou-deformidade", pick); refusal != "" {
			t.Fatalf("escolher %s na Deformidade foi recusado: %q", pick, refusal)
		}
	}
	if refusal := raceAbilityCommand(t, f, id, "lefou-deformidade", "Percepção"); !strings.Contains(refusal, "2") {
		t.Errorf("a terceira escolha da Deformidade foi aceita, ou a recusa não diz o limite: %q", refusal)
	}
	if got := storedRaceAttributeChoices(t, f, id); !strings.Contains(got, `"abilityPicks":{"lefou-deformidade":["Furtividade","olhos-vermelhos"]}`) {
		t.Errorf("raceAttributeChoices gravado = %s", got)
	}
	// O PODER TROCADO É DA FICHA: aparece no acervo.
	if !strings.Contains(powerPanel(powerScreen(t, f, id)), "Olhos Vermelhos") {
		t.Error("o poder da Tormenta trocado na Deformidade não aparece na aba Poderes")
	}

	// Tirar a perícia e pôr um SEGUNDO poder: "um desses bônus", um só.
	if refusal := raceAbilityCommand(t, f, id, "lefou-deformidade", "Furtividade"); refusal != "" {
		t.Fatalf("tirar Furtividade foi recusado: %q", refusal)
	}
	if refusal := raceAbilityCommand(t, f, id, "lefou-deformidade", "antenas"); !strings.Contains(refusal, "poder da Tormenta") {
		t.Errorf("dois poderes da Tormenta na Deformidade foram aceitos: %q", refusal)
	}
}

// REDISTRIBUIR O ATRIBUTO NÃO APAGA A ESCOLHA: os dois moram no mesmo blob.
func TestTheRaceSpreadKeepsTheAbilityPicks(t *testing.T) {
	f, id := lefouWithSpread(t)
	if refusal := raceAbilityCommand(t, f, id, "lefou-deformidade", "Furtividade"); refusal != "" {
		t.Fatalf("escolher Furtividade: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, "atributos", `{"race_attributes":["strength","dexterity","constitution"]}`); refusal != "" {
		t.Fatalf("distribuir o lefou: %q", refusal)
	}
	got := storedRaceAttributeChoices(t, f, id)
	if !strings.Contains(got, `"lefou-deformidade":["Furtividade"]`) || !strings.Contains(got, `"floatingPicks":["strength","dexterity","constitution"]`) {
		t.Errorf("a distribuição apagou a escolha da Deformidade, ou não gravou: %s", got)
	}
}

// p28 Vanguardista: "+2 em Ofício (um qualquer, a sua escolha)". Só um Ofício
// que a ficha TEM; e o Ofício escolhido não se apaga (decisão do dono).
func TestTheVanguardistPicksACraftTheSheetHas(t *testing.T) {
	f, id := barbaro(t, 3)
	seedRaca(t, f.s, id, "kliren")

	if !strings.Contains(powerScreen(t, f, id), "Vanguardista: escolher o Ofício") {
		t.Error("o kliren sem Ofício escolhido não tem a pendência")
	}
	if refusal := raceAbilityCommand(t, f, id, "kliren-vanguardista", "Ofício (alquimista)"); !strings.Contains(refusal, "Ofício") {
		t.Errorf("um Ofício que a ficha não tem foi aceito: %q", refusal)
	}
	if _, err := f.s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: "Ofício (alquimista)", Attribute: "intelligence", Trained: 0, Custom: 1,
	}); err != nil {
		t.Fatalf("semear o ofício: %v", err)
	}
	if refusal := raceAbilityCommand(t, f, id, "kliren-vanguardista", "Ofício (alquimista)"); refusal != "" {
		t.Fatalf("escolher o Ofício que a ficha tem: %q", refusal)
	}
	if refusal := sheetPost(t, f, id, "/pericias/remover/"+url.PathEscape("Ofício (alquimista)"), ""); !strings.Contains(refusal, "Vanguardista") {
		t.Errorf("apagar o Ofício escolhido no Vanguardista não foi recusado: %q", refusal)
	}
}
