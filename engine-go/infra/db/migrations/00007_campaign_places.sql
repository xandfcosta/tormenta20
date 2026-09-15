-- Lugares da cronica (ALE-124, fatia 5). Encerrar o tabuleiro passa a ARQUIVAR:
-- as posicoes congelam e a cena volta para a lista de lugares da campanha, para
-- ser reaberta com tudo onde estava. Taverna -> masmorra -> de volta a taverna.
--
-- Ate aqui encerrar DESTRUIA: o `close` apagava a linha de session_boards, e a
-- taverna que o mestre montou peca por peca morria junto. Era a unica promessa
-- da epica que o codigo contradizia.
--
-- O lugar pertence a CAMPANHA e nao a sessao, pela mesma razao do bloco de
-- criatura (00006): a taverna do Javali serve a noite inteira e volta na semana
-- seguinte, enquanto a sessao acaba. Uma taverna por sessao obrigaria o mestre a
-- remontar o cenario toda quinta.
--
-- Tabela propria e nao coluna nova em session_boards por duas razoes:
--
-- 1. O sqlc v1.31.1 nao propaga ALTER TABLE ADD COLUMN entre arquivos de
--    migracao: a coluna some do catalogo e toda query sobre ela falha com
--    "column does not exist" (ALE-124, ALE-137). Dentro do mesmo arquivo
--    funciona; de um arquivo para outro, nao.
-- 2. Sao ciclos de vida diferentes. session_boards e o que esta NA MESA agora,
--    com uma linha por sessao e escrita a cada peca movida; campaign_places e
--    acervo, escrito ao arquivar e lido ao montar a cena.
--
-- O NOME e coluna e o resto vai como JSON, mesmo arranjo do session_boards e da
-- campaign_creatures: e por ele que se lista e se ordena, e ter o nome nos dois
-- lugares criaria duas verdades sobre como o lugar se chama.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE campaign_places (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  campaignId INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  state      TEXT NOT NULL,
  createdAt  TEXT NOT NULL,
  updatedAt  TEXT NOT NULL
);

CREATE INDEX idx_campaign_places_campaignId ON campaign_places(campaignId);

-- +goose Down
DROP TABLE campaign_places;
