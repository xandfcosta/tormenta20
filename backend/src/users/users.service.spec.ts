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

describe('UsersService.list', () => {
  let service: UsersService;
  let userRepo: FakeUserRepo;

  beforeEach(async () => {
    userRepo = new FakeUserRepo();
    const moduleRef = await Test.createTestingModule({
      providers: [
        UsersService,
        { provide: PrismaService, useValue: { user: userRepo } },
      ],
    }).compile();
    service = moduleRef.get(UsersService);
  });

  it('queries user.findMany with the expected select + order', async () => {
    await service.list();
    expect(userRepo.findMany).toHaveBeenCalledWith({
      orderBy: { createdAt: 'desc' },
      select: { id: true, email: true, name: true, createdAt: true },
    });
  });

  it('passes the Prisma result through', async () => {
    const rows = await service.list();
    expect(rows).toHaveLength(2);
    expect(rows[0]!.email).toBe('a@b.com');
  });

  it('does not select passwordHash (caller cannot leak it)', async () => {
    await service.list();
    const select = userRepo.findMany.mock.calls[0]![0]!.select;
    expect(select).not.toHaveProperty('passwordHash');
  });
});
