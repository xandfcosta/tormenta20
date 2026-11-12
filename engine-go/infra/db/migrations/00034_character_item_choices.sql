-- A ESCOLHA QUE UM ITEM PEDE A QUEM O TEM (ALE-623).
--
-- O tomo hermetico da +2 na CD das magias arcanas "dessa escola" (p160), e a
-- escola e de quem o tem: decisao do dono, o jogador escolhe e pode trocar. Nao
-- e a escolha do encanto (`character_item_enchant_choices`), que e do mestre e
-- presa ao encanto: esta e do proprio item, e tirar o item leva a escolha junto.
--
-- TABELA e nao coluna em `character_items` pela armadilha do guia: o sqlc nao
-- enxerga `ALTER TABLE ADD COLUMN` de um arquivo de migracao novo. E so o item
-- que pede escolha tem linha aqui.
-- +goose Up
CREATE TABLE character_item_choices (
  itemId INTEGER PRIMARY KEY REFERENCES character_items(id) ON DELETE CASCADE,
  choice TEXT    NOT NULL
);

-- +goose Down
DROP TABLE character_item_choices;
