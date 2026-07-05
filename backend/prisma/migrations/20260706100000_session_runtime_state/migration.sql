-- Whole-flow audit → Persistence Plan P1a: Session runtime tracker
-- (initiative + round + turnIndex) now persists as a JSON blob on the
-- Session row so a server restart or a browser refresh mid-combat
-- doesn't wipe combat state.
--
-- Default is the "empty tracker" so existing rows load without needing
-- special handling — SessionStateService just parses the blob on first
-- access.

ALTER TABLE "Session" ADD COLUMN "runtimeState" TEXT NOT NULL
  DEFAULT '{"initiative":[],"round":0,"turnIndex":-1}';
