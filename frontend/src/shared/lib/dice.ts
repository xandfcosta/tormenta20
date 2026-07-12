/**
 * Rolls a raw d20 (1–20). Used to seed initiative for combatants the client
 * adds without a full sheet (encounter monsters, in-session bestiary adds) —
 * no attribute mod, since only monsters/NPCs go through this path.
 */
export function rollD20(): number {
  return Math.floor(Math.random() * 20) + 1
}
