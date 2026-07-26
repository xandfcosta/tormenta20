/**
 * Unified test seed — one full table you can log into from two browsers to
 * exercise the whole app (character select, sheet tabs, campaigns, sessions
 * in every state, and live "match mode").
 *
 *   GM     mestre@hotmail.com  / 123456
 *   Player jogador@hotmail.com / 123456
 *
 * What it builds (all idempotent — re-running finds existing rows):
 *   - 2 users (GM + player), each owning 2 fully-enriched characters
 *     (level 8, trained perícias, equipped gear incl. an expertise-boosting
 *     item, potions, an arcane spellbook, and a live scene effect).
 *   - GM owns 2 campaigns:
 *       "Mesa de Teste"      → ACTIVE session #1 (player's Bruenor joined) —
 *                              open in two browsers for match mode.
 *       "A Tormenta Rubra"   → PLANNED session #1 + ENDED session #2
 *                              (player's Lyra joined) — exercises the other
 *                              session lifecycle states.
 *
 * Everything runs through the real services (Auth / Characters / Campaigns /
 * CampaignMembers / Sessions), so passwords are hashed, characters are
 * rules-valid, and every domain invariant (invite token, one-PC-per-campaign,
 * GM-only session start) is honoured — the same paths the UI uses.
 *
 * Run from backend/ (needs the compiled dist — the generated Prisma client
 * uses ESM-style .js imports that ts-node can't resolve):
 *   pnpm build && DATABASE_URL="file:./dev.db" node dist/seed.js
 */
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { AuthService } from './auth/auth.service';
import { CampaignMembersService } from './campaign-members/campaign-members.service';
import { CampaignsService } from './campaigns/campaigns.service';
import { CharacterExpertisesService } from './characters/characters-expertises.service';
import { CharacterItemsService } from './characters/characters-items.service';
import { CharactersService } from './characters/characters.service';
import { CharactersSpellsService } from './characters/characters-spells.service';
import { SessionsService, type SessionStatus } from './sessions/sessions.service';

type SeededUser = { id: number; email: string };

/** Sub-services the enrichment drives, bundled so the helpers take one arg. */
type CharacterServices = {
  characters: CharactersService;
  expertises: CharacterExpertisesService;
  items: CharacterItemsService;
  spells: CharactersSpellsService;
};

type Attributes = {
  strength: number;
  dexterity: number;
  constitution: number;
  intelligence: number;
  wisdom: number;
  charisma: number;
};

type ClassLevel = { className: string; level: number };
type SpellRef = { id: string; prepared: boolean };
type EquippedSlot = 'vested' | 'wielded' | 'wielded2';
type GearRef = { catalogId: string; quantity: number; equipped?: EquippedSlot };

type CharacterSpec = {
  name: string;
  races: string[];
  origin: string;
  /** One entry per class — two entries exercise the multiclass level split. */
  classes: ClassLevel[];
  god?: string;
  attrs: Attributes;
  /** Fraction of hpMax the character currently has (0–1). Defaults to full;
   * set below 1 to show a damaged bar on the detail panel. */
  hpFraction?: number;
  /** Gear to add + equip. Defaults to `MARTIAL_GEAR`; override for casters,
   * pack-mules, etc. */
  gear?: GearRef[];
  /** Spellbook for casters — teaches + prepares. Omit for non-casters. */
  spells?: SpellRef[];
  /** Consume a scene catalisador so the sheet carries a live ActiveEffect. */
  sceneEffect?: boolean;
};

/** Terse attribute constructor to keep the roster specs on one line each. */
function attr(
  strength: number,
  dexterity: number,
  constitution: number,
  intelligence: number,
  wisdom: number,
  charisma: number,
): Attributes {
  return { strength, dexterity, constitution, intelligence, wisdom, charisma };
}

/** Total character level = sum of class levels (drives PV/PM pools). */
function totalLevel(spec: CharacterSpec): number {
  return spec.classes.reduce((sum, c) => sum + c.level, 0);
}

/** Derive PV/PM pools from level so higher-level characters read as tougher. */
function vitals(level: number): { hpMax: number; mpMax: number } {
  return { hpMax: 12 + level * 7, mpMax: 3 + level * 4 };
}

const PASSWORD = '123456';

