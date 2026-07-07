jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  ConflictException,
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
  charactersFindOne?: jest.Mock;
}) {
  const prisma = {
    characterSpell: {
      create: over?.create ?? jest.fn(),
      findUnique: over?.findUnique ?? jest.fn(),
      update: over?.update ?? jest.fn(),
      deleteMany: over?.deleteMany ?? jest.fn().mockResolvedValue({ count: 1 }),
    },
  };
  const characters = {
    findOne:
      over?.charactersFindOne ??
      jest.fn().mockResolvedValue({ id: 1, ownerId: 1 }),
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
