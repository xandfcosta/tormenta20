-- O DADO DA VEZ (ALE-703): os aprimoramentos escolhidos ao conjurar a magia que rola de novo
-- na vez de quem conjurou (Ferver Sangue, Poeira da Podridao, os Enxames). Eles valem em toda
-- vez, e por isso a conjuracao os grava junto da linha em vigor.
--
-- TABELA e nao coluna em `active_effects`, pela armadilha da 00036: o sqlc nao enxerga
-- `ALTER TABLE ADD COLUMN` de um arquivo de migracao novo. A linha pende do efeito em vigor e
-- sai com ele (cascata): encerrar a magia, perder a manutencao ou acabar a cena a levam junto.
-- `augments` e a lista de `sheet.AugmentPick` em JSON.
-- +goose Up
CREATE TABLE effect_cast_augments (
  effectId INTEGER PRIMARY KEY REFERENCES active_effects(id) ON DELETE CASCADE,
  augments TEXT NOT NULL
);

-- +goose Down
DROP TABLE effect_cast_augments;
