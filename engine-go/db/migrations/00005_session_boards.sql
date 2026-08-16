-- Tabuleiro tatico da sessao (ALE-124). Tabela 1:1 com sessions em vez de uma
-- coluna dentro de runtimeState, por tres motivos:
--
-- 1. A taxa de escrita e outra. Um token anda muitas vezes numa cena; a
--    iniciativa muda poucas vezes num combate. No mesmo blob, cada tique de PV
--    re-serializaria o tabuleiro inteiro, e vice-versa.
-- 2. "Sessao sem tabuleiro" fica dito no schema: a linha existe ou nao existe.
--    Dentro de um JSON isso viraria acordo tacito sobre o que vazio significa.
-- 3. A redacao por papel e OPOSTA a da iniciativa: la a linha sobrevive sem os
--    numeros (hpHidden), aqui um token oculto some inteiro, porque a existencia
--    dele e a emboscada. Dois redatores, dois estados.
--
-- E tabela e nao coluna nova em sessions porque o sqlc v1.31.1 nao propaga
-- ALTER TABLE ADD COLUMN entre arquivos de migracao: a coluna some do catalogo
-- e toda query sobre ela falha com "column does not exist". Dentro do mesmo
-- arquivo funciona; de um arquivo para outro, nao.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE session_boards (
  sessionId INTEGER PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
  state     TEXT NOT NULL,
  updatedAt TEXT NOT NULL
);

-- +goose Down
DROP TABLE session_boards;
