-- CreateTable
CREATE TABLE "CharacterExpertise" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "characterId" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "trained" BOOLEAN NOT NULL DEFAULT false,
    "others" INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT "CharacterExpertise_characterId_fkey" FOREIGN KEY ("characterId") REFERENCES "Character" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);

-- CreateIndex
CREATE UNIQUE INDEX "CharacterExpertise_characterId_name_key" ON "CharacterExpertise"("characterId", "name");

-- Backfill: every existing Character gets the canonical 28 expertise rows.
INSERT INTO "CharacterExpertise" ("characterId", "name", "trained", "others")
SELECT c."id", n."name", 0, 0
FROM "Character" c
CROSS JOIN (
    SELECT 'Acrobacia' AS name UNION ALL
    SELECT 'Adestramento' UNION ALL
    SELECT 'Atletismo' UNION ALL
    SELECT 'Atuação' UNION ALL
    SELECT 'Cavalgar' UNION ALL
    SELECT 'Conhecimento' UNION ALL
    SELECT 'Cura' UNION ALL
    SELECT 'Diplomacia' UNION ALL
    SELECT 'Enganação' UNION ALL
    SELECT 'Fortitude' UNION ALL
    SELECT 'Furtividade' UNION ALL
    SELECT 'Guerra' UNION ALL
    SELECT 'Iniciativa' UNION ALL
    SELECT 'Intimidação' UNION ALL
    SELECT 'Intuição' UNION ALL
    SELECT 'Investigação' UNION ALL
    SELECT 'Jogatina' UNION ALL
    SELECT 'Ladinagem' UNION ALL
    SELECT 'Luta' UNION ALL
    SELECT 'Misticismo' UNION ALL
    SELECT 'Nobreza' UNION ALL
    SELECT 'Percepção' UNION ALL
    SELECT 'Pilotagem' UNION ALL
    SELECT 'Pontaria' UNION ALL
    SELECT 'Reflexos' UNION ALL
    SELECT 'Religião' UNION ALL
    SELECT 'Sobrevivência' UNION ALL
    SELECT 'Vontade'
) n;
