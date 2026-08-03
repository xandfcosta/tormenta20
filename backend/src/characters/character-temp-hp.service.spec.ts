jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  ForbiddenException,
} from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { CharactersService } from './characters.service';
import {
  CharacterTempHpService,
  MANUAL_TEMP_HP_CATALOG_ID,
} from './character-temp-hp.service';
import { PrismaService } from '../prisma/prisma.service';

/**
 * CharacterTempHpService — the temp-PV pool lifecycle (livro p256: PV
 * temporários não acumulam, vale o maior):
 *
 *  F1 `applyPool`  — same source+scope replaces; smaller-than-existing is a
 *                    no-op (superseded); a winning pool displaces smaller ones
 *                    (pure rows deleted, mixed rows zeroed — never touching
 *                    non-tempHp modifiers).
 *  F2 `applyDamage` — atomic temp-first routing: pools drain highest first,
 *                    remainder lowers hpCurrent floored at 0.
 *  F3 `setManualPool` — GM ad-hoc pool through the SAME vale-o-maior path;
 *                    0 clears it.
 */

type EffectRow = {
  id: number;
  catalogId: string;
  scope: string;
  modifiers: string;
};

function poolRow(
  id: number,
  amount: number,
  catalogId = 'class.barbaro.alma-de-bronze',
  scope = 'scene',
): EffectRow {
  return {
    id,
    catalogId,
    scope,
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: 'PV temporários' },
    ]),
  };
}

function mixedRow(id: number, amount: number): EffectRow {
  return {
    id,
    catalogId: 'heroismo',
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: 'Heroísmo' },
      { target: { k: 'attack', scope: 'all' }, amount: 4, bonusType: 'untyped' },
    ]),
  };
}

/** Minimal Character row satisfying CharactersService.findOne (auth + heal). */
function fakeCharacterRow(over: Record<string, unknown> = {}) {
  return {
    id: 1,
    ownerId: 7,
    name: 'X',
    origin: 'Soldado',
    god: null,
    level: 1,
    hpMax: 20,
    hpCurrent: 20,
    mpMax: 4,
    mpCurrent: 4,
    strength: 1,
    dexterity: 1,
    constitution: 1,
    intelligence: 1,
    wisdom: 1,
    charisma: 1,
    size: 'M',
    displacement: 9,
    proficiencies: JSON.stringify(['armas-simples']),
    raceAbilityChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    // classes: [] → findOne's read-heal short-circuits, so specs can assert
    // "no character.update" without mirroring engine vitals in the fixture.
    classes: [],
    races: [],
    expertises: [],
    items: [],
    activeEffects: [],
    ...over,
  };
}

/** Named fake — jest fns per Prisma delegate, one coherent view inside the tx. */
class FakeTempHpPrisma {
  characterFindUnique = jest.fn(async () => this.characterRow);
  characterUpdate = jest.fn(async ({ data }: { data: object }) => ({
    ...(this.characterRow ?? {}),
    ...data,
  }));
  activeEffectFindMany = jest.fn(async () => this.effectRows);
  activeEffectUpsert = jest.fn(async ({ create }: { create: object }) => ({
    id: 100,
    createdAt: new Date(),
    ...create,
  }));
  activeEffectUpdate = jest.fn(async ({ data }: { data: unknown }) => data);
  activeEffectDeleteMany = jest.fn(async () => ({ count: 0 }));
  campaignMemberFindFirst = jest.fn(async (): Promise<{ id: number } | null> => null);
  transaction = jest.fn(async (cb: (tx: unknown) => Promise<unknown>) =>
    cb({
      character: {
        findUnique: this.characterFindUnique,
        update: this.characterUpdate,
      },
      activeEffect: {
        findMany: this.activeEffectFindMany,
        upsert: this.activeEffectUpsert,
        update: this.activeEffectUpdate,
        deleteMany: this.activeEffectDeleteMany,
      },
    }),
  );

  characterRow: ReturnType<typeof fakeCharacterRow> | null = fakeCharacterRow();
  effectRows: EffectRow[] = [];

  get service() {
    return {
      character: {
        findUnique: this.characterFindUnique,
        update: this.characterUpdate,
      },
      activeEffect: {
        findMany: this.activeEffectFindMany,
        upsert: this.activeEffectUpsert,
        update: this.activeEffectUpdate,
        deleteMany: this.activeEffectDeleteMany,
      },
      campaignMember: { findFirst: this.campaignMemberFindFirst },
      $transaction: this.transaction,
    };
  }
}

