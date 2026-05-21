-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_Character" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "ownerId" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "origin" TEXT NOT NULL,
    "god" TEXT,
    "level" INTEGER NOT NULL DEFAULT 1,
    "hpMax" INTEGER NOT NULL,
    "hpCurrent" INTEGER NOT NULL,
    "mpMax" INTEGER NOT NULL,
    "mpCurrent" INTEGER NOT NULL,
    "strength" INTEGER NOT NULL,
    "dexterity" INTEGER NOT NULL,
    "constitution" INTEGER NOT NULL,
    "intelligence" INTEGER NOT NULL,
    "wisdom" INTEGER NOT NULL DEFAULT 0,
    "charisma" INTEGER NOT NULL,
    "size" TEXT NOT NULL DEFAULT 'Médio',
    "displacement" INTEGER NOT NULL DEFAULT 9,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" DATETIME NOT NULL,
    CONSTRAINT "Character_ownerId_fkey" FOREIGN KEY ("ownerId") REFERENCES "User" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);
INSERT INTO "new_Character" ("charisma", "constitution", "createdAt", "dexterity", "god", "hpCurrent", "hpMax", "id", "intelligence", "level", "mpCurrent", "mpMax", "name", "origin", "ownerId", "strength", "updatedAt", "wisdom") SELECT "charisma", "constitution", "createdAt", "dexterity", "god", "hpCurrent", "hpMax", "id", "intelligence", "level", "mpCurrent", "mpMax", "name", "origin", "ownerId", "strength", "updatedAt", "wisdom" FROM "Character";
DROP TABLE "Character";
ALTER TABLE "new_Character" RENAME TO "Character";
CREATE INDEX "Character_ownerId_idx" ON "Character"("ownerId");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