// ── Enrichment data ──────────────────────────────────────────────────
// Perícias to train, each pinned to a fitting attribute — populates the skill
// list with trained rows + non-trivial totals (treino + ½ nível + atributo).
const TRAINED_EXPERTISES: { name: string; attribute: string }[] = [
  { name: 'Luta', attribute: 'strength' },
  { name: 'Atletismo', attribute: 'strength' },
  { name: 'Pontaria', attribute: 'dexterity' },
  { name: 'Reflexos', attribute: 'dexterity' },
  { name: 'Fortitude', attribute: 'constitution' },
  { name: 'Vontade', attribute: 'wisdom' },
  { name: 'Percepção', attribute: 'wisdom' },
  { name: 'Intimidação', attribute: 'charisma' },
  { name: 'Investigação', attribute: 'intelligence' },
  { name: 'Misticismo', attribute: 'intelligence' },
];

// Bandana grants Intimidação +1 → gives the perícia "outros" breakdown real
// data; cosmético is a scene consumable → becomes a live ActiveEffect.
const MARTIAL_GEAR: GearRef[] = [
  { catalogId: 'espada-curta', quantity: 1, equipped: 'wielded' },
  { catalogId: 'armadura-couro', quantity: 1, equipped: 'vested' },
  { catalogId: 'bandana', quantity: 1, equipped: 'vested' },
  { catalogId: 'balsamo-restaurador', quantity: 3 },
  { catalogId: 'cosmetico', quantity: 2 },
];

// Heavy armor + shield → exercises the "Destreza bloqueada por armadura
// pesada" Defense breakdown on the sheet.
const HEAVY_GEAR: GearRef[] = [
  { catalogId: 'armadura-completa', quantity: 1, equipped: 'vested' },
  { catalogId: 'escudo-pesado', quantity: 1, equipped: 'wielded' },
  { catalogId: 'machado-batalha', quantity: 1 },
  { catalogId: 'balsamo-restaurador', quantity: 2 },
];

// A stuffed backpack → gives the Inventário tab a long list to scroll/filter.
const HEAVY_PACK: GearRef[] = [
  { catalogId: 'clava', quantity: 1, equipped: 'wielded' },
  { catalogId: 'corda', quantity: 2 },
  { catalogId: 'tocha', quantity: 6 },
  { catalogId: 'racao-de-viagem', quantity: 10 },
  { catalogId: 'balsamo-restaurador', quantity: 5 },
  { catalogId: 'cosmetico', quantity: 3 },
  { catalogId: 'saco-de-dormir', quantity: 1 },
  { catalogId: 'mochila', quantity: 1, equipped: 'vested' },
];

// Light caster kit — no armor so Destreza applies cleanly to Defense.
const CASTER_GEAR: GearRef[] = [
  { catalogId: 'adaga', quantity: 1, equipped: 'wielded' },
  { catalogId: 'bandana', quantity: 1, equipped: 'vested' },
  { catalogId: 'balsamo-restaurador', quantity: 2 },
  { catalogId: 'cosmetico', quantity: 2 },
];

const ARCANE_SPELLBOOK: SpellRef[] = [
  { id: 'luz', prepared: true },
  { id: 'armadura-arcana', prepared: true },
  { id: 'bola-de-fogo', prepared: true },
  { id: 'relampago', prepared: true },
  { id: 'sono', prepared: false },
  { id: 'invisibilidade', prepared: false },
  { id: 'teletransporte', prepared: false },
  { id: 'imagem-espelhada', prepared: false },
];

const DIVINE_SPELLBOOK: SpellRef[] = [
  { id: 'curar-ferimentos', prepared: true },
  { id: 'escudo-da-fe', prepared: true },
  { id: 'abencoar-alimentos', prepared: true },
  { id: 'voz-divina', prepared: false },
  { id: 'tempestade-divina', prepared: false },
];

const NATURE_SPELLBOOK: SpellRef[] = [
  { id: 'caminhos-da-natureza', prepared: true },
  { id: 'controlar-plantas', prepared: true },
  { id: 'metamorfose', prepared: false },
];

const SCENE_CONSUMABLE = 'cosmetico';

