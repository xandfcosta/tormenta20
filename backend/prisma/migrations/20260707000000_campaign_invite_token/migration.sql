-- Rotating invite token for campaigns. Nullable — the GM opts in by
-- generating one from the campaign detail page; rotating replaces the
-- token so any previously-shared link goes 404.
--
-- Unique index guards against a collision between the (small) 24-char
-- random token space.

ALTER TABLE "Campaign" ADD COLUMN "inviteToken" TEXT;

CREATE UNIQUE INDEX "Campaign_inviteToken_key" ON "Campaign"("inviteToken");
