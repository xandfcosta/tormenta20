package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"html"
	"os"
	"slices"
	"strings"
	"testing"
)

// OS PODERES CONCEDIDOS. Quem se torna devoto recebe um (p96), e clérigo (p57), druida (p61) e paladino
// (p82) recebem dois. O passo Devoção marca e desmarca no mesmo botão.

func storedGrantedPowers(t *testing.T, f sceneFixture, id int64) []string {
	t.Helper()
	var blob string
	if err := f.s.db.QueryRowContext(context.Background(), `SELECT godPowers FROM characters WHERE id = ?`, id).Scan(&blob); err != nil {
		t.Fatalf("ler os poderes: %v", err)
	}
	var powers []string
	if err := json.Unmarshal([]byte(blob), &powers); err != nil {
		t.Fatalf("godPowers não é uma lista: %q", blob)
	}
	return powers
}

func devotedTo(t *testing.T, f sceneFixture, class, god string) int64 {
	t.Helper()
	id := seedCharacterAtLevel(t, f.s, f.player, "Fiel", class, 1, 0, 0)
	if refusal := powerCommand(t, f, id, devotionPath(god), ""); refusal != "" {
		t.Fatalf("%s de %s: %q", class, god, refusal)
	}
	return id
}

func TestTheClassesThatServeAGodChooseTwoGrantedPowers(t *testing.T) {
	for class, god := range map[string]string{"Clérigo": "khalmyr", "Paladino": "khalmyr", "Druida": "allihanna"} {
		f := newSceneFixture(t)
		id := devotedTo(t, f, class, god)
		if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "falta 2" {
			t.Errorf("%s com o deus e sem poder: o passo diz %q", class, step.Tag)
		}
	}

	f := newSceneFixture(t)
	id := devotedTo(t, f, "Clérigo", "khalmyr")
	if refusal := powerCommand(t, f, id, grantedPath("Coragem Total"), ""); refusal != "" {
		t.Fatalf("o primeiro: %q", refusal)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "falta 1" {
		t.Errorf("com um dos dois, o passo diz %q", step.Tag)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Dom da Verdade"), ""); refusal != "" {
		t.Fatalf("o segundo: %q", refusal)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "feito" {
		t.Errorf("com os dois, o passo diz %q", step.Tag)
	}
	if got := storedGrantedPowers(t, f, id); !slices.Equal(got, []string{"Coragem Total", "Dom da Verdade"}) {
		t.Errorf("guardou %v", got)
	}

	refusal := powerCommand(t, f, id, grantedPath("Espada Justiceira"), "")
	if !strings.Contains(refusal, "2") || !strings.Contains(refusal, "Espada Justiceira") {
		t.Errorf("o terceiro devia ser recusado citando o limite e o poder, e veio %q", refusal)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Coragem Total"), ""); refusal != "" {
		t.Fatalf("tirar um: %q", refusal)
	}
	if got := storedGrantedPowers(t, f, id); !slices.Equal(got, []string{"Dom da Verdade"}) {
		t.Errorf("tocar no marcado devia tirá-lo, e ficou %v", got)
	}
	screen := powerScreen(t, f, id)
	if !strings.Contains(html.UnescapeString(screen), "1 de 2") {
		t.Error("a lista não diz quantos dos dois foram escolhidos")
	}
}

// Os outros devotos recebem um, e tocar em outro só troca.
func TestAnyOtherDevoteeHasOneGrantedPower(t *testing.T) {
	f, id := guerreiro(t)
	if refusal := powerCommand(t, f, id, devotionPath("khalmyr"), ""); refusal != "" {
		t.Fatalf("escolher o deus: %q", refusal)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "falta 1" {
		t.Errorf("devoto sem o poder: o passo diz %q", step.Tag)
	}
	for _, power := range []string{"Coragem Total", "Dom da Verdade"} {
		if refusal := powerCommand(t, f, id, grantedPath(power), ""); refusal != "" {
			t.Fatalf("%s: %q", power, refusal)
		}
	}
	if got := storedGrantedPowers(t, f, id); !slices.Equal(got, []string{"Dom da Verdade"}) {
		t.Errorf("numa vaga só, o segundo devia trocar o primeiro, e ficou %v", got)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "feito" {
		t.Errorf("com o poder, o passo diz %q", step.Tag)
	}
}

// Trocar de deus tira da ficha o que o deus novo não concede, por decisão do dono. O que ele também concede
// fica, e ocupa a vaga: Coragem Total é de Khalmyr e de Valkaria (p99, p105).
func TestChangingTheGodRemovesThePowersItDoesNotGrant(t *testing.T) {
	f := newSceneFixture(t)
	id := devotedTo(t, f, "Clérigo", "khalmyr")
	for _, power := range []string{"Coragem Total", "Dom da Verdade"} {
		if refusal := powerCommand(t, f, id, grantedPath(power), ""); refusal != "" {
			t.Fatalf("%s: %q", power, refusal)
		}
	}
	if refusal := powerCommand(t, f, id, devotionPath("valkaria"), ""); refusal != "" {
		t.Fatalf("trocar: %q", refusal)
	}
	if got := storedGrantedPowers(t, f, id); !slices.Equal(got, []string{"Coragem Total"}) {
		t.Errorf("trocar para Valkaria devia deixar só a Coragem Total, e ficou %v", got)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "falta 1" {
		t.Errorf("com um dos dois de Valkaria, o passo diz %q", step.Tag)
	}
	if refusal := powerCommand(t, f, id, devotionPath("nenhum"), ""); refusal != "" {
		t.Fatalf("ficar sem deus: %q", refusal)
	}
	if got := storedGrantedPowers(t, f, id); len(got) != 0 {
		t.Errorf("sem deus, nenhum poder concedido fica, e ficou %v", got)
	}
}

// A ficha que já chega com o poder de outro deus, de antes desta regra, mostra-o e deixa tirar. Ele não
// ocupa a vaga do deus de agora.
func TestAPowerOfAnotherGodFromBeforeCanBeTakenOut(t *testing.T) {
	f := newSceneFixture(t)
	id := devotedTo(t, f, "Clérigo", "azgher")
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET godPowers = '["Dom da Verdade"]' WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	screen := powerScreen(t, f, id)
	if !strings.Contains(screen, "Dom da Verdade é de outro deus") {
		t.Error("o passo não avisa que o poder é de outro deus")
	}
	if step := choiceStepTitled(t, screen, "Devoção"); step.Tag != "falta 2" {
		t.Errorf("o poder de outro deus não ocupa vaga, e o passo diz %q", step.Tag)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Dom da Verdade"), ""); refusal != "" {
		t.Fatalf("tirar: %q", refusal)
	}
	if got := storedGrantedPowers(t, f, id); len(got) != 0 {
		t.Errorf("tirar deixou %v", got)
	}
}

// Quem já tem deus não troca num toque: o passo diz o que muda, e pede confirmação. Sem deus não há o que
// avisar, e o toque já escolhe.
func TestChangingTheGodTellsWhatChangesAndAsks(t *testing.T) {
	f := newSceneFixture(t)
	fresh := seedCharacterAtLevel(t, f.s, f.player, "Fiel", "Clérigo", 1, 0, 0)
	if screen := html.UnescapeString(powerScreen(t, f, fresh)); strings.Contains(screen, "$devotion_target") {
		t.Error("sem deus, não há troca a confirmar")
	}

	id := devotedTo(t, f, "Clérigo", "khalmyr")
	if refusal := powerCommand(t, f, id, grantedPath("Dom da Verdade"), ""); refusal != "" {
		t.Fatalf("o poder: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Coragem Total"), ""); refusal != "" {
		t.Fatalf("o poder: %q", refusal)
	}
	screen := html.UnescapeString(powerScreen(t, f, id))
	if !strings.Contains(screen, "$devotion_target = 'azgher'") {
		t.Error("tocar em outro deus devia abrir a confirmação")
	}
	for _, want := range []string{
		"Deus: Khalmyr → Azgher.",
		"Dom da Verdade e Coragem Total saem da ficha.",
		"Falta escolher 2 poderes concedidos de Azgher.",
	} {
		if !strings.Contains(screen, want) {
			t.Errorf("o aviso não diz %q", want)
		}
	}
	// Coragem Total também é de Valkaria (p105), e continua valendo depois da troca.
	if !strings.Contains(screen, "Coragem Total continua valendo: Valkaria também o concede.") {
		t.Error("o aviso não separa o poder que o novo deus também concede")
	}
	if !strings.Contains(screen, "Confirmar a troca") {
		t.Error("o aviso não tem o botão de confirmar")
	}
	if !strings.Contains(screen, "Deus: Khalmyr → Panteão.") || !strings.Contains(screen, "Panteão não recebe poder concedido.") {
		t.Error("o aviso do Panteão não diz que ele não concede nada")
	}
	if !strings.Contains(screen, "Khalmyr deixa de ser o deus da ficha.") || !strings.Contains(screen, "A devoção fica pendente.") {
		t.Error("o aviso de ficar sem deus não diz o que muda para um clérigo")
	}
}

// A migração guarda como lista o poder que cada ficha já tinha.
func TestTheMigrationKeepsTheGrantedPowerAsAList(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE characters (id INTEGER PRIMARY KEY, godPower TEXT NOT NULL DEFAULT '');
		INSERT INTO characters (id, godPower) VALUES (1, 'Dom da Verdade'), (2, '');`); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../infra/db/migrations/00025_god_powers_are_a_list.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	if _, err := db.ExecContext(ctx, up); err != nil {
		t.Fatalf("rodar a migração: %v", err)
	}
	for id, want := range map[int]string{1: `["Dom da Verdade"]`, 2: `[]`} {
		var got string
		if err := db.QueryRowContext(ctx, `SELECT godPowers FROM characters WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("ficha %d: %q, esperava %q", id, got, want)
		}
	}
}
