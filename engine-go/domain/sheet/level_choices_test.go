package sheet

import (
	"fmt"
	"slices"
	"testing"
)

const testIncrease = "class.barbaro.aumento-de-atributo"

func levelsOfKind(rows []LevelChoiceDTO, kind string) []int64 {
	out := []int64{}
	for _, row := range rows {
		if row.Kind == kind {
			out = append(out, row.Level)
		}
	}
	return out
}

// As linhas de classe sem nível entram em ordem, e cada nível de classe a partir do 2º dá uma
// vaga (p33): o 1º nível do Arcanista não dá, o 2º do Guerreiro dá.
func TestTheFirstClassTakesTheFirstLevelsAndChoicesFillTheLowestSlot(t *testing.T) {
	dto := CharacterDTO{
		Classes: []ClassDTO{{ClassName: "Guerreiro", Level: 3}, {ClassName: "Arcanista", Level: 1}},
		LevelChoices: []LevelChoiceDTO{
			{ID: 1, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 2, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 3, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 4, Kind: LevelKindClass, Ref: "Arcanista"},
			{ID: 5, Kind: LevelKindPower, Ref: "class.guerreiro.ataque-especial"},
			{ID: 6, Kind: LevelKindPower, Ref: "class.guerreiro.golpe-de-raspao"},
		},
	}
	rows := PlanLevelRows(dto)

	if got := levelsOfKind(rows, LevelKindClass); !slices.Equal(got, []int64{1, 2, 3, 4}) {
		t.Fatalf("os níveis foram %v", got)
	}
	var classes []string
	for _, row := range rows {
		if row.Kind == LevelKindClass {
			classes = append(classes, row.Ref)
		}
	}
	if classes[0] != "Guerreiro" || classes[2] != "Guerreiro" || classes[3] != "Arcanista" {
		t.Errorf("a sequência saiu %v, esperava a primeira classe primeiro", classes)
	}
	if got := levelsOfKind(rows, LevelKindPower); !slices.Equal(got, []int64{2, 3}) {
		t.Errorf("as escolhas foram para os níveis %v, esperava 2 e 3", got)
	}
}

// O Aumento de Atributo no mesmo atributo procura outro patamar (p38): a segunda Força de um
// nível 6 vai para o 5º, e não para o 3º, que ainda é Iniciante.
func TestTheSameAttributeIncreaseLooksForAnotherTier(t *testing.T) {
	dto := CharacterDTO{
		Classes: []ClassDTO{{ClassName: "Bárbaro", Level: 6}},
		LevelChoices: []LevelChoiceDTO{
			{ID: 1, Kind: LevelKindClass, Ref: "Bárbaro"}, {ID: 2, Kind: LevelKindClass, Ref: "Bárbaro"},
			{ID: 3, Kind: LevelKindClass, Ref: "Bárbaro"}, {ID: 4, Kind: LevelKindClass, Ref: "Bárbaro"},
			{ID: 5, Kind: LevelKindClass, Ref: "Bárbaro"}, {ID: 6, Kind: LevelKindClass, Ref: "Bárbaro"},
			{ID: 7, Kind: LevelKindPower, Ref: testIncrease},
			{ID: 8, Kind: LevelKindPick, Ref: testIncrease, Pick: "strength"},
			{ID: 9, Kind: LevelKindPick, Ref: testIncrease, Pick: "strength"},
		},
	}
	rows := PlanLevelRows(dto)
	if got := levelsOfKind(rows, LevelKindPick); !slices.Equal(got, []int64{2, 5}) {
		t.Errorf("as duas Forças foram para os níveis %v, esperava 2 e 5", got)
	}
	if got := levelsOfKind(rows, LevelKindPower); !slices.Equal(got, []int64{2}) {
		t.Errorf("a linha do poder repetível ficou no nível %v, esperava o da primeira escolha", got)
	}
}

