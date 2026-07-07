-- SP2: spellbook persistence — CharacterSpell join table so a
-- caster's learned magias survive across sessions. `catalogSpellId` is
-- a t20-data SPELL_CATALOG id (string). `prepared` gates cast for
-- Clérigo/Druida/Mago-style casters; Bardo/Feiticeiro can cast any
-- learned spell so their rows stay prepared=false without ill effect.

CREATE TABLE "CharacterSpell" (
  "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  "characterId" INTEGER NOT NULL,
  "catalogSpellId" TEXT NOT NULL,
  "prepared" BOOLEAN NOT NULL DEFAULT false,
  "learnedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "CharacterSpell_characterId_fkey"
    FOREIGN KEY ("characterId") REFERENCES "Character" ("id")
    ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX "CharacterSpell_characterId_catalogSpellId_key"
  ON "CharacterSpell"("characterId", "catalogSpellId");

CREATE INDEX "CharacterSpell_characterId_idx"
  ON "CharacterSpell"("characterId");
