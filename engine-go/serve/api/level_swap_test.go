package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

const barbarianIncreaseID = "class.barbaro.aumento-de-atributo"

func swapRow(t *testing.T, f sceneFixture, id, rowID int64, target string) string {
	t.Helper()
	path := fmt.Sprintf("/personagens/%d/escolhas/%d/troca/%s", id, rowID, target)
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, path, "").Body.String())
}

func levelPickRow(t *testing.T, f sceneFixture, id int64, option string) sheet.LevelChoiceDTO {
	t.Helper()
	for _, row := range levelRows(t, f, id) {
		if row.Kind == sheet.LevelKindPick && row.Pick == option {
			return row
		}
	}
	t.Fatalf("a ficha %d não tem a escolha %q", id, option)
	return sheet.LevelChoiceDTO{}
}

// Trocar mantém o nível, tira a marca de estimado, e a ficha nunca fica com a vaga vazia.
func TestSwappingAChoiceKeepsItsLevel(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, barbarianIncrease+"wisdom", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	before := levelPickRow(t, f, id, "strength")

	if refusal := swapRow(t, f, id, before.ID, barbarianIncreaseID+"/dexterity"); refusal != "" {
		t.Fatalf("trocar foi recusado: %q", refusal)
	}
	after := levelPickRow(t, f, id, "dexterity")
	if after.Level != before.Level || after.Estimated {
		t.Errorf("a troca ficou %+v, esperava o nível %d e sem estimativa", after, before.Level)
	}
	_, picks := storedChoices(t, f.s, id)
	if strings.Contains(picks, "strength") || !strings.Contains(picks, "wisdom") {
		t.Errorf("as escolhas ficaram %s", picks)
	}
}

// Trocar para um poder sem escolha substitui a linha, no mesmo nível.
func TestSwappingAPickForAnotherPowerRemovesTheOldPower(t *testing.T) {
	f, id := barbaro(t, 2)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	before := levelPickRow(t, f, id, "strength")
	var other book.ClassPower
	for _, power := range book.ClassPowers() {
		if power.ClassName == "Bárbaro" && power.GrantedAtLevel == nil && power.Choice == nil && len(power.Prerequisites) == 0 {
			other = power
			break
		}
	}
	if refusal := swapRow(t, f, id, before.ID, other.ID); refusal != "" {
		t.Fatalf("trocar por %s foi recusado: %q", other.Name, refusal)
	}
	powers, picks := storedChoices(t, f.s, id)
	if strings.Contains(powers, barbarianIncreaseID) || strings.Contains(picks, "strength") || !strings.Contains(powers, other.ID) {
		t.Errorf("depois da troca: poderes %s, escolhas %s", powers, picks)
	}
	row, found := rowWith(levelRows(t, f, id), sheet.LevelKindPower, other.ID)
	if !found || row.Level != before.Level {
		t.Errorf("o poder novo ficou %+v, esperava o nível %d", row, before.Level)
	}
}

// Trocar para o mesmo atributo num patamar em que ele já está é recusado (p38).
func TestSwappingRespectsTheAttributeTier(t *testing.T) {
	// No nível 5 a conta total deixa duas Forças: quem recusa é o patamar de cada uma (p38).
	f, id := barbaro(t, 5)
	for _, option := range []string{"strength", "wisdom", "dexterity", "charisma"} {
		if refusal := powerCommand(t, f, id, barbarianIncrease+option, ""); refusal != "" {
			t.Fatalf("escolher %s: %q", option, refusal)
		}
	}
	sameTier := levelPickRow(t, f, id, "wisdom")
	if refusal := swapRow(t, f, id, sameTier.ID, barbarianIncreaseID+"/strength"); !strings.Contains(refusal, "patamar") {
		t.Errorf("a Força no mesmo patamar da outra devia ser recusada, e a recusa foi %q", refusal)
	}
	otherTier := levelPickRow(t, f, id, "charisma")
	if refusal := swapRow(t, f, id, otherTier.ID, barbarianIncreaseID+"/strength"); refusal != "" {
		t.Errorf("a Força no nível 5 é de outro patamar, e foi recusada: %q", refusal)
	}
}

// O poder que outra escolha exige não se troca, e a recusa diz qual e em que nível.
func TestSwappingRefusesWhatAnotherChoiceDependsOn(t *testing.T) {
	var needed, dependent book.ClassPower
	for _, power := range book.ClassPowers() {
		if power.ClassName != "Bárbaro" || power.GrantedAtLevel != nil {
			continue
		}
		for _, requirement := range power.Prerequisites {
			if requirement.Kind != "power" {
				continue
			}
			if candidate, known := book.ClassPowers()[requirement.ID]; known && candidate.GrantedAtLevel == nil && candidate.Choice == nil {
				needed, dependent = candidate, power
			}
		}
	}
	if dependent.Name == "" {
		t.Fatal("o catálogo não tem poder de Bárbaro que exija outro: o caso não mede nada")
	}
	f, id := barbaro(t, 3)
	seedPowerChoices(t, f.s, id, fmt.Sprintf("[%q,%q]", needed.ID, dependent.ID), "{}")
	row, found := rowWith(levelRows(t, f, id), sheet.LevelKindPower, needed.ID)
	if !found {
		t.Fatalf("a ficha não tem %s", needed.Name)
	}
	refusal := swapRow(t, f, id, row.ID, barbarianIncreaseID+"/strength")
	if !strings.Contains(refusal, dependent.Name) {
		t.Errorf("a recusa devia citar %q, e foi %q", dependent.Name, refusal)
	}
}