async function makeTempHpService(
  prisma: FakeTempHpPrisma,
): Promise<CharacterTempHpService> {
  const moduleRef = await Test.createTestingModule({
    providers: [
      CharactersService,
      CharacterTempHpService,
      { provide: PrismaService, useValue: prisma.service },
    ],
  }).compile();
  return moduleRef.get(CharacterTempHpService);
}

const OWN = 'class.barbaro.alma-de-bronze';

describe('CharacterTempHpService.applyPool — vale o maior (F1, p256)', () => {
  it('replaces its own pool (same catalogId+scope) with the FRESH amount', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [poolRow(50, 99)];
    const service = await makeTempHpService(prisma);
    const result = await service.applyPool(1, 'power', OWN, 'scene', 10);
    expect(prisma.activeEffectUpsert).toHaveBeenCalledWith(
      expect.objectContaining({
        where: {
          characterId_catalogId_scope: {
            characterId: 1,
            catalogId: OWN,
            scope: 'scene',
          },
        },
        update: expect.objectContaining({
          modifiers: JSON.stringify([
            { target: { k: 'tempHp' }, amount: 10, bonusType: 'untyped', note: 'PV temporários' },
          ]),
        }),
      }),
    );
    expect(prisma.activeEffectDeleteMany).not.toHaveBeenCalled();
    expect(result).toMatchObject({ displaced: [] });
  });

  it('is a no-op when another pool is bigger — returns superseded', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [poolRow(7, 30, 'campo-de-forca')];
    const service = await makeTempHpService(prisma);
    const result = await service.applyPool(1, 'power', OWN, 'scene', 10);
    expect(result).toEqual({ superseded: true, keptEffectId: 7, keptAmount: 30 });
    expect(prisma.activeEffectUpsert).not.toHaveBeenCalled();
    expect(prisma.activeEffectDeleteMany).not.toHaveBeenCalled();
  });

  it('creates the winning pool and deletes smaller PURE pools only', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [
      poolRow(7, 8, 'campo-de-forca'),
      mixedRow(9, 5),
      { id: 11, catalogId: 'armadura-arcana', scope: 'scene', modifiers: JSON.stringify([{ target: { k: 'defense' }, amount: 5, bonusType: 'armor' }]) },
    ];
    const service = await makeTempHpService(prisma);
    const result = await service.applyPool(1, 'power', OWN, 'scene', 10);
    expect(prisma.activeEffectDeleteMany).toHaveBeenCalledWith({
      where: { id: { in: [7] } },
    });
    // Mixed Heroísmo row survives with its pool zeroed — attack bonus intact.
    expect(prisma.activeEffectUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ where: { id: 9 } }),
    );
    const written = JSON.parse(
      (prisma.activeEffectUpdate.mock.calls[0]![0] as { data: { modifiers: string } }).data.modifiers,
    );
    expect(written[0].amount).toBe(0);
    expect(written[1].amount).toBe(4);
    expect(result).toMatchObject({
      displaced: [
        { effectId: 7, removed: true },
        { effectId: 9, removed: false },
      ],
    });
  });
});

