package api

import (
	"context"
	"testing"

	"t20engine/domain/sheet"
)

func levelRows(t *testing.T, f sceneFixture, id int64) []sheet.LevelChoiceDTO {
	t.Helper()
	rows, err := f.s.queries.ListLevelChoicesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as escolhas: %v", err)
	}
	return sheet.LevelChoicesFrom(rows)
}

func rowWith(rows []sheet.LevelChoiceDTO, kind, ref string) (sheet.LevelChoiceDTO, bool) {
	for _, row := range rows {
		if row.Kind == kind && row.Ref == ref {
			return row, true
		}
	}
	return sheet.LevelChoiceDTO{}, false
}

// O gesto de escolher guarda o nível, e a escolha NÃO é estimada: ela aconteceu aqui.
func TestChoosingAPowerRecordsTheLevelItTookASlotIn(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	rows := levelRows(t, f, id)

	pick, found := rowWith(rows, sheet.LevelKindPick, "class.barbaro.aumento-de-atributo")
	if !found || pick.Level != 2 || pick.Estimated {
		t.Errorf("a escolha ficou %+v, esperava o nível 2 e não estimada", pick)
	}
	furia, found := rowWith(rows, sheet.LevelKindAutomatic, "class.barbaro.furia")
	if !found || furia.Level != 1 {
		t.Errorf("a Fúria ficou %+v, esperava automática no nível 1", furia)
	}
	classes := 0
	for _, row := range rows {
		if row.Kind == sheet.LevelKindClass {
			classes++
		}
	}
	if classes != 3 {
		t.Errorf("a ficha tem %d linhas de classe, esperava 3", classes)
	}
}

// A cópia da campanha nasce com o histórico, e mexer nela não mexe no rascunho.
func TestTheCampaignCopyCarriesTheLevelHistory(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("escolher: %q", refusal)
	}
	source := levelRows(t, f, id)

	ctx := context.Background()
	campaign := seedCampaign(t, f.s, f.player)
	if err := f.s.campaignSeating().Seat(ctx, f.player, campaign, id, ""); err != nil {
		t.Fatalf("sentar: %v", err)
	}
	var copyID int64
	if err := f.s.db.QueryRowContext(ctx,
		`SELECT id FROM characters WHERE sourceCharacterId = ? AND campaignId = ?`, id, campaign,
	).Scan(&copyID); err != nil {
		t.Fatalf("achar a cópia: %v", err)
	}

	copied := levelRows(t, f, copyID)
	if len(copied) != len(source) {
		t.Fatalf("a cópia tem %d linhas, e o original %d", len(copied), len(source))
	}
	for i := range source {
		if copied[i].Level != source[i].Level || copied[i].Kind != source[i].Kind ||
			copied[i].Ref != source[i].Ref || copied[i].Pick != source[i].Pick {
			t.Errorf("a linha %d da cópia é %+v, e a do original %+v", i, copied[i], source[i])
		}
	}

	if refusal := powerCommand(t, f, copyID, barbarianIncrease+"wisdom", ""); refusal != "" {
		t.Fatalf("escolher na cópia: %q", refusal)
	}
	if len(levelRows(t, f, id)) != len(source) {
		t.Error("escolher na cópia mexeu no rascunho")
	}
}

// As fichas que vieram da migração não têm nível. O boot as situa, e elas continuam marcadas
// como estimadas, porque o nível de verdade não está registrado em lugar nenhum.
func TestTheBootSituatesMigratedChoicesAndKeepsThemEstimated(t *testing.T) {
	f, id := barbaro(t, 3)
	ctx := context.Background()
	if _, err := f.s.db.ExecContext(ctx,
		`INSERT INTO character_level_choices (characterId, kind, ref, pick, estimated)
		 VALUES (?, 'power', 'class.barbaro.aumento-de-atributo', '', 1),
		        (?, 'pick', 'class.barbaro.aumento-de-atributo', 'strength', 1)`, id, id,
	); err != nil {
		t.Fatalf("simular a migração: %v", err)
	}
	if err := f.s.SyncLevelChoices(ctx); err != nil {
		t.Fatalf("situar: %v", err)
	}
	pick, found := rowWith(levelRows(t, f, id), sheet.LevelKindPick, "class.barbaro.aumento-de-atributo")
	if !found || pick.Level != 2 || !pick.Estimated {
		t.Errorf("a escolha migrada ficou %+v, esperava o nível 2 e estimada", pick)
	}
}
