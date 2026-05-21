-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_CharacterItem" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "characterId" INTEGER NOT NULL,
    "catalogId" TEXT,
    "name" TEXT NOT NULL,
    "quantity" INTEGER NOT NULL DEFAULT 1,
    "slots" REAL NOT NULL DEFAULT 1,
    "equipped" TEXT,
    "improvements" TEXT NOT NULL DEFAULT '[]',
    "material" TEXT,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "CharacterItem_characterId_fkey" FOREIGN KEY ("characterId") REFERENCES "Character" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);
INSERT INTO "new_CharacterItem" ("catalogId", "characterId", "createdAt", "equipped", "id", "name", "quantity", "slots") SELECT "catalogId", "characterId", "createdAt", "equipped", "id", "name", "quantity", "slots" FROM "CharacterItem";
DROP TABLE "CharacterItem";
ALTER TABLE "new_CharacterItem" RENAME TO "CharacterItem";
CREATE INDEX "CharacterItem_characterId_idx" ON "CharacterItem"("characterId");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;

