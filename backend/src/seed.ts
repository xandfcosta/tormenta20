/**
 * Test seed — three accounts on the @tormenta.com domain, log in from any
 * browser to exercise the character select + sheet.
 *
 *   GM      mestre@tormenta.com  / 123456  → many complex characters
 *   Player  jogador@tormenta.com / 123456  → a few simple + complex characters
 *   Test    teste@tormenta.com   / 123456  → empty (no characters)
 *
 * "Complex" characters are fully enriched (trained perícias + gear; casters
 * also get a spellbook and some carry a live scene effect) and span a wide
 * level / god / HP range to cover the sheet's edge cases. "Simple" characters
 * are low-level with only basic gear — the bare minimum a fresh PC starts with.
 *
 * All rows go through the real services (Auth / Characters), so passwords are
 * hashed, characters are rules-valid, and every domain invariant is honoured —
 * the same paths the UI uses. Idempotent: re-running finds existing rows.
 *
 * Run from backend/ (needs the compiled dist — the generated Prisma client
 * uses ESM-style .js imports that ts-node can't resolve):
 *   pnpm build && DATABASE_URL="file:./dev.db" node dist/seed.js
 */
import { NestFactory } from '@nestjs/core';
import type { AttributeKey } from '@tormenta20/t20-data';
import { AppModule } from './app.module';
import { AuthService } from './auth/auth.service';
import { CharacterExpertisesService } from './characters/characters-expertises.service';
import { CharacterItemsService } from './characters/characters-items.service';
import { CharactersService } from './characters/characters.service';
import { CharactersSpellsService } from './characters/characters-spells.service';

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
  /** Poder concedido da devoção (NOME — ex: 'Bênção do Mana' soma PM). */
  godPower?: string;
  /**
   * BASE attributes (point-buy, pre-race). The sheet derives the racial mod
   * once from `floatingPicks`/`ascendencia` below (fixed races need neither).
   */
  attrs: Attributes;
  /** Floating-race +1 placements (Humano/Osteon/Lefou — 3 distinct, honouring
   * the race's excluded attribute). */
  floatingPicks?: AttributeKey[];
  /** Subrace ascendência for subrace-gated races (Suraggel: aggelus/sulfure). */
  ascendencia?: string;
  /** Fraction of hpMax the character currently has (0–1). Defaults to full;
   * set below 1 to show a damaged bar on the detail panel. */
  hpFraction?: number;
  /** Gear to add + equip. Defaults to `MARTIAL_GEAR`; override for casters,
   * pack-mules, etc. */
  gear?: GearRef[];
  /** Spellbook for casters — teaches + prepares. Omit for non-casters. */
  spells?: SpellRef[];
  /** Per-class caminho/devoto picks (todo Arcanista tem um Caminho, p36 —
   * define o atributo-chave somado no PM). Re-applied on every run so
   * pre-existing rows heal too. */
  classChoices?: Record<string, { devoto?: string; caminho?: string }>;
  /** Consume a scene catalisador so the sheet carries a live ActiveEffect. */
  sceneEffect?: boolean;
  /** A bare starter PC: skip trained perícias / spells / scene effect and give
   * only basic gear. Used for the player's "simple" characters. */
  simple?: boolean;
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
// pesada" Defense breakdown on the sheet. Weapon wielded in the free hand so
// the HUD weapon row + attack chips have a live source; cosmético feeds
// applySceneEffect (which consumes SCENE_CONSUMABLE — without it a
// sceneEffect:true spec silently no-ops and activeEffects stays empty).
const HEAVY_GEAR: GearRef[] = [
  { catalogId: 'armadura-completa', quantity: 1, equipped: 'vested' },
  { catalogId: 'escudo-pesado', quantity: 1, equipped: 'wielded' },
  { catalogId: 'machado-batalha', quantity: 1, equipped: 'wielded' },
  { catalogId: 'balsamo-restaurador', quantity: 2 },
  { catalogId: 'cosmetico', quantity: 2 },
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

// Bare starter kit for a simple PC — one weapon + light armor, nothing else.
const SIMPLE_GEAR: GearRef[] = [
  { catalogId: 'espada-curta', quantity: 1, equipped: 'wielded' },
  { catalogId: 'armadura-couro', quantity: 1, equipped: 'vested' },
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
// screen" doubles as a QA checklist.

// GM — many complex characters: high level, casters with spellbooks,
// multiclass, heavy gear, live scene effects; levels/gods/HP vary widely.
const GM_CHARACTERS: CharacterSpec[] = [
  { name: 'Tanque Placas Nv10', races: ['Anão'], origin: 'Soldado', classes: [{ className: 'Guerreiro', level: 10 }], god: 'Khalmyr', attrs: attr(4, 1, 4, 1, 2, 1), gear: HEAVY_GEAR, sceneEffect: true },
  { name: 'Curandeira Divina Nv8', races: ['Elfo'], origin: 'Acólito', classes: [{ className: 'Clérigo', level: 8 }], god: 'Lena', attrs: attr(1, 2, 2, 2, 4, 3), gear: CASTER_GEAR, spells: DIVINE_SPELLBOOK, hpFraction: 0.5 },
  { name: 'Necromante Nv12 Magias', races: ['Osteon'], origin: 'Criminoso', classes: [{ className: 'Arcanista', level: 12 }], god: 'Tenebra', attrs: attr(1, 2, 3, 5, 2, 3), floatingPicks: ['intelligence', 'wisdom', 'charisma'], gear: CASTER_GEAR, spells: ARCANE_SPELLBOOK, classChoices: { Arcanista: { caminho: 'mago' } } },
  { name: 'Druida Natureza Nv9', races: ['Dahllan'], origin: 'Eremita', classes: [{ className: 'Druida', level: 9 }], god: 'Allihanna', attrs: attr(1, 3, 2, 2, 4, 2), gear: CASTER_GEAR, spells: NATURE_SPELLBOOK, sceneEffect: true },
  { name: 'Multiclasse Guer+Arc Nv8', races: ['Humano'], origin: 'Herói Camponês', classes: [{ className: 'Guerreiro', level: 4 }, { className: 'Arcanista', level: 4 }], god: 'Valkaria', attrs: attr(3, 2, 3, 3, 1, 1), floatingPicks: ['strength', 'constitution', 'intelligence'], gear: MARTIAL_GEAR, spells: ARCANE_SPELLBOOK.slice(0, 4), classChoices: { Arcanista: { caminho: 'feiticeiro' } } },
  { name: 'Paladino Sagrado Nv11', races: ['Suraggel'], origin: 'Aristocrata', classes: [{ className: 'Paladino', level: 11 }], god: 'Khalmyr', attrs: attr(3, 1, 3, 1, 2, 4), ascendencia: 'aggelus', gear: HEAVY_GEAR, spells: DIVINE_SPELLBOOK },
  { name: 'Lenda Nv20 Maximo', races: ['Anão'], origin: 'Herdeiro', classes: [{ className: 'Cavaleiro', level: 20 }], god: 'Valkaria', attrs: attr(5, 2, 5, 1, 2, 3), gear: HEAVY_GEAR, hpFraction: 0.4 },
  { name: 'Bardo Versátil Nv7', races: ['Lefou'], origin: 'Amnésico', classes: [{ className: 'Bardo', level: 7 }], attrs: attr(1, 3, 2, 2, 1, 4), floatingPicks: ['strength', 'dexterity', 'constitution'], gear: CASTER_GEAR, spells: ARCANE_SPELLBOOK.slice(0, 4), sceneEffect: true },
  { name: 'Inventor Genial Nv6', races: ['Kliren'], origin: 'Assistente de Laboratório', classes: [{ className: 'Inventor', level: 6 }], god: 'Tanna-Toh', attrs: attr(0, 3, 2, 5, 1, 1), gear: HEAVY_PACK },
];

// Player — a few simple starter PCs plus a few enriched complex ones.
const PLAYER_CHARACTERS: CharacterSpec[] = [
  { name: 'Recruta Nv1 Simples', races: ['Humano'], origin: 'Capanga', classes: [{ className: 'Guerreiro', level: 1 }], attrs: attr(3, 2, 2, 1, 1, 1), floatingPicks: ['strength', 'dexterity', 'constitution'], gear: SIMPLE_GEAR, simple: true },
  { name: 'Batedor Nv2 Simples', races: ['Sílfide'], origin: 'Batedor', classes: [{ className: 'Ladino', level: 2 }], attrs: attr(1, 4, 2, 2, 2, 1), gear: SIMPLE_GEAR, simple: true },
  { name: 'Aprendiz Nv1 Simples', races: ['Humano'], origin: 'Charlatão', classes: [{ className: 'Arcanista', level: 1 }], attrs: attr(0, 2, 2, 4, 1, 2), floatingPicks: ['intelligence', 'constitution', 'charisma'], gear: SIMPLE_GEAR, simple: true, classChoices: { Arcanista: { caminho: 'mago' } } },
  { name: 'Guerreiro Veterano Nv8', races: ['Humano'], origin: 'Soldado', classes: [{ className: 'Guerreiro', level: 8 }], god: 'Khalmyr', attrs: attr(4, 3, 3, 2, 2, 1), floatingPicks: ['strength', 'dexterity', 'constitution'], gear: MARTIAL_GEAR, sceneEffect: true },
  { name: 'Arcanista Erudito Nv9', races: ['Qareen'], origin: 'Charlatão', classes: [{ className: 'Arcanista', level: 9 }], god: 'Wynna', godPower: 'Bênção do Mana', attrs: attr(0, 3, 2, 4, 2, 3), gear: CASTER_GEAR, spells: ARCANE_SPELLBOOK, sceneEffect: true, classChoices: { Arcanista: { caminho: 'bruxo' } } },
  { name: 'Paladino Sagrado Nv10', races: ['Suraggel'], origin: 'Aristocrata', classes: [{ className: 'Paladino', level: 10 }], god: 'Khalmyr', attrs: attr(3, 1, 3, 1, 2, 4), ascendencia: 'aggelus', gear: HEAVY_GEAR, spells: DIVINE_SPELLBOOK, hpFraction: 0.6 },
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

// Enrich to the spec. Simple PCs get only basic gear; complex ones get trained
// perícias + gear, a spellbook for casters, and an optional live scene effect.
// Level is set at create time from the spec, so nothing is bumped here.
async function enrichCharacter(
  s: CharacterServices,
  ownerId: number,
  id: number,
  spec: CharacterSpec,
): Promise<void> {
  const before = await s.characters.findOne(ownerId, id);
  // Before `simple` early-return: even bare starters need their caminho.
  if (spec.classChoices) {
    await s.characters.updateAbilityChoices(ownerId, id, {
      classChoices: spec.classChoices,
    });
  }
  if (spec.simple) {
    await addGear(s, ownerId, id, before.items, spec.gear ?? SIMPLE_GEAR);
    return;
  }
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
      godPower: spec.godPower,
      classes: spec.classes,
      hpMax,
      hpCurrent: Math.round(hpMax * (spec.hpFraction ?? 1)),
      mpMax,
      mpCurrent: mpMax,
      size: 'Médio',
      displacement: 9,
      ...spec.attrs,
      raceAttributeChoices: {
        floatingPicks: spec.floatingPicks ?? [],
        ascendencia: spec.ascendencia,
      },
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

async function main() {
  const app = await NestFactory.createApplicationContext(AppModule, {
    logger: false,
  });
  const auth = app.get(AuthService);
  const s: CharacterServices = {
    characters: app.get(CharactersService),
    expertises: app.get(CharacterExpertisesService),
    items: app.get(CharacterItemsService),
    spells: app.get(CharactersSpellsService),
  };

  // ── Users ──────────────────────────────────────────────────────
  const gm = await registerOrLogin(auth, 'mestre@tormenta.com', 'Mestre');
  const player = await registerOrLogin(auth, 'jogador@tormenta.com', 'Jogador');
  // Test account stays empty — just proves login on a blank roster.
  await registerOrLogin(auth, 'teste@tormenta.com', 'Teste');

  // ── Characters ─────────────────────────────────────────────────
  for (const spec of GM_CHARACTERS) await ensureCharacter(s, gm.id, spec);
  for (const spec of PLAYER_CHARACTERS) await ensureCharacter(s, player.id, spec);

  console.log('\n─── test table ready ───');
  console.log(`GM      mestre@tormenta.com  / ${PASSWORD}  (${GM_CHARACTERS.length} complex)`);
  console.log(`Player  jogador@tormenta.com / ${PASSWORD}  (${PLAYER_CHARACTERS.length} chars)`);
  console.log(`Test    teste@tormenta.com   / ${PASSWORD}  (empty)`);

  await app.close();
}

main()
  .then(() => process.exit(0))
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
