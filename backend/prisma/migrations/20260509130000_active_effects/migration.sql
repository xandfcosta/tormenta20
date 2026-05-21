-- CreateTable
CREATE TABLE "ActiveEffect" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "characterId" INTEGER NOT NULL,
    "catalogId" TEXT NOT NULL,
    "scope" TEXT NOT NULL,
    "modifiers" TEXT NOT NULL,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "ActiveEffect_characterId_fkey" FOREIGN KEY ("characterId") REFERENCES "Character" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);

-- CreateIndex
CREATE INDEX "ActiveEffect_characterId_idx" ON "ActiveEffect"("characterId");

