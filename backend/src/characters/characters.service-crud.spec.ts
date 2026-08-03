jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  NotFoundException,
} from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { CharactersService } from './characters.service';
import { CharacterItemsService } from './characters-items.service';
import { CharacterExpertisesService } from './characters-expertises.service';
import { PrismaService } from '../prisma/prisma.service';
import type { CreateCharacterDto } from './dto/character.dto';

/**
 * Sibling spec to characters.service.spec.ts — covers the CRUD paths the
 * original file left untested:
 *
 *  - addCustomExpertise: rejects empty trim, builtin name collision, dup
 *    custom name; success path on a valid custom expertise
 *  - deleteExpertise: NotFound when row missing, BadRequest when row is
 *    builtin (custom=false), success when custom=true
 *  - updateExpertise: rejects empty patch, persists attribute alone,
 *    persists trained alone, persists both together
 *  - updateLevel: persists the new total level
 *  - updateAbilityChoices: rejects empty patch; serializes
 *    raceAbilityChoices / originChoices / classPowers to JSON strings
 *  - updateItem: NotFound when itemId belongs to another character
 *  - deleteItem: NotFound when itemId belongs to another character;
 *    success when owned
 *
 * Kept in a separate file from the main service spec so each file fits
 * comfortably under the project's 500-line cap.
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
  activeEffects: unknown[];
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
  characterCreate = jest.fn(async ({ data }: { data: unknown }) => ({
    id: 1,
    ...(data as object),
  }));
  characterUpdate = jest.fn(async ({ data }: { data: unknown }) => ({
    ...this.lastSeed!,
    ...(data as object),
  }));
  characterExpertiseFindUnique = jest.fn<
    Promise<{ id: number; custom?: boolean } | null>,
    [unknown]
  >();
  characterExpertiseCreate = jest.fn(async ({ data }: { data: unknown }) => data);
  characterExpertiseDelete = jest.fn(async () => ({ ok: true }));
  characterExpertiseUpdate = jest.fn(async ({ data }: { data: unknown }) => data);
  characterItemFindUnique = jest.fn<
    Promise<{ id: number; characterId: number; equipped?: string | null; catalogId?: string | null } | null>,
    [unknown]
  >();
  characterItemFindMany = jest.fn(async () => [] as { equipped: string | null }[]);
  characterItemUpdate = jest.fn(async ({ data }: { data: unknown }) => data);
  characterItemDelete = jest.fn(async () => ({ ok: true }));
  characterItemCreate = jest.fn(async ({ data }: { data: unknown }) => data);
  transaction = jest.fn(async (cb: (tx: unknown) => Promise<unknown>) =>
    cb({
      characterItem: {
        findUnique: this.characterItemFindUnique,
        findMany: this.characterItemFindMany,
        update: this.characterItemUpdate,
        delete: this.characterItemDelete,
        create: this.characterItemCreate,
      },
    }),
  );

  lastSeed: Character | null = null;

  seedCharacter(char: Character | null): void {
    this.lastSeed = char;
    this.characterFindUnique.mockResolvedValue(char);
  }

  get service() {
    return {
      character: {
        findUnique: this.characterFindUnique,
        create: this.characterCreate,
        update: this.characterUpdate,
      },
      characterExpertise: {
        findUnique: this.characterExpertiseFindUnique,
        create: this.characterExpertiseCreate,
        delete: this.characterExpertiseDelete,
        update: this.characterExpertiseUpdate,
      },
      characterItem: {
        findUnique: this.characterItemFindUnique,
        findMany: this.characterItemFindMany,
        update: this.characterItemUpdate,
        delete: this.characterItemDelete,
        create: this.characterItemCreate,
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

/** Item mutations moved to CharacterItemsService (auth still via
 * CharactersService.findOne); build both over the same fake prisma. */
async function makeItemsService(
  prisma: FakePrisma,
): Promise<CharacterItemsService> {
  const moduleRef = await Test.createTestingModule({
    providers: [
      CharactersService,
      CharacterItemsService,
      { provide: PrismaService, useValue: prisma.service },
    ],
  }).compile();
  return moduleRef.get(CharacterItemsService);
}

async function makeExpertisesService(
  prisma: FakePrisma,
): Promise<CharacterExpertisesService> {
  const moduleRef = await Test.createTestingModule({
    providers: [
      CharactersService,
      CharacterExpertisesService,
      { provide: PrismaService, useValue: prisma.service },
    ],
  }).compile();
  return moduleRef.get(CharacterExpertisesService);
}

