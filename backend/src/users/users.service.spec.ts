jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import { Test } from '@nestjs/testing';
import { UsersService } from './users.service';
import { PrismaService } from '../prisma/prisma.service';

class FakeUserRepo {
  findMany = jest.fn(async () => [
    { id: 1, email: 'a@b.com', name: 'A', createdAt: new Date('2025-01-01') },
    { id: 2, email: 'c@d.com', name: null, createdAt: new Date('2025-02-01') },
  ]);
}

class FakeCampaignMemberRepo {
  findMany = jest.fn(async () => []);
}

async function setup(over?: {
  memberFindMany?: jest.Mock;
  userFindMany?: jest.Mock;
}) {
  const userRepo = new FakeUserRepo();
  if (over?.userFindMany) userRepo.findMany = over.userFindMany;
  const memberRepo = new FakeCampaignMemberRepo();
  if (over?.memberFindMany) memberRepo.findMany = over.memberFindMany;
  const moduleRef = await Test.createTestingModule({
    providers: [
      UsersService,
      {
        provide: PrismaService,
        useValue: { user: userRepo, campaignMember: memberRepo },
      },
    ],
  }).compile();
  return {
    service: moduleRef.get(UsersService),
    userRepo,
    memberRepo,
  };
}

describe('UsersService.listVisibleTo (scoped to co-members)', () => {
  it('always includes the caller themselves', async () => {
    const { service, userRepo } = await setup({
      userFindMany: jest.fn(async () => [{ id: 7 }]),
    });
    await service.listVisibleTo(7);
    const lastCall = userRepo.findMany.mock.calls.at(-1)![0]!;
    expect(lastCall.where.id.in).toContain(7);
  });

  it('includes users who own characters in campaigns the caller runs', async () => {
    const memberFindMany = jest
      .fn()
      .mockResolvedValueOnce([{ character: { ownerId: 42 } }])
      .mockResolvedValueOnce([]);
    const { service, userRepo } = await setup({
      memberFindMany,
    });
    await service.listVisibleTo(7);
    const lastCall = userRepo.findMany.mock.calls.at(-1)![0]!;
    expect(lastCall.where.id.in).toContain(42);
  });

  it('includes GMs of campaigns the caller plays in', async () => {
    const memberFindMany = jest
      .fn()
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([{ campaign: { ownerId: 99 } }]);
    const { service, userRepo } = await setup({
      memberFindMany,
    });
    await service.listVisibleTo(7);
    const lastCall = userRepo.findMany.mock.calls.at(-1)![0]!;
    expect(lastCall.where.id.in).toContain(99);
  });

  it('de-duplicates ids so a user shared across many campaigns appears once', async () => {
    const memberFindMany = jest
      .fn()
      .mockResolvedValueOnce([
        { character: { ownerId: 42 } },
        { character: { ownerId: 42 } },
      ])
      .mockResolvedValueOnce([{ campaign: { ownerId: 42 } }]);
    const { service, userRepo } = await setup({
      memberFindMany,
    });
    await service.listVisibleTo(7);
    const ids = userRepo.findMany.mock.calls.at(-1)![0]!.where.id.in;
    /* Set should collapse the three 42s. */
    expect(ids.filter((id: number) => id === 42)).toHaveLength(1);
  });

  it('does not select passwordHash', async () => {
    const { service, userRepo } = await setup();
    await service.listVisibleTo(7);
    const select = userRepo.findMany.mock.calls.at(-1)![0]!.select;
    expect(select).not.toHaveProperty('passwordHash');
  });

  it('caller with zero campaigns still sees themselves', async () => {
    const { service, userRepo } = await setup();
    await service.listVisibleTo(7);
    const lastCall = userRepo.findMany.mock.calls.at(-1)![0]!;
    expect(lastCall.where.id.in).toEqual([7]);
  });
});
