package api

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// seedExpertiseRows dá à ficha as 29 linhas de perícia que a criação grava.
func seedExpertiseRows(t *testing.T, f sceneFixture, id int64) {
	t.Helper()
	for _, expertise := range sheet.BuiltinExpertises() {
		seedPericia(t, f.s, id, expertise.Name, expertise.Attribute, false)
	}
}

// trainingStepTag é o que a lista do Escolher diz na ponta do passo de perícias.
func trainingStepTag(t *testing.T, page, class string) string {
	t.Helper()
	tag := regexp.MustCompile(`(?s)` + class + ` · perícias</span>.*?ds-step-tag[^>]*>([^<]*)<`).FindStringSubmatch(page)
	if tag == nil {
		t.Fatalf("a lista do Escolher não tem o passo de perícias de %s", class)
	}
	return tag[1]
}

// trainingStepOf é o passo aberto, onde moram as opções.
func trainingStepOf(t *testing.T, page, class string) string {
	t.Helper()
	section := regexp.MustCompile(`(?s)<section[^>]*id="choice-step-class-` + class + `-training".*?</section>`).FindString(page)
	if section == "" {
		t.Fatalf("o Escolher não desenhou o passo de perícias de %s", class)
	}
	return section
}

// offersExpertise diz se o passo tem o gesto que treina esta perícia.
func offersExpertise(section, name string) bool {
	return strings.Contains(section, "/pericias/treino/"+url.PathEscape(name))
}

func trainExpertise(t *testing.T, f sceneFixture, id int64, name string) {
	t.Helper()
	if refusal := sheetPost(t, f, id, "/pericias/treino/"+url.PathEscape(name), ""); refusal != "" {
		t.Fatalf("treinar %q foi recusado: %q", name, refusal)
	}
}

