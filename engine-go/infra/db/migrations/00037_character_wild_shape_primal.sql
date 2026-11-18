-- A FORMA PRIMAL (p62, ALE-655): a Forma Selvagem com dois animais.
--
-- A segunda forma, e de qual das duas vem a arma natural e o tamanho. TABELA e nao
-- coluna em `character_wild_shape`, pela armadilha da 00036: o sqlc nao enxerga
-- `ALTER TABLE ADD COLUMN` de um arquivo de migracao novo. So quem usa as duas
-- formas tem linha aqui, e voltar ao normal a leva junto (cascata).
-- +goose Up
CREATE TABLE character_wild_shape_primal (
  characterId INTEGER PRIMARY KEY REFERENCES character_wild_shape(characterId) ON DELETE CASCADE,
  secondForm  TEXT NOT NULL,
  weaponFrom  TEXT NOT NULL DEFAULT '',
  sizeFrom    TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE character_wild_shape_primal;
