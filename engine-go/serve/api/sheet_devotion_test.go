package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

// devotionPath é o gesto de escolher o deus, ou `nenhum`.
func devotionPath(god string) string { return "devocao/" + god }

// grantedPath é o gesto de escolher o poder que o deus concede.
func grantedPath(power string) string { return "poder-concedido/" + url.PathEscape(power) }

func storedDevotion(t *testing.T, f sceneFixture, id int64) (string, string) {
	t.Helper()
	var god sql.NullString
	var power string
	if err := f.s.db.QueryRowContext(context.Background(),
		`SELECT god, godPower FROM characters WHERE id = ?`, id).Scan(&god, &power); err != nil {
		t.Fatal(err)
	}
	return god.String, power
}

// Qualquer personagem escolhe o deus e troca quando quiser, e o cabeçalho lê a mesma fonte.
func TestAnyCharacterChoosesAndChangesTheGod(t *testing.T) {
	f, id := guerreiro(t)
	if refusal := powerCommand(t, f, id, devotionPath("khalmyr"), ""); refusal != "" {
		t.Fatalf("escolher Khalmyr foi recusado: %q", refusal)
	}
	if god, _ := storedDevotion(t, f, id); god != "khalmyr" {
		t.Errorf("o deus guardado é %q", god)
	}
	list := f.requests(t, f.player, http.MethodGet, "/personagens", "").Body.String()
	if !strings.Contains(list, "Devoto de Khalmyr") {
		t.Error("a lista de personagens não diz o deus escolhido")
	}
	if refusal := powerCommand(t, f, id, devotionPath("wynna"), ""); refusal != "" {
		t.Errorf("trocar de deus foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, devotionPath("nenhum"), ""); refusal != "" {
		t.Errorf("ficar sem deus foi recusado: %q", refusal)
	}
	if god, _ := storedDevotion(t, f, id); god != "" {
		t.Errorf("sem deus, o guardado é %q", god)
	}
	if refusal := powerCommand(t, f, id, devotionPath("zeus"), ""); !strings.Contains(refusal, "zeus") {
		t.Errorf("um deus que não existe devia ser recusado pelo nome, e a recusa foi %q", refusal)
	}
}

// As classes que servem a um deus limitam a escolha: o clérigo a deus maior ou o Panteão (p57), e o
// paladino à lista dele ou Paladino do Bem (p82).
func TestTheClassLimitsTheGodItServes(t *testing.T) {
	cleric := newSceneFixture(t)
	clericID := seedCharacterAtLevel(t, cleric.s, cleric.player, "Fiel", "Clérigo", 1, 0, 0)
	if refusal := powerCommand(t, cleric, clericID, devotionPath("panteao"), ""); refusal != "" {
		t.Errorf("o clérigo não conseguiu cultuar o Panteão: %q", refusal)
	}
	if refusal := powerCommand(t, cleric, clericID, devotionPath("bem"), ""); !strings.Contains(refusal, "Clérigo") {
		t.Errorf("Paladino do Bem não é de clérigo, e a recusa foi %q", refusal)
	}

	paladin := newSceneFixture(t)
	paladinID := seedCharacterAtLevel(t, paladin.s, paladin.player, "Justo", "Paladino", 1, 0, 0)
	if refusal := powerCommand(t, paladin, paladinID, devotionPath("wynna"), ""); !strings.Contains(refusal, "Paladino") {
		t.Errorf("Wynna não é de paladino, e a recusa foi %q", refusal)
	}
	if refusal := powerCommand(t, paladin, paladinID, devotionPath("bem"), ""); refusal != "" {
		t.Errorf("Paladino do Bem foi recusado: %q", refusal)
	}

	fighter, fighterID := guerreiro(t)
	if refusal := powerCommand(t, fighter, fighterID, devotionPath("panteao"), ""); refusal == "" {
		t.Error("o Panteão é só de clérigo, e um guerreiro o escolheu")
	}
}

// Clérigo e paladino sem deus têm a pendência, e o deus da ficha a resolve: era a Curandeira da
// semente, "Devoto de Lena" no cabeçalho e devoto pendente no Escolher.
func TestTheGodSettlesThePendingDevotion(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Fiel", "Clérigo", 1, 0, 0)
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Devoção"); step.Tag != "falta 1" {
		t.Errorf("o clérigo sem deus devia ter a pendência, e o passo diz %q", step.Tag)
	}
	if _, err := f.s.db.ExecContext(context.Background(), `UPDATE characters SET god = 'lena' WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	screen := powerScreen(t, f, id)
	if step := choiceStepTitled(t, screen, "Devoção"); step.Tag != "feito" {
		t.Errorf("com o deus na ficha, o passo diz %q", step.Tag)
	}
	if strings.Contains(screen, "escolher devoto") {
		t.Error("a ficha com deus ainda pede o devoto")
	}

	fighter, fighterID := guerreiro(t)
	if step := choiceStepTitled(t, powerScreen(t, fighter, fighterID), "Devoção"); step.Tag != "feito" {
		t.Errorf("sem classe que exija deus, nada falta, e o passo diz %q", step.Tag)
	}
}

// Trocar de deus não apaga o poder concedido: o passo avisa, e tirar é da mesa.
func TestChangingTheGodKeepsTheGrantedPowerAndWarns(t *testing.T) {
	f, id := guerreiro(t)
	if refusal := powerCommand(t, f, id, grantedPath("Dom da Verdade"), ""); !strings.Contains(refusal, "deus") {
		t.Errorf("sem deus não há poder concedido, e a recusa foi %q", refusal)
	}
	if refusal := powerCommand(t, f, id, devotionPath("khalmyr"), ""); refusal != "" {
		t.Fatalf("escolher Khalmyr: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Dom da Verdade"), ""); refusal != "" {
		t.Fatalf("escolher o poder: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, grantedPath("Afinidade com a Tormenta"), ""); !strings.Contains(refusal, "Khalmyr") {
		t.Errorf("o poder de outro deus devia ser recusado, e a recusa foi %q", refusal)
	}
	if refusal := powerCommand(t, f, id, devotionPath("wynna"), ""); refusal != "" {
		t.Fatalf("trocar: %q", refusal)
	}
	if _, power := storedDevotion(t, f, id); power != "Dom da Verdade" {
		t.Errorf("trocar de deus apagou o poder, e ficou %q", power)
	}
	if screen := powerScreen(t, f, id); !strings.Contains(screen, "Dom da Verdade é de outro deus") {
		t.Error("o passo não avisa que o poder é do deus antigo")
	}
	if refusal := powerCommand(t, f, id, grantedPath("nenhum"), ""); refusal != "" {
		t.Errorf("tirar o poder: %q", refusal)
	}
	if _, power := storedDevotion(t, f, id); power != "" {
		t.Errorf("tirar deixou %q", power)
	}
}

// Quem cultua o Panteão não recebe poder concedido (p57), e o passo diz isso em vez de uma lista vazia.
func TestThePantheonGrantsNoPower(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Fiel", "Clérigo", 1, 0, 0)
	if refusal := powerCommand(t, f, id, devotionPath("panteao"), ""); refusal != "" {
		t.Fatalf("escolher o Panteão: %q", refusal)
	}
	if screen := powerScreen(t, f, id); !strings.Contains(screen, "não recebe poder concedido") {
		t.Error("o passo não diz que o Panteão não concede poder")
	}
}

// A migração leva o devoto para o deus da ficha, onde ela não tem um, e tira o campo.
func TestTheMigrationMovesTheDevoteeToTheGod(t *testing.T) {
	f := newSceneFixture(t)
	ctx := context.Background()
	devotee := seedCharacterAtLevel(t, f.s, f.player, "Devoto", "Clérigo", 1, 0, 0)
	both := seedCharacterAtLevel(t, f.s, f.player, "Os dois", "Clérigo", 1, 0, 0)
	named := seedCharacterAtLevel(t, f.s, f.player, "Pelo nome", "Guerreiro", 1, 0, 0)
	for id, statement := range map[int64]string{
		devotee: `UPDATE characters SET classChoices = '{"Clérigo":{"devoto":"khalmyr","caminho":""}}' WHERE id = ?`,
		both:    `UPDATE characters SET god = 'Lena', classChoices = '{"Clérigo":{"devoto":"khalmyr"}}' WHERE id = ?`,
		named:   `UPDATE characters SET god = 'Lin-Wu' WHERE id = ?`,
	} {
		if _, err := f.s.db.ExecContext(ctx, statement, id); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile("../../infra/db/migrations/00024_god_is_the_devotion.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	if _, err := f.s.db.ExecContext(ctx, up); err != nil {
		t.Fatalf("rodar a migração: %v", err)
	}
	for id, want := range map[int64]string{devotee: "khalmyr", both: "lena", named: "lin-wu"} {
		var god sql.NullString
		var choices string
		if err := f.s.db.QueryRowContext(ctx, `SELECT god, classChoices FROM characters WHERE id = ?`, id).Scan(&god, &choices); err != nil {
			t.Fatal(err)
		}
		if god.String != want {
			t.Errorf("a ficha %d ficou com o deus %q, e devia ser %q", id, god.String, want)
		}
		if strings.Contains(choices, "devoto") {
			t.Errorf("a ficha %d ainda guarda o devoto: %s", id, choices)
		}
	}
}

// Sair da classe não leva o deus: a devoção é da ficha. Só recusa quando o deus valia POR ela.
func TestLeavingTheClassKeepsTheGod(t *testing.T) {
	f, id := guerreiro(t)
	if refusal := enterClass(t, f, id, "Clérigo"); refusal != "" {
		t.Fatalf("entrar: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, devotionPath("khalmyr"), ""); refusal != "" {
		t.Fatalf("escolher o deus: %q", refusal)
	}
	if refusal := stepClass(t, f, id, "Clérigo", -1); refusal != "" {
		t.Fatalf("a devoção não prende a classe: %q", refusal)
	}
	var god string
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT god FROM characters WHERE id = ?", id,
	).Scan(&god); err != nil {
		t.Fatalf("ler o deus: %v", err)
	}
	if god != "khalmyr" {
		t.Errorf("o deus saiu junto com a classe: %q", god)
	}
}

// Tauron morreu (p25), então não é um dos vinte deuses do livro, e a mesa ainda pode servi-lo: ele
// entra na lista sem conceder nada, e o passo diz isso.
func TestTauronIsADevotionThatGrantsNothing(t *testing.T) {
	f, id := guerreiro(t)
	if refusal := powerCommand(t, f, id, devotionPath("tauron"), ""); refusal != "" {
		t.Fatalf("o guerreiro não conseguiu servir a Tauron: %q", refusal)
	}
	if screen := powerScreen(t, f, id); !strings.Contains(screen, "Devoto de Tauron não recebe poder concedido") {
		t.Errorf("o passo não diz que Tauron não concede nada:\n%s", screen)
	}
	refusal := powerCommand(t, f, id, grantedPath(url.PathEscape("Dom da Verdade")), "")
	if !strings.Contains(refusal, "Tauron") {
		t.Errorf("Tauron não concede o Dom da Verdade, e a recusa foi %q", refusal)
	}

	// Clérigo, paladino e druida seguem as listas do livro, e ele não está em nenhuma.
	for _, class := range []string{"Clérigo", "Paladino", "Druida"} {
		cf := newSceneFixture(t)
		classID := seedCharacterAtLevel(t, cf.s, cf.player, "Fiel", class, 1, 0, 0)
		if refusal := powerCommand(t, cf, classID, devotionPath("tauron"), ""); !strings.Contains(refusal, class) {
			t.Errorf("Tauron não está na lista de %s, e a recusa foi %q", class, refusal)
		}
	}
}
