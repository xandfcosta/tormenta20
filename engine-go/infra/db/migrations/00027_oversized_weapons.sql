-- ARMA DE CATEGORIA MAIOR (ALE-472). O catalogo nao da tamanho por item (p142), e a
-- marca mora no item da mochila, guardada.
--
-- Tabela e nao coluna pela mesma razao dos encantos (00020): o sqlc nao enxerga
-- `ALTER TABLE ADD COLUMN` de um arquivo novo. A linha existir E a marca.
-- +goose Up
CREATE TABLE character_oversized_weapons (
  itemId INTEGER PRIMARY KEY NOT NULL REFERENCES character_items(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE character_oversized_weapons;
