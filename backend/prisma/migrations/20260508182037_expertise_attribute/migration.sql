/*
  Warnings:

  - Added the required column `attribute` to the `CharacterExpertise` table without a default value. This is not possible if the table is not empty.

*/
-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_CharacterExpertise" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "characterId" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "attribute" TEXT NOT NULL,
    "trained" BOOLEAN NOT NULL DEFAULT false,
    "others" INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT "CharacterExpertise_characterId_fkey" FOREIGN KEY ("characterId") REFERENCES "Character" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);
INSERT INTO "new_CharacterExpertise" ("characterId", "id", "name", "attribute", "others", "trained")
SELECT
  "characterId",
  "id",
  "name",
  CASE "name"
    WHEN 'Acrobacia' THEN 'dexterity'
    WHEN 'Adestramento' THEN 'charisma'
    WHEN 'Atletismo' THEN 'strength'
    WHEN 'Atuação' THEN 'charisma'
    WHEN 'Cavalgar' THEN 'dexterity'
    WHEN 'Conhecimento' THEN 'intelligence'
    WHEN 'Cura' THEN 'wisdom'
    WHEN 'Diplomacia' THEN 'charisma'
    WHEN 'Enganação' THEN 'charisma'
    WHEN 'Fortitude' THEN 'constitution'
    WHEN 'Furtividade' THEN 'dexterity'
    WHEN 'Guerra' THEN 'intelligence'
    WHEN 'Iniciativa' THEN 'dexterity'
    WHEN 'Intimidação' THEN 'charisma'
    WHEN 'Intuição' THEN 'wisdom'
    WHEN 'Investigação' THEN 'intelligence'
    WHEN 'Jogatina' THEN 'charisma'
    WHEN 'Ladinagem' THEN 'dexterity'
    WHEN 'Luta' THEN 'strength'
    WHEN 'Misticismo' THEN 'intelligence'
    WHEN 'Nobreza' THEN 'intelligence'
    WHEN 'Percepção' THEN 'wisdom'
    WHEN 'Pilotagem' THEN 'dexterity'
    WHEN 'Pontaria' THEN 'dexterity'
    WHEN 'Reflexos' THEN 'dexterity'
    WHEN 'Religião' THEN 'wisdom'
    WHEN 'Sobrevivência' THEN 'wisdom'
    WHEN 'Vontade' THEN 'wisdom'
    ELSE 'strength'
  END,
  "others",
  "trained"
FROM "CharacterExpertise";
DROP TABLE "CharacterExpertise";
ALTER TABLE "new_CharacterExpertise" RENAME TO "CharacterExpertise";
CREATE UNIQUE INDEX "CharacterExpertise_characterId_name_key" ON "CharacterExpertise"("characterId", "name");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
