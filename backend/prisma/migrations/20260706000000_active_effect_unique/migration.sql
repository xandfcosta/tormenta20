-- Whole-flow audit finding BC2: consumeItem's oncePerDay check was racey.
-- Two concurrent consumes of the same one-per-day catalog item both read
-- an empty active effects list, then both inserted. Add a composite unique
-- to have the DB itself reject the duplicate; app layer catches P2002.
--
-- Cleanup pass first — any pre-existing duplicate rows from the racey
-- window get collapsed (keep the oldest by id, drop the rest).

DELETE FROM "ActiveEffect"
WHERE id NOT IN (
  SELECT MIN(id)
  FROM "ActiveEffect"
  GROUP BY characterId, catalogId, scope
);

-- CreateIndex
CREATE UNIQUE INDEX "ActiveEffect_characterId_catalogId_scope_key"
  ON "ActiveEffect"("characterId", "catalogId", "scope");
