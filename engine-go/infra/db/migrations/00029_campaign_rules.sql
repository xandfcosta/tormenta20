-- As REGRAS DA MESA: o que o mestre monta no editor guiado, condicoes e efeito.
--
-- `characterId` nula e a campanha inteira, como em campaign_grants. `enabled` nasce
-- desligada: o mestre liga a regra quando quiser que ela valha.
-- +goose Up
CREATE TABLE campaign_rules (
  id          TEXT PRIMARY KEY,
  campaignId  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  characterId INTEGER REFERENCES characters(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  enabled     INTEGER NOT NULL DEFAULT 0,
  conditions  TEXT NOT NULL,
  effect      TEXT NOT NULL,
  updatedAt   TEXT NOT NULL
);

CREATE INDEX campaign_rules_by_campaign ON campaign_rules(campaignId);

-- +goose Down
DROP TABLE campaign_rules;
