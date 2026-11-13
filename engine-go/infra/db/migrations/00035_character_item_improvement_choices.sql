-- A ESCOLHA QUE UMA MELHORIA PEDE, gravada por item e por melhoria (ALE-631).
--
-- O Harmonizado diz "escolha uma magia. Seu custo diminui em -1 PM" (p165), e
-- a magia e de quem tem o esoterico. Nao e a escolha do item
-- (`character_item_choices`): o tomo hermetico harmonizado pede as duas, a
-- escola dele e a magia da melhoria.
--
-- A melhoria mora numa coluna JSON de `character_items`, e nao numa tabela, e
-- por isso nao ha chave estrangeira para ela: quem tira a melhoria tira a
-- escolha junto (`SaveItemOverlays`). Tirar o item leva tudo pela cascata.
-- +goose Up
CREATE TABLE character_item_improvement_choices (
  itemId        INTEGER NOT NULL REFERENCES character_items(id) ON DELETE CASCADE,
  improvementId TEXT    NOT NULL,
  choice        TEXT    NOT NULL,
  PRIMARY KEY (itemId, improvementId)
);

-- +goose Down
DROP TABLE character_item_improvement_choices;
