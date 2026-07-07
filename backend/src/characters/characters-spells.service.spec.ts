jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  ConflictException,
  ForbiddenException,
  NotFoundException,
} from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { CharactersSpellsService } from './characters-spells.service';
import { CharactersService } from './characters.service';
import { PrismaService } from '../prisma/prisma.service';

async function setup(over?: {
  create?: jest.Mock;
  findUnique?: jest.Mock;
  update?: jest.Mock;
  deleteMany?: jest.Mock;
  characterUpdate?: jest.Mock;
  charactersFindOne?: jest.Mock;
}) {
  const prisma = {
    characterSpell: {
      create: over?.create ?? jest.fn(),
      findUnique: over?.findUnique ?? jest.fn(),
      update: over?.update ?? jest.fn(),
      deleteMany: over?.deleteMany ?? jest.fn().mockResolvedValue({ count: 1 }),
    },
    character: {
      update: over?.characterUpdate ?? jest.fn().mockResolvedValue({}),
    },
  };
  const characters = {
    findOne:
      over?.charactersFindOne ??
      jest.fn().mockResolvedValue({
        id: 1,
        ownerId: 1,
        level: 6,
        mpCurrent: 20,
        mpMax: 20,
        classes: [{ className: 'Arcanista', level: 6 }],
      }),
  };
  const moduleRef = await Test.createTestingModule({
    providers: [
      CharactersSpellsService,
      { provide: PrismaService, useValue: prisma },
      { provide: CharactersService, useValue: characters },
    ],
  }).compile();
  return {
    service: moduleRef.get(CharactersSpellsService),
    prisma,
    characters,
  };
}

describe('CharactersSpellsService.learnSpell', () => {
  it('creates the row with prepared=false on a valid catalog id', async () => {
    const create = jest.fn().mockResolvedValue({
      id: 10,
      characterId: 1,
      catalogSpellId: 'luz',
      prepared: false,
    });
    const { service } = await setup({ create });
    await service.learnSpell(1, 1, 'luz');
    expect(create).toHaveBeenCalledWith({
      data: { characterId: 1, catalogSpellId: 'luz', prepared: false },
    });
  });

  it('rejects an unknown spell id (not in catalog)', async () => {
    const create = jest.fn();
    const { service } = await setup({ create });
    await expect(
      service.learnSpell(1, 1, 'not-a-real-spell'),
    ).rejects.toBeInstanceOf(BadRequestException);
    expect(create).not.toHaveBeenCalled();
  });

  it('translates P2002 into 409 Conflict (already learned)', async () => {
    const create = jest.fn().mockRejectedValue({ code: 'P2002' });
    const { service } = await setup({ create });
    await expect(
      service.learnSpell(1, 1, 'luz'),
    ).rejects.toBeInstanceOf(ConflictException);
  });

  it('propagates the ownership Forbidden from charactersFindOne', async () => {
    const charactersFindOne = jest.fn().mockRejectedValue(
      new BadRequestException('nope'),
    );
    const { service } = await setup({ charactersFindOne });
    await expect(
      service.learnSpell(1, 1, 'luz'),
    ).rejects.toBeInstanceOf(BadRequestException);
  });
});

describe('CharactersSpellsService.unlearnSpell', () => {
  it('deletes any matching row and returns the count', async () => {
    const deleteMany = jest.fn().mockResolvedValue({ count: 1 });
    const { service } = await setup({ deleteMany });
    await expect(service.unlearnSpell(1, 1, 'luz')).resolves.toEqual({
      catalogSpellId: 'luz',
      removed: 1,
    });
    expect(deleteMany).toHaveBeenCalledWith({
      where: { characterId: 1, catalogSpellId: 'luz' },
    });
  });

  it('is a no-op when no row matches (count 0, no throw)', async () => {
    const deleteMany = jest.fn().mockResolvedValue({ count: 0 });
    const { service } = await setup({ deleteMany });
    await expect(service.unlearnSpell(1, 1, 'luz')).resolves.toEqual({
      catalogSpellId: 'luz',
      removed: 0,
    });
  });
});

describe('CharactersSpellsService.setSpellPrepared', () => {
  it('toggles the flag when the row exists', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 10, prepared: false });
    const update = jest
      .fn()
      .mockResolvedValue({ id: 10, prepared: true });
    const { service } = await setup({ findUnique, update });
    await service.setSpellPrepared(1, 1, 'luz', true);
    expect(update).toHaveBeenCalledWith({
      where: { id: 10 },
      data: { prepared: true },
    });
  });

  it('throws NotFound when the spell is not in the character spellbook', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(
      service.setSpellPrepared(1, 1, 'luz', true),
    ).rejects.toBeInstanceOf(NotFoundException);
  });
});

