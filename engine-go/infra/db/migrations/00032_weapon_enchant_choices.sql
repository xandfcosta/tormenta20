-- A ESCOLHA QUE UM ENCANTO PEDE, gravada por arma (ALE-619).
--
-- A Anticriatura e "letal contra um tipo de criatura (ou uma raca de
-- humanoides)" (p335), e o tipo e escolhido por quem encanta. TABELA e nao
-- coluna em `character_item_enchants` pela armadilha do guia: o sqlc nao
-- enxerga `ALTER TABLE ADD COLUMN` de um arquivo de migracao novo. E a tabela
-- diz pelo schema o que a coluna diria por convencao: so o encanto que pede
-- escolha tem linha aqui.
--
-- A chave estrangeira aponta para o ENCANTO da arma, e nao para a arma: tirar o
-- encanto leva a escolha junto, e regravar os encantos (apagar e reinserir)
-- pede a escolha de novo.
-- +goose Up
CREATE TABLE character_item_enchant_choices (
  itemId    INTEGER NOT NULL,
  enchantId TEXT    NOT NULL,
  choice    TEXT    NOT NULL,
  PRIMARY KEY (itemId, enchantId),
  FOREIGN KEY (itemId, enchantId) REFERENCES character_item_enchants(itemId, enchantId) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE character_item_enchant_choices;
