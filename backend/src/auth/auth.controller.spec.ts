jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import { ConfigService } from '@nestjs/config';
import { Test } from '@nestjs/testing';
import type { Response } from 'express';
import { AuthController } from './auth.controller';
import { AuthService } from './auth.service';
import type { AuthUser } from './auth.service';

class FakeAuthService {
  register = jest.fn(
    async (input: { email: string; password: string; name?: string }): Promise<AuthUser> => ({
      id: 1,
      email: input.email,
      name: input.name ?? null,
    }),
  );
  validate = jest.fn(
    async (email: string, _pw: string): Promise<AuthUser> => ({
      id: 2,
      email,
      name: null,
    }),
  );
  signToken = jest.fn((user: AuthUser) => `tok.${user.id}`);
  findById = jest.fn();
}

class FakeConfigService {
  private values: Record<string, string | undefined> = {};
  set(key: string, value: string | undefined): void {
    this.values[key] = value;
  }
  get<T>(key: string): T | undefined {
    return this.values[key] as T | undefined;
  }
}

function fakeResponse(): Response {
  return {
    cookie: jest.fn(),
    clearCookie: jest.fn(),
  } as unknown as Response;
}

async function buildController(): Promise<{
  controller: AuthController;
  auth: FakeAuthService;
  config: FakeConfigService;
}> {
  const auth = new FakeAuthService();
  const config = new FakeConfigService();
  const moduleRef = await Test.createTestingModule({
    controllers: [AuthController],
    providers: [
      { provide: AuthService, useValue: auth },
      { provide: ConfigService, useValue: config },
    ],
  }).compile();
  return { controller: moduleRef.get(AuthController), auth, config };
}

describe('AuthController.register', () => {
  it('registers the user, signs a token, and sets the session cookie', async () => {
    const { controller, auth } = await buildController();
    const res = fakeResponse();
    const user = await controller.register(
      { email: 'a@b.com', password: 'pw', name: 'Alice' },
      res,
    );
    expect(user.email).toBe('a@b.com');
    expect(auth.register).toHaveBeenCalledWith({
      email: 'a@b.com',
      password: 'pw',
      name: 'Alice',
    });
    expect(auth.signToken).toHaveBeenCalledWith(user);
    expect(res.cookie).toHaveBeenCalledWith(
      't20_session',
      'tok.1',
      expect.objectContaining({ httpOnly: true, sameSite: 'lax' }),
    );
  });

  it('uses COOKIE_NAME from config when set', async () => {
    const { controller, config } = await buildController();
    config.set('COOKIE_NAME', 'custom_cookie');
    const res = fakeResponse();
    await controller.register({ email: 'a@b.com', password: 'pw' }, res);
    expect(res.cookie).toHaveBeenCalledWith(
      'custom_cookie',
      expect.any(String),
      expect.any(Object),
    );
  });

  it('sets secure=true only in production', async () => {
    const { controller, config } = await buildController();
    config.set('NODE_ENV', 'production');
    const res = fakeResponse();
    await controller.register({ email: 'a@b.com', password: 'pw' }, res);
    const opts = (res.cookie as jest.Mock).mock.calls[0]![2];
    expect(opts.secure).toBe(true);
  });

  it('sets secure=false outside production', async () => {
    const { controller, config } = await buildController();
    config.set('NODE_ENV', 'development');
    const res = fakeResponse();
    await controller.register({ email: 'a@b.com', password: 'pw' }, res);
    const opts = (res.cookie as jest.Mock).mock.calls[0]![2];
    expect(opts.secure).toBe(false);
  });
});

describe('AuthController.login', () => {
  it('validates credentials and sets the session cookie', async () => {
    const { controller, auth } = await buildController();
    const res = fakeResponse();
    const user = await controller.login(
      { email: 'a@b.com', password: 'pw' },
      res,
    );
    expect(auth.validate).toHaveBeenCalledWith('a@b.com', 'pw');
    expect(user.email).toBe('a@b.com');
    expect(res.cookie).toHaveBeenCalledWith(
      't20_session',
      'tok.2',
      expect.any(Object),
    );
  });
});

describe('AuthController.logout', () => {
  it('clears the session cookie with matching options', async () => {
    const { controller } = await buildController();
    const res = fakeResponse();
    controller.logout(res);
    expect(res.clearCookie).toHaveBeenCalledWith(
      't20_session',
      expect.objectContaining({ maxAge: 0, httpOnly: true }),
    );
  });
});

describe('AuthController.me', () => {
  it('returns the current user verbatim', async () => {
    const { controller } = await buildController();
    const user: AuthUser = { id: 5, email: 'me@b.com', name: 'Me' };
    expect(controller.me(user)).toBe(user);
  });
});
