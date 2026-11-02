-- +goose Up
-- +goose StatementBegin
-- UM LINK SERVE PARA VARIAS CONTAS (ALE-581). `usedAt` e `usedBy` so guardavam uma
-- conta, entao quem entrou passa a morar numa tabela propria, e revogar deixa de
-- apagar a linha, para o registro de quem entrou sobreviver.
--
-- Tabela nova em vez de RENAME porque o sqlc perde o RENAME do catalogo.
CREATE TABLE account_invite_links (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  token     TEXT NOT NULL UNIQUE,
  createdBy INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  createdAt TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  revokedAt TEXT
);

CREATE TABLE account_invite_uses (
  linkId INTEGER NOT NULL REFERENCES account_invite_links(id) ON DELETE CASCADE,
  userId INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  usedAt TEXT NOT NULL,
  PRIMARY KEY (linkId, userId)
);

-- O link ja gasto nasce revogado: ele foi mandado valendo uma vez so, e nao pode
-- acordar reutilizavel num historico de conversa.
INSERT INTO account_invite_links (id, token, createdBy, createdAt, expiresAt, revokedAt)
SELECT id, token, createdBy, createdAt, expiresAt, usedAt FROM account_invites;

INSERT INTO account_invite_uses (linkId, userId, usedAt)
SELECT id, usedBy, usedAt FROM account_invites WHERE usedBy IS NOT NULL AND usedAt IS NOT NULL;

DROP TABLE account_invites;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE account_invites (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  token     TEXT NOT NULL UNIQUE,
  createdBy INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  createdAt TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  usedAt    TEXT,
  usedBy    INTEGER REFERENCES users(id) ON DELETE SET NULL
);

-- O modelo antigo so guarda uma conta por link, e revogar era apagar a linha.
INSERT INTO account_invites (id, token, createdBy, createdAt, expiresAt, usedAt, usedBy)
SELECT l.id, l.token, l.createdBy, l.createdAt, l.expiresAt,
  (SELECT MIN(u.usedAt) FROM account_invite_uses u WHERE u.linkId = l.id),
  (SELECT u.userId FROM account_invite_uses u WHERE u.linkId = l.id ORDER BY u.usedAt LIMIT 1)
FROM account_invite_links l
WHERE l.revokedAt IS NULL OR EXISTS (SELECT 1 FROM account_invite_uses u WHERE u.linkId = l.id);

DROP TABLE account_invite_uses;
DROP TABLE account_invite_links;
-- +goose StatementEnd
