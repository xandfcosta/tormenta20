package campaign

import (
	"context"
	"database/sql"
	"fmt"
	"t20engine/infra/db/dbvalue"
)

// cloneCharacterTx é o clone SEM dono da transação, para quem precisa que a
// cópia e o que vem depois dela caiam ou vivam juntos.
//
// Existe por causa da ALE-156: a entrada na mesa clonava numa transação e
// inseria o membro em outra, então um `CreateMember` que falhasse deixava a
// cópia órfã — e a cópia órfã é pior que nada, porque a deduplicação passa a
// responder "já está na mesa" e o herói fica impedido de entrar PARA SEMPRE,
// sem membro nenhum para remover.
func cloneCharacterTx(ctx context.Context, tx *sql.Tx, sourceID, campaignID int64) (int64, error) {
	now := dbvalue.NowISO()

	res, err := tx.ExecContext(ctx, `
INSERT INTO characters (
  ownerId, name, origin, god, godPower, tibar, level,
  strength, dexterity, constitution, intelligence, wisdom, charisma,
  size, displacement, proficiencies, raceAbilityChoices, raceAttributeChoices,
  secondaryRaceChoices, originChoices, classPowers, classChoices, powerChoices,
  activeConditions, sourceCharacterId, campaignId, createdAt, updatedAt)
SELECT
  ownerId, name, origin, god, godPower, tibar, level,
  strength, dexterity, constitution, intelligence, wisdom, charisma,
  size, displacement, proficiencies, raceAbilityChoices, raceAttributeChoices,
  secondaryRaceChoices, originChoices, classPowers, classChoices, powerChoices,
  activeConditions, ?, ?, ?, ?
FROM characters WHERE id = ?`, sourceID, campaignID, now, now, sourceID)
	if err != nil {
		return 0, fmt.Errorf("clonar a ficha do molde %d: %w", sourceID, err)
	}
	destID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// AS TABELAS FILHAS, uma por `INSERT…SELECT` sob o id novo. As que têm
	// carimbo próprio — itens, efeitos, magias — levam um `now` fresco: a cópia
	// nasce agora, e herdar o carimbo do molde faria a mesa dizer que o item
	// entrou na mochila antes de o herói existir nela.
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
		// O DANO vem junto, e é o que faz um herói ferido entrar na mesa ferido.
		// Ele era copiado nas colunas `hpCurrent`/`mpCurrent` do INSERT acima,
		// que saíram na 00015: o poço é derivado e o que se guarda é a dívida.
		// Sem esta linha, entrar numa campanha viraria uma cura (ALE-355).
		{"damage", `INSERT INTO character_damage (characterId, hpDamage, mpSpent)
			SELECT ?, hpDamage, mpSpent FROM character_damage WHERE characterId = ?`, []any{destID, sourceID}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.sql, step.args...); err != nil {
			return 0, fmt.Errorf("clonar %s do molde %d: %w", step.what, sourceID, err)
		}
	}

	return destID, nil
}