describe('CharactersSpellsService.castSpell', () => {
  // Uses 'luz' from the fixture catalog: circle 1, base 1 PM, augments
  // include a +1 aumenta and a +0 muda (classOnly). All catalog spells
  // in these tests come from the real SPELL_CATALOG import.

  it('debits base PM on happy path (Arcanista, learned, no augments)', async () => {
    const learned = { id: 10, prepared: false };
    const characterUpdate = jest.fn().mockResolvedValue({});
    const charactersFindOne = jest.fn().mockResolvedValue({
      id: 1,
      ownerId: 1,
      level: 6,
      mpCurrent: 20,
      classes: [{ className: 'Arcanista', level: 6 }],
    });
    const findUnique = jest.fn().mockResolvedValue(learned);
    const { service } = await setup({
      findUnique,
      characterUpdate,
      charactersFindOne,
    });
    await service.castSpell(1, 1, 'luz', []);
    expect(characterUpdate).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { mpCurrent: 19 },
    });
  });

  it('rejects insufficient PM (400)', async () => {
    const learned = { id: 10, prepared: false };
    const findUnique = jest.fn().mockResolvedValue(learned);
    const charactersFindOne = jest.fn().mockResolvedValue({
      id: 1,
      ownerId: 1,
      level: 6,
      mpCurrent: 0,
      classes: [{ className: 'Arcanista', level: 6 }],
    });
    const { service } = await setup({ findUnique, charactersFindOne });
    await expect(
      service.castSpell(1, 1, 'luz', []),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('rejects total PM above per-spell limit (400)', async () => {
    // level 2 → limit 1. 'luz' base 1 PM + 1 augment (+1) = 2 > limit.
    const learned = { id: 10, prepared: false };
    const findUnique = jest.fn().mockResolvedValue(learned);
    const charactersFindOne = jest.fn().mockResolvedValue({
      id: 1,
      ownerId: 1,
      level: 2,
      mpCurrent: 20,
      classes: [{ className: 'Arcanista', level: 2 }],
    });
    const { service } = await setup({ findUnique, charactersFindOne });
    await expect(
      service.castSpell(1, 1, 'luz', [{ augmentIndex: 0, stacks: 1 }]),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('rejects spell not learned (404)', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(
      service.castSpell(1, 1, 'luz', []),
    ).rejects.toBeInstanceOf(NotFoundException);
  });

  it('rejects unlearned catalog spell first (BadRequest for unknown id)', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(
      service.castSpell(1, 1, 'not-a-real-spell', []),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('Clérigo requires prepared=true (403 otherwise)', async () => {
    const learned = { id: 10, prepared: false };
    const findUnique = jest.fn().mockResolvedValue(learned);
    const charactersFindOne = jest.fn().mockResolvedValue({
      id: 1,
      ownerId: 1,
      level: 6,
      mpCurrent: 20,
      classes: [{ className: 'Clérigo', level: 6 }],
    });
    const { service } = await setup({ findUnique, charactersFindOne });
    await expect(
      service.castSpell(1, 1, 'luz', []),
    ).rejects.toBeInstanceOf(ForbiddenException);
  });

  it('Clérigo cast succeeds when prepared=true', async () => {
    const learned = { id: 10, prepared: true };
    const characterUpdate = jest.fn().mockResolvedValue({});
    const charactersFindOne = jest.fn().mockResolvedValue({
      id: 1,
      ownerId: 1,
      level: 6,
      mpCurrent: 20,
      classes: [{ className: 'Clérigo', level: 6 }],
    });
    const findUnique = jest.fn().mockResolvedValue(learned);
    const { service } = await setup({
      findUnique,
      characterUpdate,
      charactersFindOne,
    });
    await service.castSpell(1, 1, 'luz', []);
    expect(characterUpdate).toHaveBeenCalled();
  });

  it('augment stacks sum correctly into total PM', async () => {
    // luz augment index 0 is +1 aumenta. Level 20 → limit 10.
    // base 1 + augment +1 × 2 stacks = 3. mpCurrent 20 → 17 after.
    const learned = { id: 10, prepared: false };
    const characterUpdate = jest.fn().mockResolvedValue({});
    const charactersFindOne = jest.fn().mockResolvedValue({
      id: 1,
      ownerId: 1,
      level: 20,
      mpCurrent: 20,
      classes: [{ className: 'Arcanista', level: 20 }],
    });
    const findUnique = jest.fn().mockResolvedValue(learned);
    const { service } = await setup({
      findUnique,
      characterUpdate,
      charactersFindOne,
    });
    await service.castSpell(1, 1, 'luz', [{ augmentIndex: 0, stacks: 2 }]);
    expect(characterUpdate).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { mpCurrent: 17 },
    });
  });

  it("rejects 'muda' augment stacked > 1 (400)", async () => {
    // luz augment index 2 is 'muda' (classOnly:arcanos, +0). Stack 2 → 400.
    const learned = { id: 10, prepared: false };
    const findUnique = jest.fn().mockResolvedValue(learned);
    const { service } = await setup({ findUnique });
    await expect(
      service.castSpell(1, 1, 'luz', [{ augmentIndex: 2, stacks: 2 }]),
    ).rejects.toBeInstanceOf(BadRequestException);
  });
});
