-- +goose Up
-- +goose StatementBegin
-- OS IDS ACENTUADOS DO CATALOGO VIRAM ASCII (ALE-152).
--
-- Dos 1646 slugs do catalogo, tres carregavam acento e os outros 1643 nao:
-- `enfeiticado` (era com cedilha), `heroi-campones` (era com circunflexo) e a
-- ativacao derivada dele. A grafia irregular e o que faz toda copia, toda URL e
-- todo teste escrito de memoria errarem NAQUELES ids e em nenhum outro -- e o de
-- condicao ja quebrou uma vez, dando 400 ao ser aplicado (ALE-122).
--
-- ESTA MIGRACAO EXISTE PELO BANCO QUE EU NAO VEJO. Medido antes de escreve-la:
-- ZERO linhas com esses ids no banco de desenvolvimento, no do e2e e no seed
-- gerado. O banco de producao do dono pode ter condicao aplicada numa ficha ou
-- numa sessao salva, e sem esta troca ela viraria orfa EM SILENCIO -- o filtro
-- descarta id desconhecido, que e o certo para id inventado e o errado aqui.
--
-- `replace` e IDEMPOTENTE e nao precisa de guarda: num banco sem o id antigo ele
-- reescreve a coluna com o mesmo valor. E por isso a migracao nao tem `WHERE`:
-- a condicao custaria mais que o trabalho que ela evitaria, em tabelas deste
-- tamanho.
--
-- AS COLUNAS FORAM MAPEADAS, E DUAS FICARAM DE FORA COM MOTIVO:
--
--   characters.activeConditions   entra -- e a lista de condicoes da ficha
--   sessions.runtimeState         entra -- guarda a fila, com as condicoes de
--                                          cada combatente
--   characters.origin             NAO entra: ela guarda o NOME ("Heroi
--                                 Campones"), nao o slug. Conferido no banco de
--                                 desenvolvimento.
--   character_power_uses.powerId  NAO entra: ela guarda id de PODER, e o
--                                 `origin.heroi-campones.coracao-heroico` e id
--                                 de ATIVACAO, que o `LookupActivation` resolve
--                                 a partir do catalogo. Zero linhas com
--                                 `campon` nos dois bancos que eu vejo.
--
-- Se o banco de producao tiver `powerId` com a grafia antiga, o poder some da
-- ficha em silencio -- e o sintoma seria "o poder nao aparece mais", longe da
-- causa. Vale conferir la antes de subir:
--
--   select powerId from character_power_uses where powerId like '%campon%';
UPDATE characters
SET activeConditions = replace(activeConditions, '"enfeitiçado"', '"enfeiticado"');

UPDATE sessions
SET runtimeState = replace(runtimeState, '"enfeitiçado"', '"enfeiticado"');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- A VOLTA reescreve o id antigo, e ela e honesta sobre o que nao consegue: um
-- dado gravado DEPOIS desta migracao, ja com a grafia nova, e indistinguivel de
-- um dado migrado. Voltar devolve os dois para a grafia acentuada, que e o
-- estado que o catalogo daquele momento espera.
UPDATE characters
SET activeConditions = replace(activeConditions, '"enfeiticado"', '"enfeitiçado"');

UPDATE sessions
SET runtimeState = replace(runtimeState, '"enfeiticado"', '"enfeitiçado"');
-- +goose StatementEnd