describe('CharacterExpertisesService.addCustomExpertise', () => {
  it('rejects an empty trimmed name', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeExpertisesService(prisma);
    await expect(
      service.addCustomExpertise(1, 1, {
        name: '   ',
        attribute: 'strength',
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: { name: ['Name is required'] },
      }),
    });
  });

  it('rejects a builtin expertise name', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeExpertisesService(prisma);
    await expect(
      service.addCustomExpertise(1, 1, {
        name: 'Atletismo',
        attribute: 'strength',
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: {
          name: expect.arrayContaining([expect.stringContaining('builtin')]),
        },
      }),
    });
  });

  it('rejects a duplicate custom name (already on this character)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterExpertiseFindUnique.mockResolvedValue({ id: 99 });
    const service = await makeExpertisesService(prisma);
    await expect(
      service.addCustomExpertise(1, 1, {
        name: 'Cozinhar Sopa',
        attribute: 'wisdom',
      }),
    ).rejects.toMatchObject({
      response: expect.objectContaining({
        fieldErrors: {
          name: expect.arrayContaining([expect.stringContaining('already exists')]),
        },
      }),
    });
  });

  it('persists trimmed name, attribute, trained=true, custom=true', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterExpertiseFindUnique.mockResolvedValue(null);
    const service = await makeExpertisesService(prisma);
    await service.addCustomExpertise(1, 1, {
      name: '  Cozinhar Sopa  ',
      attribute: 'wisdom',
    });
    const payload = prisma.characterExpertiseCreate.mock.calls[0]![0] as {
      data: unknown;
    };
    expect(payload.data).toEqual({
      characterId: 1,
      name: 'Cozinhar Sopa',
      attribute: 'wisdom',
      trained: true,
      custom: true,
    });
  });
});

describe('CharacterExpertisesService.deleteExpertise', () => {
  it('throws NotFound when the expertise row is missing', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterExpertiseFindUnique.mockResolvedValue(null);
    const service = await makeExpertisesService(prisma);
    await expect(
      service.deleteExpertise(1, 1, 'Ghost'),
    ).rejects.toBeInstanceOf(NotFoundException);
  });

  it('rejects deleting a builtin expertise (custom=false)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterExpertiseFindUnique.mockResolvedValue({
      id: 7,
      custom: false,
    });
    const service = await makeExpertisesService(prisma);
    await expect(
      service.deleteExpertise(1, 1, 'Atletismo'),
    ).rejects.toBeInstanceOf(BadRequestException);
    expect(prisma.characterExpertiseDelete).not.toHaveBeenCalled();
  });

  it('deletes a custom expertise (custom=true)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    prisma.characterExpertiseFindUnique.mockResolvedValue({
      id: 8,
      custom: true,
    });
    const service = await makeExpertisesService(prisma);
    const result = await service.deleteExpertise(1, 1, 'Cozinhar Sopa');
    expect(result).toEqual({ name: 'Cozinhar Sopa' });
    expect(prisma.characterExpertiseDelete).toHaveBeenCalledWith({
      where: { id: 8 },
    });
  });
});

describe('CharacterExpertisesService.updateExpertise', () => {
  it('rejects an empty patch (no fields to update)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeExpertisesService(prisma);
    await expect(
      service.updateExpertise(1, 1, { name: 'Atletismo' }),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('persists attribute alone', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeExpertisesService(prisma);
    await service.updateExpertise(1, 1, {
      name: 'Atletismo',
      attribute: 'dexterity',
    });
    const payload = prisma.characterExpertiseUpdate.mock.calls[0]![0] as {
      data: unknown;
    };
    expect(payload.data).toEqual({ attribute: 'dexterity' });
  });

  it('persists trained alone', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeExpertisesService(prisma);
    await service.updateExpertise(1, 1, {
      name: 'Atletismo',
      trained: true,
    });
    const payload = prisma.characterExpertiseUpdate.mock.calls[0]![0] as {
      data: unknown;
    };
    expect(payload.data).toEqual({ trained: true });
  });

  it('persists both attribute and trained when both supplied', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeExpertisesService(prisma);
    await service.updateExpertise(1, 1, {
      name: 'Atletismo',
      attribute: 'strength',
      trained: false,
    });
    const payload = prisma.characterExpertiseUpdate.mock.calls[0]![0] as {
      data: unknown;
    };
    expect(payload.data).toEqual({ attribute: 'strength', trained: false });
  });
});

describe('CharactersService.updateLevel', () => {
  it('persists the new total level', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await service.updateLevel(1, 1, { level: 5 });
    // .at(-1): the findOne read-heal may prepend its own character.update.
    const payload = prisma.characterUpdate.mock.calls.at(-1)![0] as {
      data: { level: number };
    };
    expect(payload.data).toEqual({ level: 5 });
  });
});

