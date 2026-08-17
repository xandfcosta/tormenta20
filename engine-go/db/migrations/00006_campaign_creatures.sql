-- Bloco de criatura do mestre (ALE-137). O livro modela NPC e monstro na MESMA
-- forma: "BANDIDO - ND 1/4 - Humanoide (humano) Medio" (p289) tem o mesmo bloco
-- do Ogro, com Pericias, Equipamento e Tesouro; conjurador ganha Pontos de Mana
-- (Centauro Xama, p290). Entao um NPC do mestre nao e ficha de personagem: e um
-- bloco de criatura, e modelar como personagem exigiria escolher raca, classe e
-- origem para um capanga que o livro descreve com ND.
--
-- Tabela propria e nao coluna em characters por duas razoes:
--
-- 1. O sqlc v1.31.1 nao propaga ALTER TABLE ADD COLUMN entre arquivos de
--    migracao: a coluna some do catalogo e toda query sobre ela falha com
--    "column does not exist" (ALE-124). Foi por isso que o tabuleiro virou
--    session_boards.
-- 2. Sao coisas diferentes. Personagem tem classe, nivel, pericias treinadas e
--    o motor de regras calculando por cima; criatura tem ND e numeros escritos
--    a mao. Junta-las obrigaria metade das colunas a mentir de um lado ou do
--    outro.
--
-- O bloco em si vai como JSON e o NOME como coluna: o nome e por onde se lista
-- e se ordena, e ter o nome nos dois lugares criaria duas verdades. Mesmo
-- arranjo do session_boards, e pela mesma razao de forma: o bloco tem listas
-- (ataques, pericias, habilidades) que crescem, e a ALE-151 ainda vai encher
-- campos que a importacao do bestiario perdeu.
--
-- A criatura pertence a CAMPANHA, nao a sessao: o vilao recorrente volta na
-- semana seguinte, e o ogro que o mestre modificou continua modificado.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE campaign_creatures (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  campaignId INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  block      TEXT NOT NULL,
  createdAt  TEXT NOT NULL,
  updatedAt  TEXT NOT NULL
);
CREATE INDEX idx_campaign_creatures_campaignId ON campaign_creatures(campaignId);

-- +goose Down
DROP TABLE campaign_creatures;