// Só se troca, e não se remove: tirar um poder escolhido é recusado.
func TestAChosenPowerCannotBeRemoved(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	tira := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/poderes/escolha/%s/strength/tira", id, barbarianIncreaseID), "")
	if tira.Code == http.StatusOK && !strings.Contains(tira.Body.String(), "Trocar") {
		t.Errorf("a rota de tirar ainda responde")
	}
	if _, picks := storedChoices(t, f.s, id); !strings.Contains(picks, "strength") {
		t.Errorf("a escolha saiu: %s", picks)
	}

	var plain book.ClassPower
	for _, power := range book.ClassPowers() {
		if power.ClassName == "Bárbaro" && power.GrantedAtLevel == nil && power.Choice == nil && len(power.Prerequisites) == 0 {
			plain = power
			break
		}
	}
	if refusal := powerCommand(t, f, id, "escolhe/"+plain.ID, ""); refusal != "" {
		t.Fatalf("escolher %s: %q", plain.Name, refusal)
	}
	if refusal := powerCommand(t, f, id, "escolhe/"+plain.ID, ""); !strings.Contains(refusal, "Trocar") {
		t.Errorf("tocar de novo no poder devia recusar apontando a troca, e foi %q", refusal)
	}
}

// O `−` só tira o último nível, e o que foi escolhido nele sai junto.
func TestLoweringTakesTheLastLevelWithItsChoices(t *testing.T) {
	f, id := barbaro(t, 5)
	for i := 0; i < 2; i++ {
		if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
			t.Fatalf("Força: %q", refusal)
		}
	}
	if refusal := stepClass(t, f, id, "Bárbaro", -1); refusal != "" {
		t.Fatalf("descer foi recusado: %q", refusal)
	}
	if got := classesOfTheSheet(t, f, id)["Bárbaro"]; got != 4 {
		t.Errorf("a classe ficou no nível %d, esperava 4", got)
	}
	if _, picks := storedChoices(t, f.s, id); strings.Count(picks, "strength") != 1 {
		t.Errorf("a escolha do nível 5 devia sair junto: %s", picks)
	}
}

// Numa ficha multiclasse, só a classe do último nível desce.
func TestOnlyTheClassOfTheLastLevelGoesDown(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := enterClass(t, f, id, "Ladino"); refusal != "" {
		t.Fatalf("entrar: %q", refusal)
	}
	if refusal := stepClass(t, f, id, "Bárbaro", -1); !strings.Contains(refusal, "Ladino") {
		t.Errorf("descer o Bárbaro devia apontar o Ladino, e a recusa foi %q", refusal)
	}
}

// A seção mostra os níveis do mais recente para o mais antigo, com a automática, a escolha com
// Trocar, a vaga pendente, e a marca de estimado.
func TestTheByLevelSectionShowsEveryLevel(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities&visao=nivel", id), "").Body.String()
	for _, want := range []string{"Por nível", "Nível 3", "Nível 1", "Fúria", "automática", "Aumento de Atributo", "Trocar", "Escolha pendente"} {
		if !strings.Contains(page, want) {
			t.Errorf("a seção não traz %q", want)
		}
	}
	if strings.Index(page, "Nível 3") > strings.Index(page, ">Nível 1<") && strings.Contains(page, ">Nível 1<") {
		t.Error("os níveis não vieram do mais recente para o mais antigo")
	}
	if strings.Contains(page, "nível estimado") {
		t.Error("uma escolha feita agora não é estimada")
	}
}

// Trocar abre o diálogo dentro da cena, com as candidatas daquela linha, e não sai da Mesa.
func TestSwappingOpensItsDialogInsideTheScene(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	row := levelPickRow(t, f, id, "strength")
	page := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=abilities&embutida=1&trocar=%d", id, row.ID), "").Body.String()
	wants := []string{
		"Trocar a escolha do nível " + fmt.Sprint(row.Level),
		fmt.Sprintf("/personagens/%d/escolhas/%d/troca/", id, row.ID),
		"embutida=1",
	}
	for _, want := range wants {
		if !strings.Contains(page, want) {
			t.Errorf("o diálogo de troca não traz %q", want)
		}
	}
	if without := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities&visao=nivel", id), "").Body.String(); strings.Contains(without, "swap-choice") {
		t.Error("o diálogo de troca veio sem ninguém pedir")
	}
}

// O `−` diz antes o que some, e a classe que não é a do último nível não tem o gesto.
func TestTheLowerDialogListsWhatLeaves(t *testing.T) {
	f, id := barbaro(t, 3)
	for _, option := range []string{"strength", "wisdom"} {
		if refusal := powerCommand(t, f, id, barbarianIncrease+option, ""); refusal != "" {
			t.Fatalf("escolher %s: %q", option, refusal)
		}
	}
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
	for _, want := range []string{"Descer para o nível 2?", "Aumento de Atributo, Sabedoria", "Cancelar"} {
		if !strings.Contains(page, want) {
			t.Errorf("o aviso de descer não traz %q", want)
		}
	}
}