describe('CharactersService.updateAbilityChoices', () => {
  it('rejects an empty patch (no fields to update)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await expect(
      service.updateAbilityChoices(1, 1, {}),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('serializes raceAbilityChoices / originChoices / classPowers to JSON', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter());
    const service = await makeService(prisma);
    await service.updateAbilityChoices(1, 1, {
      raceAbilityChoices: ['humano-versatil'],
      originChoices: ['origin-soldado-pericia-Fortitude'],
      classPowers: ['class.guerreiro.especializacao-em-arma'],
    });
    // .at(-1): the findOne read-heal may prepend its own character.update.
    const payload = prisma.characterUpdate.mock.calls.at(-1)![0] as {
      data: {
        raceAbilityChoices: string;
        originChoices: string;
        classPowers: string;
      };
    };
    expect(JSON.parse(payload.data.raceAbilityChoices)).toEqual([
      'humano-versatil',
    ]);
    expect(JSON.parse(payload.data.originChoices)).toEqual([
      'origin-soldado-pericia-Fortitude',
    ]);
    expect(JSON.parse(payload.data.classPowers)).toEqual([
      'class.guerreiro.especializacao-em-arma',
    ]);
  });
});

describe('CharacterItemsService.updateItem — cross-character guard', () => {
  it('throws NotFound when the itemId belongs to another character', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    prisma.characterItemFindUnique.mockResolvedValue({
      id: 99,
      characterId: 2,
    });
    const service = await makeItemsService(prisma);
    await expect(
      service.updateItem(1, 1, 99, { name: 'Adaga' }),
    ).rejects.toBeInstanceOf(NotFoundException);
  });

  it('rejects an empty patch (no fields to update)', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    prisma.characterItemFindUnique.mockResolvedValue({
      id: 5,
      characterId: 1,
      equipped: null,
    });
    const service = await makeItemsService(prisma);
    await expect(
      service.updateItem(1, 1, 5, {}),
    ).rejects.toBeInstanceOf(BadRequestException);
  });
});

describe('CharacterItemsService.deleteItem — cross-character guard', () => {
  it('throws NotFound when itemId belongs to another character', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    prisma.characterItemFindUnique.mockResolvedValue({
      id: 50,
      characterId: 2,
    });
    const service = await makeItemsService(prisma);
    await expect(service.deleteItem(1, 1, 50)).rejects.toBeInstanceOf(
      NotFoundException,
    );
    expect(prisma.characterItemDelete).not.toHaveBeenCalled();
  });

  it('deletes the item when owned by the target character', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter(makeCharacter({ id: 1 }));
    prisma.characterItemFindUnique.mockResolvedValue({
      id: 51,
      characterId: 1,
    });
    const service = await makeItemsService(prisma);
    const result = await service.deleteItem(1, 1, 51);
    expect(result).toEqual({ id: 51 });
    expect(prisma.characterItemDelete).toHaveBeenCalledWith({
      where: { id: 51 },
    });
  });
});

describe('CharactersService.updateConditions — condições do livro (p394)', () => {
  it('persiste ids válidos do catálogo', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter({ id: 1, ownerId: 7 } as never);
    const service = await makeService(prisma);
    const result = await service.updateConditions(7, 1, {
      activeConditions: ['caido', 'agarrado'],
    });
    expect(JSON.parse(result.activeConditions)).toEqual(['caido', 'agarrado']);
    expect(prisma.characterUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        data: { activeConditions: JSON.stringify(['caido', 'agarrado']) },
      }),
    );
  });

  it('rejeita id desconhecido com o valor ofensor', async () => {
    const prisma = new FakePrisma();
    prisma.seedCharacter({ id: 1, ownerId: 7 } as never);
    const service = await makeService(prisma);
    await expect(
      service.updateConditions(7, 1, { activeConditions: ['dormindo-demais'] }),
    ).rejects.toThrow(/dormindo-demais/);
  });
});

