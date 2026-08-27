-- Varios tabuleiros abertos na mesma sessao (ALE-205).
--
-- A 00005 declarou 1:1 no SCHEMA e nao por convencao: `sessionId INTEGER
-- PRIMARY KEY`. Isso foi certo enquanto o mestre tinha UMA cena na mesa; o caso
-- de uso desta issue e o grupo que se separou, com a cripta e a taverna abertas
-- ao mesmo tempo, e ele nao cabe numa linha por sessao.
--
-- Tabela NOVA e nao `ALTER TABLE`: o sqlc v1.31.1 nao propaga `ADD COLUMN`
-- entre arquivos de migracao (ver o guia do pacote), e trocar a chave primaria
-- de uma tabela no SQLite e um rebuild com `RENAME TO`, que o gerador tambem
-- nao acompanha. Criar, copiar e derrubar e o caminho que o gerador entende.
--
-- O `boardId` e TEXT e nao um autoincremento porque o tabuleiro nasce em
-- MEMORIA e so depois e gravado: o `Open` cunha o id no mesmo instante em que
-- monta a cena, com o mesmo `newID` das pecas, e a gravacao e um upsert sobre
-- ele. Um id do banco obrigaria o store a esperar o disco para saber quem e a
-- cena que ja esta na tela.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE open_boards (
  sessionId INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  boardId   TEXT NOT NULL,
  state     TEXT NOT NULL,
  -- A ORDEM DE ABERTURA, e ela e a ordem das abas na tela: a primeira e tambem
  -- a aba PADRAO de quem ainda nao escolheu, entao instabilidade aqui troca a
  -- cena debaixo de quem estava olhando.
  --
  -- Um CONTADOR e nao um carimbo de tempo, e isso foi MEDIDO: com `openedAt` em
  -- milissegundos, duas cenas abertas no mesmo milissegundo empatam e o
  -- desempate cai no `boardId`, que e um UUID e nao tem ordem nenhuma. O teste
  -- da hidratacao pegou exatamente esse caso. O contador e do servidor
  -- (`max(openSeq) + 1` da sessao), nunca empata, e nao depende do relogio.
  openSeq   INTEGER NOT NULL,
  updatedAt TEXT NOT NULL,
  PRIMARY KEY (sessionId, boardId)
);

-- A cena que estava na mesa vira a PRIMEIRA aba, e nao um tabuleiro perdido: o
-- mestre que estava jogando quando o servidor subiu com esta migracao encontra
-- a taverna onde ele a deixou.
INSERT INTO open_boards (sessionId, boardId, state, openSeq, updatedAt)
SELECT sessionId, 'primeiro', state, 1, updatedAt FROM session_boards;

DROP TABLE session_boards;

-- +goose Down
CREATE TABLE session_boards (
  sessionId INTEGER PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
  state     TEXT NOT NULL,
  updatedAt TEXT NOT NULL
);

-- Volta so a PRIMEIRA aba de cada sessao, porque e so ela que cabe: descer e
-- perder as outras, e o `MIN(openSeq)` escolhe a mais antiga em vez de deixar o
-- SQLite escolher.
INSERT INTO session_boards (sessionId, state, updatedAt)
SELECT sessionId, state, updatedAt FROM open_boards
WHERE openSeq = (SELECT MIN(b.openSeq) FROM open_boards b WHERE b.sessionId = open_boards.sessionId);

DROP TABLE open_boards;
