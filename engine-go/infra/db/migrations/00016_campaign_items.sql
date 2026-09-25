-- A EMENDA DE CATALOGO DA CAMPANHA (ALE-387).
--
-- A mesa ACRESCENTA modificador a um verbete que o livro ja tem: numa campanha
-- o medalhao de prata concede +1 em Luta alem do +1 de limite de PM da p160.
--
-- ACRESCENTA e nao substitui, e a diferenca e o defeito que ela evita: uma
-- campanha que redeclarasse o verbete inteiro copiaria preco, espacos e eixo de
-- equipar, e no dia em que o livro fosse corrigido a copia nao seguiria.
--
-- O ID E O MESMO DO LIVRO, de proposito. A linha do personagem continua
-- apontando para `medalhao-de-prata`, entao o mesmo heroi levado a outra mesa
-- volta a ter o item do livro sozinho -- fora de campanha, so o livro (decisao
-- do dono).
--
-- O `adds` e um ARRAY JSON de modificadores, no mesmo formato de `items.json`:
-- um formato so para aprender, e o motor ja sabe le-lo.
--
-- Tabela e nao coluna, e comentario sem acento: as duas razoes sao do
-- engine-go/CLAUDE.md, secao do sqlc.

-- +goose Up
CREATE TABLE campaign_items (
  campaignId INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  itemId     TEXT NOT NULL,
  adds       TEXT NOT NULL,
  updatedAt  TEXT NOT NULL,
  PRIMARY KEY (campaignId, itemId)
);

-- +goose Down
DROP TABLE campaign_items;
