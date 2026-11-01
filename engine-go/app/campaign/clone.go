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
  secondaryRaceChoices, originChoices, classChoices,
  activeConditions, sourceCharacterId, campaignId, createdAt, updatedAt)
SELECT
  ownerId, name, origin, god, godPower, tibar, level,
  strength, dexterity, constitution, intelligence, wisdom, charisma,
  size, displacement, proficiencies, raceAbilityChoices, raceAttributeChoices,
  secondaryRaceChoices, originChoices, classChoices,
  '[]', ?, ?, ?, ?
FROM characters WHERE id = ?`, sourceID, campaignID, now, now, sourceID)
	if err != nil {
		return 0, fmt.Errorf("clonar a ficha do molde %d: %w", sourceID, err)
	}
	destID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// AS TABELAS FILHAS, uma por `INSERT…SELECT` sob o id novo. Dano, efeitos e condições NÃO
	// vêm: o que se gastou no rascunho fica nele, e a cópia nasce inteira (ALE-564). As que têm
	// carimbo próprio — itens, magias — levam um `now` fresco: a cópia
	// nasce agora, e herdar o carimbo do molde faria a mesa dizer que o item
	// entrou na mochila antes de o herói existir nela.
	steps := []struct {
		what string
		sql  string
		args []any
	}{
		{"races", `INSERT INTO character_races (characterId, race)
			SELECT ?, race FROM character_races WHERE characterId = ?`, []any{destID, sourceID}},
		{"expertises", `INSERT INTO character_expertises (characterId, name, attribute, trained, custom)
			SELECT ?, name, attribute, trained, custom FROM character_expertises WHERE characterId = ?`, []any{destID, sourceID}},
		{"items", `INSERT INTO character_items (characterId, catalogId, name, quantity, slots, equipped, improvements, material, createdAt)
			SELECT ?, catalogId, name, quantity, slots, equipped, improvements, material, ? FROM character_items WHERE characterId = ?`, []any{destID, now, sourceID}},
		{"spells", `INSERT INTO character_spells (characterId, catalogSpellId, prepared, learnedAt)
			SELECT ?, catalogSpellId, prepared, ? FROM character_spells WHERE characterId = ?`, []any{destID, now, sourceID}},
		// O HISTÓRICO POR NÍVEL vem junto: é daqui que saem os poderes da ficha, e a cópia
		// tem de poder trocar a escolha de um nível como o original.
		{"level choices", `INSERT INTO character_level_choices (characterId, level, kind, ref, pick, estimated)
			SELECT ?, level, kind, ref, pick, estimated FROM character_level_choices WHERE characterId = ? ORDER BY id`,
			[]any{destID, sourceID}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.sql, step.args...); err != nil {
			return 0, fmt.Errorf("clonar %s do molde %d: %w", step.what, sourceID, err)
		}
	}

	return destID, nil
}
