-- +goose Up
-- +goose StatementBegin
-- AS ESCOLHAS POR NIVEL (ALE-556), e esta tabela passa a ser a unica fonte.
--
-- `classPowers` e `powerChoices` eram conjuntos sem ordem: a ficha sabia o que foi
-- escolhido e nao sabia quando. Uma linha por escolha, com o nivel do personagem em
-- que ela foi feita:
--
--   class      a classe que recebeu aquele nivel (ref = nome da classe)
--   power      o poder escolhido (ref = id do poder)
--   pick       a escolha dentro de um poder (ref = id do poder, pick = a opcao)
--   automatic  a habilidade que a classe da naquele nivel (ref = id do poder)
--
-- `level` nulo e o que ainda nao foi situado: o codigo o resolve ao subir (ele precisa
-- do catalogo, que nao mora em SQL), e `estimated` marca o que veio desta migracao,
-- porque o nivel dessas escolhas nao esta registrado em lugar nenhum.
CREATE TABLE character_level_choices (
  id          INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  level       INTEGER,
  kind        TEXT NOT NULL CHECK (kind IN ('class', 'power', 'pick', 'automatic')),
  ref         TEXT NOT NULL,
  pick        TEXT NOT NULL DEFAULT '',
  estimated   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_character_level_choices_characterId ON character_level_choices(characterId);

INSERT INTO character_level_choices (characterId, kind, ref, estimated)
SELECT c.id, 'power', power.value, 1
FROM characters c, json_each(c.classPowers) power
ORDER BY c.id, CAST(power.key AS INTEGER);

INSERT INTO character_level_choices (characterId, kind, ref, pick, estimated)
SELECT c.id, 'pick', power.key, option.value, 1
FROM characters c, json_each(c.powerChoices) power, json_each(power.value) option
ORDER BY c.id, power.key, CAST(option.key AS INTEGER);

-- Em minusculas de proposito: o sqlc guarda o nome assim, e recusa a outra grafia. Para o
-- SQLite tanto faz.
ALTER TABLE characters DROP COLUMN classpowers;
ALTER TABLE characters DROP COLUMN powerchoices;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE characters ADD COLUMN classPowers TEXT NOT NULL DEFAULT '[]';
ALTER TABLE characters ADD COLUMN powerChoices TEXT NOT NULL DEFAULT '{}';

UPDATE characters SET classPowers = COALESCE((
  SELECT json_group_array(ref) FROM (
    SELECT ref FROM character_level_choices
    WHERE characterId = characters.id AND kind = 'power' ORDER BY id
  )), '[]');

UPDATE characters SET powerChoices = COALESCE((
  SELECT json_group_object(ref, json(picks)) FROM (
    SELECT ref, json_group_array(pick) AS picks FROM (
      SELECT ref, pick FROM character_level_choices
      WHERE characterId = characters.id AND kind = 'pick' ORDER BY id
    ) GROUP BY ref
  )), '{}');

DROP TABLE character_level_choices;
-- +goose StatementEnd