// ── Character rosters ────────────────────────────────────────────────
// Names describe the test scenario each character exercises, so the "select
// screen" doubles as a QA checklist. Every character is enriched (trained
// perícias + gear; casters also get a spellbook), and levels/gods/HP vary
// widely to cover the sheet's edge cases.
const GM_CHARACTERS: CharacterSpec[] = [
  { name: 'Tanque Placas Nv10', races: ['Anão'], origin: 'Soldado', classes: [{ className: 'Guerreiro', level: 10 }], god: 'Khalmyr', attrs: attr(4, 1, 4, 1, 2, 1), gear: HEAVY_GEAR, sceneEffect: true },
  { name: 'Curandeira Divina', races: ['Elfo'], origin: 'Acólito', classes: [{ className: 'Clérigo', level: 8 }], god: 'Lena', attrs: attr(1, 2, 2, 2, 4, 3), gear: CASTER_GEAR, spells: DIVINE_SPELLBOOK, hpFraction: 0.5 },
  { name: 'Necromante Nv12 Magias', races: ['Osteon'], origin: 'Criminoso', classes: [{ className: 'Arcanista', level: 12 }], god: 'Tenebra', attrs: attr(1, 2, 3, 5, 2, 3), gear: CASTER_GEAR, spells: ARCANE_SPELLBOOK },
  { name: 'Barbaro Ferido 15pct', races: ['Minotauro'], origin: 'Gladiador', classes: [{ className: 'Bárbaro', level: 5 }], attrs: attr(4, 2, 4, 0, 1, 0), gear: MARTIAL_GEAR, hpFraction: 0.15 },
  { name: 'Druida Natureza', races: ['Dahllan'], origin: 'Eremita', classes: [{ className: 'Druida', level: 9 }], god: 'Allihanna', attrs: attr(1, 3, 2, 2, 4, 2), gear: CASTER_GEAR, spells: NATURE_SPELLBOOK, sceneEffect: true },
  { name: 'Bardo Sem Devocao', races: ['Lefou'], origin: 'Amnésico', classes: [{ className: 'Bardo', level: 3 }], attrs: attr(1, 3, 2, 2, 1, 4), gear: CASTER_GEAR, spells: ARCANE_SPELLBOOK.slice(0, 3) },
  { name: 'Multiclasse Guer+Arc', races: ['Humano'], origin: 'Herói Camponês', classes: [{ className: 'Guerreiro', level: 4 }, { className: 'Arcanista', level: 4 }], god: 'Valkaria', attrs: attr(3, 2, 3, 3, 1, 1), gear: MARTIAL_GEAR, spells: ARCANE_SPELLBOOK.slice(0, 4) },
];

const PLAYER_CHARACTERS: CharacterSpec[] = [
  { name: 'Guerreiro Campanha A', races: ['Humano'], origin: 'Soldado', classes: [{ className: 'Guerreiro', level: 8 }], god: 'Khalmyr', attrs: attr(4, 3, 3, 2, 2, 1), gear: MARTIAL_GEAR, sceneEffect: true },
  { name: 'Arcanista Campanha B', races: ['Humano'], origin: 'Charlatão', classes: [{ className: 'Arcanista', level: 8 }], god: 'Wynna', attrs: attr(0, 3, 2, 4, 2, 3), gear: CASTER_GEAR, spells: ARCANE_SPELLBOOK, sceneEffect: true },
  { name: 'Ladino Furtivo Nv7', races: ['Sílfide'], origin: 'Batedor', classes: [{ className: 'Ladino', level: 7 }], attrs: attr(1, 5, 2, 2, 2, 1), gear: MARTIAL_GEAR },
  { name: 'Paladino Sagrado Nv11', races: ['Suraggel'], origin: 'Aristocrata', classes: [{ className: 'Paladino', level: 11 }], god: 'Khalmyr', attrs: attr(3, 1, 3, 1, 2, 4), gear: HEAVY_GEAR, spells: DIVINE_SPELLBOOK },
  { name: 'Recruta Nv1 Minimo', races: ['Humano'], origin: 'Capanga', classes: [{ className: 'Guerreiro', level: 1 }], attrs: attr(3, 2, 2, 1, 1, 1), gear: MARTIAL_GEAR },
  { name: 'Lenda Nv20 Maximo', races: ['Anão'], origin: 'Herdeiro', classes: [{ className: 'Cavaleiro', level: 20 }], god: 'Valkaria', attrs: attr(5, 2, 5, 1, 2, 3), gear: HEAVY_GEAR, hpFraction: 0.4 },
  { name: 'Bucaneiro Ferido Meio', races: ['Qareen'], origin: 'Artista', classes: [{ className: 'Bucaneiro', level: 5 }], attrs: attr(2, 4, 2, 1, 1, 3), gear: MARTIAL_GEAR, hpFraction: 0.5 },
  { name: 'Mochila Cheia Itens', races: ['Golem'], origin: 'Capanga', classes: [{ className: 'Lutador', level: 4 }], attrs: attr(3, 2, 4, 0, 1, 0), gear: HEAVY_PACK, hpFraction: 0.8 },
  { name: 'Inventor Genial Nv6', races: ['Kiiren'], origin: 'Assistente de Laboratório', classes: [{ className: 'Inventor', level: 6 }], god: 'Tanna-Toh', attrs: attr(0, 3, 2, 5, 1, 1), gear: HEAVY_PACK },
];

