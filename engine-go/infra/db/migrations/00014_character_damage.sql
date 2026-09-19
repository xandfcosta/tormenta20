-- +goose Up
-- +goose StatementBegin
-- O QUE SE GUARDA PASSA A SER O DANO, E NAO O ATUAL (ALE-355).
--
-- Decisao do dono: o PV maximo e DERIVADO. Se o catalogo mudou, as regras do
-- mundo mudaram, logo os personagens acompanham. Hoje `hpMax` e uma COLUNA,
-- recomputada so em gesto de escrita -- nascer, passo de atributo, degrau de
-- nivel --, e um catalogo que mude o poco de uma classe nao alcanca ninguem ate
-- o proximo desses gestos.
--
-- ISSO NAO E HIPOTESE: o `bardo-versatil-nv7` da propria semente estava com 31
-- PM gravados contra 33 derivados, porque o numero foi calculado quando o
-- Carisma total dele era 3. O oraculo carregava as duas respostas lado a lado e
-- nenhuma suite comparava as duas.
--
-- # Por que o ATUAL nao pode continuar sendo o que se guarda
--
-- As duas regras de "maximo novo" recebem o maximo ANTIGO para diferenciar:
--
--   ShiftedByNewMax  ->  HpCurrent + (pvMax - atuais.HpMax)
--   ClampedToNewMax  ->  min(HpCurrent, pvMax)
--
-- Com o maximo derivado nao existe maximo antigo. Guardar o DANO resolve por
-- construcao: `pvAtual = pvMaximoDerivado - dano`. Subir de nivel passa a
-- entregar os PV novos preenchidos sem regra nenhuma, e um catalogo que mude o
-- poco alcanca todo mundo no desenho seguinte.
--
-- # TABELA e nao coluna, e a razao e o gerador
--
-- O sqlc v1.31.1 nao propaga `ALTER TABLE ADD COLUMN` entre arquivos de
-- migracao: a coluna some do catalogo e toda query sobre ela falha com
-- "column does not exist" -- apontando a QUERY, nao a migracao (ALE-124).
-- Medido aqui tambem, com `hpDamage` como coluna: `column "hpdamage" does not
-- exist`. Mesma saida que a 00005 tomou com o tabuleiro.
--
-- E a tabela compra o que a coluna nao dava: "sem dano" vira fato do SCHEMA --
-- a linha existe ou nao existe --, em vez de um zero que e indistinguivel de um
-- personagem que nunca foi tocado. Nascer nao precisa lembrar de escrever nada.
--
-- # O piso em zero
--
-- Um personagem com `hpCurrent > hpMax` -- qualquer um cujo maximo tenha
-- encolhido sem o atual ser aparado -- produziria dano NEGATIVO, que a tela
-- desenharia como barra alem do fim.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).
CREATE TABLE character_damage (
  characterId INTEGER PRIMARY KEY NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  hpDamage    INTEGER NOT NULL DEFAULT 0,
  mpSpent     INTEGER NOT NULL DEFAULT 0
);

-- A conta reversivel e exata ENQUANTO `hpMax` ainda existir na linha, e e por
-- isso que esta migracao nao apaga nada: as colunas velhas sao a rede de
-- seguranca ate o codigo passar a ler o derivado. Elas saem numa migracao
-- propria, depois.
--
-- So quem TEM dano ganha linha: um personagem intacto nao precisa de registro.
INSERT INTO character_damage (characterId, hpDamage, mpSpent)
SELECT id, MAX(0, hpMax - hpCurrent), MAX(0, mpMax - mpCurrent)
  FROM characters
 WHERE hpCurrent < hpMax OR mpCurrent < mpMax;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- A volta so APAGA a tabela, e nao reescreve o atual a partir dela.
--
-- Reescrever seria a conta ao contrario e parece mais simetrico -- mas ela so
-- vale num mundo em que o dano ja e a autoridade. Enquanto o codigo ainda
-- gravar `hpCurrent`, um dano velho sobrescreveria o atual de verdade: um
-- personagem sem linha de dano voltaria com PV cheio depois de uma sessao
-- inteira apanhando.
--
-- Como a IDA nao tocou em `hpCurrent`, apagar e o inverso exato nos dois mundos.
DROP TABLE character_damage;
-- +goose StatementEnd
