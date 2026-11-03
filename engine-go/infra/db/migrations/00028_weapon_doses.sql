-- AS DOSES DA INJECAO ALQUIMICA (p165). A arma guarda o que foi carregado nela, e o
-- item da mochila ja saiu. Uma linha por dose, porque a melhoria tem dois espacos e
-- eles podem guardar coisas diferentes; o `id` diz qual entrou primeiro.
-- +goose Up
CREATE TABLE character_item_doses (
  id        INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  itemId    INTEGER NOT NULL REFERENCES character_items(id) ON DELETE CASCADE,
  catalogId TEXT NOT NULL
);
CREATE INDEX idx_character_item_doses_itemId ON character_item_doses(itemId);

-- +goose Down
DROP TABLE character_item_doses;
