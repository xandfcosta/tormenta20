-- Regras opcionais da campanha (ALE-221, fatia 1: a carga).
--
-- Numerada 00009 e nao 00008 porque a ALE-222, feita em paralelo noutra
-- sessao, ja tinha reservado o 00008 para as tres tabelas de estado de jogo.
-- Duas migracoes com o mesmo numero nao convivem: o goose aplica uma so.
--
-- O livro nao manda aplicar tudo. Sobre os limites de carga ele diz, com todas
-- as letras: "O mestre pode ignorar essa regra, desde que os jogadores nao
-- abusem" (p141). Ate aqui o app aplicava sempre, o que escolhe um estilo de
-- jogo pelo mestre -- e escolhe o mais punitivo.
--
-- Uma LINHA POR REGRA DESLIGADA, e nao um blob com o mapa das ligadas. Tres
-- razoes:
--
-- 1. O padrao nasce certo. Campanha sem linha nenhuma = todas as regras em
--    vigor, que e o padrao do livro. Um mapa das ligadas teria de ser
--    preenchido na criacao da campanha e retro-preenchido nas antigas, e uma
--    regra nova entraria DESLIGADA em todo mundo, em silencio.
-- 2. Regra nova nao pede migracao: e uma string nova na coluna `rule`.
-- 3. Da para perguntar em SQL "quais regras TODAS as campanhas deste
--    personagem desligaram", que e como a ficha resolve pertencer a mais de
--    uma campanha.
--
-- Tabela propria e nao coluna em campaigns porque o sqlc v1.31.1 nao propaga
-- ALTER TABLE ADD COLUMN entre arquivos de migracao -- a coluna some do
-- catalogo e toda query sobre ela falha com "column does not exist"
-- (ALE-124, ALE-137).
--
-- Nao ha CHECK sobre o valor de `rule`: o conjunto de regras conhecidas mora no
-- Go, que ignora o que nao reconhece. Um CHECK aqui obrigaria uma migracao por
-- regra, que e exatamente o que este desenho evita.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE campaign_ignored_rules (
  campaignId INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  rule       TEXT NOT NULL,
  updatedAt  TEXT NOT NULL,
  PRIMARY KEY (campaignId, rule)
);

-- +goose Down
DROP TABLE campaign_ignored_rules;