describe('CharacterTempHpService.setManualPool — GM ad-hoc pool (F3)', () => {
  it('creates the manual pool through the vale-o-maior path', async () => {
    const prisma = new FakeTempHpPrisma();
    const service = await makeTempHpService(prisma);
    const result = await service.setManualPool(1, 12);
    expect(prisma.activeEffectUpsert).toHaveBeenCalledWith(
      expect.objectContaining({
        create: expect.objectContaining({
          source: 'manual',
          catalogId: MANUAL_TEMP_HP_CATALOG_ID,
          scope: 'scene',
          modifiers: JSON.stringify([
            { target: { k: 'tempHp' }, amount: 12, bonusType: 'untyped', note: 'PV temporários (manual)' },
          ]),
        }),
      }),
    );
    expect(result).toMatchObject({ displaced: [] });
  });

  it('re-setting replaces the manual pool with the fresh value', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [poolRow(60, 5, MANUAL_TEMP_HP_CATALOG_ID)];
    const service = await makeTempHpService(prisma);
    await service.setManualPool(1, 12);
    expect(prisma.activeEffectUpsert).toHaveBeenCalledWith(
      expect.objectContaining({
        update: expect.objectContaining({
          modifiers: expect.stringContaining('"amount":12'),
        }),
      }),
    );
  });

  it('is superseded by a bigger existing pool (vale o maior)', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [poolRow(50, 30)];
    const service = await makeTempHpService(prisma);
    const result = await service.setManualPool(1, 10);
    expect(result).toEqual({ superseded: true, keptEffectId: 50, keptAmount: 30 });
    expect(prisma.activeEffectUpsert).not.toHaveBeenCalled();
  });

  it('0 deletes the manual pool and reports the removed ids', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.activeEffectFindMany.mockResolvedValue([{ id: 60 }] as never);
    const service = await makeTempHpService(prisma);
    const result = await service.setManualPool(1, 0);
    expect(result).toEqual({ cleared: true, removedEffectIds: [60] });
    expect(prisma.activeEffectDeleteMany).toHaveBeenCalledWith({
      where: { characterId: 1, catalogId: MANUAL_TEMP_HP_CATALOG_ID },
    });
  });

  it('rejects a non-integer/negative value with the offending value', async () => {
    const prisma = new FakeTempHpPrisma();
    const service = await makeTempHpService(prisma);
    await expect(service.setManualPool(1, -3)).rejects.toMatchObject({
      response: expect.objectContaining({
        message: expect.stringContaining('-3'),
      }),
    });
  });
});

describe('CharacterTempHpService.applyDamage — atomic routing (F2)', () => {
  it('rejects a non-positive/non-integer amount with the offending value', async () => {
    const prisma = new FakeTempHpPrisma();
    const service = await makeTempHpService(prisma);
    await expect(service.applyDamage(7, 1, 0)).rejects.toBeInstanceOf(
      BadRequestException,
    );
    await expect(service.applyDamage(7, 1, 2.5)).rejects.toMatchObject({
      response: expect.objectContaining({
        message: expect.stringContaining('2.5'),
      }),
    });
    expect(prisma.transaction).not.toHaveBeenCalled();
  });

  it('forbids a caller who is neither owner nor campaign GM', async () => {
    const prisma = new FakeTempHpPrisma();
    const service = await makeTempHpService(prisma);
    await expect(service.applyDamage(99, 1, 5)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
    expect(prisma.transaction).not.toHaveBeenCalled();
  });

  it('pool covers all: hp untouched, partial drain persisted', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [poolRow(50, 10)];
    const service = await makeTempHpService(prisma);
    const result = await service.applyDamage(7, 1, 7);
    expect(result).toEqual({
      hpCurrent: 20,
      tempHpRemaining: 3,
      drained: [{ effectId: 50, newAmount: 3, removed: false }],
    });
    expect(prisma.activeEffectUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ where: { id: 50 } }),
    );
    // hp unchanged → no character write inside the tx.
    expect(prisma.characterUpdate).not.toHaveBeenCalled();
  });

  it('drains multiple pools highest first, remainder to hp', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.effectRows = [poolRow(1, 4), poolRow(2, 30, 'campo-de-forca')];
    const service = await makeTempHpService(prisma);
    const result = await service.applyDamage(7, 1, 36);
    expect(result.drained).toEqual([
      { effectId: 2, newAmount: 0, removed: true },
      { effectId: 1, newAmount: 0, removed: true },
    ]);
    expect(prisma.activeEffectDeleteMany).toHaveBeenCalledWith({
      where: { id: { in: [2, 1] } },
    });
    expect(result.hpCurrent).toBe(18);
    expect(prisma.characterUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ data: { hpCurrent: 18 } }),
    );
  });

  it('passes straight through to hp when there is no pool', async () => {
    const prisma = new FakeTempHpPrisma();
    const service = await makeTempHpService(prisma);
    const result = await service.applyDamage(7, 1, 5);
    expect(result).toEqual({ hpCurrent: 15, tempHpRemaining: 0, drained: [] });
  });

  it('floors hp at 0 on overkill', async () => {
    const prisma = new FakeTempHpPrisma();
    prisma.characterRow = fakeCharacterRow({ hpCurrent: 3 });
    const service = await makeTempHpService(prisma);
    const result = await service.applyDamage(7, 1, 99);
    expect(result.hpCurrent).toBe(0);
  });
});
