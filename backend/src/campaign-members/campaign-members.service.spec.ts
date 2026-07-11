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
  campaignsResolveAccess?: jest.Mock;
  campaignFindUnique?: jest.Mock;
  memberFindFirst?: jest.Mock;
}) {
  const prisma = {
    campaign: {
      findUnique:
        over?.campaignFindUnique ??
        // caller 1 is the owner by default → joins their own campaign
        // without a token; token tests override ownerId to a stranger.
        jest.fn().mockResolvedValue({ id: 1, ownerId: 1, inviteToken: null }),
    },
    campaignMember: {
      findMany: over?.memberFindMany ?? jest.fn(),
      findUnique: over?.memberFindUnique ?? jest.fn(),
      findFirst: over?.memberFindFirst ?? jest.fn().mockResolvedValue(null),
      create: over?.memberCreate ?? jest.fn(),
      update: over?.memberUpdate ?? jest.fn(),
      delete: over?.memberDelete ?? jest.fn(),
    },
    character: {
      findUnique:
        over?.characterFindUnique ??
        jest.fn().mockResolvedValue({ id: 10, ownerId: 1 }),
    },
  };
  const campaigns = {
    findOne:
      over?.campaignsFindOne ??
      jest.fn().mockResolvedValue({ id: 1, ownerId: 1 }),
    resolveAccess:
      over?.campaignsResolveAccess ??
      jest.fn().mockResolvedValue({
        campaign: { id: 1, ownerId: 1 },
        role: 'gm',
      }),
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

  it('propagates Forbidden when caller is neither GM nor member', async () => {
    const campaignsResolveAccess = jest.fn().mockRejectedValue(
      new ForbiddenException(),
    );
    const { service } = await setup({ campaignsResolveAccess });
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

  it('rejects a second player character from the same user (one PC per campaign)', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    // caller already owns a player-role member in this campaign
    const memberFindFirst = jest.fn().mockResolvedValue({ id: 3 });
    const memberCreate = jest.fn();
    const { service } = await setup({
      memberFindUnique,
      memberFindFirst,
      memberCreate,
    });
    await expect(
      service.add(1, 1, { characterId: 10 }),
    ).rejects.toBeInstanceOf(ConflictException);
    expect(memberCreate).not.toHaveBeenCalled();
  });

  it('allows a second character when added as a GM-role NPC', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const memberFindFirst = jest.fn().mockResolvedValue({ id: 3 });
    const memberCreate = jest.fn().mockResolvedValue(makeMember({ role: 'gm' }));
    const { service } = await setup({
      memberFindUnique,
      memberFindFirst,
      memberCreate,
    });
    await service.add(1, 1, { characterId: 10, role: 'gm' });
    // GM-role exempt from the one-PC rule → findFirst not consulted
    expect(memberFindFirst).not.toHaveBeenCalled();
    expect(memberCreate).toHaveBeenCalled();
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

  it('rejects when the caller does not own the character (OC1 consent)', async () => {
    /* Pre-OC1: the check was "caller is GM of campaign" — any GM
     * could slot any user's PC into their table. Post-OC1: caller
     * must own the character. GM adding another user's char → 403. */
    const characterFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 10, ownerId: 999 });
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ characterFindUnique, memberFindUnique });
    await expect(
      service.add(1, 1, { characterId: 10 }),
    ).rejects.toBeInstanceOf(ForbiddenException);
  });

  it('accepts a matching invite token and persists the row', async () => {
    const campaignFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 999, inviteToken: 'good-token' });
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const memberCreate = jest.fn().mockResolvedValue(makeMember());
    const { service } = await setup({
      campaignFindUnique,
      memberFindUnique,
      memberCreate,
    });
    await service.add(1, 1, { characterId: 10, inviteToken: 'good-token' });
    expect(memberCreate).toHaveBeenCalledWith({
      data: { campaignId: 1, characterId: 10, role: 'player' },
    });
  });

  it('rejects a non-owner self-join with no invite token (closes the hole)', async () => {
    /* A stranger who owns a character must NOT be able to join a campaign
     * by id alone — that would grant member-aware read of its sessions. */
    const campaignFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 999, inviteToken: 'secret' });
    const memberCreate = jest.fn();
    const { service } = await setup({ campaignFindUnique, memberCreate });
    await expect(
      service.add(1, 1, { characterId: 10 }),
    ).rejects.toBeInstanceOf(ForbiddenException);
    expect(memberCreate).not.toHaveBeenCalled();
  });

  it('lets the campaign owner add their own character without a token', async () => {
    const campaignFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, ownerId: 1, inviteToken: null });
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const memberCreate = jest.fn().mockResolvedValue(makeMember());
    const { service } = await setup({
      campaignFindUnique,
      memberFindUnique,
      memberCreate,
    });
    await service.add(1, 1, { characterId: 10 });
    expect(memberCreate).toHaveBeenCalled();
  });

  it('rejects a stale/rotated invite token', async () => {
    /* GM rotated the token → the old link should stop working. */
    const campaignFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, inviteToken: 'new-token' });
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const memberCreate = jest.fn();
    const { service } = await setup({
      campaignFindUnique,
      memberFindUnique,
      memberCreate,
    });
    await expect(
      service.add(1, 1, { characterId: 10, inviteToken: 'old-token' }),
    ).rejects.toBeInstanceOf(ForbiddenException);
    expect(memberCreate).not.toHaveBeenCalled();
  });

  it('rejects an invite token when the campaign has none set', async () => {
    const campaignFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, inviteToken: null });
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ campaignFindUnique, memberFindUnique });
    await expect(
      service.add(1, 1, { characterId: 10, inviteToken: 'anything' }),
    ).rejects.toBeInstanceOf(ForbiddenException);
  });

  it('invite-token flow still rejects a non-owner character (consent preserved)', async () => {
    /* Even with a valid invite, the caller can only bring in a char
     * they own. Prevents relaying invites to grab someone else's PC. */
    const campaignFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 1, inviteToken: 'good-token' });
    const characterFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 10, ownerId: 999 });
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({
      campaignFindUnique,
      characterFindUnique,
      memberFindUnique,
    });
    await expect(
      service.add(1, 1, { characterId: 10, inviteToken: 'good-token' }),
    ).rejects.toBeInstanceOf(ForbiddenException);
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
  /* Post-OI1: `remove` fetches the member with nested owners so it can
   * authorize either the campaign GM or the character owner. Every
   * spec below mocks that shape explicitly. */
  const memberRowGmOwnedByOne = {
    id: 1,
    campaignId: 1,
    campaign: { ownerId: 1 },
    character: { ownerId: 42 },
  };

  it('GM (campaign owner) can remove any member', async () => {
    const memberFindUnique = jest
      .fn()
      .mockResolvedValue(memberRowGmOwnedByOne);
    const memberDelete = jest.fn().mockResolvedValue({});
    const { service } = await setup({ memberFindUnique, memberDelete });
    await expect(service.remove(1, 1, 1)).resolves.toEqual({ id: 1 });
    expect(memberDelete).toHaveBeenCalledWith({ where: { id: 1 } });
  });

  it('player (character owner) can leave the campaign (OI1)', async () => {
    /* Pre-OI1 this was rejected — only the GM could remove. */
    const memberFindUnique = jest
      .fn()
      .mockResolvedValue(memberRowGmOwnedByOne);
    const memberDelete = jest.fn().mockResolvedValue({});
    const { service } = await setup({ memberFindUnique, memberDelete });
    await expect(service.remove(42, 1, 1)).resolves.toEqual({ id: 1 });
  });

  it('third party (neither GM nor char owner) is rejected', async () => {
    const memberFindUnique = jest
      .fn()
      .mockResolvedValue(memberRowGmOwnedByOne);
    const { service } = await setup({ memberFindUnique });
    await expect(service.remove(999, 1, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });

  it('NotFound when the member row is missing', async () => {
    const memberFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ memberFindUnique });
    await expect(service.remove(1, 1, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('NotFound when the member belongs to another campaign', async () => {
    const memberFindUnique = jest
      .fn()
      .mockResolvedValue({ ...memberRowGmOwnedByOne, campaignId: 999 });
    const { service } = await setup({ memberFindUnique });
    await expect(service.remove(1, 1, 1)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });
});

describe('CampaignMembersService.listForCharacter', () => {
  it('returns campaigns joined with the member row for the given character', async () => {
    const rows = [
      { id: 1, campaignId: 1, characterId: 10, role: 'player', campaign: { id: 1, name: 'C' } },
    ];
    const memberFindMany = jest.fn().mockResolvedValue(rows);
    const characterFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 10, ownerId: 5 });
    const { service } = await setup({ memberFindMany, characterFindUnique });
    await expect(service.listForCharacter(5, 10)).resolves.toEqual(rows);
    expect(memberFindMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { characterId: 10 },
        include: expect.objectContaining({ campaign: expect.any(Object) }),
      }),
    );
  });

  it('throws NotFound when the character does not exist', async () => {
    const characterFindUnique = jest.fn().mockResolvedValue(null);
    const { service } = await setup({ characterFindUnique });
    await expect(service.listForCharacter(1, 99)).rejects.toBeInstanceOf(
      NotFoundException,
    );
  });

  it('throws Forbidden when the character belongs to another user', async () => {
    const characterFindUnique = jest
      .fn()
      .mockResolvedValue({ id: 10, ownerId: 999 });
    const { service } = await setup({ characterFindUnique });
    await expect(service.listForCharacter(1, 10)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });
});
