-- +goose Up
-- +goose StatementBegin
-- AS QUATRO COLUNAS DE VITAIS SAEM (ALE-355, ultima fatia).
--
-- Desde a 00014 o que se GUARDA e o dano, e o poco e DERIVADO do catalogo. As
-- colunas continuaram como ESPELHO, mantidas pelo funil, por uma razao so: os
-- leitores que pegavam a linha por JOIN -- os cartoes do Grupo e a entrada da
-- iniciativa -- ainda liam dali. Eles passaram a perguntar o poco derivado em
-- lote, entao o espelho ficou sem leitor.
--
-- Espelho sem leitor nao e inofensivo: e uma SEGUNDA resposta para "quantos PV
-- este personagem tem", e a primeira coisa que o proximo autor faz e ler a
-- coluna, porque ela esta ali e e barata. Foi exatamente assim que o bardo da
-- semente ficou com 31 PM gravados contra 33 derivados.
--
-- # MINUSCULAS, e isso nao e estilo
--
-- O sqlc v1.31.1 casa o nome da coluna do `DROP COLUMN` em minusculas contra o
-- catalogo dele. `ALTER TABLE characters DROP COLUMN hpMax` -- a grafia com que
-- a coluna foi DECLARADA na 00001 -- falha com `column "hpMax" of relation
-- "characters" does not exist`. Medido aqui. E parente da armadilha do ADD
-- COLUMN que a 00014 documenta, e desta vez ela reclama em voz alta.
ALTER TABLE characters DROP COLUMN hpmax;
ALTER TABLE characters DROP COLUMN hpcurrent;
ALTER TABLE characters DROP COLUMN mpmax;
ALTER TABLE characters DROP COLUMN mpcurrent;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- A VOLTA restaura o SCHEMA, e nao os numeros -- de proposito.
--
-- Nenhum numero se perde: o maximo sai do catalogo e o atual e `maximo - dano`,
-- e a tabela `character_damage` continua de pe. O que nao existe e um jeito de
-- SQL derivar o maximo, porque ele depende do motor de regras.
--
-- E restaurar o valor ANTIGO seria pior que restaurar zero: aquele numero era
-- justamente o que discordava do derivado. O codigo anterior a esta fatia
-- recomputa a coluna no proximo gesto de escrita, que foi como ela se encheu da
-- primeira vez.
ALTER TABLE characters ADD COLUMN hpMax INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN hpCurrent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN mpMax INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN mpCurrent INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd
