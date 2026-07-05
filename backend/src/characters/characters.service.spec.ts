jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  ForbiddenException,
  NotFoundException,
} from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { CharactersService } from './characters.service';
import { PrismaService } from '../prisma/prisma.service';

/**
 * CharactersService — domain guardrails. These specs pin the rules the
 * controller relies on but Prisma cannot enforce:
 *
 *  - `findOne` rejects cross-tenant reads (ForbiddenException, not NotFound,
 *    so the API exposes that the id exists — controller wraps with `@Auth`)
 *  - `updateClassLevel` rejects classes the character does not have, and
 *    keeps the total level ≤ 20 (PDF p7)
 *  - `updateVitals` rejects HP/MP above the current max
 *  - `updateProficiencies` rejects unknown PROFICIENCY_CATEGORIES and dedups
 *  - `assertEquipLimits` (via addItem) enforces the 4-vested / 2-hands caps
 *  - `consumeItem` decrements quantity, deletes on 1→0, blocks repeat
 *    oncePerDay, clamps HP gain to hpMax, and creates an ActiveEffect for
 *    non-instant scopes
 *  - `removeActiveEffect` 404s when the effect belongs to another character
 *  - `endScene` deletes only `scene`-scoped effects (day effects persist)
 *
 * All Prisma I/O is faked — these are domain unit tests, not integration
 * tests. The `backfillProficiencies` short-circuit (proficiencies !== '[]')
 * keeps the read path simple: every fixture starts with a non-empty
 * proficiencies string.
 */

type Character = {
  id: number;
  ownerId: number;
  name: string;
  origin: string;
  god: string | null;
  level: number;
  hpMax: number;
  hpCurrent: number;
  mpMax: number;
  mpCurrent: number;
  strength: number;
  dexterity: number;
  constitution: number;
  intelligence: number;
  wisdom: number;
  charisma: number;
  size: string;
  displacement: number;
  proficiencies: string;
  raceAbilityChoices: string;
  originChoices: string;
  classPowers: string;
  classChoices: string;
  classes: { className: string; level: number }[];
  races: { race: string }[];
  expertises: unknown[];
  items: {
    id: number;
    catalogId: string | null;
    name: string;
    quantity: number;
    slots: number;
    equipped: 'vested' | 'wielded' | 'wielded2' | null;
    improvements: string;
    material: string | null;
  }[];
  activeEffects: {
    id: number;
    catalogId: string;
    scope: string;
    modifiers: string;
    createdAt: Date;
  }[];
};

function makeCharacter(over: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'X',
    origin: 'Soldado',
    god: null,
    level: 1,
    hpMax: 12,
    hpCurrent: 12,
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
    classes: [{ className: 'Guerreiro', level: 1 }],
    races: [{ race: 'Humano' }],
    expertises: [],
    items: [],
    activeEffects: [],
    ...over,
  };
}

class FakePrisma {
  characterFindUnique = jest.fn<Promise<Character | null>, [unknown]>();
  characterUpdate = jest.fn<Promise<Character>, [unknown]>(
    async ({ data }: { data: Partial<Character> }) => ({
      ...this.lastSeed!,
      ...data,
    }),
  );
  characterItemFindMany = jest.fn<
    Promise<{ equipped: string | null }[]>,
    [unknown]
  >(async () => []);
  characterItemFindUnique = jest.fn<
    Promise<{ id: number; characterId: number; equipped?: string | null; catalogId?: string | null } | null>,
    [unknown]
  >();
  characterItemUpdate = jest.fn(async ({ data }: { data: unknown }) => data);
  characterItemDelete = jest.fn(async () => ({ ok: true }));
  characterItemCreate = jest.fn(async ({ data }: { data: unknown }) => data);
  activeEffectFindUnique = jest.fn<
    Promise<{ id: number; characterId: number } | null>,
    [unknown]
  >();
  activeEffectCreate = jest.fn(async ({ data }: { data: unknown }) => data);
  activeEffectDelete = jest.fn(async () => ({ ok: true }));
  activeEffectDeleteMany = jest.fn(async () => ({ count: 0 }));
  characterClassFindMany = jest.fn<
    Promise<{ className: string; level: number }[]>,
    [unknown]
  >(async () => (this.lastSeed?.classes ?? []).map((c) => ({
    className: c.className,
    level: c.level,
  })));
  transaction = jest.fn(async (cb: (tx: unknown) => Promise<unknown>) =>
    cb({
      activeEffect: {
        create: this.activeEffectCreate,
      },
      characterItem: {
        /* BI1: addItem/updateItem now hold equip-limit check + write
         * in one tx. Fake exposes findMany/findUnique/update/delete/
         * create so both sides see a coherent view. */
        findMany: this.characterItemFindMany,
        findUnique: this.characterItemFindUnique,
        update: this.characterItemUpdate,
        delete: this.characterItemDelete,
        create: this.characterItemCreate,
      },
      character: {
        /* BC1: consumeItem re-reads hp/mp/max inside the tx to avoid
         * read-modify-write races. Fake exposes findUnique so the
         * service can pull fresh vitals from the "same row" the test
         * seeded (`lastSeed`). */
        findUnique: async () => this.lastSeed,
        update: this.characterUpdate,
      },
      characterClass: {
        /* BI2: updateClassLevel re-reads classes inside the tx so
         * parallel per-class updates can't slip past the L20 cap. */
        findMany: this.characterClassFindMany,
      },
    }),
  );

