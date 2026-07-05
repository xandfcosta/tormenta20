jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import { JwtService } from '@nestjs/jwt';
import { WsException } from '@nestjs/websockets';
import { Test } from '@nestjs/testing';
import type { Socket } from 'socket.io';
import { RealtimeGateway, sessionRoom } from './realtime.gateway';
import { PrismaService } from '../prisma/prisma.service';
import { SessionsService } from '../sessions/sessions.service';

/**
 * Fase C1 gateway smoke tests. Cover:
 *   - room name helper
 *   - handleConnection: valid token → data.user set
 *   - handleConnection: bad token → disconnect + unauthorized emit
 *   - joinSession: delegates ownership to SessionsService.findOne
 *   - joinSession: SessionsService throw → WsException
 *   - joinSession/leaveSession: invalid body shape → WsException
 */

const SECRET = 'test-secret-just-for-jest';

async function setup(over?: {
  sessionsFindOne?: jest.Mock;
  userFindUnique?: jest.Mock;
}) {
  const prisma = {
    user: {
      findUnique:
        over?.userFindUnique ??
        jest
          .fn()
          .mockResolvedValue({ id: 7, email: 'gm@example.com', name: 'GM' }),
    },
  };
  const sessions = {
    findOne: over?.sessionsFindOne ?? jest.fn().mockResolvedValue({ id: 1 }),
  };
  const module = await Test.createTestingModule({
    providers: [
      RealtimeGateway,
      { provide: JwtService, useValue: new JwtService({ secret: SECRET }) },
      { provide: PrismaService, useValue: prisma },
      { provide: SessionsService, useValue: sessions },
    ],
  }).compile();
  return {
    gateway: module.get(RealtimeGateway),
    prisma,
    sessions,
    jwt: module.get(JwtService),
  };
}

function fakeSocket(over: Partial<Socket> & { auth?: { token?: string } } = {}) {
  const emit = jest.fn();
  const disconnect = jest.fn();
  const join = jest.fn().mockResolvedValue(undefined);
  const leave = jest.fn().mockResolvedValue(undefined);
  return {
    id: over.id ?? 'socket-1',
    handshake: {
      auth: over.auth ?? {},
      headers: {},
    },
    data: {},
    emit,
    disconnect,
    join,
    leave,
  } as unknown as Socket & { data: { user?: unknown } };
}

describe('sessionRoom', () => {
  it('builds a per-session namespace', () => {
    expect(sessionRoom(42)).toBe('session:42');
  });
});

describe('RealtimeGateway.handleConnection', () => {
  it('accepts a socket carrying a valid JWT and stores the user', async () => {
    const { gateway, jwt } = await setup();
    const token = jwt.sign({ sub: 7, email: 'gm@example.com' });
    const socket = fakeSocket({ auth: { token } });
    await gateway.handleConnection(socket);
    expect((socket as unknown as { data: { user?: { id: number } } }).data.user?.id).toBe(7);
    expect(socket.disconnect).not.toHaveBeenCalled();
  });

  it('disconnects sockets that fail JWT verification', async () => {
    const { gateway } = await setup();
    const socket = fakeSocket({ auth: { token: 'obviously-broken' } });
    await gateway.handleConnection(socket);
    expect(socket.emit).toHaveBeenCalledWith(
      'unauthorized',
      expect.any(Object),
    );
    expect(socket.disconnect).toHaveBeenCalledWith(true);
  });

  it('disconnects when the token points at a removed user', async () => {
    const { gateway, jwt } = await setup({
      userFindUnique: jest.fn().mockResolvedValue(null),
    });
    const token = jwt.sign({ sub: 99, email: 'gone@example.com' });
    const socket = fakeSocket({ auth: { token } });
    await gateway.handleConnection(socket);
    expect(socket.disconnect).toHaveBeenCalled();
  });
});

describe('RealtimeGateway.joinSession', () => {
  it('validates campaignId + sessionId as integers', async () => {
    const { gateway } = await setup();
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.joinSession(
        socket as unknown as Parameters<typeof gateway.joinSession>[0],
        {} as unknown as Parameters<typeof gateway.joinSession>[1],
      ),
    ).rejects.toBeInstanceOf(WsException);
  });

  it('joins the session room when SessionsService accepts the caller', async () => {
    const { gateway, sessions } = await setup();
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.joinSession(
      socket as unknown as Parameters<typeof gateway.joinSession>[0],
      { campaignId: 1, sessionId: 5 },
    );
    expect(sessions.findOne).toHaveBeenCalledWith(7, 1, 5);
    expect(socket.join).toHaveBeenCalledWith('session:5');
    expect(result).toEqual({ joined: 'session:5' });
  });

  it('rejects when SessionsService throws (Forbidden / NotFound)', async () => {
    const { gateway } = await setup({
      sessionsFindOne: jest.fn().mockRejectedValue(new Error('nope')),
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.joinSession(
        socket as unknown as Parameters<typeof gateway.joinSession>[0],
        { campaignId: 1, sessionId: 5 },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });
});

describe('RealtimeGateway.leaveSession', () => {
  it('validates body shape', async () => {
    const { gateway } = await setup();
    const socket = fakeSocket();
    await expect(
      gateway.leaveSession(
        socket as unknown as Parameters<typeof gateway.leaveSession>[0],
        {} as unknown as Parameters<typeof gateway.leaveSession>[1],
      ),
    ).rejects.toBeInstanceOf(WsException);
  });

  it('leaves the room', async () => {
    const { gateway } = await setup();
    const socket = fakeSocket();
    const result = await gateway.leaveSession(
      socket as unknown as Parameters<typeof gateway.leaveSession>[0],
      { sessionId: 5 },
    );
    expect(socket.leave).toHaveBeenCalledWith('session:5');
    expect(result).toEqual({ left: 'session:5' });
  });
});
