-- QUEM SUSTENTA E QUEM CONJUROU (ALE-700): a manutencao e o limite de uma sustentada (p227)
-- passam a contar so a linha que o Conjurar da propria ficha abriu ou tocou, e essa marca e a
-- linha em `effect_cast_augments`. A magia so aplicada numa ficha veio de outro conjurador.
--
-- As sustentadas gravadas antes desta regra nao tem a marca, e sem ela deixariam de cobrar
-- manutencao. Ganham-na as que a propria ficha conhece no grimorio: e quem conjurou. A
-- sustentada de quem nao conhece a magia era a do aliado, e segue sem marca.
-- +goose Up
INSERT INTO effect_cast_augments (effectId, augments)
SELECT e.id, '[]' FROM active_effects e
WHERE e.source = 'spell' AND e.scope = 'sustained'
  AND NOT EXISTS (SELECT 1 FROM effect_cast_augments c WHERE c.effectId = e.id)
  AND EXISTS (
    SELECT 1 FROM character_spells s
    WHERE s.characterId = e.characterId AND s.catalogSpellId = e.catalogId
  );

-- +goose Down
-- A marca vazia nao distingue a desta migracao da de uma conjuracao sem aprimoramento, e
-- apaga-la tiraria a manutencao de quem conjurou: a volta nao desfaz nada.
SELECT 1;