// Quem já tem nível não sai dele, e o plano é idempotente: rodar de novo não muda nada.
func TestAChoiceKeepsItsLevelAndThePlanIsIdempotent(t *testing.T) {
	dto := CharacterDTO{
		Classes: []ClassDTO{{ClassName: "Guerreiro", Level: 4}},
		LevelChoices: []LevelChoiceDTO{
			{ID: 1, Level: 1, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 2, Level: 2, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 3, Level: 3, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 4, Level: 4, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 5, Level: 4, Kind: LevelKindPower, Ref: "class.guerreiro.ataque-especial"},
			{ID: 6, Kind: LevelKindPower, Ref: "class.guerreiro.golpe-de-raspao"},
		},
	}
	first := PlanLevelRows(dto)
	if got := levelsOfKind(first, LevelKindPower); !slices.Equal(got, []int64{4, 2}) {
		t.Fatalf("as escolhas ficaram nos níveis %v, esperava 4 e 2", got)
	}
	dto.LevelChoices = first
	second := PlanLevelRows(dto)
	if len(first) != len(second) {
		t.Fatalf("o plano mudou de %d para %d linhas", len(first), len(second))
	}
	for i := range first {
		if first[i].Level != second[i].Level || first[i].Ref != second[i].Ref || first[i].Kind != second[i].Kind {
			t.Errorf("a linha %d mudou: %+v → %+v", i, first[i], second[i])
		}
	}
}

// Um nível que saiu do meio desce os seguintes, e quem estava nele volta a procurar vaga.
func TestRemovingAClassLevelMovesTheFollowingOnes(t *testing.T) {
	dto := CharacterDTO{
		Classes: []ClassDTO{{ClassName: "Guerreiro", Level: 2}, {ClassName: "Ladino", Level: 2}},
		LevelChoices: []LevelChoiceDTO{
			{ID: 1, Level: 1, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 2, Level: 2, Kind: LevelKindClass, Ref: "Ladino"},
			{ID: 4, Level: 4, Kind: LevelKindClass, Ref: "Guerreiro"},
			{ID: 5, Level: 5, Kind: LevelKindClass, Ref: "Ladino"},
			{ID: 6, Level: 4, Kind: LevelKindPower, Ref: "class.guerreiro.ataque-especial"},
			{ID: 7, Level: 3, Kind: LevelKindPower, Ref: "class.guerreiro.golpe-de-raspao"},
		},
	}
	rows := PlanLevelRows(dto)
	if got := levelsOfKind(rows, LevelKindClass); !slices.Equal(got, []int64{1, 2, 3, 4}) {
		t.Fatalf("os níveis ficaram %v", got)
	}
	// O 4º virou 3º. O que estava no 3º, que saiu, foi para a vaga livre: o 2º do Ladino
	// (nível 4 do personagem).
	if got := levelsOfKind(rows, LevelKindPower); !slices.Equal(got, []int64{3, 4}) {
		t.Errorf("as escolhas ficaram nos níveis %v, esperava 3 e 4", got)
	}
}

// Os textos que o motor lê saem das linhas, e voltar a escrevê-los não muda quem já está lá.
func TestTheBlobsComeFromTheRowsAndTheDiffOnlyTouchesWhatChanged(t *testing.T) {
	rows := []LevelChoiceDTO{
		{ID: 1, Level: 2, Kind: LevelKindPower, Ref: testIncrease},
		{ID: 2, Level: 2, Kind: LevelKindPick, Ref: testIncrease, Pick: "strength"},
		{ID: 3, Level: 5, Kind: LevelKindPick, Ref: testIncrease, Pick: "strength"},
		{ID: 4, Level: 1, Kind: LevelKindAutomatic, Ref: "class.barbaro.furia"},
	}
	powers, picks := choiceBlobsOf(rows)
	if powers != `["`+testIncrease+`"]` || picks != `{"`+testIncrease+`":["strength","strength"]}` {
		t.Fatalf("os textos saíram %s e %s", powers, picks)
	}
	if diff := DiffChoiceBlobs(rows, &powers, &picks); len(diff.RemoveIDs)+len(diff.Add) != 0 {
		t.Errorf("reescrever o mesmo texto mudou as linhas: %+v", diff)
	}

	// Tirar UMA Força apaga uma linha só, a mais recente.
	fewer := `{"` + testIncrease + `":["strength"]}`
	diff := DiffChoiceBlobs(rows, nil, &fewer)
	if len(diff.RemoveIDs) != 1 || diff.RemoveIDs[0] != 3 || len(diff.Add) != 0 {
		t.Errorf("tirar uma Força deu %+v, esperava apagar a linha 3", diff)
	}
}

