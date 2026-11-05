-- As MELHORIAS DA MESA: o que o mestre cria para se colocar num item, ao lado das do livro.
--
-- `characterId` nula e a mesa inteira, como em campaign_rules. `family` vazia cabe em
-- qualquer item. O item guarda o id dela em character_items.improvements, como guarda
-- o de uma melhoria do livro.
-- +goose Up
CREATE TABLE campaign_improvements (
  id          TEXT PRIMARY KEY,
  campaignId  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  characterId INTEGER REFERENCES characters(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  family      TEXT NOT NULL,
  effect      TEXT NOT NULL,
  updatedAt   TEXT NOT NULL
);

CREATE INDEX campaign_improvements_by_campaign ON campaign_improvements(campaignId);

-- +goose Down
DROP TABLE campaign_improvements;
