-- +goose Up
-- +goose StatementBegin
-- O PAPEL do membro sai (ALE-287).
--
-- A coluna era sempre 'player': o unico escritor fixava a string e o
-- SetMemberRole nunca teve chamador. E a autorizacao NUNCA a leu -- o roleIn
-- decide por "e o dono da campanha?", entao escrever 'gm' aqui nao mudaria nada
-- do que a pessoa pode fazer.
--
-- Uma coluna que nao decide nada so pode divergir do que decide.
ALTER TABLE campaign_members DROP COLUMN role;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaign_members ADD COLUMN role TEXT NOT NULL DEFAULT 'player';
-- +goose StatementEnd
