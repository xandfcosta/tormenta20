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

/* Same string token SessionsService uses. Kept local so the spec
 * doesn't import the real module and drag Prisma into the test env. */
const SESSION_STATE_TOKEN = 'SESSION_STATE_SERVICE';

async function setup(over?: {
  findMany?: jest.Mock;
  findUnique?: jest.Mock;
  create?: jest.Mock;
  update?: jest.Mock;
  delete?: jest.Mock;
  campaignsFindOne?: jest.Mock;
  campaignsResolveAccess?: jest.Mock;
  characterFindUnique?: jest.Mock;
  characterUpdate?: jest.Mock;
  state?: unknown;
}) {
  const prisma = {
    session: {
      findMany: over?.findMany ?? jest.fn(),
      findUnique: over?.findUnique ?? jest.fn(),
      create: over?.create ?? jest.fn(),
      update: over?.update ?? jest.fn(),
      delete: over?.delete ?? jest.fn(),
    },
    /* P3: batch commit path reads/writes Character rows on Session.end
     * when WS_VITALS_WRITETHROUGH=1. Defaults resolve to shapes the
     * commit loop tolerates (no matching row = skip). */
    character: {
      findUnique: over?.characterFindUnique ?? jest.fn().mockResolvedValue(null),
      update: over?.characterUpdate ?? jest.fn().mockResolvedValue({}),
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
  const providers: Parameters<
    typeof Test.createTestingModule
  >[0]['providers'] = [
    SessionsService,
    { provide: PrismaService, useValue: prisma },
    { provide: CampaignsService, useValue: campaigns },
  ];
  if (over?.state !== undefined) {
    providers.push({ provide: SESSION_STATE_TOKEN, useValue: over.state });
  }
  const moduleRef = await Test.createTestingModule({
    providers,
  }).compile();
  return { service: moduleRef.get(SessionsService), prisma, campaigns };
}

describe('SessionsService.listForCaller — member-aware read', () => {
  it('returns sessions for a player member (via resolveAccess, not owner check)', async () => {
    const findMany = jest.fn().mockResolvedValue([{ id: 1 }]);
    const campaignsResolveAccess = jest
      .fn()
      .mockResolvedValue({ campaign: { id: 1, ownerId: 999 }, role: 'player' });
    const { service } = await setup({ findMany, campaignsResolveAccess });
    await expect(service.listForCaller(5, 1)).resolves.toEqual([{ id: 1 }]);
    expect(campaignsResolveAccess).toHaveBeenCalledWith(5, 1);
  });

  it('propagates Forbidden when caller is neither GM nor member', async () => {
    const campaignsResolveAccess = jest
      .fn()
      .mockRejectedValue(new ForbiddenException());
    const { service } = await setup({ campaignsResolveAccess });
    await expect(service.listForCaller(9, 1)).rejects.toBeInstanceOf(
      ForbiddenException,
    );
  });
});

describe('SessionsService.findOneForCaller — member-aware read', () => {
  it('returns { session, role } for a player member', async () => {
    const session = { id: 5, campaignId: 1 };
    const findUnique = jest.fn().mockResolvedValue(session);
    const campaignsResolveAccess = jest
      .fn()
      .mockResolvedValue({ campaign: { id: 1, ownerId: 999 }, role: 'player' });
    const { service } = await setup({ findUnique, campaignsResolveAccess });
    await expect(service.findOneForCaller(5, 1, 5)).resolves.toEqual({
      session,
      role: 'player',
    });
  });

  it('throws NotFound when the session belongs to another campaign', async () => {
    const findUnique = jest
      .fn()
      .mockResolvedValue({ id: 5, campaignId: 2 });
    const { service } = await setup({ findUnique });
    await expect(service.findOneForCaller(1, 1, 5)).rejects.toBeInstanceOf(
      NotFoundException,
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

  it('reopens an ended session (P4) — status ended → active, clears endedAt', async () => {
    /* Pre-P4 this rejected — GMs pause mid-combat and want to resume
     * without minting a new session (which would break initiative
     * continuity). Post-P4: reopen restores the session, clears
     * endedAt so the UI can show "ongoing", and preserves the runtime
     * tracker via P1a/P1b persistence. */
    const findUnique = jest
      .fn()
      .mockResolvedValue(makeSession({ status: 'ended' }));
    const update = jest
      .fn()
      .mockResolvedValue(makeSession({ status: 'active', endedAt: null }));
    const { service } = await setup({ findUnique, update });
    await service.start(1, 1, 1);
    expect(update).toHaveBeenCalledWith({
      where: { id: 1 },
      data: { status: 'active', endedAt: null },
    });
  });
});

describe('SessionsService.clearTracker', () => {
  it('resets the tracker + persists the empty state without touching lifecycle', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeSession({ status: 'active' }));
    const stateResetInitiative = jest.fn();
    const statePersist = jest.fn().mockResolvedValue(undefined);
    const state = {
      resetInitiative: stateResetInitiative,
      persist: statePersist,
    } as unknown as import('../realtime/session-state.service').SessionStateService;
    const { service } = await setup({ findUnique, state });
    await expect(service.clearTracker(1, 1, 1)).resolves.toEqual({ id: 1 });
    expect(stateResetInitiative).toHaveBeenCalledWith(1);
    expect(statePersist).toHaveBeenCalledWith(1);
  });

  it('no-ops when SessionStateService is not wired (still returns { id })', async () => {
    const findUnique = jest.fn().mockResolvedValue(makeSession({ status: 'planned' }));
    const { service } = await setup({ findUnique });
    await expect(service.clearTracker(1, 1, 1)).resolves.toEqual({ id: 1 });
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

  describe('P3 batch commit — WS_VITALS_WRITETHROUGH gated', () => {
    const OLD_ENV = process.env.WS_VITALS_WRITETHROUGH;
    afterEach(() => {
      if (OLD_ENV === undefined) delete process.env.WS_VITALS_WRITETHROUGH;
      else process.env.WS_VITALS_WRITETHROUGH = OLD_ENV;
    });

    function stateWithEntries(entries: unknown[]) {
      return {
        resetInitiative: jest.fn(),
        persist: jest.fn().mockResolvedValue(undefined),
        getState: () => ({
          initiative: entries,
          round: 1,
          turnIndex: 0,
        }),
      } as unknown;
    }

    it('when flag off: skips character write-through (no-op)', async () => {
      delete process.env.WS_VITALS_WRITETHROUGH;
      const findUnique = jest
        .fn()
        .mockResolvedValue(makeSession({ status: 'active' }));
      const update = jest.fn().mockResolvedValue({});
      const characterUpdate = jest.fn().mockResolvedValue({});
      const state = stateWithEntries([
        {
          id: 'x',
          label: 'PC',
          initiative: 12,
          type: 'character',
          characterId: 10,
          hpCurrent: 5,
          hpMax: 20,
        },
      ]);
      const { service } = await setup({
        findUnique,
        update,
        characterUpdate,
        state,
      });
      await service.end(1, 1, 1);
      expect(characterUpdate).not.toHaveBeenCalled();
    });

    it('when flag on: writes hpCurrent back to Character, clamped to fresh hpMax', async () => {
      process.env.WS_VITALS_WRITETHROUGH = '1';
      const findUnique = jest
        .fn()
        .mockResolvedValue(makeSession({ status: 'active' }));
      const update = jest.fn().mockResolvedValue({});
      const characterFindUnique = jest
        .fn()
        .mockResolvedValue({ hpMax: 15, mpMax: 5 });
      const characterUpdate = jest.fn().mockResolvedValue({});
      const state = stateWithEntries([
        {
          id: 'x',
          label: 'PC',
          initiative: 12,
          type: 'character',
          characterId: 10,
          /* Entry cache says hpMax 30 but fresh DB says 15 — clamp
           * source-of-truth. */
          hpMax: 30,
          hpCurrent: 25,
        },
      ]);
      const { service } = await setup({
        findUnique,
        update,
        characterFindUnique,
        characterUpdate,
        state,
      });
      await service.end(1, 1, 1);
      expect(characterUpdate).toHaveBeenCalledWith({
        where: { id: 10 },
        data: { hpCurrent: 15 },
      });
    });

    it('when flag on: skips entries without characterId', async () => {
      process.env.WS_VITALS_WRITETHROUGH = '1';
      const findUnique = jest
        .fn()
        .mockResolvedValue(makeSession({ status: 'active' }));
      const update = jest.fn().mockResolvedValue({});
      const characterUpdate = jest.fn().mockResolvedValue({});
      const state = stateWithEntries([
        {
          id: 'y',
          label: 'Goblin',
          initiative: 8,
          type: 'npc',
          hpCurrent: 3,
          hpMax: 10,
        },
      ]);
      const { service } = await setup({
        findUnique,
        update,
        characterUpdate,
        state,
      });
      await service.end(1, 1, 1);
      expect(characterUpdate).not.toHaveBeenCalled();
    });

    it('when flag on: individual failure is logged + skipped, rest of batch continues', async () => {
      process.env.WS_VITALS_WRITETHROUGH = '1';
      const findUnique = jest
        .fn()
        .mockResolvedValue(makeSession({ status: 'active' }));
      const update = jest.fn().mockResolvedValue({});
      const characterFindUnique = jest
        .fn()
        .mockRejectedValueOnce(new Error('DB blip'))
        .mockResolvedValueOnce({ hpMax: 20, mpMax: 0 });
      const characterUpdate = jest.fn().mockResolvedValue({});
      const state = stateWithEntries([
        {
          id: 'a',
          label: 'PC1',
          initiative: 12,
          type: 'character',
          characterId: 10,
          hpMax: 20,
          hpCurrent: 4,
        },
        {
          id: 'b',
          label: 'PC2',
          initiative: 8,
          type: 'character',
          characterId: 11,
          hpMax: 20,
          hpCurrent: 8,
        },
      ]);
      const { service } = await setup({
        findUnique,
        update,
        characterFindUnique,
        characterUpdate,
        state,
      });
      await service.end(1, 1, 1);
      /* First entry failed at findUnique → no update; second succeeded. */
      expect(characterUpdate).toHaveBeenCalledTimes(1);
      expect(characterUpdate).toHaveBeenCalledWith({
        where: { id: 11 },
        data: { hpCurrent: 8 },
      });
    });
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
