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
import { CharactersService } from '../characters/characters.service';
import { AuthService } from '../auth/auth.service';
import { SessionStateService } from './session-state.service';

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
  sessionsFindOneForCaller?: jest.Mock;
  userFindUnique?: jest.Mock;
  characterFindUnique?: jest.Mock;
  campaignFindUnique?: jest.Mock;
  campaignMemberFindUnique?: jest.Mock;
  campaignMemberFindMany?: jest.Mock;
  activeEffectDeleteMany?: jest.Mock;
  characterUpdate?: jest.Mock;
  charactersEndScene?: jest.Mock;
  charactersEndDay?: jest.Mock;
  charactersRestVitals?: jest.Mock;
  charactersAssertOwner?: jest.Mock;
  authFindById?: jest.Mock;
}) {
  const prisma = {
    user: {
      findUnique:
        over?.userFindUnique ??
        jest
          .fn()
          .mockResolvedValue({ id: 7, email: 'gm@example.com', name: 'GM' }),
    },
    session: {
      /* P1b: SessionStateService.persist writes here after every
       * mutation. Resolves quietly so gateway specs stay green. */
      findUnique: jest.fn().mockResolvedValue(null),
      update: jest.fn().mockResolvedValue({}),
    },
    /* P2: initiative-add with characterId hits these three tables. */
    character: {
      findUnique: over?.characterFindUnique ?? jest.fn(),
      update: over?.characterUpdate ?? jest.fn().mockResolvedValue({}),
    },
    campaign: {
      findUnique: over?.campaignFindUnique ?? jest.fn(),
    },
    campaignMember: {
      findUnique: over?.campaignMemberFindUnique ?? jest.fn(),
      findMany: over?.campaignMemberFindMany ?? jest.fn().mockResolvedValue([]),
    },
    activeEffect: {
      deleteMany:
        over?.activeEffectDeleteMany ??
        jest.fn().mockResolvedValue({ count: 0 }),
    },
  };
  const sessions = {
    findOne: over?.sessionsFindOne ?? jest.fn().mockResolvedValue({ id: 1 }),
    findOneForCaller:
      over?.sessionsFindOneForCaller ??
      jest.fn().mockResolvedValue({ session: { id: 1 }, role: 'gm' }),
  };
  const characters = {
    endScene: over?.charactersEndScene ?? jest.fn().mockResolvedValue({}),
    endDay: over?.charactersEndDay ?? jest.fn().mockResolvedValue({}),
    restVitals:
      over?.charactersRestVitals ??
      jest.fn().mockResolvedValue({ hpCurrent: 0, mpCurrent: 0 }),
    assertOwner:
      over?.charactersAssertOwner ?? jest.fn().mockResolvedValue(undefined),
  };
  const auth = {
    findById:
      over?.authFindById ??
      jest
        .fn()
        .mockResolvedValue({ id: 7, email: 'gm@example.com', name: 'GM' }),
  };
  const module = await Test.createTestingModule({
    providers: [
      RealtimeGateway,
      SessionStateService,
      { provide: JwtService, useValue: new JwtService({ secret: SECRET }) },
      { provide: PrismaService, useValue: prisma },
      { provide: SessionsService, useValue: sessions },
      { provide: CharactersService, useValue: characters },
      { provide: AuthService, useValue: auth },
    ],
  }).compile();
  const gateway = module.get(RealtimeGateway);
  /* Inject a fake Server so broadcast handlers can be probed. */
  const emit = jest.fn();
  const roomServer = { emit } as unknown as { emit: jest.Mock };
  const to = jest.fn().mockReturnValue(roomServer);
  gateway.server = { to } as unknown as typeof gateway.server;
  return {
    gateway,
    prisma,
    sessions,
    characters,
    jwt: module.get(JwtService),
    state: module.get(SessionStateService),
    to,
    emit,
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
      authFindById: jest.fn().mockResolvedValue(null),
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
    expect(sessions.findOneForCaller).toHaveBeenCalledWith(7, 1, 5);
    expect(socket.join).toHaveBeenCalledWith('session:5');
    expect(result).toEqual({ joined: 'session:5' });
  });

  it('rejects when SessionsService throws (Forbidden / NotFound)', async () => {
    const { gateway } = await setup({
      sessionsFindOneForCaller: jest.fn().mockRejectedValue(new Error('nope')),
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

describe('RealtimeGateway.initiativeAdd', () => {
  it('adds an entry, broadcasts session-state, returns the state', async () => {
    const { gateway, to, emit } = await setup();
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const state = await gateway.initiativeAdd(
      socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entry: { label: 'Goblin', initiative: 12, type: 'npc' },
      },
    );
    expect(state.initiative).toHaveLength(1);
    expect(to).toHaveBeenCalledWith('session:5');
    expect(emit).toHaveBeenCalledWith('session-state', state);
  });

  it('rejects when entry.label is missing (NPC path)', async () => {
    const { gateway } = await setup();
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.initiativeAdd(
        socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
        {
          campaignId: 1,
          sessionId: 5,
          entry: { initiative: 12, type: 'npc' } as unknown as {
            label: string;
            initiative: number;
            type: 'npc';
          },
        },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });

  it('P2: resolves characterId — pre-populates label + hp/mp from Character DB', async () => {
    const { gateway, state } = await setup({
      characterFindUnique: jest.fn().mockResolvedValue({
        id: 10,
        name: 'Alric',
        ownerId: 7,
        hpCurrent: 22,
        hpMax: 30,
        mpCurrent: 5,
        mpMax: 8,
      }),
      campaignFindUnique: jest.fn().mockResolvedValue({ id: 1, ownerId: 99 }),
      campaignMemberFindUnique: jest.fn().mockResolvedValue({ id: 55 }),
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await gateway.initiativeAdd(
      socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entry: { characterId: 10, initiative: 14 } as unknown as {
          label: string;
          initiative: number;
          type: 'character';
        },
      },
    );
    const entry = state.getState(5).initiative[0]!;
    expect(entry.label).toBe('Alric');
    expect(entry.characterId).toBe(10);
    expect(entry.hpCurrent).toBe(22);
    expect(entry.hpMax).toBe(30);
    expect(entry.type).toBe('character');
  });

  it('P2: allows GM (campaign owner) to add someone else\'s character', async () => {
    const { gateway, state } = await setup({
      characterFindUnique: jest.fn().mockResolvedValue({
        id: 10,
        name: 'PlayerPC',
        ownerId: 42, /* different from caller */
        hpCurrent: 10,
        hpMax: 10,
        mpCurrent: 0,
        mpMax: 0,
      }),
      campaignFindUnique: jest.fn().mockResolvedValue({ id: 1, ownerId: 7 }),
      campaignMemberFindUnique: jest.fn().mockResolvedValue({ id: 55 }),
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await gateway.initiativeAdd(
      socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entry: { characterId: 10, initiative: 8 } as unknown as {
          label: string;
          initiative: number;
          type: 'character';
        },
      },
    );
    expect(state.getState(5).initiative[0]!.characterId).toBe(10);
  });

  it('P2: rejects when caller is neither char owner nor GM', async () => {
    const { gateway } = await setup({
      characterFindUnique: jest.fn().mockResolvedValue({
        id: 10,
        name: 'PC',
        ownerId: 42,
        hpCurrent: 10,
        hpMax: 10,
        mpCurrent: 0,
        mpMax: 0,
      }),
      campaignFindUnique: jest.fn().mockResolvedValue({ id: 1, ownerId: 99 }),
      campaignMemberFindUnique: jest.fn().mockResolvedValue({ id: 55 }),
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.initiativeAdd(
        socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
        {
          campaignId: 1,
          sessionId: 5,
          entry: { characterId: 10, initiative: 8 } as unknown as {
            label: string;
            initiative: number;
            type: 'character';
          },
        },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });

  it('P2: rejects when character is not a member of the campaign', async () => {
    const { gateway } = await setup({
      characterFindUnique: jest.fn().mockResolvedValue({
        id: 10,
        name: 'Stranger',
        ownerId: 7,
        hpCurrent: 5,
        hpMax: 5,
        mpCurrent: 0,
        mpMax: 0,
      }),
      campaignFindUnique: jest.fn().mockResolvedValue({ id: 1, ownerId: 7 }),
      campaignMemberFindUnique: jest.fn().mockResolvedValue(null),
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.initiativeAdd(
        socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
        {
          campaignId: 1,
          sessionId: 5,
          entry: { characterId: 10, initiative: 8 } as unknown as {
            label: string;
            initiative: number;
            type: 'character';
          },
        },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });
});

describe('RealtimeGateway.vitalsPatch', () => {
  it('applies clamped absolute values + broadcasts', async () => {
    const { gateway, state, to, emit } = await setup();
    const s = state.addEntry(5, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      hpCurrent: 10,
      hpMax: 20,
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.vitalsPatch(
      socket as unknown as Parameters<typeof gateway.vitalsPatch>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entryId: s.initiative[0]!.id,
        patch: { hpCurrent: 999 },
      },
    );
    expect(result.initiative[0]?.hpCurrent).toBe(20);
    expect(to).toHaveBeenCalledWith('session:5');
    expect(emit).toHaveBeenCalledWith('session-state', result);
  });

  it('rejects missing entryId', async () => {
    const { gateway } = await setup();
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.vitalsPatch(
        socket as unknown as Parameters<typeof gateway.vitalsPatch>[0],
        {
          campaignId: 1,
          sessionId: 5,
          entryId: '',
          patch: { hpCurrent: 5 },
        },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });
});

describe('RealtimeGateway.vitalsDelta', () => {
  it('applies hpDelta and broadcasts', async () => {
    const { gateway, state } = await setup();
    const s = state.addEntry(5, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      hpCurrent: 20,
      hpMax: 30,
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.vitalsDelta(
      socket as unknown as Parameters<typeof gateway.vitalsDelta>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entryId: s.initiative[0]!.id,
        hpDelta: -8,
      },
    );
    expect(result.initiative[0]?.hpCurrent).toBe(12);
  });

  it('applies mpDelta', async () => {
    const { gateway, state } = await setup();
    const s = state.addEntry(5, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      mpCurrent: 4,
      mpMax: 10,
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.vitalsDelta(
      socket as unknown as Parameters<typeof gateway.vitalsDelta>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entryId: s.initiative[0]!.id,
        mpDelta: 5,
      },
    );
    expect(result.initiative[0]?.mpCurrent).toBe(9);
  });

  it('lets a player edit their own character vitals', async () => {
    const { gateway, state } = await setup({
      sessionsFindOneForCaller: jest
        .fn()
        .mockResolvedValue({ session: { id: 5 }, role: 'player' }),
      characterFindUnique: jest.fn().mockResolvedValue({ ownerId: 7 }),
    });
    const s = state.addEntry(5, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      characterId: 10,
      hpCurrent: 20,
      hpMax: 30,
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.vitalsDelta(
      socket as unknown as Parameters<typeof gateway.vitalsDelta>[0],
      { campaignId: 1, sessionId: 5, entryId: s.initiative[0]!.id, hpDelta: -5 },
    );
    expect(result.initiative[0]?.hpCurrent).toBe(15);
  });

  it('blocks a player from editing a combatant they do not own', async () => {
    const { gateway, state } = await setup({
      sessionsFindOneForCaller: jest
        .fn()
        .mockResolvedValue({ session: { id: 5 }, role: 'player' }),
      // CharactersService.assertOwner rejects → not the player's character.
      charactersAssertOwner: jest.fn().mockRejectedValue(new Error('nope')),
    });
    const s = state.addEntry(5, {
      label: 'Ally',
      initiative: 12,
      type: 'character',
      characterId: 11,
      hpCurrent: 10,
      hpMax: 20,
    });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.vitalsDelta(
        socket as unknown as Parameters<typeof gateway.vitalsDelta>[0],
        {
          campaignId: 1,
          sessionId: 5,
          entryId: s.initiative[0]!.id,
          hpDelta: -5,
        },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });

  it('blocks a player from editing NPC vitals (GM-only)', async () => {
    const { gateway, state } = await setup({
      sessionsFindOneForCaller: jest
        .fn()
        .mockResolvedValue({ session: { id: 5 }, role: 'player' }),
    });
    const s = state.addEntry(5, { label: 'Goblin', initiative: 9, type: 'npc' });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.vitalsDelta(
        socket as unknown as Parameters<typeof gateway.vitalsDelta>[0],
        {
          campaignId: 1,
          sessionId: 5,
          entryId: s.initiative[0]!.id,
          hpDelta: -5,
        },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });
});

describe('RealtimeGateway.initiativeNextTurn + reset', () => {
  it('rejects initiative-next-turn from a player (GM-only)', async () => {
    const { gateway, state } = await setup({
      sessionsFindOneForCaller: jest
        .fn()
        .mockResolvedValue({ session: { id: 5 }, role: 'player' }),
    });
    state.addEntry(5, { label: 'A', initiative: 20, type: 'npc' });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await expect(
      gateway.initiativeNextTurn(
        socket as unknown as Parameters<typeof gateway.initiativeNextTurn>[0],
        { campaignId: 1, sessionId: 5 },
      ),
    ).rejects.toBeInstanceOf(WsException);
  });

  it('advances turn and broadcasts', async () => {
    const { gateway, state, to } = await setup();
    state.addEntry(5, { label: 'A', initiative: 20, type: 'npc' });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.initiativeNextTurn(
      socket as unknown as Parameters<typeof gateway.initiativeNextTurn>[0],
      { campaignId: 1, sessionId: 5 },
    );
    expect(result.turnIndex).toBe(0);
    expect(result.round).toBe(1);
    expect(to).toHaveBeenCalledWith('session:5');
  });

  it('reset clears the tracker', async () => {
    const { gateway, state } = await setup();
    state.addEntry(5, { label: 'A', initiative: 20, type: 'npc' });
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    const result = await gateway.initiativeReset(
      socket as unknown as Parameters<typeof gateway.initiativeReset>[0],
      { campaignId: 1, sessionId: 5 },
    );
    expect(result.initiative).toEqual([]);
    expect(result.round).toBe(0);
    expect(result.turnIndex).toBe(-1);
  });
});

describe('RealtimeGateway persistence-warning broadcast', () => {
  it('broadcasts persistence-warning { dirty:true } when persist fails', async () => {
    const sessionUpdate = jest
      .fn()
      .mockRejectedValue(new Error('DB down'));
    /* setup uses the default userFindUnique; override prisma.session.update
     * by re-wiring after setup. */
    const { gateway, state, to, emit } = await setup();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ((gateway as any).state as unknown as { prisma: { session: { update: jest.Mock } } })
      .prisma.session.update = sessionUpdate;
    const socket = fakeSocket();
    (socket as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    await gateway.initiativeAdd(
      socket as unknown as Parameters<typeof gateway.initiativeAdd>[0],
      {
        campaignId: 1,
        sessionId: 5,
        entry: { label: 'Goblin', initiative: 12, type: 'npc' },
      },
    );
    /* Yield to the microtask queue so the .then on persist runs. */
    await new Promise((r) => setImmediate(r));
    expect(to).toHaveBeenCalledWith('session:5');
    expect(emit).toHaveBeenCalledWith('persistence-warning', {
      sessionId: 5,
      dirty: true,
    });
    expect(state.isDirty(5)).toBe(true);
  });
});

describe('RealtimeGateway presence', () => {
  const withUser = (id: string) => {
    const s = fakeSocket({ id });
    (s as unknown as { data: { user: unknown } }).data.user = {
      id: 7,
      name: 'GM',
      email: 'gm@x',
    };
    return s;
  };

  it('broadcasts presence with the joiner in the roster', async () => {
    const { gateway, emit } = await setup();
    const socket = withUser('sock-a');
    await gateway.joinSession(
      socket as unknown as Parameters<typeof gateway.joinSession>[0],
      { campaignId: 1, sessionId: 5 },
    );
    expect(emit).toHaveBeenCalledWith('presence', {
      sessionId: 5,
      users: [{ userId: 7, name: 'GM', role: 'gm' }],
    });
  });

  it('drops the user from the roster on disconnect', async () => {
    const { gateway, emit } = await setup();
    const socket = withUser('sock-a');
    await gateway.joinSession(
      socket as unknown as Parameters<typeof gateway.joinSession>[0],
      { campaignId: 1, sessionId: 5 },
    );
    emit.mockClear();
    gateway.handleDisconnect(
      socket as unknown as Parameters<typeof gateway.handleDisconnect>[0],
    );
    expect(emit).toHaveBeenCalledWith('presence', { sessionId: 5, users: [] });
  });

  it('dedupes multiple tabs of the same user into one roster entry', async () => {
    const { gateway, emit } = await setup();
    await gateway.joinSession(
      withUser('sock-a') as unknown as Parameters<typeof gateway.joinSession>[0],
      { campaignId: 1, sessionId: 5 },
    );
    await gateway.joinSession(
      withUser('sock-b') as unknown as Parameters<typeof gateway.joinSession>[0],
      { campaignId: 1, sessionId: 5 },
    );
    const last = emit.mock.calls.filter((c) => c[0] === 'presence').pop();
    expect(last?.[1]).toEqual({
      sessionId: 5,
      users: [{ userId: 7, name: 'GM', role: 'gm' }],
    });
  });
});

describe('RealtimeGateway.initiativePopulate', () => {
  const gmSocket = () => {
    const s = fakeSocket();
    (s as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    return s as unknown as Parameters<
      typeof RealtimeGateway.prototype.initiativePopulate
    >[0];
  };

  it('adds player characters to the tracker, skipping ones already present', async () => {
    const { gateway, state } = await setup({
      campaignMemberFindMany: jest.fn().mockResolvedValue([
        {
          character: {
            id: 10,
            name: 'Mira',
            hpCurrent: 20,
            hpMax: 30,
            mpCurrent: 5,
            mpMax: 10,
          },
        },
        {
          character: {
            id: 11,
            name: 'Kael',
            hpCurrent: 15,
            hpMax: 25,
            mpCurrent: 0,
            mpMax: 8,
          },
        },
      ]),
    });
    state.addEntry(5, {
      label: 'Mira',
      initiative: 3,
      type: 'character',
      characterId: 10,
    });
    const result = await gateway.initiativePopulate(gmSocket(), {
      campaignId: 1,
      sessionId: 5,
    });
    const ids = result.initiative.map((e) => e.characterId);
    expect(ids).toContain(11);
    expect(ids.filter((id) => id === 10)).toHaveLength(1);
  });

  it('rejects a non-GM', async () => {
    const { gateway } = await setup({
      sessionsFindOneForCaller: jest
        .fn()
        .mockResolvedValue({ session: { id: 5 }, role: 'player' }),
    });
    await expect(
      gateway.initiativePopulate(gmSocket(), { campaignId: 1, sessionId: 5 }),
    ).rejects.toBeInstanceOf(WsException);
  });
});

describe('RealtimeGateway.sessionRest', () => {
  const gmSocket = () => {
    const s = fakeSocket();
    (s as unknown as { data: { user: unknown } }).data.user = { id: 7 };
    return s as unknown as Parameters<
      typeof RealtimeGateway.prototype.sessionRest
    >[0];
  };

  it('scene rest delegates end-scene per member, no heal, broadcasts (GM)', async () => {
    const { gateway, emit, characters } = await setup({
      campaignMemberFindMany: jest
        .fn()
        .mockResolvedValue([{ characterId: 10 }, { characterId: 11 }]),
    });
    const result = await gateway.sessionRest(gmSocket(), {
      campaignId: 1,
      sessionId: 5,
      scope: 'scene',
    });
    expect(characters.endScene).toHaveBeenCalledWith(7, 10);
    expect(characters.endScene).toHaveBeenCalledWith(7, 11);
    expect(characters.restVitals).not.toHaveBeenCalled();
    expect(result).toEqual({ rested: 'scene', characters: 2, healed: 0 });
    expect(emit).toHaveBeenCalledWith('session-rest', {
      sessionId: 5,
      scope: 'scene',
      condition: undefined,
    });
  });

  it('day rest delegates end-day + restVitals per member', async () => {
    const { gateway, characters } = await setup({
      campaignMemberFindMany: jest.fn().mockResolvedValue([{ characterId: 10 }]),
      charactersRestVitals: jest
        .fn()
        .mockResolvedValue({ hpCurrent: 18, mpCurrent: 10 }),
    });
    const result = await gateway.sessionRest(gmSocket(), {
      campaignId: 1,
      sessionId: 5,
      scope: 'day',
      condition: 'confortavel',
    });
    expect(characters.endDay).toHaveBeenCalledWith(7, 10);
    expect(characters.restVitals).toHaveBeenCalledWith(7, 10, 'confortavel');
    expect(result).toMatchObject({ rested: 'day', healed: 1 });
  });

  it('day rest defaults the condition to normal when omitted', async () => {
    const { gateway, characters } = await setup({
      campaignMemberFindMany: jest.fn().mockResolvedValue([{ characterId: 10 }]),
    });
    await gateway.sessionRest(gmSocket(), {
      campaignId: 1,
      sessionId: 5,
      scope: 'day',
    });
    expect(characters.restVitals).toHaveBeenCalledWith(7, 10, 'normal');
  });

  it('rejects a non-GM', async () => {
    const { gateway } = await setup({
      sessionsFindOneForCaller: jest
        .fn()
        .mockResolvedValue({ session: { id: 5 }, role: 'player' }),
    });
    await expect(
      gateway.sessionRest(gmSocket(), {
        campaignId: 1,
        sessionId: 5,
        scope: 'scene',
      }),
    ).rejects.toBeInstanceOf(WsException);
  });
});
