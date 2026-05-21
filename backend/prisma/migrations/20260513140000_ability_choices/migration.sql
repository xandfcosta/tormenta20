-- AlterTable
ALTER TABLE "Character" ADD COLUMN "raceAbilityChoices" TEXT NOT NULL DEFAULT '[]';
ALTER TABLE "Character" ADD COLUMN "originChoices" TEXT NOT NULL DEFAULT '[]';
ALTER TABLE "Character" ADD COLUMN "classPowers" TEXT NOT NULL DEFAULT '[]';