  /** The most recent seed returned by `characterFindUnique` — used as the
   *  base object for synthesized `characterUpdate` results so the service
   *  sees a coherent post-update view. */
  lastSeed: Character | null = null;

  seedCharacter(char: Character | null): void {
    this.lastSeed = char;
    this.characterFindUnique.mockResolvedValue(char);
  }

  get service(): {
    character: {
      findUnique: typeof this.characterFindUnique;
      update: typeof this.characterUpdate;
    };
    characterItem: {
      findMany: typeof this.characterItemFindMany;
      findUnique: typeof this.characterItemFindUnique;
      update: typeof this.characterItemUpdate;
      delete: typeof this.characterItemDelete;
      create: typeof this.characterItemCreate;
    };
    activeEffect: {
      findUnique: typeof this.activeEffectFindUnique;
      create: typeof this.activeEffectCreate;
      delete: typeof this.activeEffectDelete;
      deleteMany: typeof this.activeEffectDeleteMany;
    };
    $transaction: typeof this.transaction;
  } {
    return {
      character: {
        findUnique: this.characterFindUnique,
        update: this.characterUpdate,
      },
      characterItem: {
        findMany: this.characterItemFindMany,
        findUnique: this.characterItemFindUnique,
        update: this.characterItemUpdate,
        delete: this.characterItemDelete,
        create: this.characterItemCreate,
      },
      activeEffect: {
        findUnique: this.activeEffectFindUnique,
        create: this.activeEffectCreate,
        delete: this.activeEffectDelete,
        deleteMany: this.activeEffectDeleteMany,
      },
      $transaction: this.transaction,
    };
  }
}

async function makeService(prisma: FakePrisma): Promise<CharactersService> {
  const moduleRef = await Test.createTestingModule({
    providers: [
      CharactersService,
      { provide: PrismaService, useValue: prisma.service },
    ],
  }).compile();
  return moduleRef.get(CharactersService);
}

describe('CharactersService.findOne — ownership guard', () => {
  it('throws NotFoundException when the character does not exist', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(null);
    const service = await makeService(prisma);
    await expect(service.findOne(1, 999)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('throws ForbiddenException when ownerId does not match', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 5, ownerId: 7 }));
    const service = await makeService(prisma);
    await expect(service.findOne(99, 5)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });

  it('returns the character when the owner matches', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 5, ownerId: 7, name: 'Mira' }));
    const service = await makeService(prisma);
    const result = await service.findOne(7, 5);
    expect(result.name).toBe('Mira');
  });
});

