jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import {
  BadRequestException,
  ForbiddenException,
  NotFoundException,
} from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { SessionsService } from './sessions.service';
import { PrismaService } from '../prisma/prisma.service';
import { CampaignsService } from '../campaigns/campaigns.service';

type SessionRow = {
  id: number;
  campaignId: number;
  title: string | null;
  sessionNumber: number;
  notes: string | null;
  status: 'planned' | 'active' | 'ended';
  startedAt: Date | null;
  endedAt: Date | null;
  createdAt: Date;
  updatedAt: Date;
};

function makeSession(over: Partial<SessionRow> = {}): SessionRow {
  return {
    id: 1,
    campaignId: 1,
    title: null,
    sessionNumber: 1,
    notes: null,
    status: 'planned',
    startedAt: null,
    endedAt: null,
    createdAt: new Date(),
    updatedAt: new Date(),
    ...over,
  };
}

async function setup(over?: {
  findMany?: jest.Mock;
  findUnique?: jest.Mock;
  create?: jest.Mock;
  update?: jest.Mock;
  delete?: jest.Mock;
  campaignsFindOne?: jest.Mock;
}) {
  const prisma = {
    session: {
      findMany: over?.findMany ?? jest.fn(),
      findUnique: over?.findUnique ?? jest.fn(),
      create: over?.create ?? jest.fn(),
      update: over?.update ?? jest.fn(),
      delete: over?.delete ?? jest.fn(),
    },
  };
  const campaigns = {
    findOne:
      over?.campaignsFindOne ??
      jest.fn().mockResolvedValue({ id: 1, ownerId: 1 }),
  };
  const moduleRef = await Test.createTestingModule({
    providers: [
      SessionsService,
      { provide: PrismaService, useValue: prisma },
      { provide: CampaignsService, useValue: campaigns },
    ],
  }).compile();
  return { service: moduleRef.get(SessionsService), prisma, campaigns };
}

describe('SessionsService.list', () => {
  it('scopes by campaign + orders by sessionNumber asc', async () => {
    const findMany = jest.fn().mockResolvedValue([]);
    const { service } = await setup({ findMany });
    await service.list(1, 1);
    expect(findMany).toHaveBeenCalledWith({
      where: { campaignId: 1 },
      orderBy: { sessionNumber: 'asc' },
    });
  });

  it('propagates ForbiddenException from campaigns.findOne', async () => {
    const campaignsFindOne = jest.fn().mockRejectedValue(
      new ForbiddenException(),
    );
    const { service } = await setup({ campaignsFindOne });
    await expect(service.list(99, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });
});

describe('SessionsService.findOne', () => {
  it('returns the row when session belongs to the campaign', async () => {
    const row = makeSession();
    const findUnique = jest.fn().mockResolvedValue(row);
    const { service } = await setup({ findUnique });
    await expect(service.findOne(1, 1, 1)).resolves.toBe(row);
  });

  it('throws NotFound when the row does not exist', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(service.findOne(1, 1, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('throws NotFound when the session belongs to another campaign', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeSession({ campaignId: 999 }));
    const { service } = await setup({ findUnique });
    await expect(service.findOne(1, 1, 1)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });
});

describe('SessionsService.create', () => {
  it('creates with sessionNumber + trimmed title / notes', async () => {
    const create = jest.fn().mockResolvedValue(makeSession());
    const { service } = await setup({ create });
    await service.create(1, 1, {
      sessionNumber: 2,
      title: '  Ataque de goblins  ',
      notes: '  \n  ',
    });
    expect(create).toHaveBeenCalledWith({
      data: {
        campaignId: 1,
        sessionNumber: 2,
        title: 'Ataque de goblins',
        notes: null,
      },
    });
  });

  it('drops missing title / notes to null', async () => {
    const create = jest.fn().mockResolvedValue(makeSession());
    const { service } = await setup({ create });
    await service.create(1, 1, { sessionNumber: 1 });
    expect(create).toHaveBeenCalledWith({
      data: {
        campaignId: 1,
        sessionNumber: 1,
        title: null,
        notes: null,
      },
    });
  });
});

describe('SessionsService.update', () => {
  it('rejects an empty patch', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeSession());
    const { service } = await setup({ findUnique });
    await expect(service.update(1, 1, 1, {})).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });

  it('applies sessionNumber alone', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeSession());
    const update = jest.fn().mockResolvedValue(makeSession({ sessionNumber: 5 }));
    const { service } = await setup({ findUnique, update });
    await service.update(1, 1, 1, { sessionNumber: 5 });
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { sessionNumber: 5 },
    });
  });

  it('trims title to null when whitespace-only', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeSession());
    const update = jest.fn().mockResolvedValue(makeSession());
    const { service } = await setup({ findUnique, update });
    await service.update(1, 1, 1, { title: '   ' });
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { title: null },
    });
  });
});

describe('SessionsService.start', () => {
  it('transitions planned → active + sets startedAt', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeSession({ status: 'planned' }));
    const update = jest.fn().mockResolvedValue(makeSession({ status: 'active' }));
    const { service } = await setup({ findUnique, update });
    await service.start(1, 1, 1);
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: expect.objectContaining({
        status: 'active',
        startedAt: expect.any(Date),
      }),
    });
  });

  it('is idempotent when already active', async () => {
    const existing = makeSession({ status: 'active' });
    const findUnique = jest.fn().mockResolvedValue(existing);
    const update = jest.fn();
    const { service } = await setup({ findUnique, update });
    await expect(service.start(1, 1, 1)).resolves.toEqual(existing);
    expect(update).not.toHaveBeenCalled();
  });

  it('rejects starting an already-ended session', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeSession({ status: 'ended' }));
    const { service } = await setup({ findUnique });
    await expect(service.start(1, 1, 1)).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });
});

describe('SessionsService.end', () => {
  it('transitions active → ended + sets endedAt', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeSession({ status: 'active' }));
    const update = jest.fn().mockResolvedValue(makeSession({ status: 'ended' }));
    const { service } = await setup({ findUnique, update });
    await service.end(1, 1, 1);
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: expect.objectContaining({
        status: 'ended',
        endedAt: expect.any(Date),
      }),
    });
  });

  it('is idempotent when already ended', async () => {
    const existing = makeSession({ status: 'ended' });
    const findUnique = jest.fn().mockResolvedValue(existing);
    const update = jest.fn();
    const { service } = await setup({ findUnique, update });
    await expect(service.end(1, 1, 1)).resolves.toEqual(existing);
    expect(update).not.toHaveBeenCalled();
  });

  it('rejects ending a planned session', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeSession({ status: 'planned' }));
    const { service } = await setup({ findUnique });
    await expect(service.end(1, 1, 1)).rejects.toBeInstanceOf(
      BadRequestException,
    );
  });
});

describe('SessionsService.remove', () => {
  it('deletes and returns { id }', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeSession());
    const del = jest.fn().mockResolvedValue({});
    const { service } = await setup({ findUnique, delete: del });
    await expect(service.remove(1, 1, 1)).resolves.toEqual({ id: 1 });
    expect(del).toHaveBeenCalledWith({ where: { id: 1 } });
  });
});
