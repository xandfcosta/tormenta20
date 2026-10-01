-- O DANO NAO LETAL, como PARCELA do dano acumulado (ALE-423, p236).
--
-- "Dano nao letal conta para determinar quando voce cai inconsciente, mas nao
-- para determinar quando voce comeca a sangrar ou morre. Efeitos de cura
-- recuperam primeiro pontos de vida perdidos por dano nao letal."
--
-- TABELA IRMA e nao coluna no `character_damage`, e a razao e a armadilha que o
-- guia registra -- MEDIDA de novo aqui, porque ela e mais sutil do que parece:
-- o sqlc v1.31.1 GERA o campo no modelo a partir de um `ALTER TABLE ADD COLUMN`
-- de migracao nova, e mesmo assim recusa toda QUERY que cite a coluna, com
-- "column \"nonlethaldamage\" does not exist" apontando a linha da query. Ver o
-- modelo gerado nao prova nada; so escrever a consulta prova.
--
-- O preco da tabela irma e repetir o ON DELETE CASCADE e abrir a hipotese de
-- "nao letal sem dano" -- que a chave primaria compartilhada nao impede, mas
-- que o funil de escrita de vital nunca produz: quem escreve uma escreve a
-- outra.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE character_nonlethal_damage (
  characterId INTEGER PRIMARY KEY NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  amount      INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE character_nonlethal_damage;