func setIntelligence(t *testing.T, f sceneFixture, id, value int64) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET intelligence = ? WHERE id = ?`, value, id); err != nil {
		t.Fatalf("semear a Inteligência: %v", err)
	}
}

// O Bárbaro escolhe 4 perícias da lista dele (p40), e Fortitude e Luta já vêm.
func TestTheBarbarianPicksFourExpertisesInTheChoicesDialog(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)

	page := parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Bárbaro"); tag != "0 de 4" {
		t.Errorf("o passo diz %q, e o Bárbaro escolhe 4 (p40)", tag)
	}
	section := trainingStepOf(t, page, "Bárbaro")
	if !offersExpertise(section, "Adestramento") {
		t.Error("Adestramento é da lista do Bárbaro e não foi oferecida")
	}
	if offersExpertise(section, "Fortitude") {
		t.Error("Fortitude já vem com a classe e não se escolhe")
	}
	if offersExpertise(section, "Cura") {
		t.Error("Cura está fora da lista, e sem Inteligência não há como escolhê-la")
	}

	trainExpertise(t, f, id, "Adestramento")
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "1 de 4" {
		t.Errorf("depois de uma escolha o passo diz %q", tag)
	}

	for _, name := range []string{"Atletismo", "Cavalgar", "Iniciativa"} {
		trainExpertise(t, f, id, name)
	}
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "feito" {
		t.Errorf("com as 4 escolhidas o passo diz %q", tag)
	}
}

// "Você também recebe um número de perícias treinadas igual a sua Inteligência.
// Perícias ganhas por Inteligência não precisam pertencer à lista de sua classe." (p114)
func TestIntelligenceAddsExpertisesFromOutsideTheClassList(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	setIntelligence(t, f, id, 2)

	page := parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Bárbaro"); tag != "0 de 6" {
		t.Errorf("com Inteligência 2 o passo diz %q, e são 4 da classe mais 2", tag)
	}
	if !offersExpertise(trainingStepOf(t, page, "Bárbaro"), "Cura") {
		t.Error("com Inteligência, Cura pode ser escolhida fora da lista (p114)")
	}

	trainExpertise(t, f, id, "Cura")
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "1 de 6" {
		t.Errorf("a perícia de fora da lista não contou: o passo diz %q", tag)
	}
}

// A quinta perícia da lista conta como a da Inteligência: a ficha que já foi marcada
// à mão não precisa ser desfeita para fechar a conta.
func TestAnExtraPickFromTheListCountsForIntelligence(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	names := []string{"Adestramento", "Atletismo", "Cavalgar", "Iniciativa", "Percepção"}
	seedExpertiseRows(t, f, id)
	setIntelligence(t, f, id, 1)

	for _, name := range names {
		trainExpertise(t, f, id, name)
	}

	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "feito" {
		t.Errorf("4 da classe mais 1 de Inteligência deu %q", tag)
	}
}

// O Guerreiro escolhe Luta ou Pontaria, e mais 2 da lista.
func TestTheWarriorAlsoPicksBetweenTwoExpertises(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 1, 0, 0)
	seedExpertiseRows(t, f, id)

	page := parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Guerreiro"); tag != "0 de 3" {
		t.Errorf("o passo diz %q, e são a escolha entre duas mais 2 da lista", tag)
	}
	section := trainingStepOf(t, page, "Guerreiro")
	if !offersExpertise(section, "Luta") || !offersExpertise(section, "Pontaria") {
		t.Error("o passo não oferece as duas opções da escolha")
	}

	trainExpertise(t, f, id, "Luta")
	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Guerreiro"); tag != "1 de 3" {
		t.Errorf("escolher Luta deu %q", tag)
	}
}

// A multiclasse "não ganha as perícias treinadas ou proficiências da nova classe" (p35).
func TestOnlyTheFirstClassAsksForExpertises(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedClasse(t, f.s, id, "Guerreiro", 1)

	page := parallelSheet(t, f, id, "abilities")
	seedExpertiseRows(t, f, id)
	page = parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Bárbaro"); tag != "0 de 4" {
		t.Errorf("o passo diz %q, e a regra é a do Bárbaro, que escolhe 4", tag)
	}
	if strings.Contains(page, "Guerreiro · perícias") {
		t.Error("a segunda classe pediu perícias, e a multiclasse não as ganha (p35)")
	}
}

// Perícia que a origem já treinou não gasta uma escolha.
func TestAnExpertiseTrainedByTheOriginIsNotOffered(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET originChoices = '["origin-soldado-pericia-pontaria"]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear o benefício da origem: %v", err)
	}

	section := trainingStepOf(t, parallelSheet(t, f, id, "abilities"), "Bárbaro")
	if offersExpertise(section, "Pontaria") {
		t.Error("Pontaria já é treinada pela origem e não devia gastar uma escolha")
	}
	if !offersExpertise(section, "Adestramento") {
		t.Error("o controle sumiu: Adestramento devia continuar oferecida")
	}
}

// Um Ofício criado na ficha conta como o Ofício da lista da classe.
func TestACraftOnTheSheetCountsAsTheCraftOfTheList(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	seedOficio(t, f.s, id, "Ofício (ferreiro)", "intelligence")

	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "1 de 4" {
		t.Errorf("o Ofício treinado não contou: o passo diz %q", tag)
	}
}

// A perícia inventada não é do livro, e o Ofício é uma das 29 (p115).
func TestAnInventedExpertiseIsOutOfTheCount(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	seedOficio(t, f.s, id, "Marinheiro", "intelligence")

	page := parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Bárbaro"); tag != "0 de 4" {
		t.Errorf("a perícia inventada contou: o passo diz %q", tag)
	}
	if offersExpertise(trainingStepOf(t, page, "Bárbaro"), "Marinheiro") {
		t.Error("a perícia inventada apareceu como escolha da classe")
	}
}

// O Ofício é fixo no Inventor, e um Ofício criado à parte conta na Inteligência.
func TestAFixedCraftDoesNotConsumeACreatedCraft(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Inventor", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	seedOficio(t, f.s, id, "Ofício (ferreiro)", "intelligence")
	setIntelligence(t, f, id, 2)

	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Inventor"); tag != "1 de 6" {
		t.Errorf("com Inteligência 2 e um Ofício criado, o passo diz %q", tag)
	}
}

// O Treinamento em Perícia (p131) grava na mesma coluna, e não é escolha da classe.
func TestAnExpertiseTrainedByAPowerDoesNotCountAsAClassPick(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 2, 0, 0)
	seedExpertiseRows(t, f, id)
	if refusal := sheetPost(t, f, id, "/poderes/escolha/treinamento-em-pericia/Cavalgar", ""); refusal != "" {
		t.Fatalf("escolher o Treinamento em Perícia foi recusado: %q", refusal)
	}

	page := parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Bárbaro"); tag != "0 de 4" {
		t.Errorf("a perícia do poder contou: o passo diz %q", tag)
	}
	if offersExpertise(trainingStepOf(t, page, "Bárbaro"), "Cavalgar") {
		t.Error("o passo ofereceu a perícia que o poder treinou, e destreiná-la é recusado")
	}
}

// A Memória Póstuma (p29) também grava na coluna.
func TestTheExpertiseOfThePosthumousMemoryDoesNotCount(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedRaca(t, f.s, id, "osteon")
	seedExpertiseRows(t, f, id)
	if refusal := raceAbilityCommand(t, f, id, "osteon-memoria-postuma", "Cavalgar"); refusal != "" {
		t.Fatalf("escolher na Memória Póstuma foi recusado: %q", refusal)
	}

	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "0 de 4" {
		t.Errorf("a perícia da Memória Póstuma contou: o passo diz %q", tag)
	}
}

// A origem treinou e a coluna também está marcada: continua sem gastar escolha.
func TestAnExpertiseOfTheOriginMarkedByHandDoesNotCount(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET originChoices = '["origin-soldado-pericia-pontaria"]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear o benefício da origem: %v", err)
	}
	trainExpertise(t, f, id, "Pontaria")

	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "0 de 4" {
		t.Errorf("a perícia da origem contou: o passo diz %q", tag)
	}
}

// Se a origem treina as duas opções, não sobra o que escolher, e a pendência não
// pode ficar aberta para sempre.
func TestTheOriginTrainingBothOptionsClosesTheChoice(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET originChoices = '["origin-soldado-pericia-luta","origin-soldado-pericia-pontaria"]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear os benefícios da origem: %v", err)
	}

	page := parallelSheet(t, f, id, "abilities")
	if tag := trainingStepTag(t, page, "Guerreiro"); tag != "0 de 2" {
		t.Errorf("o passo diz %q, e só sobram as 2 da lista", tag)
	}
	if strings.Contains(trainingStepOf(t, page, "Guerreiro"), "Luta ou Pontaria") {
		t.Error("o passo desenhou uma escolha sem nenhuma opção")
	}
}

// "Se sua Inteligência for positiva": a negativa não tira perícia da classe (p17).
func TestANegativeIntelligenceTakesNothingFromTheClass(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	seedExpertiseRows(t, f, id)
	setIntelligence(t, f, id, -2)

	if tag := trainingStepTag(t, parallelSheet(t, f, id, "abilities"), "Bárbaro"); tag != "0 de 4" {
		t.Errorf("com Inteligência −2 o passo diz %q", tag)
	}
}

// A ficha sem nenhuma linha de perícia não tem o que escolher, e não ganha um passo
// que nunca fecharia.
func TestASheetWithoutExpertiseRowsHasNoStep(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)

	if strings.Contains(parallelSheet(t, f, id, "abilities"), "Bárbaro · perícias") {
		t.Error("a ficha sem linhas de perícia ganhou o passo")
	}
}

// O crachá do Escolher conta o treino que falta, e deixa de contar quando ele acaba.
func TestTheMissingTrainingCountsOnTheChoicesBadge(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Korg", "Bárbaro", 1, 0, 0)
	names := []string{"Adestramento", "Atletismo", "Cavalgar", "Iniciativa"}
	seedExpertiseRows(t, f, id)
	badge := regexp.MustCompile(`class="ds-badge"[^>]*>(\d+)<`)

	before := badge.FindStringSubmatch(parallelSheet(t, f, id, "abilities"))
	if before == nil {
		t.Fatal("o Escolher não mostrou crachá de pendências")
	}
	for _, name := range names {
		trainExpertise(t, f, id, name)
	}
	after := badge.FindStringSubmatch(parallelSheet(t, f, id, "abilities"))
	remaining := 0
	if after != nil {
		remaining, _ = strconv.Atoi(after[1])
	}
	pending, _ := strconv.Atoi(before[1])
	if remaining != pending-1 {
		t.Errorf("o crachá foi de %d para %d, e devia cair um com o treino feito", pending, remaining)
	}
}