describe('CharactersService.updateClassLevel — PDF p7 (sum ≤ 20)', () => {
  it('rejects classes the character does not have', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await expect(
      service.updateClassLevel(1, 1, { className: 'Bardo', level: 3 }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { className: ['Class not on character'] },
      }),
    });
  });

  it('rejects updates that push total level above 20', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        classes: [
          { className: 'Guerreiro', level: 15 },
          { className: 'Bardo', level: 4 },
        ],
      }),
    );
    const service = await makeService(prisma);
    await expect(
      service.updateClassLevel(1, 1, { className: 'Bardo', level: 6 }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        message: 'Total level 21 exceeds 20',
      }),
    });
  });

  it('persists the new total = sum of other class levels + new level', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        classes: [
          { className: 'Guerreiro', level: 10 },
          { className: 'Bardo', level: 4 },
        ],
      }),
    );
    const service = await makeService(prisma);
    await service.updateClassLevel(1, 1, { className: 'Bardo', level: 7 });
    const payload = prisma.characterUpdate.mock.calls[0]![0] as {
      data: { level: number };
    };
    expect(payload.data.level).toBe(17);
  });

  it('re-reads classes inside the tx so parallel per-class updates cap correctly (BI2)', async () => {
    /* Pre-BI2: findOne read `[Guerreiro 10, Bardo 4]`. Two concurrent
     * updateClassLevel — Guerreiro→15, Bardo→5 — both computed
     * newTotal against the same stale snapshot (10+5=15 and 15+4=19).
     * Both wrote; the winning row's committed total ignored the
     * other's uncommitted delta.
     *
     * Post-BI2: the tx re-reads classes fresh. Here we simulate the
     * second call landing after the first committed by mocking
     * characterClassFindMany to return the post-first snapshot. Cap
     * enforcement now correctly rejects. */
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        classes: [
          { className: 'Guerreiro', level: 10 },
          { className: 'Bardo', level: 4 },
        ],
      }),
    );
    /* The second updateClassLevel call sees Guerreiro already bumped
     * to 15 by an earlier commit. Trying to also push Bardo to 7
     * would make total 22 → cap error. */
    prisma.characterClassFindMany.mockResolvedValueOnce([
      { className: 'Guerreiro', level: 15 },
      { className: 'Bardo', level: 4 },
    ]);
    const service = await makeService(prisma);
    await expect(
      service.updateClassLevel(1, 1, { className: 'Bardo', level: 7 }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        message: 'Total level 22 exceeds 20',
      }),
    });
  });
});

describe('CharactersService.updateVitals — clamp to max', () => {
  it('rejects hpCurrent above hpMax', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ hpMax: 10 }));
    const service = await makeService(prisma);
    await expect(
      service.updateVitals(1, 1, { hpCurrent: 11 }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { hpCurrent: ['HP current cannot exceed HP max'] },
      }),
    });
  });

  it('rejects mpCurrent above mpMax', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ mpMax: 4 }));
    const service = await makeService(prisma);
    await expect(
      service.updateVitals(1, 1, { mpCurrent: 5 }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { mpCurrent: ['MP current cannot exceed MP max'] },
      }),
    });
  });

  it('rejects an empty patch (no fields to update)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await expect(service.updateVitals(1, 1, {})).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });

  it('persists hpCurrent and mpCurrent when within bounds', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ hpMax: 12, mpMax: 4 }));
    const service = await makeService(prisma);
    await service.updateVitals(1, 1, { hpCurrent: 6, mpCurrent: 2 });
    const payload = prisma.characterUpdate.mock.calls[0]![0] as {
      data: { hpCurrent: number; mpCurrent: number };
    };
    expect(payload.data).toEqual({ hpCurrent: 6, mpCurrent: 2 });
  });
});

describe('CharactersService.updateProficiencies', () => {
  it('rejects unknown proficiency categories', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await expect(
      service.updateProficiencies(1, 1, {
        proficiencies: ['armas-simples', 'not-a-category'],
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: {
          proficiencies: ['Unknown category "not-a-category"'],
        },
      }),
    });
  });

  it('dedups duplicates before persisting', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await service.updateProficiencies(1, 1, {
      proficiencies: ['armas-simples', 'armas-simples', 'armas-marciais'],
    });
    const payload = prisma.characterUpdate.mock.calls[0]![0] as {
      data: { proficiencies: string };
    };
    expect(JSON.parse(payload.data.proficiencies)).toEqual([
      'armas-simples',
      'armas-marciais',
    ]);
  });
});

