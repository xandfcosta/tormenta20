-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_ActiveEffect" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "characterId" INTEGER NOT NULL,
    "source" TEXT NOT NULL DEFAULT 'consumable',
    "catalogId" TEXT NOT NULL,
    "scope" TEXT NOT NULL,
    "modifiers" TEXT NOT NULL,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "ActiveEffect_characterId_fkey" FOREIGN KEY ("characterId") REFERENCES "Character" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);
INSERT INTO "new_ActiveEffect" ("catalogId", "characterId", "createdAt", "id", "modifiers", "scope") SELECT "catalogId", "characterId", "createdAt", "id", "modifiers", "scope" FROM "ActiveEffect";
DROP TABLE "ActiveEffect";
ALTER TABLE "new_ActiveEffect" RENAME TO "ActiveEffect";
CREATE INDEX "ActiveEffect_characterId_idx" ON "ActiveEffect"("characterId");
CREATE UNIQUE INDEX "ActiveEffect_characterId_catalogId_scope_key" ON "ActiveEffect"("characterId", "catalogId", "scope");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
