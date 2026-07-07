import * as bcrypt from 'bcrypt';
import * as dotenv from 'dotenv';
import { PrismaBetterSqlite3 } from '@prisma/adapter-better-sqlite3';
import { PrismaClient } from '../src/generated/prisma/client';

dotenv.config();

function resolveSqlitePath(url: string | undefined): string {
  if (!url) throw new Error('DATABASE_URL is required');
  if (url === ':memory:') return ':memory:';
  if (url.startsWith('file:')) return url.slice('file:'.length);
  return url;
}

/**
 * Deterministic seed for local dev + Playwright visual regression.
 *
 * Every seed run produces the same ids so E2E specs can hardcode
 * `/characters/1`, `/campaigns/1`, `/campaigns/1/sessions/1`. Uses
 * upserts everywhere so re-running against an existing DB is a no-op
 * on ids (no autoincrement drift).
 *
 * Credentials are fixed in this file and are safe to commit — the
 * seed only ever runs against dev/test databases (never against prod
 * per Prisma seed conventions; production DBs don't run `db seed`).
 */

const prisma = new PrismaClient({
  adapter: new PrismaBetterSqlite3({
    url: resolveSqlitePath(process.env.DATABASE_URL),
  }),
});

const SEED_EMAIL = 'seed@t20.dev';
const SEED_PASSWORD = 'seed-password-1234';
const SEED_NAME = 'Seed Runner';

async function main() {
  const passwordHash = await bcrypt.hash(SEED_PASSWORD, 4);

  const user = await prisma.user.upsert({
    where: { email: SEED_EMAIL },
    update: { name: SEED_NAME, passwordHash },
    create: { email: SEED_EMAIL, name: SEED_NAME, passwordHash },
  });

  const character = await prisma.character.upsert({
    where: { id: 1 },
    update: {},
    create: {
      id: 1,
      ownerId: user.id,
      name: 'Vanya Ravenna',
      origin: 'artesao',
      god: 'Wynna',
      level: 5,
      hpMax: 42,
      hpCurrent: 42,
      mpMax: 15,
      mpCurrent: 15,
      strength: 3,
      dexterity: 2,
      constitution: 2,
      intelligence: 1,
      wisdom: 0,
      charisma: 1,
      size: 'Médio',
      displacement: 9,
    },
  });

  await prisma.characterRace.upsert({
    where: { characterId_race: { characterId: character.id, race: 'humano' } },
    update: {},
    create: { characterId: character.id, race: 'humano' },
  });

  await prisma.characterClass.upsert({
    where: {
      characterId_className: { characterId: character.id, className: 'Guerreiro' },
    },
    update: { level: 5 },
    create: { characterId: character.id, className: 'Guerreiro', level: 5 },
  });

  for (const expertise of [
    { name: 'Luta', attribute: 'strength', trained: true },
    { name: 'Fortitude', attribute: 'constitution', trained: true },
    { name: 'Reflexos', attribute: 'dexterity', trained: false },
    { name: 'Vontade', attribute: 'wisdom', trained: false },
    { name: 'Percepção', attribute: 'wisdom', trained: true },
    { name: 'Atletismo', attribute: 'strength', trained: true },
  ]) {
    await prisma.characterExpertise.upsert({
      where: {
        characterId_name: { characterId: character.id, name: expertise.name },
      },
      update: {
        attribute: expertise.attribute,
        trained: expertise.trained,
      },
      create: {
        characterId: character.id,
        name: expertise.name,
        attribute: expertise.attribute,
        trained: expertise.trained,
      },
    });
  }

  for (const item of [
    { catalogId: 'espada-longa', name: 'Espada Longa', equipped: 'wielded' },
    { catalogId: 'armadura-de-couro', name: 'Armadura de Couro', equipped: 'vested' },
    { catalogId: null, name: 'Corda de Cânhamo (15m)', equipped: null },
  ]) {
    /* Items have no natural composite unique — a simple deleteMany +
     * createMany rebuild keeps the seed idempotent without a
     * unique-constraint fight. */
  }
  await prisma.characterItem.deleteMany({ where: { characterId: character.id } });
  await prisma.characterItem.createMany({
    data: [
      { characterId: character.id, catalogId: 'espada-longa', name: 'Espada Longa', quantity: 1, slots: 2, equipped: 'wielded' },
      { characterId: character.id, catalogId: 'armadura-de-couro', name: 'Armadura de Couro', quantity: 1, slots: 3, equipped: 'vested' },
      { characterId: character.id, catalogId: null, name: 'Corda de Cânhamo (15m)', quantity: 1, slots: 1, equipped: null },
    ],
  });

  const campaign = await prisma.campaign.upsert({
    where: { id: 1 },
    update: { name: 'A Marcha da Tormenta', description: 'Campanha seed' },
    create: {
      id: 1,
      ownerId: user.id,
      name: 'A Marcha da Tormenta',
      description: 'Campanha seed',
    },
  });

  await prisma.campaignMember.upsert({
    where: {
      campaignId_characterId: { campaignId: campaign.id, characterId: character.id },
    },
    update: { role: 'player' },
    create: {
      campaignId: campaign.id,
      characterId: character.id,
      role: 'player',
    },
  });

  await prisma.session.upsert({
    where: { id: 1 },
    update: {
      title: 'Chegada à Vila',
      status: 'active',
      sessionNumber: 1,
    },
    create: {
      id: 1,
      campaignId: campaign.id,
      title: 'Chegada à Vila',
      sessionNumber: 1,
      status: 'active',
      startedAt: new Date(),
    },
  });

  console.log(`Seed complete. Login: ${SEED_EMAIL} / ${SEED_PASSWORD}`);
}

main()
  .catch((err) => {
    console.error(err);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
  });