describe('CharactersService.addItem — equip caps (4 vested / 2 hands)', () => {
  it('rejects a 5th vested item', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterItemFindMany.mockResolvedValue([
      { equipped: 'vested' },
      { equipped: 'vested' },
      { equipped: 'vested' },
      { equipped: 'vested' },
    ]);
    const service = await makeService(prisma);
    await expect(
      service.addItem(1, 1, {
        name: 'Amuleto',
        quantity: 1,
        slots: 0.5,
        equipped: 'vested',
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { equipped: ['Limite de 4 itens vestidos atingido'] },
      }),
    });
  });

  it('rejects adding a 1H item when both hands are already used', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterItemFindMany.mockResolvedValue([
      { equipped: 'wielded' },
      { equipped: 'wielded' },
    ]);
    const service = await makeService(prisma);
    await expect(
      service.addItem(1, 1, {
        name: 'Adaga',
        quantity: 1,
        slots: 0.5,
        equipped: 'wielded',
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { equipped: ['Limite de 2 mãos atingido'] },
      }),
    });
  });

  it('rejects a 2H item when a 1H is already wielded (sum > 2)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterItemFindMany.mockResolvedValue([
      { equipped: 'wielded' },
    ]);
    const service = await makeService(prisma);
    await expect(
      service.addItem(1, 1, {
        name: 'Espadão',
        quantity: 1,
        slots: 2,
        equipped: 'wielded2',
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { equipped: ['Limite de 2 mãos atingido'] },
      }),
    });
  });

  it('runs the equip-limit check inside the tx (BI1 race guard)', async () => {
    /* Pre-BI1: assertEquipLimits queried `this.prisma` while the write
     * used `this.prisma`, so two concurrent addItem/updateItem calls
     * could both observe an empty equip set and both commit wielded2.
     * Post-BI1: both check + write share the tx client. This spec
     * captures the wiring — the tx must run at least one findMany
     * against the tx client (recorded by our FakePrisma). */
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterItemFindMany.mockResolvedValue([]);
    const service = await makeService(prisma);
    await service.addItem(1, 1, {
      name: 'Adaga',
      quantity: 1,
      slots: 0.5,
      equipped: 'wielded',
    });
    expect(prisma.transaction).toHaveBeenCalled();
    expect(prisma.characterItemFindMany).toHaveBeenCalled();
    expect(prisma.characterItemCreate).toHaveBeenCalled();
  });
});

