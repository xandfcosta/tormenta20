-- Convite de CONTA (ALE-120): o registro deixou de ser aberto quando a mesa
-- passou a ser servida na LAN, e passar a exigir convite é o que impede um
-- vizinho de criar conta em http://<ip>:3001. Não confundir com o convite de
-- CAMPANHA (campaigns.inviteToken), que traz um usuário JÁ EXISTENTE para uma
-- mesa: este aqui é o que faz a conta existir, e por isso é uso único.
--
-- usedBy fica NULL quando a conta é apagada depois (SET NULL), porque o que
-- importa auditar é que o convite foi gasto, não quem sobrou no banco.

-- +goose Up
CREATE TABLE account_invites (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  token     TEXT NOT NULL UNIQUE,
  createdBy INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  createdAt TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  usedAt    TEXT,
  usedBy    INTEGER REFERENCES users(id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE account_invites;