// O poder SEM NÍVEL vai para a vaga mais baixa em que ele PODE estar: não antes do nível mínimo
// dele, nem antes do poder que ele pede (p38). O Mestre em Escola pede o 8º nível de arcanista
// e o Especialista em Escola; o Arcanista Pleno da semente saía com o Mestre no 2º nível e o
// Especialista no 3º. As linhas são as da semente, na ordem dela: o poder e a escola de cada um.
func TestAPowerWithoutLevelWaitsForItsMinimumAndItsPrerequisite(t *testing.T) {
	const specialist, master = "class.arcanista.especialista-em-escola", "class.arcanista.mestre-em-escola"
	choices := []LevelChoiceDTO{}
	for i := int64(1); i <= 10; i++ {
		choices = append(choices, LevelChoiceDTO{ID: i, Kind: LevelKindClass, Ref: "Arcanista"})
	}
	choices = append(choices,
		LevelChoiceDTO{ID: 11, Kind: LevelKindPower, Ref: specialist},
		LevelChoiceDTO{ID: 12, Kind: LevelKindPower, Ref: master},
		LevelChoiceDTO{ID: 13, Kind: LevelKindPick, Ref: specialist, Pick: "evocacao"},
		LevelChoiceDTO{ID: 14, Kind: LevelKindPick, Ref: master, Pick: "evocacao"},
	)
	rows := PlanLevelRows(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 10}}, LevelChoices: choices})

	levelOf := map[string]int64{}
	for _, row := range rows {
		if row.Kind == LevelKindPower || row.Kind == LevelKindPick {
			levelOf[row.Kind+" "+row.Ref] = row.Level
		}
	}
	// O CONTROLE: os dois poderes ganharam nível. Sem ele, "o Mestre não está no 2º" seria
	// verdade com o Mestre sem nível nenhum.
	if levelOf["power "+specialist] == 0 || levelOf["power "+master] == 0 {
		t.Fatalf("um dos poderes ficou sem nível: %v", levelOf)
	}
	if levelOf["power "+master] < 8 {
		t.Errorf("o Mestre em Escola ficou no %dº nível, e ele pede o 8º de arcanista (p38)", levelOf["power "+master])
	}
	if levelOf["power "+specialist] >= levelOf["power "+master] {
		t.Errorf("o Especialista em Escola ficou no %dº nível e o Mestre no %dº, e o Mestre pede o Especialista antes (p38)",
			levelOf["power "+specialist], levelOf["power "+master])
	}
	if levelOf["pick "+master] != levelOf["power "+master] {
		t.Errorf("a escola do Mestre ficou no %dº nível e o poder no %dº: a escolha anda com o poder",
			levelOf["pick "+master], levelOf["power "+master])
	}
}

// O PRÉ-REQUISITO decide sozinho quando o nível mínimo já não separa os dois: do 2º ao 8º nível
// a ficha está cheia, e sobram o 9º e o 10º. O Mestre em Escola, gravado primeiro, pegaria o 9º
// pelo nível mínimo; ele espera o Especialista, que fica com o 9º, e vai para o 10º (p38).
func TestAPowerWithoutLevelGoesAfterThePowerItAsks(t *testing.T) {
	const specialist, master = "class.arcanista.especialista-em-escola", "class.arcanista.mestre-em-escola"
	choices := []LevelChoiceDTO{}
	for i := int64(1); i <= 10; i++ {
		choices = append(choices, LevelChoiceDTO{ID: i, Kind: LevelKindClass, Ref: "Arcanista"})
	}
	for level := int64(2); level <= 8; level++ {
		choices = append(choices, LevelChoiceDTO{ID: 100 + level, Kind: LevelKindPower, Level: level, Ref: fmt.Sprintf("poder-do-%d", level)})
	}
	choices = append(choices,
		LevelChoiceDTO{ID: 11, Kind: LevelKindPower, Ref: master},
		LevelChoiceDTO{ID: 12, Kind: LevelKindPick, Ref: master, Pick: "evocacao"},
		LevelChoiceDTO{ID: 13, Kind: LevelKindPower, Ref: specialist},
		LevelChoiceDTO{ID: 14, Kind: LevelKindPick, Ref: specialist, Pick: "evocacao"},
	)
	rows := PlanLevelRows(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 10}}, LevelChoices: choices})

	levelOf := map[string]int64{}
	for _, row := range rows {
		if row.Kind == LevelKindPower {
			levelOf[row.Ref] = row.Level
		}
	}
	if levelOf[specialist] != 9 || levelOf[master] != 10 {
		t.Errorf("o Especialista ficou no %dº e o Mestre no %dº; com o 2º ao 8º ocupados, eram o 9º e o 10º (p38)",
			levelOf[specialist], levelOf[master])
	}
}