describe('CharactersService.consumeItem — quantity + oncePerDay + clamp', () => {
  it('decrements quantity when more than 1 remains', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        hpCurrent: 5,
        hpMax: 12,
        items: [
          {
            id: 10,
            catalogId: 'balsamo-restaurador',
            name: 'Bálsamo restaurador',
            quantity: 3,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await service.consumeItem(1, 1, 10, { hpRolled: 4 });
    expect(prisma.characterItemUpdate).toHaveBeenCalledWith({
      where: { id: 10 },
      data: { quantity: 2 },
    });
    expect(prisma.characterItemDelete).not.toHaveBeenCalled();
  });

  it('deletes the row when quantity drops from 1 to 0', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        hpCurrent: 5,
        hpMax: 12,
        items: [
          {
            id: 11,
            catalogId: 'balsamo-restaurador',
            name: 'Bálsamo restaurador',
            quantity: 1,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await service.consumeItem(1, 1, 11, { hpRolled: 4 });
    expect(prisma.characterItemDelete).toHaveBeenCalledWith({
      where: { id: 11 },
    });
    expect(prisma.characterItemUpdate).not.toHaveBeenCalled();
  });

  it('clamps HP gain to hpMax (Bálsamo overshoot)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        hpCurrent: 10,
        hpMax: 12,
        items: [
          {
            id: 12,
            catalogId: 'balsamo-restaurador',
            name: 'Bálsamo restaurador',
            quantity: 1,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await service.consumeItem(1, 1, 12, { hpRolled: 8 });
    expect(prisma.characterUpdate).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { hpCurrent: 12 },
    });
  });

  it('blocks a second oncePerDay use within the day (Gorad quente)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        items: [
          {
            id: 20,
            catalogId: 'gorad-quente',
            name: 'Gorad quente',
            quantity: 2,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
        activeEffects: [
          {
            id: 1,
            catalogId: 'gorad-quente',
            scope: 'day',
            modifiers: '[]',
            createdAt: new Date('2026-06-24'),
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await expect(service.consumeItem(1, 1, 20, {})).rejects.toMatchObject({
      response: expect.objectContaining({
        message: '"Gorad quente" already active for the day',
      }),
    });
  });

  it('reads fresh hp inside the tx so parallel consumes stack (BC1)', async () => {
    /* Scenario: character on 5 HP (max 20). Two Bálsamo (heal 5 HP)
     * fire in parallel. Old code read `hpCurrent=5` once outside the
     * tx and both wrote `Math.min(20, 5+5)=10` — one gain lost. Fix
     * re-reads inside the tx, so the second consume observes
     * `hpCurrent=10` and writes `Math.min(20, 10+5)=15`. */
    const prisma = new FakePrisma();
    let hpInDb = 5;
    /* seedCharacter puts the initial state in lastSeed. But the tx
     * needs to see updates from the first call. Fake it by overriding
     * lastSeed after each characterUpdate. */
    prisma.seedCharacter(
      makeCharacter({
        hpMax: 20,
        hpCurrent: 5,
        items: [
          {
            id: 50,
            catalogId: 'balsamo-restaurador',
            name: 'Bálsamo restaurador',
            quantity: 2,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    prisma.characterUpdate.mockImplementation(async ({ data }) => {
      const patch = data as { hpCurrent?: number; mpCurrent?: number };
      if (patch.hpCurrent !== undefined) hpInDb = patch.hpCurrent;
      const next = { ...prisma.lastSeed!, hpCurrent: hpInDb };
      prisma.lastSeed = next;
      return next;
    });
    const service = await makeService(prisma);

    /* First consume: hp 5 → 10. */
    await service.consumeItem(1, 1, 50, { hpRolled: 5 });
    expect(hpInDb).toBe(10);
    /* Second consume: hp 10 → 15 because the tx re-reads. */
    await service.consumeItem(1, 1, 50, { hpRolled: 5 });
    expect(hpInDb).toBe(15);
  });

  it('translates Prisma P2002 into "already active" when race sneaks past app check (BC2)', async () => {
    /* We use "Cosmético" (scope='scene', has modifiers) because the DB
     * unique constraint fires on ActiveEffect insert — which only
     * happens when the catalog entry has modifiers. Consumables with no
     * modifiers never insert an ActiveEffect and therefore can't race
     * on this index. */
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        items: [
          {
            id: 22,
            catalogId: 'cosmetico',
            name: 'Cosmético',
            quantity: 1,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    /* Simulate the race: no active effect at app-check time, but the
     * DB's composite unique index catches the duplicate on insert. */
    prisma.activeEffectCreate.mockRejectedValueOnce(
      Object.assign(new Error('unique violation'), { code: 'P2002' }),
    );
    const service = await makeService(prisma);
    await expect(service.consumeItem(1, 1, 22, {})).rejects.toMatchObject({
      response: expect.objectContaining({
        message: '"Cosmético" already active for the day',
      }),
    });
  });

  it('creates an ActiveEffect for non-instant scopes (Cosmético → scene)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        items: [
          {
            id: 30,
            catalogId: 'cosmetico',
            name: 'Cosmético',
            quantity: 1,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await service.consumeItem(1, 1, 30, {});
    expect(prisma.activeEffectCreate).toHaveBeenCalledWith({
      data: expect.objectContaining({
        characterId: 1,
        catalogId: 'cosmetico',
        scope: 'scene',
      }),
    });
  });

  it('rejects consume on a custom (no-catalog) item', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        items: [
          {
            id: 40,
            catalogId: null,
            name: 'Homebrew',
            quantity: 1,
            slots: 0.5,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await expect(service.consumeItem(1, 1, 40, {})).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });

  it('rejects consume on a non-consumable catalog item', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(
      makeCharacter({
        items: [
          {
            id: 41,
            catalogId: 'espada-longa',
            name: 'Espada longa',
            quantity: 1,
            slots: 1,
            equipped: null,
            improvements: '[]',
            material: null,
          },
        ],
      }),
    );
    const service = await makeService(prisma);
    await expect(service.consumeItem(1, 1, 41, {})).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });
});

describe('CharactersService.removeActiveEffect — cross-character guard', () => {
  it('throws NotFound when the effect belongs to another character', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    prisma.activeEffectFindUnique.mockResolvedValue({
      id: 99,
      characterId: 2,
    });
    const service = await makeService(prisma);
    await expect(service.removeActiveEffect(1, 1, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('deletes the effect when owned by the target character', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    prisma.activeEffectFindUnique.mockResolvedValue({
      id: 50,
      characterId: 1,
    });
    const service = await makeService(prisma);
    const result = await service.removeActiveEffect(1, 1, 50);
    expect(result).toEqual({ id: 50 });
    expect(prisma.activeEffectDelete).toHaveBeenCalledWith({
      where: { id: 50 },
    });
  });
});

describe('CharactersService.endScene / endDay — scope filtering', () => {
  it('endScene deletes only scene-scoped effects', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    const service = await makeService(prisma);
    await service.endScene(1, 1);
    expect(prisma.activeEffectDeleteMany).toHaveBeenCalledWith({
      where: { characterId: 1, scope: 'scene' },
    });
  });

  it('endDay deletes scene and day effects', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    const service = await makeService(prisma);
    await service.endDay(1, 1);
    expect(prisma.activeEffectDeleteMany).toHaveBeenCalledWith({
      where: { characterId: 1, scope: { in: ['scene', 'day'] } },
    });
  });
});
