-- A EMENDA DE POPULACAO: o que a mesa CONCEDE, e a quem (ALE-387).
--
-- A de verbete (00016) muda o que uma COISA e, e vale para o mundo inteiro.
-- Esta muda quem TEM as coisas, e por isso carrega escopo.
--
-- O ESCOPO E A COLUNA `characterId`: nula quer dizer a campanha inteira, e
-- preenchida quer dizer aquela ficha. Nula como "todos" e nao uma linha por
-- personagem porque o mestre escreve UMA regra para a mesa -- expandi-la em N
-- linhas faria a mesma regra ser editada em N lugares, e a ficha que entrasse
-- depois nasceria sem ela.
--
-- O `characterId` aponta para o CLONE que vive na campanha, e nao para o molde
-- do elenco: o molde nao esta em mesa nenhuma. O ON DELETE CASCADE dos dois
-- lados e o que impede a regra de sobreviver ao que ela descreve.
--
-- O `label` e obrigatorio porque a ficha MOSTRA a procedencia de cada termo.
-- Sem ele o jogador ve um numero aparecer sem nome, que e a pior forma de uma
-- regra da mesa existir.
--
-- Tabela propria e nao coluna, e comentario sem acento: as duas razoes sao do
-- engine-go/CLAUDE.md, secao do sqlc.

-- +goose Up
CREATE TABLE campaign_grants (
  id          TEXT PRIMARY KEY,
  campaignId  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  characterId INTEGER REFERENCES characters(id) ON DELETE CASCADE,
  label       TEXT NOT NULL,
  modifiers   TEXT NOT NULL,
  updatedAt   TEXT NOT NULL
);

CREATE INDEX campaign_grants_by_campaign ON campaign_grants(campaignId);

-- +goose Down
DROP TABLE campaign_grants;