async function registerOrLogin(
  auth: AuthService,
  email: string,
  name: string,
): Promise<SeededUser> {
  try {
    const user = await auth.register({ email, password: PASSWORD, name });
    console.log(`✓ created user #${user.id} <${email}>`);
    return user;
  } catch {
    const user = await auth.validate(email, PASSWORD);
    console.log(`• user already exists: #${user.id} <${email}>`);
    return user;
  }
}

async function trainExpertises(
  s: CharacterServices,
  ownerId: number,
  id: number,
): Promise<void> {
  for (const e of TRAINED_EXPERTISES) {
    await s.expertises.updateExpertise(ownerId, id, {
      name: e.name,
      attribute: e.attribute,
      trained: true,
    });
  }
}

async function addGear(
  s: CharacterServices,
  ownerId: number,
  id: number,
  existing: readonly { catalogId: string | null }[],
  gear: GearRef[],
): Promise<void> {
  const have = new Set(existing.map((i) => i.catalogId));
  for (const g of gear) {
    if (have.has(g.catalogId)) continue;
    await s.items.addItem(ownerId, id, {
      catalogId: g.catalogId,
      quantity: g.quantity,
      equipped: g.equipped,
    });
  }
}

async function teachSpells(
  s: CharacterServices,
  ownerId: number,
  id: number,
  spellbook: SpellRef[],
): Promise<void> {
  for (const sp of spellbook) {
    let known = true;
    try {
      await s.spells.learnSpell(ownerId, id, sp.id);
    } catch {
      // Either already known (unique clash on re-run) or the caster's circle
      // is too low for this spell. Only prepare if it's actually in the book.
      const char = await s.characters.findOne(ownerId, id);
      known = char.spells.some((x) => x.catalogSpellId === sp.id);
    }
    if (sp.prepared && known) {
      await s.spells.setSpellPrepared(ownerId, id, sp.id, true);
    }
  }
}

// Consume one scene catalisador so the sheet carries a live ActiveEffect —
// only when there's none yet and the potion is on hand (idempotent).
async function applySceneEffect(
  s: CharacterServices,
  ownerId: number,
  id: number,
): Promise<void> {
  const char = await s.characters.findOne(ownerId, id);
  if (char.activeEffects.length > 0) return;
  const potion = char.items.find((i) => i.catalogId === SCENE_CONSUMABLE);
  if (!potion) return;
  await s.items.consumeItem(ownerId, id, potion.id, {});
}

// Enrich to the spec: trained perícias + gear for everyone, a spellbook for
// casters, and an optional live scene effect. Level is set at create time
// from the spec, so nothing is bumped here — varied levels survive.
async function enrichCharacter(
  s: CharacterServices,
  ownerId: number,
  id: number,
  spec: CharacterSpec,
): Promise<void> {
  const before = await s.characters.findOne(ownerId, id);
  await trainExpertises(s, ownerId, id);
  await addGear(s, ownerId, id, before.items, spec.gear ?? MARTIAL_GEAR);
  if (spec.spells) await teachSpells(s, ownerId, id, spec.spells);
  if (spec.sceneEffect) await applySceneEffect(s, ownerId, id);
}

/** Create (if missing) + enrich a character for `ownerId`; returns its id. */
async function ensureCharacter(
  s: CharacterServices,
  ownerId: number,
  spec: CharacterSpec,
): Promise<{ id: number; name: string }> {
  const existing = (await s.characters.list(ownerId)).find(
    (c) => c.name === spec.name,
  );
  const level = totalLevel(spec);
  const { hpMax, mpMax } = vitals(level);
  const char =
    existing ??
    (await s.characters.create(ownerId, {
      name: spec.name,
      races: spec.races,
      origin: spec.origin,
      god: spec.god,
      classes: spec.classes,
      hpMax,
      hpCurrent: Math.round(hpMax * (spec.hpFraction ?? 1)),
      mpMax,
      mpCurrent: mpMax,
      size: 'Médio',
      displacement: 9,
      ...spec.attrs,
    }));
  const label = spec.classes
    .map((c) => `${c.className} ${c.level}`)
    .join(' / ');
  console.log(
    existing
      ? `• character #${char.id} "${char.name}" already exists`
      : `✓ created character #${char.id} "${char.name}" (${label})`,
  );
  await enrichCharacter(s, ownerId, char.id, spec);
  return { id: char.id, name: char.name };
}

async function ensureCampaign(
  campaigns: CampaignsService,
  gmId: number,
  name: string,
  description: string,
): Promise<{ id: number; name: string }> {
  const found = (await campaigns.list(gmId)).find((c) => c.name === name);
  if (found) {
    console.log(`• campaign #${found.id} "${found.name}" already exists`);
    return { id: found.id, name: found.name };
  }
  const created = await campaigns.create(gmId, { name, description });
  console.log(`✓ created campaign #${created.id} "${created.name}"`);
  return created;
}

