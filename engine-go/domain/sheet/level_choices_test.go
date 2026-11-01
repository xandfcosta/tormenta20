package sheet

import "testing"

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

func sameLevels(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

	if got := levelsOfKind(rows, LevelKindClass); !sameLevels(got, []int64{1, 2, 3, 4}) {
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
	if got := levelsOfKind(rows, LevelKindPower); !sameLevels(got, []int64{2, 3}) {
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
	if got := levelsOfKind(rows, LevelKindPick); !sameLevels(got, []int64{2, 5}) {
		t.Errorf("as duas Forças foram para os níveis %v, esperava 2 e 5", got)
	}
	if got := levelsOfKind(rows, LevelKindPower); !sameLevels(got, []int64{2}) {
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
	if got := levelsOfKind(first, LevelKindPower); !sameLevels(got, []int64{4, 2}) {
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
	if got := levelsOfKind(rows, LevelKindClass); !sameLevels(got, []int64{1, 2, 3, 4}) {
		t.Fatalf("os níveis ficaram %v", got)
	}
	// O 4º virou 3º. O que estava no 3º, que saiu, foi para a vaga livre: o 2º do Ladino
	// (nível 4 do personagem).
	if got := levelsOfKind(rows, LevelKindPower); !sameLevels(got, []int64{3, 4}) {
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
