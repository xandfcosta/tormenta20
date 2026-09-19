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
-- # A JANELA da conta reversivel
--
-- `hpDamage = hpMax - hpCurrent` e exata ENQUANTO `hpMax` ainda existir na
-- linha. Por isso esta migracao acrescenta e preenche, e NAO apaga: as colunas
-- velhas sao a rede de seguranca ate o codigo passar a ler o derivado. Elas
-- saem numa migracao propria, depois.
--
-- # O piso em zero
--
-- Um personagem com `hpCurrent > hpMax` -- o caso do bardo, e de qualquer um
-- cujo maximo tenha encolhido sem o atual ser aparado -- produziria dano
-- NEGATIVO, que a tela desenharia como barra alem do fim. O `MAX(0, ...)` e a
-- mesma invariante que o `withinRange` ja cobra em memoria.
ALTER TABLE characters ADD COLUMN hpDamage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN mpSpent  INTEGER NOT NULL DEFAULT 0;

UPDATE characters
   SET hpDamage = MAX(0, hpMax - hpCurrent),
       mpSpent  = MAX(0, mpMax - mpCurrent);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- A volta so APAGA as colunas, e nao reescreve o atual a partir delas.
--
-- Reescrever seria a conta ao contrario e parece mais simetrico -- mas ela so
-- vale num mundo em que o dano ja e a autoridade. Enquanto o codigo ainda
-- gravar `hpCurrent`, um `hpDamage` velho sobrescreveria o atual de verdade:
-- um personagem sem dano nenhum na coluna nova voltaria com PV cheio depois de
-- uma sessao inteira apanhando.
--
-- Como a IDA nao tocou em `hpCurrent`, apagar e o inverso exato nos dois
-- mundos.
ALTER TABLE characters DROP COLUMN hpDamage;
ALTER TABLE characters DROP COLUMN mpSpent;
-- +goose StatementEnd
