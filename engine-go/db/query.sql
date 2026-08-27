-- Comments here are ASCII-ONLY on purpose: sqlc measures the query with byte
-- offsets and rune-counted comments, so one accented letter above a query
-- silently truncates the generated SQL by one character (ALE-120 lost the
-- "ULL" of an `IS NULL`, which still compiled).
--
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
-- Roster = templates only; campaign snapshots (sourceCharacterId set) are
-- reached through their campaign, not the personal list (ALE-33).
SELECT * FROM characters
WHERE ownerId = ? AND sourceCharacterId IS NULL
ORDER BY updatedAt DESC;

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

-- name: SetItemQuantity :exec
UPDATE character_items SET quantity = sqlc.arg('quantity') WHERE id = sqlc.arg('id');

-- name: CreateActiveEffect :one
INSERT INTO active_effects (characterId, catalogId, scope, modifiers, createdAt)
VALUES (?, ?, ?, ?, ?)
RETURNING id, catalogId, scope, modifiers, createdAt;

-- name: SetVitalsCurrent :exec
UPDATE characters SET hpCurrent = sqlc.arg('hpCurrent'), mpCurrent = sqlc.arg('mpCurrent'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: ListCharacterMaxes :many
SELECT id, hpMax, mpMax FROM characters WHERE id IN (sqlc.slice('ids'));

-- name: UpsertActiveEffect :one
INSERT INTO active_effects (characterId, source, catalogId, scope, modifiers, createdAt)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (characterId, catalogId, scope) DO UPDATE SET modifiers = excluded.modifiers, source = excluded.source
RETURNING id, catalogId, scope, modifiers, createdAt;

-- name: ListEffectIdsByCatalog :many
SELECT id FROM active_effects WHERE characterId = ? AND catalogId = ?;

-- name: DeleteEffectsByCatalog :exec
DELETE FROM active_effects WHERE characterId = ? AND catalogId = ?;

-- name: DeleteEffectsByScope :exec
DELETE FROM active_effects WHERE characterId = ? AND scope = ?;

-- name: DeleteSceneAndDayEffects :exec
DELETE FROM active_effects WHERE characterId = ? AND scope IN ('scene', 'day');

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

-- name: SetMpCurrent :exec
UPDATE characters SET mpCurrent = sqlc.arg('mpCurrent'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: SetCharacterLevel :exec
UPDATE characters SET level = sqlc.arg('level'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: SetCharacterClassLevel :execrows
UPDATE character_classes SET level = sqlc.arg('level')
WHERE characterId = sqlc.arg('characterId') AND className = sqlc.arg('className');

-- name: SetCharacterVitals :exec
UPDATE characters
SET hpMax = sqlc.arg('hpMax'), hpCurrent = sqlc.arg('hpCurrent'),
    mpMax = sqlc.arg('mpMax'), mpCurrent = sqlc.arg('mpCurrent'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: SetCharacterTibar :exec
UPDATE characters SET tibar = sqlc.arg('tibar'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: SetProficiencies :exec
UPDATE characters SET proficiencies = sqlc.arg('proficiencies'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- name: CreateCharacter :one
INSERT INTO characters (
  ownerId, name, origin, god, godPower, tibar, level, hpMax, hpCurrent, mpMax, mpCurrent,
  strength, dexterity, constitution, intelligence, wisdom, charisma, size, displacement,
  proficiencies, raceAttributeChoices, secondaryRaceChoices, originChoices, classPowers,
  classChoices, powerChoices, createdAt, updatedAt
) VALUES (
  sqlc.arg('ownerId'), sqlc.arg('name'), sqlc.arg('origin'), sqlc.arg('god'), sqlc.arg('godPower'),
  sqlc.arg('tibar'), sqlc.arg('level'), sqlc.arg('hpMax'), sqlc.arg('hpCurrent'), sqlc.arg('mpMax'),
  sqlc.arg('mpCurrent'), sqlc.arg('strength'), sqlc.arg('dexterity'), sqlc.arg('constitution'),
  sqlc.arg('intelligence'), sqlc.arg('wisdom'), sqlc.arg('charisma'), sqlc.arg('size'),
  sqlc.arg('displacement'), sqlc.arg('proficiencies'), sqlc.arg('raceAttributeChoices'),
  sqlc.arg('secondaryRaceChoices'), sqlc.arg('originChoices'), sqlc.arg('classPowers'),
  sqlc.arg('classChoices'), sqlc.arg('powerChoices'), sqlc.arg('createdAt'), sqlc.arg('updatedAt')
)
RETURNING id;

-- name: CreateRace :exec
INSERT INTO character_races (characterId, race) VALUES (?, ?);

-- name: CreateClass :exec
INSERT INTO character_classes (characterId, className, level) VALUES (?, ?, ?);

-- name: GetActiveEffectMeta :one
SELECT id, characterId FROM active_effects WHERE id = ? LIMIT 1;

-- name: GetActiveEffect :one
SELECT id, characterId, catalogId, scope, modifiers, createdAt
FROM active_effects WHERE id = ? LIMIT 1;

-- campaigns / users (B.4)

-- name: ListCampaignsForUser :many
SELECT * FROM campaigns c
WHERE c.ownerId = sqlc.arg('userId')
   OR c.id IN (
     SELECT m.campaignId FROM campaign_members m
     JOIN characters ch ON ch.id = m.characterId WHERE ch.ownerId = sqlc.arg('userId')
   )
ORDER BY c.updatedAt DESC;

-- name: GetCampaign :one
SELECT * FROM campaigns WHERE id = ? LIMIT 1;

-- name: IsCampaignMember :one
SELECT EXISTS (
  SELECT 1 FROM campaign_members m JOIN characters ch ON ch.id = m.characterId
  WHERE m.campaignId = ? AND ch.ownerId = ?
) AS isMember;

-- name: CreateCampaign :one
INSERT INTO campaigns (ownerId, name, description, createdAt, updatedAt)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteCampaign :exec
DELETE FROM campaigns WHERE id = ?;

-- name: SetInviteToken :one
UPDATE campaigns SET inviteToken = sqlc.arg('inviteToken'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id')
RETURNING id, inviteToken;

-- name: GetCampaignByToken :one
SELECT id, name FROM campaigns WHERE inviteToken = ? LIMIT 1;

-- name: CallerCharacterInCampaign :one
SELECT ch.id, ch.name, ch.level FROM campaign_members m
JOIN characters ch ON ch.id = m.characterId
WHERE m.campaignId = ? AND ch.ownerId = ? LIMIT 1;

-- name: ListUsersByIDs :many
SELECT id, email, name, createdAt FROM users WHERE id IN (sqlc.slice('ids')) ORDER BY createdAt DESC;

-- campaign members (B.4)

-- name: ListMembers :many
SELECT m.id, m.campaignId, m.characterId, m.role, m.addedAt,
       ch.name AS charName, ch.level AS charLevel,
       ch.hpCurrent AS charHpCurrent, ch.hpMax AS charHpMax,
       ch.mpCurrent AS charMpCurrent, ch.mpMax AS charMpMax
FROM campaign_members m JOIN characters ch ON ch.id = m.characterId
WHERE m.campaignId = ? ORDER BY m.addedAt ASC;

-- name: GetMember :one
SELECT * FROM campaign_members WHERE id = ? LIMIT 1;

-- name: HasPlayerPc :one
SELECT EXISTS (
  SELECT 1 FROM campaign_members m JOIN characters ch ON ch.id = m.characterId
  WHERE m.campaignId = ? AND m.role = 'player' AND ch.ownerId = ?
) AS hasPc;

-- name: IsCharacterMember :one
SELECT EXISTS (SELECT 1 FROM campaign_members WHERE campaignId = ? AND characterId = ?) AS isMember;

-- name: CreateMember :one
INSERT INTO campaign_members (campaignId, characterId, role, addedAt) VALUES (?, ?, ?, ?) RETURNING *;

-- name: SetMemberRole :one
UPDATE campaign_members SET role = sqlc.arg('role') WHERE id = sqlc.arg('id') RETURNING *;

-- name: DeleteMember :exec
DELETE FROM campaign_members WHERE id = ?;

-- name: GetMemberOwners :one
SELECT m.id, m.campaignId, c.ownerId AS campaignOwner, ch.ownerId AS characterOwner
FROM campaign_members m JOIN campaigns c ON c.id = m.campaignId JOIN characters ch ON ch.id = m.characterId
WHERE m.id = ? LIMIT 1;

-- sessions (B.4)

-- name: ListSessions :many
SELECT * FROM sessions WHERE campaignId = ? ORDER BY sessionNumber ASC;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ? LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (campaignId, sessionNumber, title, notes, createdAt, updatedAt)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: StartSessionFresh :one
UPDATE sessions SET status = 'active', startedAt = sqlc.arg('startedAt'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id') RETURNING *;

-- name: ReopenSession :one
UPDATE sessions SET status = 'active', endedAt = NULL, updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id') RETURNING *;

-- name: EndSession :one
UPDATE sessions SET status = 'ended', endedAt = sqlc.arg('endedAt'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id') RETURNING *;

-- name: HasLiveSessionForCharacter :one
-- ALE-216: the character is at a table with a session RUNNING. 'active' is the
-- only live status; 'planned' and 'ended' are not a table in progress.
SELECT EXISTS (
  SELECT 1 FROM campaign_members m
  JOIN sessions s ON s.campaignId = m.campaignId
  WHERE m.characterId = ? AND s.status = 'active'
) AS atLiveTable;

-- name: FirstLiveSessionForUser :one
-- ALE-231: the Hub's "Continuar sessao" answered in ONE query.
-- The SPA had no endpoint for this, so it fanned out one sessions request per
-- campaign and picked the first live one client-side (see the deleted
-- createActiveSession). Campaign order is the SAME as ListCampaignsForUser --
-- most recently touched first -- because that is the order the Hub walked, and
-- changing it would change which session "Continuar" resumes.
SELECT s.id AS sessionId, s.campaignId AS campaignId
FROM sessions s
JOIN campaigns c ON c.id = s.campaignId
WHERE s.status = 'active'
  AND (c.ownerId = sqlc.arg('userId')
       OR c.id IN (
         SELECT m.campaignId FROM campaign_members m
         JOIN characters ch ON ch.id = m.characterId
         WHERE ch.ownerId = sqlc.arg('userId')
       ))
ORDER BY c.updatedAt DESC
LIMIT 1;

-- name: LiveSessionsForUser :many
-- ALE-234: which of my campaigns have a session RUNNING, all in one query.
-- The SPA asked this with one request PER CAMPAIGN, twice over: once in the Hub
-- (createActiveSession) and once in the campaign scene
-- (createActiveSessionByCampaign). Two identical fan-outs in two screens is the
-- sign that the hole was in the API, not in the screens.
SELECT s.campaignId AS campaignId, MIN(s.id) AS sessionId
FROM sessions s
JOIN campaigns c ON c.id = s.campaignId
WHERE s.status = 'active'
  AND (c.ownerId = sqlc.arg('userId')
       OR c.id IN (
         SELECT m.campaignId FROM campaign_members m
         JOIN characters ch ON ch.id = m.characterId
         WHERE ch.ownerId = sqlc.arg('userId')
       ))
GROUP BY s.campaignId;

-- name: ResetSessionTracker :exec
UPDATE sessions SET runtimeState = sqlc.arg('runtimeState'), updatedAt = sqlc.arg('updatedAt')
WHERE id = sqlc.arg('id');

-- board (ALE-124, varios abertos na ALE-205)

-- A ORDEM e a de ABERTURA, que e a ordem das abas na tela. Ver o comentario do
-- `openSeq` na migracao 00010: contador e nao carimbo de tempo, porque carimbo
-- empata.
-- name: ListOpenBoards :many
SELECT boardId, state, openSeq FROM open_boards WHERE sessionId = ? ORDER BY openSeq;

-- O upsert NAO toca `openSeq`: gravar o tabuleiro e dizer que ele mudou, nunca
-- que ele nasceu de novo. Sem o `excluded` de fora, a ordem das abas mudaria a
-- cada peca que anda.
-- name: SaveOpenBoard :exec
INSERT INTO open_boards (sessionId, boardId, state, openSeq, updatedAt) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(sessionId, boardId) DO UPDATE SET state = excluded.state, updatedAt = excluded.updatedAt;

-- name: DeleteOpenBoard :exec
DELETE FROM open_boards WHERE sessionId = ? AND boardId = ?;

-- name: ListCampaignsForCharacter :many
SELECT m.id, m.campaignId, m.characterId, m.role, m.addedAt,
       c.name AS campaignName, c.description AS campaignDescription, c.updatedAt AS campaignUpdatedAt
FROM campaign_members m
JOIN campaigns c ON c.id = m.campaignId
WHERE m.characterId = ?
ORDER BY m.addedAt ASC;

-- account invites (ALE-120)

-- name: CreateAccountInvite :one
INSERT INTO account_invites (token, createdBy, createdAt, expiresAt)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetAccountInvite :one
SELECT * FROM account_invites WHERE token = ? LIMIT 1;

-- Single use for real: `usedAt IS NULL` is what makes the second registration
-- with the same link a no-op, and the affected rows say which one won.
-- name: SpendAccountInvite :execrows
UPDATE account_invites SET usedAt = ?, usedBy = ?
WHERE id = ? AND usedAt IS NULL;

-- name: ListAllCampaigns :many
SELECT * FROM campaigns ORDER BY updatedAt DESC;

-- administration screen (ALE-120)

-- name: ListUsersWithCounts :many
SELECT u.id, u.email, u.name, u.createdAt,
  (SELECT COUNT(*) FROM campaigns c WHERE c.ownerId = u.id) AS campaigns,
  (SELECT COUNT(*) FROM characters ch WHERE ch.ownerId = u.id) AS characters
FROM users u ORDER BY u.createdAt;

-- name: ListOpenAccountInvites :many
SELECT * FROM account_invites
WHERE usedAt IS NULL AND expiresAt > sqlc.arg('now')
ORDER BY createdAt DESC;

-- Deleting an account moves its mesas to the caller first, so the chronicle
-- survives the player leaving the table.
-- name: TransferCampaigns :execrows
UPDATE campaigns SET ownerId = sqlc.arg('newOwnerId'), updatedAt = sqlc.arg('updatedAt')
WHERE ownerId = sqlc.arg('oldOwnerId');

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: TableCounts :one
SELECT
  (SELECT COUNT(*) FROM users) AS users,
  (SELECT COUNT(*) FROM campaigns) AS campaigns,
  (SELECT COUNT(*) FROM characters) AS characters;

-- password reset (ALE-120)

-- name: CreatePasswordReset :one
INSERT INTO password_resets (token, userId, createdBy, createdAt, expiresAt)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPasswordReset :one
SELECT * FROM password_resets WHERE token = ? LIMIT 1;

-- Same single-use guard as the account invite: the UPDATE, not the read, is
-- what decides who won.
-- name: SpendPasswordReset :execrows
UPDATE password_resets SET usedAt = ? WHERE id = ? AND usedAt IS NULL;

-- name: UpdateUserPassword :exec
UPDATE users SET passwordHash = ?, updatedAt = ? WHERE id = ?;

-- Creature blocks a GM authors for a campaign (ALE-137). Listed by name because
-- that is how the GM looks for one; the rest of the block is JSON.
-- name: ListCampaignCreatures :many
SELECT * FROM campaign_creatures WHERE campaignId = ? ORDER BY name;

-- name: GetCampaignCreature :one
SELECT * FROM campaign_creatures WHERE id = ? LIMIT 1;

-- name: CreateCampaignCreature :one
INSERT INTO campaign_creatures (campaignId, name, block, createdAt, updatedAt)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateCampaignCreature :one
UPDATE campaign_creatures SET name = ?, block = ?, updatedAt = ?
WHERE id = ?
RETURNING *;

-- name: DeleteCampaignCreature :exec
DELETE FROM campaign_creatures WHERE id = ?;

-- Lugares da cronica (ALE-124, fatia 5): a cena montada que sobrevive ao fim do
-- tabuleiro. Listados por nome porque e por ele que o mestre procura a taverna;
-- o resto da cena e JSON, mesmo arranjo do bloco de criatura.
-- name: ListCampaignPlaces :many
SELECT * FROM campaign_places WHERE campaignId = ? ORDER BY name;

-- name: GetCampaignPlace :one
SELECT * FROM campaign_places WHERE id = ? LIMIT 1;

-- Arquivar sobrescreve o lugar de mesmo nome na mesma cronica: o mestre que
-- reabre a taverna, move duas pecas e encerra de novo espera UMA taverna, e nao
-- uma pilha de tavernas quase iguais.
-- name: SaveCampaignPlace :one
INSERT INTO campaign_places (campaignId, name, state, createdAt, updatedAt)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateCampaignPlace :one
UPDATE campaign_places SET state = ?, updatedAt = ?
WHERE id = ?
RETURNING *;

-- name: FindCampaignPlaceByName :one
SELECT * FROM campaign_places WHERE campaignId = ? AND name = ? LIMIT 1;

-- name: DeleteCampaignPlace :exec
DELETE FROM campaign_places WHERE id = ?;

-- regras opcionais da campanha (ALE-221)

-- name: ListIgnoredRulesForCampaign :many
SELECT rule FROM campaign_ignored_rules WHERE campaignId = ? ORDER BY rule;

-- name: ClearIgnoredRulesForCampaign :exec
DELETE FROM campaign_ignored_rules WHERE campaignId = ?;

-- name: IgnoreRuleInCampaign :exec
INSERT INTO campaign_ignored_rules (campaignId, rule, updatedAt) VALUES (?, ?, ?)
ON CONFLICT(campaignId, rule) DO UPDATE SET updatedAt = excluded.updatedAt;

-- A regra so esta desligada para a ficha se TODAS as campanhas dela a
-- desligaram. A mais estrita vence, e a ficha que nao pertence a campanha
-- nenhuma calcula com tudo em vigor -- que e o padrao do livro. Escolher "a
-- primeira campanha" seria arbitrario e mudaria com a ordem das linhas.
-- name: ListIgnoredRulesForCharacter :many
SELECT r.rule
FROM campaign_ignored_rules r
WHERE r.campaignId IN (SELECT m.campaignId FROM campaign_members m WHERE m.characterId = sqlc.arg('characterId'))
GROUP BY r.rule
HAVING COUNT(DISTINCT r.campaignId) = (SELECT COUNT(*) FROM campaign_members m2 WHERE m2.characterId = sqlc.arg('characterId'))
ORDER BY r.rule;

-- name: IsGmAtLiveTableForCharacter :one
-- ALE-223: the caller OWNS a campaign that this character belongs to AND that
-- campaign has a session running. GM of campaign A with a live session in
-- campaign B does not count -- the two joins are on the same membership row.
SELECT EXISTS (
  SELECT 1 FROM campaign_members m
  JOIN campaigns c ON c.id = m.campaignId
  JOIN sessions s ON s.campaignId = m.campaignId
  WHERE m.characterId = sqlc.arg('characterId')
    AND c.ownerId = sqlc.arg('ownerId')
    AND s.status = 'active'
) AS gmAtLiveTable;

-- Estado de mesa da ficha (ALE-222). Saiu do localStorage: o servidor e dono.
-- CUIDADO: conditionals (o opt-in do JOGADOR) nao e conditions (as do LIVRO,
-- que vivem na coluna characters.activeConditions). Ver C6 no GLOSSARIO.md.

-- name: ListCharacterConditionals :many
SELECT conditionalId FROM character_conditionals
WHERE characterId = ? ORDER BY conditionalId;

-- name: AddCharacterConditional :exec
INSERT INTO character_conditionals (characterId, conditionalId)
VALUES (?, ?) ON CONFLICT DO NOTHING;

-- name: RemoveCharacterConditional :exec
DELETE FROM character_conditionals WHERE characterId = ? AND conditionalId = ?;

-- name: ClearCharacterConditionals :exec
DELETE FROM character_conditionals WHERE characterId = ?;

-- name: ListCharacterPowerUses :many
SELECT powerId, scope, used FROM character_power_uses
WHERE characterId = ? ORDER BY powerId, scope;

-- O upsert SOMA em vez de escrever o total: quem chama diz "gastei mais um",
-- nunca "agora sao tres". Mandar o total deixaria dois cliques rapidos gravarem
-- o mesmo numero e perderem um uso.
-- name: BumpCharacterPowerUse :exec
INSERT INTO character_power_uses (characterId, powerId, scope, used)
VALUES (?, ?, ?, 1)
ON CONFLICT (characterId, powerId, scope)
DO UPDATE SET used = used + 1;

-- name: ClearCharacterPowerUsesByScope :exec
DELETE FROM character_power_uses WHERE characterId = ? AND scope = ?;

-- name: ClearCharacterPowerUses :exec
DELETE FROM character_power_uses WHERE characterId = ?;

-- name: ListCharacterStances :many
SELECT flag, steps, pmPaid FROM character_stances
WHERE characterId = ? ORDER BY flag;

-- name: UpsertCharacterStance :exec
INSERT INTO character_stances (characterId, flag, steps, pmPaid)
VALUES (?, ?, ?, ?)
ON CONFLICT (characterId, flag)
DO UPDATE SET steps = excluded.steps, pmPaid = excluded.pmPaid;

-- name: RemoveCharacterStance :exec
DELETE FROM character_stances WHERE characterId = ? AND flag = ?;

-- name: ClearCharacterStances :exec
DELETE FROM character_stances WHERE characterId = ?;
