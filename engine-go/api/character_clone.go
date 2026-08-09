package api

import (
	"context"
	"fmt"
)

// cloneCharacterForCampaign snapshots a character — its row plus every child
// table (races, classes, expertises, items, active effects, spells) — into a
// new campaign-scoped copy, so edits during play stay isolated to that mesa
// (ALE-33). The copy stamps `sourceCharacterId` (the template it came from) and
// `campaignId`. Raw INSERT…SELECT keeps every column verbatim; sqlc's SQLite
// parser rejects RETURNING on INSERT…SELECT, so we read the new id via
// LastInsertId. All inserts run in one transaction.
func (s *Server) cloneCharacterForCampaign(ctx context.Context, sourceID, campaignID int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	now := nowISO()

	res, err := tx.ExecContext(ctx, `
INSERT INTO characters (
  ownerId, name, origin, god, godPower, tibar, level,
  hpMax, hpCurrent, mpMax, mpCurrent,
  strength, dexterity, constitution, intelligence, wisdom, charisma,
  size, displacement, proficiencies, raceAbilityChoices, raceAttributeChoices,
  secondaryRaceChoices, originChoices, classPowers, classChoices, powerChoices,
  activeConditions, sourceCharacterId, campaignId, createdAt, updatedAt)
SELECT
  ownerId, name, origin, god, godPower, tibar, level,
  hpMax, hpCurrent, mpMax, mpCurrent,
  strength, dexterity, constitution, intelligence, wisdom, charisma,
  size, displacement, proficiencies, raceAbilityChoices, raceAttributeChoices,
  secondaryRaceChoices, originChoices, classPowers, classChoices, powerChoices,
  activeConditions, ?, ?, ?, ?
FROM characters WHERE id = ?`, sourceID, campaignID, now, now, sourceID)
	if err != nil {
		return 0, fmt.Errorf("clone character row (source %d): %w", sourceID, err)
	}
	destID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Child tables — INSERT…SELECT copies each row under the new characterId.
	// Rows with their own createdAt (items, effects, spells) take a fresh `now`.
	steps := []struct {
		what string
		sql  string
		args []any
	}{
		{"races", `INSERT INTO character_races (characterId, race)
			SELECT ?, race FROM character_races WHERE characterId = ?`, []any{destID, sourceID}},
		{"classes", `INSERT INTO character_classes (characterId, className, level)
			SELECT ?, className, level FROM character_classes WHERE characterId = ?`, []any{destID, sourceID}},
		{"expertises", `INSERT INTO character_expertises (characterId, name, attribute, trained, custom)
			SELECT ?, name, attribute, trained, custom FROM character_expertises WHERE characterId = ?`, []any{destID, sourceID}},
		{"items", `INSERT INTO character_items (characterId, catalogId, name, quantity, slots, equipped, improvements, material, createdAt)
			SELECT ?, catalogId, name, quantity, slots, equipped, improvements, material, ? FROM character_items WHERE characterId = ?`, []any{destID, now, sourceID}},
		{"effects", `INSERT INTO active_effects (characterId, source, catalogId, scope, modifiers, createdAt)
			SELECT ?, source, catalogId, scope, modifiers, ? FROM active_effects WHERE characterId = ?`, []any{destID, now, sourceID}},
		{"spells", `INSERT INTO character_spells (characterId, catalogSpellId, prepared, learnedAt)
			SELECT ?, catalogSpellId, prepared, ? FROM character_spells WHERE characterId = ?`, []any{destID, now, sourceID}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.sql, step.args...); err != nil {
			return 0, fmt.Errorf("clone %s (source %d): %w", step.what, sourceID, err)
		}
	}

	return destID, tx.Commit()
}

// campaignHasCopyOf reports whether `sourceID` was already snapshotted into
// `campaignID` — the snapshot-model dedupe (a template joins a mesa once).
func (s *Server) campaignHasCopyOf(ctx context.Context, sourceID, campaignID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM characters WHERE sourceCharacterId = ? AND campaignId = ?)`,
		sourceID, campaignID).Scan(&exists)
	return exists, err
}
