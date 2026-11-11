-- A FORMA SELVAGEM em que o druida esta (p62-63, ALE-603).
--
-- Uma linha por personagem, e so enquanto ele esta transformado: voltar ao normal
-- apaga a linha. TABELA e nao coluna em `characters` pela armadilha do guia: o sqlc
-- nao enxerga `ALTER TABLE ADD COLUMN` de um arquivo de migracao novo. E a forma e
-- estado de MESA, como as posturas: o rascunho nao a guarda.
--
-- `form`, `tier` e `movement` sao as chaves do motor (`engine.WildForm`,
-- `engine.WildTier`, `engine.WildMovement`); `pmPaid` e o que a transformacao
-- custou, para a tela dizer e para voltar ao normal NAO devolver.
-- +goose Up
CREATE TABLE character_wild_shape (
  characterId INTEGER PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,
  form        TEXT    NOT NULL,
  tier        TEXT    NOT NULL,
  damageType  TEXT    NOT NULL,
  movement    TEXT    NOT NULL DEFAULT '',
  pmPaid      INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE character_wild_shape;
