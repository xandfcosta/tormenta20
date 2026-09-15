-- O estado de mesa da ficha sai do navegador (ALE-222). Decisao do dono,
-- 2026-08-22: "o servidor mantem estado, ponto final."
--
-- Tres coisas viviam em localStorage e contrariavam isso. Os comentarios dos
-- stores registravam a decisao CONTRARIA, tomada na migracao do zustand
-- ("It is a CLIENT choice, not server state" / "table-trust, not server
-- state"); esta migracao a sobrescreve, e os comentarios de la sao atualizados
-- em vez de apagados, para o historico nao mentir.
--
-- TRES TABELAS e nao uma, porque os ciclos de vida sao diferentes:
--   * situacional  e escolha duravel, ninguem a limpa;
--   * uso de poder zera no descanso de cena (os "1/cena") e de dia (ambos);
--   * postura morre com a cena.
-- Uma tabela so exigiria uma coluna de tipo para dizer qual regra de limpeza
-- se aplica, e essa coluna seria a unica leitora dela mesma.
--
-- Tabelas novas e nao colunas em characters: o sqlc v1.31.1 nao propaga
-- ALTER TABLE ADD COLUMN entre arquivos de migracao (ALE-124), e alem disso
-- sao relacoes 1:N.
--
-- CUIDADO COM O NOME: as CONDICOES do livro (p394-395, Caido/Atordoado/Cego)
-- ja existem e sao OUTRA COISA - elas chegam de fora e moram na coluna JSON
-- characters.activeConditions. character_conditionals e o opt-in do JOGADOR,
-- que muda o calculo da propria ficha. Ver a colisao C6 no GLOSSARIO.md.
--
-- Elas seguem em coluna e este vai para tabela por causa do sqlc, nao por
-- diferenca de forma: coluna nova em arquivo de migracao novo desaparece do
-- catalogo (ALE-124), entao "espelhar o activeConditions" nao era opcao.
--
-- Comentario em ASCII de proposito: o sqlc conta bytes e runas diferente e
-- trunca SQL em silencio quando ha acento acima da query (ALE-120).

-- +goose Up
CREATE TABLE character_conditionals (
  characterId   INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  conditionalId TEXT    NOT NULL,
  PRIMARY KEY (characterId, conditionalId)
);

-- A chave inclui o escopo porque o MESMO poder tem duas contas independentes:
-- um "1/cena" gasto tres vezes no dia soma 3 em day e 1 em scene.
CREATE TABLE character_power_uses (
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  powerId     TEXT    NOT NULL,
  scope       TEXT    NOT NULL,
  used        INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (characterId, powerId, scope)
);

-- O que foi PAGO para entrar na postura. Existe para sair nao devolver PM, e
-- por isso guarda o preco e nao o estado ligado/desligado - quem diz se ela
-- esta ligada e o situacional correspondente, na primeira tabela.
CREATE TABLE character_stances (
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  flag        TEXT    NOT NULL,
  steps       INTEGER NOT NULL DEFAULT 0,
  pmPaid      INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (characterId, flag)
);

-- +goose Down
DROP TABLE character_stances;
DROP TABLE character_power_uses;
DROP TABLE character_conditionals;
