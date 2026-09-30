-- O ENCANTO QUE UMA ARMA CARREGA (ALE-416).
--
-- TABELA e nao coluna, e a razao e a armadilha que o guia registra: o sqlc nao
-- enxerga `ALTER TABLE ADD COLUMN` de um arquivo de migracao novo, e toda query
-- que citasse a coluna falharia com "column does not exist" apontando a QUERY e
-- nao a migracao. As colunas `improvements` e `material` escaparam disso porque
-- nasceram no `CREATE TABLE` da 00001.
--
-- E a tabela paga por si: a chave primaria DIZ que um encanto nao se repete na
-- mesma arma, em vez de isso ser convencao sobre um JSON; e o teto de tres da
-- p334 vira contagem em SQL em vez de `len` sobre uma lista desserializada.
--
-- POR QUE NAO REUSAR `improvements`: o livro conta e limita as duas coisas em
-- separado -- "uma espada longa com quatro melhorias e tres encantos (o maximo
-- possivel)" (p334) --, e o servidor ja cobra o teto de melhoria. Um encanto
-- naquele vetor entraria na conta errada e quebraria a regra em silencio.
-- +goose Up
CREATE TABLE character_item_enchants (
  itemId    INTEGER NOT NULL REFERENCES character_items(id) ON DELETE CASCADE,
  enchantId TEXT    NOT NULL,
  PRIMARY KEY (itemId, enchantId)
);

-- +goose Down
DROP TABLE character_item_enchants;
