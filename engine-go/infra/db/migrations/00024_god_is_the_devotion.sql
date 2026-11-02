-- +goose Up
-- +goose StatementBegin
-- O DEUS DA FICHA E A FONTE UNICA DA DEVOCAO (ALE-518).
--
-- O devoto morava em dois lugares: `god`, com o NOME do deus, e `classChoices`, com o ID
-- por classe, e os dois podiam discordar. Fica o `god`, com o ID, que e o que as opcoes e os
-- pre-requisitos usam. Onde os dois existiam, vale o `god`.
UPDATE characters SET god = lower(god) WHERE god IS NOT NULL AND god <> '';

UPDATE characters
SET god = (
  SELECT json_extract(choice.value, '$.devoto')
  FROM json_each(characters.classChoices) AS choice
  WHERE COALESCE(json_extract(choice.value, '$.devoto'), '') <> ''
  LIMIT 1
)
WHERE COALESCE(god, '') = '' AND classChoices LIKE '%devoto%';

UPDATE characters
SET classChoices = (
  SELECT json_group_object(choice.key, json(json_remove(choice.value, '$.devoto')))
  FROM json_each(characters.classChoices) AS choice
)
WHERE classChoices LIKE '%devoto%';
-- +goose StatementEnd

-- +goose Down
-- O devoto de cada classe nao volta: o backup do deploy e quem guarda.
