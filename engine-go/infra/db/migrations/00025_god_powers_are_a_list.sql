-- +goose Up
-- +goose StatementBegin
-- O PODER CONCEDIDO VIRA LISTA (ALE-578). Clerigo, druida e paladino recebem dois (p57, p61, p82), e a
-- coluna guardava um nome so. Ela passa a ser uma lista em JSON, como `classPowers`, e cada ficha fica
-- com o poder que ja tinha.
ALTER TABLE characters RENAME COLUMN godPower TO godPowers;
UPDATE characters SET godPowers = CASE WHEN godPowers = '' THEN '[]' ELSE json_array(godPowers) END;
-- +goose StatementEnd

-- +goose Down
-- A lista nao volta a caber numa coluna de um nome so: o backup do deploy e quem guarda.
