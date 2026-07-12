// Prisma client requires generated code that isn't available in the unit-
// test sandbox. Stub the PrismaService module before AuthService loads.
jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import { ConflictException, UnauthorizedException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { Test } from '@nestjs/testing';
import * as bcrypt from 'bcrypt';
import { AuthService } from './auth.service';
import { PrismaService } from '../prisma/prisma.service';

/**
 * Pure-unit specs for AuthService. PrismaService and JwtService are
 * stubbed; bcrypt is real (its hashing is cheap enough at low rounds and
 * its compare logic is what we want to verify against a real hash).
 */
type FakeUserRow = {
  id: number;
  email: string;
  name: string | null;
  passwordHash: string;
};

class FakePrismaUserRepo {
  rows: FakeUserRow[] = [];
  nextId = 1;

  findUnique = jest.fn(
    ({
      where,
    }: {
      where: { email?: string; id?: number };
    }) => {
      if (where.email !== undefined) {
        const row = this.rows.find((r) => r.email === where.email);
        return Promise.resolve(row ?? null);
      }
      if (where.id !== undefined) {
        const row = this.rows.find((r) => r.id === where.id);
        if (!row) return Promise.resolve(null);
        return Promise.resolve({ id: row.id, email: row.email, name: row.name });
      }
      return Promise.resolve(null);
    },
  );

  create = jest.fn(
    ({
      data,
    }: {
      data: { email: string; passwordHash: string; name: string | null };
    }) => {
      const row: FakeUserRow = {
        id: this.nextId++,
        email: data.email,
        name: data.name,
        passwordHash: data.passwordHash,
      };
      this.rows.push(row);
      return Promise.resolve({ id: row.id, email: row.email, name: row.name });
    },
  );
}

class FakeJwtService {
  sign = jest.fn(
    (payload: { sub: number; email: string }) =>
      `signed.${payload.sub}.${payload.email}`,
  );
}

async function buildService(): Promise<{
  service: AuthService;
  userRepo: FakePrismaUserRepo;
  jwt: FakeJwtService;
}> {
  const userRepo = new FakePrismaUserRepo();
  const jwt = new FakeJwtService();
  const moduleRef = await Test.createTestingModule({
    providers: [
      AuthService,
      { provide: PrismaService, useValue: { user: userRepo } },
      { provide: JwtService, useValue: jwt },
    ],
  }).compile();
  return {
    service: moduleRef.get(AuthService),
    userRepo,
    jwt,
  };
}

describe('AuthService.register', () => {
  it('creates a new user when email is free', async () => {
    const { service, userRepo } = await buildService();
    const user = await service.register({
      email: 'a@b.com',
      password: 'hunter2',
      name: 'Alice',
    });
    expect(user).toEqual({ id: 1, email: 'a@b.com', name: 'Alice' });
    expect(userRepo.rows).toHaveLength(1);
  });

  it('hashes the password (does not store cleartext)', async () => {
    const { service, userRepo } = await buildService();
    await service.register({ email: 'a@b.com', password: 'hunter2' });
    const stored = userRepo.rows[0]!;
    expect(stored.passwordHash).not.toBe('hunter2');
    const matches = await bcrypt.compare('hunter2', stored.passwordHash);
    expect(matches).toBe(true);
  });

  it('throws ConflictException when email is taken', async () => {
    const { service } = await buildService();
    await service.register({ email: 'dup@b.com', password: 'pw' });
    await expect(
      service.register({ email: 'dup@b.com', password: 'pw' }),
    ).rejects.toBeInstanceOf(ConflictException);
  });

  it('stores name as null when not provided', async () => {
    const { service, userRepo } = await buildService();
    await service.register({ email: 'n@b.com', password: 'pw' });
    expect(userRepo.rows[0]!.name).toBeNull();
  });

  it('maps a raced P2002 insert to ConflictException (not a 500)', async () => {
    const { service, userRepo } = await buildService();
    // Pre-check finds nobody (findUnique → null), but the insert loses a
    // race and the unique index rejects with P2002.
    userRepo.create.mockRejectedValueOnce({ code: 'P2002' });
    await expect(
      service.register({ email: 'race@b.com', password: 'pw' }),
    ).rejects.toBeInstanceOf(ConflictException);
  });
});

describe('AuthService.validate', () => {
  it('returns AuthUser when password matches', async () => {
    const { service } = await buildService();
    await service.register({
      email: 'v@b.com',
      password: 'correct',
      name: 'V',
    });
    const user = await service.validate('v@b.com', 'correct');
    expect(user).toEqual({ id: 1, email: 'v@b.com', name: 'V' });
  });

  it('throws UnauthorizedException for unknown email', async () => {
    const { service } = await buildService();
    await expect(
      service.validate('ghost@b.com', 'whatever'),
    ).rejects.toBeInstanceOf(UnauthorizedException);
  });

  it('throws UnauthorizedException for wrong password', async () => {
    const { service } = await buildService();
    await service.register({ email: 'v@b.com', password: 'correct' });
    await expect(
      service.validate('v@b.com', 'wrong'),
    ).rejects.toBeInstanceOf(UnauthorizedException);
  });

  it('uses identical error message for both failure modes (no enumeration)', async () => {
    const { service } = await buildService();
    await service.register({ email: 'v@b.com', password: 'correct' });

    const unknownErr = await service
      .validate('ghost@b.com', 'pw')
      .catch((e) => e as Error);
    const wrongErr = await service
      .validate('v@b.com', 'wrong')
      .catch((e) => e as Error);
    expect(unknownErr.message).toBe(wrongErr.message);
  });
});

describe('AuthService.signToken', () => {
  it('signs a JwtPayload with sub + email', async () => {
    const { service, jwt } = await buildService();
    const token = service.signToken({ id: 42, email: 'a@b.com', name: null });
    expect(token).toBe('signed.42.a@b.com');
    expect(jwt.sign).toHaveBeenCalledWith({ sub: 42, email: 'a@b.com' });
  });
});

describe('AuthService.findById', () => {
  it('returns user without passwordHash field', async () => {
    const { service } = await buildService();
    await service.register({
      email: 'x@b.com',
      password: 'pw',
      name: 'X',
    });
    const user = await service.findById(1);
    expect(user).toEqual({ id: 1, email: 'x@b.com', name: 'X' });
  });

  it('returns null for unknown id', async () => {
    const { service } = await buildService();
    expect(await service.findById(999)).toBeNull();
  });
});
