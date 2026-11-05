-- O HOMEBREW QUE SAIU DO CODIGO volta como duas regras da mesa, DESLIGADAS, nas campanhas que
-- tem ficha com o medalhao de prata ou com a Equilibrada. O mestre as liga (decisao do dono).
--
-- O id comeca em "regra-0-" porque a lista se le na ordem do id, e estas sao as mais antigas. A
-- campanha em que o mestre ja escreveu uma regra com a Equilibrada, ou com o medalhao, nao ganha
-- outra: as duas ligadas somariam.
-- +goose Up
INSERT INTO campaign_rules (id, campaignId, characterId, name, enabled, conditions, effect, updatedAt)
SELECT 'regra-0-equilibrada-' || c.id, c.id, NULL, 'Equilibrada anula a desbalanceada', 0,
  '[{"kind":"improvement","value":"melhoria-equilibrada"},{"kind":"trait","value":"desbalanceada"}]',
  '{"kind":"adds","target":{"k":"attack"},"amount":2}', strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM campaigns c
WHERE EXISTS (
  SELECT 1 FROM character_items i JOIN characters ch ON ch.id = i.characterId
  WHERE ch.campaignId = c.id AND i.improvements LIKE '%"melhoria-equilibrada"%'
)
AND NOT EXISTS (
  SELECT 1 FROM campaign_rules r WHERE r.campaignId = c.id AND r.conditions LIKE '%melhoria-equilibrada%'
);

INSERT INTO campaign_rules (id, campaignId, characterId, name, enabled, conditions, effect, updatedAt)
SELECT 'regra-0-medalhao-' || c.id, c.id, NULL, 'Medalhao de prata vestido', 0,
  '[{"kind":"item","value":"medalhao-de-prata"}]',
  '{"kind":"wielded","target":{"k":""},"amount":0}', strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM campaigns c
WHERE EXISTS (
  SELECT 1 FROM character_items i JOIN characters ch ON ch.id = i.characterId
  WHERE ch.campaignId = c.id AND i.catalogId = 'medalhao-de-prata'
)
AND NOT EXISTS (
  SELECT 1 FROM campaign_rules r WHERE r.campaignId = c.id AND r.conditions LIKE '%medalhao-de-prata%'
);

-- +goose Down
DELETE FROM campaign_rules WHERE id LIKE 'regra-0-equilibrada-%' OR id LIKE 'regra-0-medalhao-%';
