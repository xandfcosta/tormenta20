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
  memberFindFirst?: jest.Mock;
}) {
  const prisma = {
    campaign: {
      findMany: over?.findMany ?? jest.fn(),
      findUnique: over?.findUnique ?? jest.fn(),
      create: over?.create ?? jest.fn(),
      update: over?.update ?? jest.fn(),
      delete: over?.delete ?? jest.fn(),
    },
    campaignMember: {
      findFirst: over?.memberFindFirst ?? jest.fn(),
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
  it('returns owned + member campaigns, tagged with the caller role', async () => {
    const owned = makeCampaign({ id: 1, ownerId: 7 });
    const joined = makeCampaign({ id: 2, ownerId: 99 });
    const findMany = jest.fn().mockResolvedValue([owned, joined]);
    const { service } = await setup({ findMany });
    const result = await service.list(7);
    expect(findMany).toHaveBeenCalledWith({
      where: {
        OR: [
          { ownerId: 7 },
          { members: { some: { character: { ownerId: 7 } } } },
        ],
      },
      orderBy: { updatedAt: 'desc' },
    });
    expect(result).toEqual([
      { ...owned, role: 'gm' },
      { ...joined, role: 'player' },
    ]);
  });
});

describe('CampaignsService.resolveAccess', () => {
  it('returns role=gm when the caller owns the campaign', async () => {
    const row = makeCampaign({ id: 1, ownerId: 5 });
    const findUnique = jest.fn().mockResolvedValue(row);
    const { service } = await setup({ findUnique });
    await expect(service.resolveAccess(5, 1)).resolves.toEqual({
      campaign: row,
      role: 'gm',
    });
  });

  it('returns role=player when a character of the caller is a member', async () => {
    const row = makeCampaign({ id: 1, ownerId: 999 });
    const findUnique = jest.fn().mockResolvedValue(row);
    const memberFindFirst = jest.fn().mockResolvedValue({ id: 42 });
    const { service } = await setup({ findUnique, memberFindFirst });
    await expect(service.resolveAccess(5, 1)).resolves.toEqual({
      campaign: row,
      role: 'player',
    });
    expect(memberFindFirst).toHaveBeenCalledWith({
      where: { campaignId: 1, character: { ownerId: 5 } },
      select: { id: true },
    });
  });

  it('throws NotFound when the campaign does not exist', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(service.resolveAccess(5, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('throws Forbidden when the caller is neither owner nor member', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign({ ownerId: 999 }));
    const memberFindFirst = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique, memberFindFirst });
    await expect(service.resolveAccess(5, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
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

describe('CampaignsService.assertCanJoin', () => {
  it('allows the owner to join without a token', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 5, inviteToken: null });
    const { service } = await setup({ findUnique });
    await expect(service.assertCanJoin(5, 1)).resolves.toBeUndefined();
  });

  it('allows a non-owner presenting the current invite token', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 999, inviteToken: 'tok' });
    const { service } = await setup({ findUnique });
    await expect(service.assertCanJoin(5, 1, 'tok')).resolves.toBeUndefined();
  });

  it('rejects a non-owner with a wrong/absent token', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 999, inviteToken: 'tok' });
    const { service } = await setup({ findUnique });
    await expect(service.assertCanJoin(5, 1, 'nope')).rejects.toBeInstanceOf(
      ForbiddenException,
    );
    await expect(service.assertCanJoin(5, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });

  it('rejects a non-owner when the campaign has no token set', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 999, inviteToken: null });
    const { service } = await setup({ findUnique });
    await expect(service.assertCanJoin(5, 1, 'anything')).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });

  it('throws NotFound when the campaign does not exist', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(service.assertCanJoin(5, 99)).rejects.toBeInstanceOf(
      NotFoundException,
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

describe('CampaignsService.rotateInviteToken', () => {
  it('generates a token, persists it, and returns {campaignId, token}', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const update = jest
      .fn()
      .mockImplementation(async ({ data }: { data: { inviteToken: string } }) => ({
        id: 1,
        inviteToken: data.inviteToken,
      }));
    const { service } = await setup({ findUnique, update });
    const result = await service.rotateInviteToken(1, 1);
    expect(result.campaignId).toBe(1);
    expect(typeof result.token).toBe('string');
    expect(result.token.length).toBeGreaterThan(16);
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { inviteToken: result.token },
      select: { id: true, inviteToken: true },
    });
  });

  it('rejects when the caller is not the GM', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeCampaign({ ownerId: 999 }));
    const update = jest.fn();
    const { service } = await setup({ findUnique, update });
    await expect(service.rotateInviteToken(1, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
    expect(update).not.toHaveBeenCalled();
  });

  it('each rotation produces a different token', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeCampaign());
    const update = jest
      .fn()
      .mockImplementation(async ({ data }: { data: { inviteToken: string } }) => ({
        id: 1,
        inviteToken: data.inviteToken,
      }));
    const { service } = await setup({ findUnique, update });
    const a = await service.rotateInviteToken(1, 1);
    const b = await service.rotateInviteToken(1, 1);
    expect(a.token).not.toBe(b.token);
  });
});

describe('CampaignsService.resolveInviteToken', () => {
  it('returns null for an empty token without hitting the DB', async () => {
    const findUnique = jest.fn();
    const { service } = await setup({ findUnique });
    await expect(service.resolveInviteToken('')).resolves.toBeNull();
    expect(findUnique).not.toHaveBeenCalled();
  });

  it('returns {campaignId, campaignName} on match', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 42, name: 'Vale de Sombras' });
    const { service } = await setup({ findUnique });
    await expect(
      service.resolveInviteToken('abc123token'),
    ).resolves.toEqual({ campaignId: 42, campaignName: 'Vale de Sombras' });
    expect(findUnique).toHaveBeenCalledWith({
      where: { inviteToken: 'abc123token' },
      select: { id: true, name: true },
    });
  });

  it('returns null on unknown/rotated token (no throw)', async () => {
    const findUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ findUnique });
    await expect(
      service.resolveInviteToken('stale-token'),
    ).resolves.toBeNull();
  });
});
