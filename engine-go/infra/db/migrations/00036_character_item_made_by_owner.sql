-- A ARMA FEITA POR QUEM A USA (ALE-649).
--
-- O Ferreiro e o Cano Raiado valem na "arma feita por voce mesmo" (p69), e o app
-- nao sabe quem fabricou um item: a marca e do jogador, na folha do item
-- (decisao do dono). Tirar o item leva a marca junto.
--
-- TABELA e nao coluna em `character_items`, pela mesma armadilha da escolha do
-- item: o sqlc nao enxerga `ALTER TABLE ADD COLUMN` de um arquivo de migracao
-- novo. E so o item marcado tem linha aqui.
-- +goose Up
CREATE TABLE character_item_made_by_owner (
  itemId INTEGER PRIMARY KEY REFERENCES character_items(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE character_item_made_by_owner;