/** Player joins `campaignId` with `characterId` via a fresh invite token. */
async function ensureMember(
  campaigns: CampaignsService,
  members: CampaignMembersService,
  gmId: number,
  campaignId: number,
  playerId: number,
  characterId: number,
): Promise<void> {
  const roster = await members.list(gmId, campaignId);
  if (roster.some((m) => m.characterId === characterId)) {
    console.log(`• character #${characterId} already on campaign #${campaignId}`);
    return;
  }
  const { token } = await campaigns.rotateInviteToken(gmId, campaignId);
  await members.add(playerId, campaignId, { characterId, inviteToken: token });
  console.log(`✓ character #${characterId} joined campaign #${campaignId}`);
}

/** Ensure a session exists in campaign and drive it to `target` state. */
async function ensureSession(
  sessions: SessionsService,
  gmId: number,
  campaignId: number,
  sessionNumber: number,
  title: string,
  target: SessionStatus,
): Promise<{ id: number; status: SessionStatus }> {
  const list = await sessions.listForCaller(gmId, campaignId);
  let session =
    list.find((s) => s.sessionNumber === sessionNumber) ??
    (await sessions.create(gmId, campaignId, { sessionNumber, title }));
  if (target === 'active' && session.status !== 'active') {
    session = await sessions.start(gmId, campaignId, session.id);
  } else if (target === 'ended' && session.status !== 'ended') {
    if (session.status === 'planned') {
      await sessions.start(gmId, campaignId, session.id);
    }
    session = await sessions.end(gmId, campaignId, session.id);
  }
  console.log(`✓ session #${session.id} (nº ${sessionNumber}) → ${session.status}`);
  return { id: session.id, status: session.status as SessionStatus };
}

async function main() {
  const app = await NestFactory.createApplicationContext(AppModule, {
    logger: false,
  });
  const auth = app.get(AuthService);
  const campaigns = app.get(CampaignsService);
  const members = app.get(CampaignMembersService);
  const sessions = app.get(SessionsService);
  const s: CharacterServices = {
    characters: app.get(CharactersService),
    expertises: app.get(CharacterExpertisesService),
    items: app.get(CharacterItemsService),
    spells: app.get(CharactersSpellsService),
  };

  // ── Users ──────────────────────────────────────────────────────
  const gm = await registerOrLogin(auth, 'mestre@hotmail.com', 'Mestre');
  const player = await registerOrLogin(auth, 'jogador@hotmail.com', 'Jogador');

  // ── Characters (full roster each, all enriched) ────────────────
  for (const spec of GM_CHARACTERS) await ensureCharacter(s, gm.id, spec);
  const playerChars: { id: number; name: string }[] = [];
  for (const spec of PLAYER_CHARACTERS) {
    playerChars.push(await ensureCharacter(s, player.id, spec));
  }
  // First two player PCs are the campaign members (see roster names).
  const [campaignPcA, campaignPcB] = playerChars;

  // ── Campaign A: "Mesa de Teste" — active session for match mode ─
  const mesa = await ensureCampaign(
    campaigns,
    gm.id,
    'Mesa de Teste',
    'Campanha de teste para o multiplayer e o modo sessão ao vivo.',
  );
  await ensureMember(campaigns, members, gm.id, mesa.id, player.id, campaignPcA.id);
  const active = await ensureSession(
    sessions,
    gm.id,
    mesa.id,
    1,
    'Primeira sessão',
    'active',
  );

  // ── Campaign B: "A Tormenta Rubra" — planned + ended sessions ───
  const rubra = await ensureCampaign(
    campaigns,
    gm.id,
    'A Tormenta Rubra',
    'Segunda campanha para exercitar os estados de sessão.',
  );
  await ensureMember(campaigns, members, gm.id, rubra.id, player.id, campaignPcB.id);
  await ensureSession(sessions, gm.id, rubra.id, 1, 'Preparação', 'planned');
  await ensureSession(sessions, gm.id, rubra.id, 2, 'O Ataque', 'ended');

  console.log('\n─── test table ready ───');
  console.log(`GM      mestre@hotmail.com  / ${PASSWORD}`);
  console.log(`Player  jogador@hotmail.com / ${PASSWORD}`);
  console.log(`Match   /campaigns/${mesa.id}/sessions/${active.id} (active)`);

  await app.close();
}

main()
  .then(() => process.exit(0))
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
