-- +goose Up
-- +goose StatementBegin
-- O RASCUNHO SO MONTA (ALE-564): quem joga e a copia da campanha.
--
-- Primeiro quem esta em campanha passa a dizer isso em `campaignId`, que e o que a
-- regra le. Entrar numa campanha sempre clonou, entao em producao nao ha o que mudar;
-- quem tinha membro sem `campaignId` era o banco semeado. Sem este passo o heroi da
-- semente perderia o dano no DELETE abaixo e deixaria de poder jogar.
UPDATE characters
SET campaignId = (SELECT MIN(m.campaignId) FROM campaign_members m WHERE m.characterId = characters.id)
WHERE campaignId IS NULL AND id IN (SELECT characterId FROM campaign_members);

DELETE FROM character_damage WHERE characterId IN (SELECT id FROM characters WHERE campaignId IS NULL);
DELETE FROM character_nonlethal_damage WHERE characterId IN (SELECT id FROM characters WHERE campaignId IS NULL);
DELETE FROM active_effects WHERE characterId IN (SELECT id FROM characters WHERE campaignId IS NULL);
DELETE FROM character_conditionals WHERE characterId IN (SELECT id FROM characters WHERE campaignId IS NULL);
DELETE FROM character_power_uses WHERE characterId IN (SELECT id FROM characters WHERE campaignId IS NULL);
DELETE FROM character_stances WHERE characterId IN (SELECT id FROM characters WHERE campaignId IS NULL);
UPDATE characters SET activeConditions = '[]' WHERE campaignId IS NULL;
-- +goose StatementEnd

-- +goose Down
-- O que foi apagado nao volta: o backup do deploy e quem guarda.
