package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// O DEUS DO SÍMBOLO SAGRADO (p157): a folha do item pede o deus, o da ficha vem primeiro, e a frase diz se
// o +1 em resistência vale. Quem escolhe é o jogador, e trocar é escolher de novo.

// clericOf é um clérigo de `level`, devoto de Wynna, o único personagem do jogador na mesa.
func clericOf(t *testing.T, level int64, powers string) sceneFixture {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)
	id := seedCharacterAtLevel(t, s, player, "Clériga de Wynna", "Clérigo", level, 0, 0)
	seedMember(t, s, campaignID, id)
	f := sceneFixture{s: s, gm: gm, player: player, campaignID: campaignID, sessionID: sessionID, charID: id}
	if powers != "" {
		seedPowerChoices(t, s, id, powers, `{}`)
	}
	devoteOf(t, f, id, "wynna", "Bênção do Mana")
	return f
}

func storedItemChoice(t *testing.T, f sceneFixture, item int64) string {
	t.Helper()
	var choice sql.NullString
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT choice FROM character_item_choices WHERE itemId = ?", item).Scan(&choice); err != nil && err != sql.ErrNoRows {
		t.Fatalf("ler a escolha do item %d: %v", item, err)
	}
	return choice.String
}

func TestTheHolySymbolSheetChoosesTheGod(t *testing.T) {
	f := clericOf(t, 1, "")
	symbol := itemSemeia(t, f, f.charID, "simbolo-sagrado", "Símbolo sagrado", "vested")
	godPath := func(god string) string { return fmt.Sprintf("/personagens/%d/itens/%d/deus/%s", f.charID, symbol, god) }

	sheet := itemSheetOfBag(t, f, f.charID, symbol)
	if !strings.Contains(sheet, godPath("wynna")) || !strings.Contains(sheet, "Wynna · o seu") {
		t.Error("a folha do símbolo não oferece Wynna, o deus da ficha, primeiro")
	}
	if !strings.Contains(sheet, godPath("khalmyr")) {
		t.Error("a folha do símbolo não oferece os outros deuses")
	}
	if !strings.Contains(sheet, "Empunhar (1 mão)") {
		t.Error("o símbolo sagrado também se empunha (p157), e a folha não oferece")
	}
	if !strings.Contains(sheet, "p157") {
		t.Error("a folha do símbolo não cita a p157, onde o livro o descreve")
	}

	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/deus/khalmyr", symbol)); refusal != "" {
		t.Fatalf("escolher Khalmyr foi recusado: %q", refusal)
	}
	if got := storedItemChoice(t, f, symbol); got != "khalmyr" {
		t.Errorf("a escolha gravada foi %q; want khalmyr", got)
	}
	if !strings.Contains(itemSheetOfBag(t, f, f.charID, symbol), "Khalmyr não é o deus da ficha") {
		t.Error("com o símbolo de outro deus, a folha não diz que o +1 não vale")
	}

	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/deus/zeus", symbol)); !strings.Contains(refusal, "deus") {
		t.Errorf("um deus que não existe deu %q; want a recusa", refusal)
	}
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "")
	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/deus/wynna", sword)); !strings.Contains(refusal, "não pede deus") {
		t.Errorf("escolher deus numa espada deu %q; want a recusa", refusal)
	}
}

// O Símbolo Sagrado Energizado (p58): o Usar cobra 1 PM, e até o fim da cena as magias divinas custam 1 a
// menos enquanto o símbolo do deus da ficha estiver na mão. A Tempestade Divina é de 2º círculo: 3 PM.
func TestTheEnergizedSymbolCutsTheDivineCast(t *testing.T) {
	f := clericOf(t, 5, `["class.clerigo.simbolo-sagrado-energizado"]`)
	symbol := itemSemeia(t, f, f.charID, "simbolo-sagrado", "Símbolo sagrado", "wielded")
	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/deus/wynna", symbol)); refusal != "" {
		t.Fatalf("escolher Wynna foi recusado: %q", refusal)
	}
	learnSpell(t, f, "tempestade-divina")
	spell(t, f, f.charID, "prepara/tempestade-divina")

	before := mpOf(t, f.s, f.charID)
	if refusal := powerCommand(t, f, f.charID, "usa/class.clerigo.simbolo-sagrado-energizado", ""); refusal != "" {
		t.Fatalf("energizar o símbolo foi recusado: %q", refusal)
	}
	if spent := before - mpOf(t, f.s, f.charID); spent != 1 {
		t.Errorf("energizar cobrou %d PM; a p58 diz 1", spent)
	}

	before = mpOf(t, f.s, f.charID)
	if refusal := castFromTheSheet(t, f, "tempestade-divina"); refusal != "" {
		t.Fatalf("conjurar a Tempestade Divina foi recusado: %q", refusal)
	}
	if spent := before - mpOf(t, f.s, f.charID); spent != 2 {
		t.Errorf("com o símbolo energizado na mão, a Tempestade Divina custou %d PM; want 2", spent)
	}
}
