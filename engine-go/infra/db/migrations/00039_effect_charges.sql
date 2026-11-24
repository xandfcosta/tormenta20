-- AS CARGAS DA MAGIA EM VIGOR (ALE-706): os dardos de lava da Lanca Ignea (p196), que o
-- aprimoramento cria em vez do efeito normal e que se gastam um por vez, ate descarregar.
--
-- TABELA e nao coluna em `active_effects`, pela armadilha da 00036 (o sqlc nao enxerga
-- `ALTER TABLE ADD COLUMN` de um arquivo novo). A linha pende do efeito em vigor e sai com
-- ele (cascata); o ultimo dardo encerra o efeito, e as cargas vao junto.
-- +goose Up
CREATE TABLE effect_charges (
  effectId  INTEGER PRIMARY KEY REFERENCES active_effects(id) ON DELETE CASCADE,
  remaining INTEGER NOT NULL
);

-- +goose Down
DROP TABLE effect_charges;
