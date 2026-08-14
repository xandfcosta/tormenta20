-- Redefinicao de senha por link (ALE-120). O admin NUNCA digita nem ve a senha
-- de ninguem: ele gera um link de uso unico e quem recebe escolhe a propria.
--
-- Tabela separada do account_invites de proposito: convite cria conta e nao tem
-- dono ainda; este aponta para uma conta que JA existe. Uma coluna `kind`
-- juntaria as duas com uma FK que so vale em metade das linhas.

-- +goose Up
CREATE TABLE password_resets (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  token     TEXT NOT NULL UNIQUE,
  userId    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  createdBy INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  createdAt TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  usedAt    TEXT
);

-- +goose Down
DROP TABLE password_resets;
