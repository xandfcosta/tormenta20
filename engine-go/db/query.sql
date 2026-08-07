-- Queries compiled by sqlc into db/sqlcgen. One camelCase column set means the
-- generated json tags already match the frontend contract (hpMax, catalogSpellId).
-- Grouped by domain; grows per Fase B slice.

-- users / auth (B.2)

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (email, name, passwordHash, createdAt, updatedAt)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- characters read model (B.3)

-- name: ListCharactersByOwner :many
SELECT * FROM characters WHERE ownerId = ? ORDER BY updatedAt DESC;

-- name: GetCharacter :one
SELECT * FROM characters WHERE id = ? LIMIT 1;

-- name: GetCharacterOwner :one
SELECT ownerId FROM characters WHERE id = ? LIMIT 1;

-- name: IsCampaignGmForCharacter :one
SELECT EXISTS (
  SELECT 1 FROM campaign_members m
  JOIN campaigns c ON c.id = m.campaignId
  WHERE m.characterId = ? AND c.ownerId = ?
) AS isGm;

-- name: ListRacesByCharacter :many
SELECT race FROM character_races WHERE characterId = ? ORDER BY id ASC;

-- name: ListClassesByCharacter :many
SELECT className, level FROM character_classes WHERE characterId = ? ORDER BY id ASC;

-- name: ListExpertisesByCharacter :many
SELECT name, attribute, trained, custom FROM character_expertises WHERE characterId = ? ORDER BY name ASC;

-- name: ListItemsByCharacter :many
SELECT id, catalogId, name, quantity, slots, equipped, improvements, material
FROM character_items WHERE characterId = ? ORDER BY id ASC;

-- name: ListActiveEffectsByCharacter :many
SELECT id, catalogId, scope, modifiers, createdAt
FROM active_effects WHERE characterId = ? ORDER BY id ASC;

-- name: ListSpellsByCharacter :many
SELECT id, catalogSpellId, prepared, learnedAt
FROM character_spells WHERE characterId = ? ORDER BY learnedAt ASC;

-- character mutations (B.3)

-- name: UpdateVitals :one
UPDATE characters
SET hpCurrent = COALESCE(sqlc.narg('hpCurrent'), hpCurrent),
    mpCurrent = COALESCE(sqlc.narg('mpCurrent'), mpCurrent),
    updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id')
RETURNING hpCurrent, mpCurrent;

-- name: CreateItem :one
INSERT INTO character_items (characterId, catalogId, name, quantity, slots, equipped, improvements, material, createdAt)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, catalogId, name, quantity, slots, equipped, improvements, material;

-- name: GetItem :one
SELECT id, characterId, catalogId, name, quantity, slots, equipped, improvements, material
FROM character_items WHERE id = ? LIMIT 1;

-- name: DeleteItem :exec
DELETE FROM character_items WHERE id = ?;

-- name: ListEquippedItems :many
SELECT id, equipped FROM character_items WHERE characterId = ? AND equipped IS NOT NULL;

-- name: CreateSpell :one
INSERT INTO character_spells (characterId, catalogSpellId, prepared, learnedAt)
VALUES (?, ?, ?, ?)
RETURNING id, characterId, catalogSpellId, prepared, learnedAt;

-- name: SetSpellPreparedByCatalog :one
UPDATE character_spells SET prepared = sqlc.arg('prepared')
WHERE characterId = sqlc.arg('characterId') AND catalogSpellId = sqlc.arg('catalogSpellId')
RETURNING id, characterId, catalogSpellId, prepared, learnedAt;

-- name: DeleteSpell :execrows
DELETE FROM character_spells WHERE characterId = ? AND catalogSpellId = ?;

-- name: UpdateConditions :exec
UPDATE characters SET activeConditions = sqlc.arg('activeConditions'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: GetExpertiseMeta :one
SELECT id, custom FROM character_expertises WHERE characterId = ? AND name = ? LIMIT 1;

-- name: CreateExpertise :one
INSERT INTO character_expertises (characterId, name, attribute, trained, custom)
VALUES (?, ?, ?, ?, ?)
RETURNING name, attribute, trained, custom;

-- name: UpdateExpertise :one
UPDATE character_expertises
SET attribute = COALESCE(sqlc.narg('attribute'), attribute),
    trained = COALESCE(sqlc.narg('trained'), trained)
WHERE characterId = sqlc.arg('characterId') AND name = sqlc.arg('name')
RETURNING name, attribute, trained, custom;

-- name: DeleteExpertiseByID :exec
DELETE FROM character_expertises WHERE id = ?;

-- name: UpdateEffectModifiers :exec
UPDATE active_effects SET modifiers = sqlc.arg('modifiers') WHERE id = sqlc.arg('id');

-- name: DeleteEffectByID :exec
DELETE FROM active_effects WHERE id = ?;

-- name: SetHpCurrent :exec
UPDATE characters SET hpCurrent = sqlc.arg('hpCurrent'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');
