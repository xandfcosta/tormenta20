-- +goose Up
-- Initial schema ported 1:1 from backend/prisma/schema.prisma. SQLite affinities:
-- Int→INTEGER, Float→REAL, String→TEXT, Boolean→INTEGER(0/1), DateTime→TEXT (ISO-8601,
-- set in Go). Autoincrement PKs, ON DELETE CASCADE + uniques + indexes match Prisma.
-- PRAGMA foreign_keys is enabled per-connection in db.go.

CREATE TABLE users (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  email        TEXT NOT NULL UNIQUE,
  name         TEXT,
  passwordHash TEXT NOT NULL,
  createdAt    TEXT NOT NULL,
  updatedAt    TEXT NOT NULL
);

CREATE TABLE campaigns (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  ownerId     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  description TEXT,
  inviteToken TEXT UNIQUE,
  createdAt   TEXT NOT NULL,
  updatedAt   TEXT NOT NULL
);
CREATE INDEX idx_campaigns_ownerId ON campaigns(ownerId);

CREATE TABLE characters (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  ownerId              INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name                 TEXT NOT NULL,
  origin               TEXT NOT NULL,
  god                  TEXT,
  godPower             TEXT NOT NULL DEFAULT '',
  tibar                REAL NOT NULL DEFAULT 0,
  level                INTEGER NOT NULL DEFAULT 1,
  hpMax                INTEGER NOT NULL,
  hpCurrent            INTEGER NOT NULL,
  mpMax                INTEGER NOT NULL,
  mpCurrent            INTEGER NOT NULL,
  strength             INTEGER NOT NULL,
  dexterity            INTEGER NOT NULL,
  constitution         INTEGER NOT NULL,
  intelligence         INTEGER NOT NULL,
  wisdom               INTEGER NOT NULL DEFAULT 0,
  charisma             INTEGER NOT NULL,
  size                 TEXT NOT NULL DEFAULT 'Médio',
  displacement         INTEGER NOT NULL DEFAULT 9,
  proficiencies        TEXT NOT NULL DEFAULT '[]',
  raceAbilityChoices   TEXT NOT NULL DEFAULT '[]',
  raceAttributeChoices TEXT NOT NULL DEFAULT '{}',
  secondaryRaceChoices TEXT NOT NULL DEFAULT '[]',
  originChoices        TEXT NOT NULL DEFAULT '[]',
  classPowers          TEXT NOT NULL DEFAULT '[]',
  classChoices         TEXT NOT NULL DEFAULT '{}',
  powerChoices         TEXT NOT NULL DEFAULT '{}',
  activeConditions     TEXT NOT NULL DEFAULT '[]',
  createdAt            TEXT NOT NULL,
  updatedAt            TEXT NOT NULL
);
CREATE INDEX idx_characters_ownerId ON characters(ownerId);

CREATE TABLE campaign_members (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  campaignId  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  role        TEXT NOT NULL DEFAULT 'player',
  addedAt     TEXT NOT NULL,
  UNIQUE (campaignId, characterId)
);
CREATE INDEX idx_campaign_members_campaignId ON campaign_members(campaignId);
CREATE INDEX idx_campaign_members_characterId ON campaign_members(characterId);

CREATE TABLE sessions (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  campaignId    INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  title         TEXT,
  sessionNumber INTEGER NOT NULL,
  notes         TEXT,
  status        TEXT NOT NULL DEFAULT 'planned',
  startedAt     TEXT,
  endedAt       TEXT,
  createdAt     TEXT NOT NULL,
  updatedAt     TEXT NOT NULL,
  runtimeState  TEXT NOT NULL DEFAULT '{"initiative":[],"round":0,"turnIndex":-1}'
);
CREATE INDEX idx_sessions_campaignId ON sessions(campaignId);

CREATE TABLE character_spells (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  characterId    INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  catalogSpellId TEXT NOT NULL,
  prepared       INTEGER NOT NULL DEFAULT 0,
  learnedAt      TEXT NOT NULL,
  UNIQUE (characterId, catalogSpellId)
);
CREATE INDEX idx_character_spells_characterId ON character_spells(characterId);

CREATE TABLE active_effects (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  source      TEXT NOT NULL DEFAULT 'consumable',
  catalogId   TEXT NOT NULL,
  scope       TEXT NOT NULL,
  modifiers   TEXT NOT NULL,
  createdAt   TEXT NOT NULL,
  UNIQUE (characterId, catalogId, scope)
);
CREATE INDEX idx_active_effects_characterId ON active_effects(characterId);

CREATE TABLE character_items (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  characterId  INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  catalogId    TEXT,
  name         TEXT NOT NULL,
  quantity     INTEGER NOT NULL DEFAULT 1,
  slots        REAL NOT NULL DEFAULT 1,
  equipped     TEXT,
  improvements TEXT NOT NULL DEFAULT '[]',
  material     TEXT,
  createdAt    TEXT NOT NULL
);
CREATE INDEX idx_character_items_characterId ON character_items(characterId);

CREATE TABLE character_races (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  race        TEXT NOT NULL,
  UNIQUE (characterId, race)
);

CREATE TABLE character_classes (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  className   TEXT NOT NULL,
  level       INTEGER NOT NULL DEFAULT 1,
  UNIQUE (characterId, className)
);

CREATE TABLE character_expertises (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  characterId INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  attribute   TEXT NOT NULL,
  trained     INTEGER NOT NULL DEFAULT 0,
  custom      INTEGER NOT NULL DEFAULT 0,
  UNIQUE (characterId, name)
);

-- +goose Down
DROP TABLE character_expertises;
DROP TABLE character_classes;
DROP TABLE character_races;
DROP TABLE character_items;
DROP TABLE active_effects;
DROP TABLE character_spells;
DROP TABLE sessions;
DROP TABLE campaign_members;
DROP TABLE characters;
DROP TABLE campaigns;
DROP TABLE users;
