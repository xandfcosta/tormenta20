-- O OPT-IN DO JOGADOR PASSA A USAR O ENDERECO DE TERMO (ALE-387).
--
-- Havia DOIS enderecos para a mesma coisa. O `ConditionalID`
-- (`fonte::alvo::nota::valor::tipo`) enderecava o condicional que o jogador
-- liga; o `TermID` (`fonte::alvo::escala::condicao`) enderaca o termo que o
-- mestre cala. Um conceito, duas grafias -- e a do jogador era a fragil, porque
-- carregava o VALOR: uma errata do livro a fazia deixar de casar em silencio.
--
-- Sobrou o `TermID`. As linhas gravadas na forma velha nunca casariam com nada,
-- entao elas SAEM: linha que nao casa e lixo invisivel, e a ficha mostraria o
-- situacional desligado para sempre sem ninguem entender por que.
--
-- O que isso custa a quem ja jogava: quem estava com um situacional ou uma
-- postura LIGADA precisa liga-la de novo. E um clique, e e estado de jogo --
-- nao ha ficha, dano nem item envolvido.
--
-- A troca foi MEDIDA antes de ser feita: a tabela tinha zero linhas no banco de
-- desenvolvimento. O guarda que prendia a forma antiga dizia, com razao, que
-- muda-la exigiria uma migracao; esta e ela.

-- +goose Up
DELETE FROM character_conditionals;

-- +goose Down
-- Nao ha volta: a forma antiga nao se reconstroi a partir da nova, porque o
-- endereco novo nao carrega o valor nem a nota que a antiga usava.
SELECT 1;
