-- CreateTable
CREATE TABLE "CampaignMember" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "campaignId" INTEGER NOT NULL,
    "characterId" INTEGER NOT NULL,
    "role" TEXT NOT NULL DEFAULT 'player',
    "addedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "CampaignMember_campaignId_fkey" FOREIGN KEY ("campaignId") REFERENCES "Campaign" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "CampaignMember_characterId_fkey" FOREIGN KEY ("characterId") REFERENCES "Character" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);

-- CreateIndex
CREATE INDEX "CampaignMember_campaignId_idx" ON "CampaignMember"("campaignId");

-- CreateIndex
CREATE INDEX "CampaignMember_characterId_idx" ON "CampaignMember"("characterId");

-- CreateIndex
CREATE UNIQUE INDEX "CampaignMember_campaignId_characterId_key" ON "CampaignMember"("campaignId", "characterId");
