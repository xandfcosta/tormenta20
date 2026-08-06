/**
 * Bench payload generator — dumps the real seeded characters as
 * `CharacterInput` payloads + their `ComputedSheet` oracle, so the Go-vs-Node
 * engine HTTP benchmark hits identical, representative inputs and can assert
 * byte-for-byte output parity.
 *
 * Payloads land in `<repo>/bench/payloads/<slug>.json` (POST body for both
 * servers) and the oracle in `<repo>/bench/expected/<slug>.json`. Checked in,
 * so the benchmark is DB-free after this runs once.
 *
 * Run from backend/ (needs the compiled dist, like seed.ts):
 *   pnpm build && DATABASE_URL="file:./dev.db" node dist/dump-bench-payloads.js
 */
import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { NestFactory } from '@nestjs/core';
import { computeCharacterSheet } from '@tormenta20/t20-data';
import { AppModule } from './app.module';
import {
  type CharacterDbRow,
  toCharacterInput,
} from './characters/character-sheet.mapper';
import { PrismaService } from './prisma/prisma.service';

/** Same relations the read path (`characterInclude`) hydrates — insertion
 *  order is semantic (races[0] primary, classes[0] primary). */
const dumpInclude = {
  races: { select: { race: true }, orderBy: { id: 'asc' } },
  classes: {
    select: { className: true, level: true },
    orderBy: { id: 'asc' },
  },
  expertises: {
    select: { name: true, attribute: true, trained: true, custom: true },
    orderBy: { name: 'asc' },
  },
  items: {
    select: {
      id: true,
      catalogId: true,
      name: true,
      quantity: true,
      slots: true,
      equipped: true,
      improvements: true,
      material: true,
    },
    orderBy: { id: 'asc' },
  },
} as const;

/** kebab-case slug from a character name (accent-stripped) for stable paths. */
function slugify(name: string): string {
  return name
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
}

async function main() {
  const app = await NestFactory.createApplicationContext(AppModule, {
    logger: false,
  });
  const prisma = app.get(PrismaService);

  const rows = (await prisma.character.findMany({
    include: dumpInclude,
    orderBy: { id: 'asc' },
  })) as unknown as (CharacterDbRow & { name: string })[];

  const outDir = join(__dirname, '..', '..', 'bench');
  const payloadDir = join(outDir, 'payloads');
  const expectedDir = join(outDir, 'expected');
  mkdirSync(payloadDir, { recursive: true });
  mkdirSync(expectedDir, { recursive: true });

  const manifest: { slug: string; name: string; level: number }[] = [];
  for (const row of rows) {
    const input = toCharacterInput(row);
    const sheet = computeCharacterSheet(input);
    const slug = slugify(row.name);
    writeFileSync(
      join(payloadDir, `${slug}.json`),
      `${JSON.stringify(input, null, 2)}\n`,
    );
    writeFileSync(
      join(expectedDir, `${slug}.json`),
      `${JSON.stringify(sheet, null, 2)}\n`,
    );
    manifest.push({ slug, name: row.name, level: input.level });
  }
  writeFileSync(
    join(outDir, 'manifest.json'),
    `${JSON.stringify(manifest, null, 2)}\n`,
  );

  // eslint-disable-next-line no-console -- CLI progress, plain text by design
  console.log(`dumped ${manifest.length} payloads → ${outDir}`);
  await app.close();
}

main().catch((err) => {
  // eslint-disable-next-line no-console -- CLI failure surface
  console.error(err);
  process.exit(1);
});