describe('CharactersService.create — creation-time ability choices', () => {
  const baseDto = (extra: Partial<CreateCharacterDto>): CreateCharacterDto =>
    ({
      name: 'Test',
      races: ['Humano'],
      origin: 'Acólito',
      classes: [{ className: 'Arcanista', level: 1 }],
      hpMax: 8,
      hpCurrent: 8,
      mpMax: 6,
      mpCurrent: 6,
      strength: 0,
      dexterity: 0,
      constitution: 0,
      intelligence: 0,
      wisdom: 0,
      charisma: 0,
      size: 'Médio',
      displacement: 9,
      ...extra,
    }) as CreateCharacterDto;

  it('persists classPowers, originChoices, classChoices, and trained perícias', async () => {
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(
      7,
      baseDto({
        classPowers: ['class.arcanista.caminho-mago'],
        originChoices: ['acolito-cura'],
        classChoices: { Arcanista: { caminho: 'mago' } },
        trainedExpertises: ['Misticismo'],
        powerChoices: { 'class.arcanista.especialista-em-escola': ['evocacao'] },
      }),
    );
    const { data } = prisma.characterCreate.mock.calls[0][0] as {
      data: {
        classPowers: string;
        originChoices: string;
        classChoices: string;
        powerChoices: string;
        expertises: { create: { name: string; trained: boolean }[] };
      };
    };
    expect(JSON.parse(data.classPowers)).toEqual(['class.arcanista.caminho-mago']);
    expect(JSON.parse(data.originChoices)).toEqual(['acolito-cura']);
    expect(JSON.parse(data.classChoices)).toEqual({ Arcanista: { caminho: 'mago' } });
    expect(JSON.parse(data.powerChoices)).toEqual({
      'class.arcanista.especialista-em-escola': ['evocacao'],
    });
    const mist = data.expertises.create.find((e) => e.name === 'Misticismo');
    expect(mist?.trained).toBe(true);
    const other = data.expertises.create.find((e) => e.name !== 'Misticismo');
    expect(other?.trained).toBe(false);
  });

  it('defaults to empty picks when omitted', async () => {
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(7, baseDto({}));
    const { data } = prisma.characterCreate.mock.calls[0][0] as {
      data: {
        classPowers: string;
        expertises: { create: { trained: boolean }[] };
      };
    };
    expect(JSON.parse(data.classPowers)).toEqual([]);
    expect(data.expertises.create.every((e) => e.trained === false)).toBe(true);
  });

  it('persists starting items + tibar (Equipamento p140)', async () => {
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(
      7,
      baseDto({
        tibar: 14,
        items: [
          { catalogId: 'mochila', name: 'Mochila', quantity: 1, slots: 1 },
          {
            catalogId: 'espada-longa',
            name: 'Espada longa',
            quantity: 1,
            slots: 1,
            equipped: 'wielded',
          },
          { name: 'Símbolo sagrado', quantity: 1, slots: 1 },
        ],
      }),
    );
    const { data } = prisma.characterCreate.mock.calls[0][0] as {
      data: {
        tibar: number;
        items: { create: { catalogId: string | null; name: string; equipped: string | null }[] };
      };
    };
    expect(data.tibar).toBe(14);
    expect(data.items.create).toHaveLength(3);
    expect(data.items.create[1]).toMatchObject({
      catalogId: 'espada-longa',
      equipped: 'wielded',
    });
    expect(data.items.create[2]).toMatchObject({ catalogId: null });
  });

  it('persists the devoção poder concedido (godPower, p96)', async () => {
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(
      7,
      baseDto({ god: 'Khalmyr', godPower: 'Coragem Total' }),
    );
    const { data } = prisma.characterCreate.mock.calls[0][0] as {
      data: { god: string | null; godPower: string };
    };
    expect(data.god).toBe('Khalmyr');
    expect(data.godPower).toBe('Coragem Total');
  });

  it('godPower defaults to empty when not devoto', async () => {
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(7, baseDto({}));
    const { data } = prisma.characterCreate.mock.calls[0][0] as {
      data: { godPower: string };
    };
    expect(data.godPower).toBe('');
  });

  it('persists the deformidade choice inside the race-choice JSON blobs', async () => {
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(
      7,
      baseDto({
        races: ['Minotauro', 'Lefou'],
        secondaryRaceChoices: [
          {
            race: 'Lefou',
            floatingPicks: ['strength', 'constitution', 'wisdom'],
            deformidade: {
              pericias: ['Furtividade'],
              tormentaPower: 'dentes-afiados',
            },
          },
        ],
      }),
    );
    const { data } = prisma.characterCreate.mock.calls[0][0] as {
      data: { secondaryRaceChoices: string };
    };
    expect(JSON.parse(data.secondaryRaceChoices)).toEqual([
      {
        race: 'Lefou',
        floatingPicks: ['strength', 'constitution', 'wisdom'],
        deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
      },
    ]);
  });

  it('queries relations ordered by insertion id — races[0]/classes[0] are the primaries', async () => {
    // Regression: without orderBy, SQLite returns CharacterRace rows in
    // unique-index order ([characterId, race] → alphabetical), silently
    // swapping the primary race (e.g. 'Lefou' < 'Minotauro').
    const prisma = new FakePrisma();
    const service = await makeService(prisma);
    await service.create(7, baseDto({ races: ['Minotauro', 'Lefou'] }));
    const { include } = prisma.characterCreate.mock.calls[0][0] as {
      include: {
        races: { orderBy?: { id?: string } };
        classes: { orderBy?: { id?: string } };
      };
    };
    expect(include.races.orderBy).toEqual({ id: 'asc' });
    expect(include.classes.orderBy).toEqual({ id: 'asc' });
  });
});
