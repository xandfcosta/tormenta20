jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  ForbiddenException,
  NotFoundException,
} from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { CampaignsService } from './campaigns.service';
import { PrismaService } from '../prisma/prisma.service';

type CampaignRow = {
  id: number;
  ownerId: number;
  name: string;
  description: string | null;
  createdAt: Date;
  updatedAt: Date;
};

function makeCampaign(over: Partial<CampaignRow> = {}): CampaignRow {
  return {
    id: 1,
    ownerId: 1,
    name: 'Campanha do Vale',
    description: null,
    createdAt: new Date(),
    updatedAt: new Date(),
    ...over,
  };
}

async function setup(over?: {
  findUnique?: jest.Mock;
  findMany?: jest.Mock;
  create?: jest.Mock;
  update?: jest.Mock;
  delete?: jest.Mock;
}) {
  const prisma = {
    campaign: {
      findMany: over?.findMany ?? jest.fn(),
      findUnique: over?.findUnique ?? jest.fn(),
      create: over?.create ?? jest.fn(),
      update: over?.update ?? jest.fn(),
      delete: over?.delete ?? jest.fn(),
    },
  };
  const moduleRef = await Test.createTestingModule({
    providers: [
      CampaignsService,
      { provide: PrismaService, useValue: prisma },
    ],
  }).compile();
  return { service: moduleRef.get(CampaignsService), prisma };
}

describe('CampaignsService.list', () => {
  it('scopes the query to ownerId + orders by updatedAt desc', async () => {
    const findMany = jest.fn().mockResolvedValue([makeCampaign()]);
    const { service } = await setup({ findMany });
    await service.list(7);
    expect(findMany).toHaveBeenCalledWith({
      where: { ownerId: 7 },
      orderBy: { updatedAt: 'desc' },
    });
  });
});

describe('CampaignsService.findOne', () => {
  it('returns the row when the owner matches', async () => {
    const row = makeCampaign({ ownerId: 5 });
    const findUnique = jest.fn().mockResolvedValue(row);
    const { service } = await setup({ findUnique });
    await expect(service.findOne(5, 1)).resolves.toEqual(row);
  });

  it('throws NotFound when the row does not exist', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(service.findOne(5, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('throws Forbidden when the row belongs to another user', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeCampaign({ ownerId: 999 }));
    const { service } = await setup({ findUnique });
    await expect(service.findOne(5, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });
});

describe('CampaignsService.create', () => {
  it('trims name / description and assigns the owner from the JWT user', async () => {
    const create = jest
      .fn()
      .mockResolvedValue(makeCampaign({ name: 'Ok' }));
    const { service } = await setup({ create });
    await service.create(42, { name: '  Ok  ', description: '  hi  ' });
    expect(create).toHaveBeenCalledWith({
      data: { ownerId: 42, name: 'Ok', description: 'hi' },
    });
  });

  it('persists description as null when omitted', async () => {
    const create = jest.fn().mockResolvedValue(makeCampaign());
    const { service } = await setup({ create });
    await service.create(42, { name: 'Solo' });
    expect(create).toHaveBeenCalledWith({
      data: { ownerId: 42, name: 'Solo', description: null },
    });
  });
});

describe('CampaignsService.update', () => {
  it('rejects an empty patch', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const { service } = await setup({ findUnique });
    await expect(service.update(1, 1, {})).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });

  it('applies name alone', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const update = jest.fn().mockResolvedValue(makeCampaign({ name: 'New' }));
    const { service } = await setup({ findUnique, update });
    await service.update(1, 1, { name: '  New  ' });
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { name: 'New' },
    });
  });

  it('applies description alone', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const update = jest.fn().mockResolvedValue(makeCampaign());
    const { service } = await setup({ findUnique, update });
    await service.update(1, 1, { description: 'stuff' });
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { description: 'stuff' },
    });
  });

  it('trims description to null when the payload is whitespace', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const update = jest.fn().mockResolvedValue(makeCampaign());
    const { service } = await setup({ findUnique, update });
    await service.update(1, 1, { description: '   ' });
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { description: null },
    });
  });

  it('enforces ownership before mutation', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeCampaign({ ownerId: 999 }));
    const update = jest.fn();
    const { service } = await setup({ findUnique, update });
    await expect(
      service.update(1, 1, { name: 'Nope' }),
    ).rejects.toBeInstanceOf(ForbiddenException);
    expect(update).not.toHaveBeenCalled();
  });
});

describe('CampaignsService.remove', () => {
  it('deletes when the owner matches and returns { id }', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const del = jest.fn().mockResolvedValue({});
    const { service } = await setup({ findUnique, delete: del });
    await expect(service.remove(1, 1)).resolves.toEqual({ id: 1 });
    expect(del).toHaveBeenCalledWith({ where: { id: 1 } });
  });

  it('enforces ownership before deleting', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeCampaign({ ownerId: 999 }));
    const del = jest.fn();
    const { service } = await setup({ findUnique, delete: del });
    await expect(service.remove(1, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
    expect(del).not.toHaveBeenCalled();
  });
});
