-- +goose Up
-- +goose StatementBegin
-- O CHAO DO TABULEIRO PASSA A SE CHAMAR EM INGLES (ALE-301).
--
-- A varredura de idioma da ALE-301 traduziu as classes CSS da casa, e a do chao
-- tem uma metade que NAO e CSS: a classe e montada como `ground-` + o valor de
-- `BoardState.Terrain`, que e GRAVADO. Traduzir so a classe deixaria o servidor
-- escrevendo `ground-pedra` contra uma folha que so sabe pintar `ground-stone`
-- -- e o modo de falhar desta familia e o pior: o quadrado aparece SEM TEXTURA,
-- sem erro em lugar nenhum.
--
-- Decisao do dono, ALE-301: traduzir tudo e migrar o dado, em vez de deixar o
-- identificador metade em cada lingua.
--
-- ESTA MIGRACAO EXISTE PELO BANCO QUE EU NAO VEJO, como a 00012. O estado do
-- tabuleiro e um JSON na coluna `state` de DUAS tabelas -- `open_boards`, que e
-- a cena no ar, e `campaign_places`, que e o acervo de lugares guardados. O
-- banco do dono tem taverna e cripta gravadas de sessoes passadas; sem esta
-- troca elas reabrem sem chao.
--
-- `json_set` sobre `$.terrain` e cirurgico: ele nao reescreve o resto do JSON e
-- nao mexe em linha cujo terreno ja esteja em ingles, porque o `WHERE` casa o
-- valor antigo. Rodar duas vezes nao muda nada.
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'stone')  WHERE json_extract(state, '$.terrain') = 'pedra';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'tavern') WHERE json_extract(state, '$.terrain') = 'taverna';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'forest') WHERE json_extract(state, '$.terrain') = 'floresta';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'wilds')  WHERE json_extract(state, '$.terrain') = 'ermo';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'crypt')  WHERE json_extract(state, '$.terrain') = 'cripta';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'paper')  WHERE json_extract(state, '$.terrain') = 'papel';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'stone')  WHERE json_extract(state, '$.terrain') = 'pedra';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'tavern') WHERE json_extract(state, '$.terrain') = 'taverna';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'forest') WHERE json_extract(state, '$.terrain') = 'floresta';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'wilds')  WHERE json_extract(state, '$.terrain') = 'ermo';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'crypt')  WHERE json_extract(state, '$.terrain') = 'cripta';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'paper')  WHERE json_extract(state, '$.terrain') = 'papel';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'pedra')    WHERE json_extract(state, '$.terrain') = 'stone';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'taverna')  WHERE json_extract(state, '$.terrain') = 'tavern';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'floresta') WHERE json_extract(state, '$.terrain') = 'forest';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'ermo')     WHERE json_extract(state, '$.terrain') = 'wilds';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'cripta')   WHERE json_extract(state, '$.terrain') = 'crypt';
UPDATE open_boards     SET state = json_set(state, '$.terrain', 'papel')    WHERE json_extract(state, '$.terrain') = 'paper';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'pedra')    WHERE json_extract(state, '$.terrain') = 'stone';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'taverna')  WHERE json_extract(state, '$.terrain') = 'tavern';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'floresta') WHERE json_extract(state, '$.terrain') = 'forest';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'ermo')     WHERE json_extract(state, '$.terrain') = 'wilds';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'cripta')   WHERE json_extract(state, '$.terrain') = 'crypt';
UPDATE campaign_places SET state = json_set(state, '$.terrain', 'papel')    WHERE json_extract(state, '$.terrain') = 'paper';
-- +goose StatementEnd
