-- O QUE A MESA DESLIGA, termo a termo (ALE-387).
--
-- A granularidade e o TERMO e nao a fonte: "nesta mesa o medalhao nao da o
-- bonus de Luta, mas continua dando o limite de PM". Calar a fonte inteira
-- seria a versao grossa da mesma ideia, e ela ja nao servia ao primeiro caso
-- que o dono trouxe.
--
-- O `term` e o endereco estavel de um termo -- `fonte::alvo::escala::condicao`,
-- montado pelo `engine.TermID`. Ele NAO carrega o valor nem a prosa de
-- proposito: assim uma errata do livro nao evapora o que o mestre escreveu.
-- Quem garante que esse endereco enderaca UM termo so e o
-- TestEveryCatalogTermHasAUniqueAddress.
--
-- O ESCOPO e a mesma coluna da 00017: `characterId` nulo e a campanha inteira,
-- preenchido e aquela ficha.
--
-- Sem chave unica, e de proposito: calar duas vezes e calar. Uma UNIQUE com
-- coluna nula nem cumpriria o papel, porque no SQLite dois NULL sao distintos.
--
-- Tabela e nao coluna, e comentario sem acento: as duas razoes sao do
-- engine-go/CLAUDE.md, secao do sqlc.

-- +goose Up
CREATE TABLE campaign_silences (
  campaignId  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  characterId INTEGER REFERENCES characters(id) ON DELETE CASCADE,
  term        TEXT NOT NULL,
  updatedAt   TEXT NOT NULL
);

CREATE INDEX campaign_silences_by_campaign ON campaign_silences(campaignId);

-- +goose Down
DROP TABLE campaign_silences;
