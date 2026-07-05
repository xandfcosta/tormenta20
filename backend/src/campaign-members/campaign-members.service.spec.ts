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
import { CampaignMembersService } from './campaign-members.service';
import { PrismaService } from '../prisma/prisma.service';
import { CampaignsService } from '../campaigns/campaigns.service';

type MemberRow = {
  id: number;
  campaignId: number;
  characterId: number;
  role: 'player' | 'gm';
  addedAt: Date;
};

function makeMember(over: Partial<MemberRow> = {}): MemberRow {
  return {
    id: 1,
    campaignId: 1,
    characterId: 10,
    role: 'player',
    addedAt: new Date(),
    ...over,
  };
}

async function setup(over?: {
  memberFindMany?: jest.Mock;
  memberFindUnique?: jest.Mock;
  memberCreate?: jest.Mock;
  memberUpdate?: jest.Mock;
  memberDelete?: jest.Mock;
  characterFindUnique?: jest.Mock;
  campaignsFindOne?: jest.Mock;
}) {
  const prisma = {
    campaignMember: {
      findMany: over?.memberFindMany ?? jest.fn(),
      findUnique: over?.memberFindUnique ?? jest.fn(),
      create: over?.memberCreate ?? jest.fn(),
      update: over?.memberUpdate ?? jest.fn(),
      delete: over?.memberDelete ?? jest.fn(),
    },
    character: {
      findUnique:
        over?.characterFindUnique ??
        jest.fn().mockResolvedValue({ id: 10 }),
    },
  };
  const campaigns = {
    findOne:
      over?.campaignsFindOne ??
      jest.fn().mockResolvedValue({ id: 1, ownerId: 1 }),
  };
  const moduleRef = await Test.createTestingModule({
    providers: [
      CampaignMembersService,
      { provide: PrismaService, useValue: prisma },
      { provide: CampaignsService, useValue: campaigns },
    ],
  }).compile();
  return { service: moduleRef.get(CampaignMembersService), prisma, campaigns };
}

describe('CampaignMembersService.list', () => {
  it('scopes by campaign + includes character roster fields', async () => {
    const memberFindMany = jest.fn().mockResolvedValue([]);
    const { service } = await setup({ memberFindMany });
    await service.list(1, 1);
    expect(memberFindMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { campaignId: 1 },
        orderBy: { addedAt: 'asc' },
        include: expect.objectContaining({
          character: expect.any(Object),
        }),
      }),
    );
  });

  it('propagates Forbidden from campaigns.findOne', async () => {
    const campaignsFindOne = jest.fn().mockRejectedValue(
      new ForbiddenException(),
    );
    const { service } = await setup({ campaignsFindOne });
    await expect(service.list(9, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });
});

describe('CampaignMembersService.add', () => {
  it('defaults role to player and persists the row', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const memberCreate = jest
      .fn()
      .mockResolvedValue(makeMember());
    const { service } = await setup({ memberFindUnique, memberCreate });
    await service.add(1, 1, { characterId: 10 });
    expect(memberCreate).toHaveBeenCalledWith({
      data: { campaignId: 1, characterId: 10, role: 'player' },
    });
  });

  it('accepts an explicit role', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const memberCreate = jest
      .fn()
      .mockResolvedValue(makeMember({ role: 'gm' }));
    const { service } = await setup({ memberFindUnique, memberCreate });
    await service.add(1, 1, { characterId: 10, role: 'gm' });
    expect(memberCreate).toHaveBeenCalledWith({
      data: { campaignId: 1, characterId: 10, role: 'gm' },
    });
  });

  it('rejects when the character does not exist', async () => {
    const characterFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ characterFindUnique });
    await expect(
      service.add(1, 1, { characterId: 99 }),
    ).rejects.toBeInstanceOf(BadRequestException);
  });

  it('rejects when the character is already a member', async () => {
    const memberFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 5 });
    const { service } = await setup({ memberFindUnique });
    await expect(
      service.add(1, 1, { characterId: 10 }),
    ).rejects.toBeInstanceOf(ConflictException);
  });
});

describe('CampaignMembersService.findOne', () => {
  it('returns the row when scoped to the campaign', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(makeMember());
    const { service } = await setup({ memberFindUnique });
    await expect(service.findOne(1, 1, 1)).resolves.toEqual(
      expect.objectContaining({ id: 1, campaignId: 1 }),
    );
  });

  it('throws NotFound when the row does not exist', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ memberFindUnique });
    await expect(service.findOne(1, 1, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('throws NotFound when the row belongs to another campaign', async () => {
    const memberFindUnique = jest
      .fn()
      .mockResolvedValue(makeMember({ campaignId: 999 }));
    const { service } = await setup({ memberFindUnique });
    await expect(service.findOne(1, 1, 1)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });
});

describe('CampaignMembersService.updateRole', () => {
  it('applies the role patch', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(makeMember());
    const memberUpdate = jest
      .fn()
      .mockResolvedValue(makeMember({ role: 'gm' }));
    const { service } = await setup({ memberFindUnique, memberUpdate });
    await service.updateRole(1, 1, 1, { role: 'gm' });
    expect(memberUpdate).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { role: 'gm' },
    });
  });
});

describe('CampaignMembersService.remove', () => {
  it('deletes when the member is scoped to the campaign', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(makeMember());
    const memberDelete = jest.fn().mockResolvedValue({});
    const { service } = await setup({ memberFindUnique, memberDelete });
    await expect(service.remove(1, 1, 1)).resolves.toEqual({ id: 1 });
    expect(memberDelete).toHaveBeenCalledWith({ where: { id: 1 } });
  });
});
