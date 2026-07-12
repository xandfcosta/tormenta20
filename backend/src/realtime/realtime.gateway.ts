import { Logger } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import {
  ConnectedSocket,
  MessageBody,
  OnGatewayConnection,
  OnGatewayDisconnect,
  SubscribeMessage,
  WebSocketGateway,
  WebSocketServer,
  WsException,
} from '@nestjs/websockets';
import type { Server, Socket } from 'socket.io';
import { PrismaService } from '../prisma/prisma.service';
import { SessionsService } from '../sessions/sessions.service';
import { CharactersService } from '../characters/characters.service';
import {
  CharacterEffectsService,
  type RestCondition,
} from '../characters/characters-effects.service';
import { AuthService } from '../auth/auth.service';
import type { AuthUser } from '../auth/auth-user.type';
import {
  SessionStateService,
  type AddEntryInput,
  type SessionRuntimeState,
  type UpdateEntryInput,
} from './session-state.service';
import { PresenceRegistry } from './presence-registry';
import { verifyHandshake } from './ws-auth';

type AuthedSocket = Socket & {
  data: { user: AuthUser; role?: 'gm' | 'player' };
};

export function sessionRoom(sessionId: number): string {
  return `session:${sessionId}`;
}

type SessionScopedBody = { campaignId: number; sessionId: number };

/**
 * Realtime session gateway. Fase C1 wired auth + join/leave. Fase C2
 * adds initiative state management + broadcasts. Every mutating
 * handler re-checks ownership via `SessionsService.findOne` — that's
 * duplicated work with the join-time check, but it means someone who
 * spoofs `sessionId` on a stale socket can't hijack another table.
 */
@WebSocketGateway({ cors: { origin: true, credentials: true } })
export class RealtimeGateway implements OnGatewayConnection, OnGatewayDisconnect {
  private readonly logger = new Logger(RealtimeGateway.name);

  @WebSocketServer()
  server!: Server;

  /** Who's-online bookkeeping (maps + dedupe) lives here; the gateway
   * just broadcasts the rosters it returns. */
  private readonly presence = new PresenceRegistry();

  constructor(
    private readonly jwt: JwtService,
    private readonly prisma: PrismaService,
    private readonly sessions: SessionsService,
    private readonly state: SessionStateService,
    private readonly characters: CharactersService,
    private readonly effects: CharacterEffectsService,
    private readonly auth: AuthService,
  ) {}

  async handleConnection(socket: Socket) {
    try {
      const user = await verifyHandshake(socket, this.jwt, this.auth);
      (socket as AuthedSocket).data.user = user;
      this.logger.log(`socket ${socket.id} authenticated as user ${user.id}`);
    } catch (err) {
      this.logger.warn(
        `socket ${socket.id} rejected: ${(err as Error).message}`,
      );
      socket.emit('unauthorized', { message: (err as Error).message });
      socket.disconnect(true);
    }
  }

  handleDisconnect(socket: Socket) {
    this.logger.log(`socket ${socket.id} disconnected`);
    for (const { sessionId, roster } of this.presence.disconnect(socket.id)) {
      this.emitPresence(sessionId, roster);
    }
  }

  @SubscribeMessage('join-session')
  async joinSession(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody,
  ) {
    await this.assertSessionAccess(socket, body);
    const room = sessionRoom(body.sessionId);
    await socket.join(room);
    this.trackPresence(socket, body.sessionId);
    return { joined: room };
  }

  @SubscribeMessage('leave-session')
  async leaveSession(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: { sessionId: number },
  ) {
    if (!body || !Number.isInteger(body.sessionId)) {
      throw new WsException('sessionId is required');
    }
    const room = sessionRoom(body.sessionId);
    await socket.leave(room);
    const roster = this.presence.leave(body.sessionId, socket.id);
    if (roster) this.emitPresence(body.sessionId, roster);
    return { left: room };
  }

  @SubscribeMessage('get-session-state')
  async getSessionState(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody,
  ) {
    await this.assertSessionAccess(socket, body);
    /* P5: on every full-state pull, refresh hpMax/mpMax from the DB
     * Character rows. Cheap batched read; ensures a level-up
     * mid-session doesn't leave the tracker capped at the old max.
     * hpCurrent is intentionally untouched. */
    return this.state.refreshCharacterMaxes(body.sessionId);
  }

  @SubscribeMessage('initiative-add')
  async initiativeAdd(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody()
    body: SessionScopedBody & {
      entry: Partial<AddEntryInput> & { characterId?: number };
    },
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    if (!body.entry) throw new WsException('entry is required');
    const entry = await this.materializeEntry(
      socket.data.user.id,
      body.campaignId,
      body.entry,
    );
    const state = this.state.addEntry(body.sessionId, entry);
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  /**
   * Resolve an initiative-add payload into the concrete `AddEntryInput`
   * the state service expects. When `characterId` is set:
   *   - The character must be a member of the campaign the session
   *     belongs to. Prevents adding a random PC from another table.
   *   - The caller must own the character OR be the campaign GM.
   *   - `label` / hp / mp defaults are lifted from the Character row
   *     so the GM doesn't retype known stats.
   *
   * NPCs (no `characterId`) still require `label` explicitly.
   */
  private async materializeEntry(
    callerId: number,
    campaignId: number,
    input: Partial<AddEntryInput> & { characterId?: number },
  ): Promise<AddEntryInput> {
    if (input.characterId === undefined) {
      if (typeof input.label !== 'string' || !input.label.trim()) {
        throw new WsException('entry.label is required for NPC entries');
      }
      if (typeof input.initiative !== 'number') {
        throw new WsException('entry.initiative is required');
      }
      const type: 'character' | 'npc' = input.type ?? 'npc';
      return {
        ...input,
        label: input.label,
        initiative: input.initiative,
        type,
      };
    }
    const characterId = input.characterId;
    if (typeof input.initiative !== 'number') {
      throw new WsException('entry.initiative is required');
    }
    const [character, campaign, member] = await Promise.all([
      this.prisma.character.findUnique({
        where: { id: characterId },
        select: {
          id: true,
          name: true,
          ownerId: true,
          hpCurrent: true,
          hpMax: true,
          mpCurrent: true,
          mpMax: true,
        },
      }),
      this.prisma.campaign.findUnique({
        where: { id: campaignId },
        select: { id: true, ownerId: true },
      }),
      this.prisma.campaignMember.findUnique({
        where: {
          campaignId_characterId: {
            campaignId,
            characterId,
          },
        },
        select: { id: true },
      }),
    ]);
    if (!character) {
      throw new WsException(`Character ${characterId} not found`);
    }
    if (!campaign) {
      throw new WsException(`Campaign ${campaignId} not found`);
    }
    if (!member) {
      throw new WsException(
        `Character ${characterId} is not a member of campaign ${campaignId}`,
      );
    }
    if (
      callerId !== character.ownerId &&
      callerId !== campaign.ownerId
    ) {
      throw new WsException(
        `You are neither the campaign GM nor the character's owner`,
      );
    }
    return {
      label: input.label?.trim() || character.name,
      initiative: input.initiative,
      type: 'character',
      characterId,
      hpCurrent: input.hpCurrent ?? character.hpCurrent,
      hpMax: input.hpMax ?? character.hpMax,
      mpCurrent: input.mpCurrent ?? character.mpCurrent,
      mpMax: input.mpMax ?? character.mpMax,
    };
  }

  @SubscribeMessage('initiative-update')
  async initiativeUpdate(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody()
    body: SessionScopedBody & { entryId: string; patch: UpdateEntryInput },
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    if (!body.entryId) throw new WsException('entryId is required');
    const state = this.state.updateEntry(
      body.sessionId,
      body.entryId,
      body.patch ?? {},
    );
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  @SubscribeMessage('initiative-remove')
  async initiativeRemove(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody & { entryId: string },
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    if (!body.entryId) throw new WsException('entryId is required');
    const state = this.state.removeEntry(body.sessionId, body.entryId);
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  @SubscribeMessage('initiative-next-turn')
  async initiativeNextTurn(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody,
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    const state = this.state.nextTurn(body.sessionId);
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  @SubscribeMessage('initiative-reset')
  async initiativeReset(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody,
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    const state = this.state.resetInitiative(body.sessionId);
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  /**
   * Pull the campaign's player characters (role='player') into the
   * tracker in one shot — GM-only. Each PC joins with initiative 0 (the
   * GM rolls/orders after) and live hp/mp lifted from the Character row.
   * Characters already in the tracker are skipped, so re-running is
   * idempotent (e.g. a player joined mid-session).
   */
  @SubscribeMessage('initiative-populate')
  async initiativePopulate(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody,
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    const members = await this.prisma.campaignMember.findMany({
      where: { campaignId: body.campaignId, role: 'player' },
      select: {
        character: {
          select: {
            id: true,
            name: true,
            hpCurrent: true,
            hpMax: true,
            mpCurrent: true,
            mpMax: true,
          },
        },
      },
    });
    const existing = new Set(
      this.state
        .getState(body.sessionId)
        .initiative.map((e) => e.characterId)
        .filter((id): id is number => id !== undefined),
    );
    let state = this.state.getState(body.sessionId);
    for (const { character } of members) {
      if (existing.has(character.id)) continue;
      state = this.state.addEntry(body.sessionId, {
        label: character.name,
        initiative: 0,
        type: 'character',
        characterId: character.id,
        hpCurrent: character.hpCurrent,
        hpMax: character.hpMax,
        mpCurrent: character.mpCurrent,
        mpMax: character.mpMax,
      });
    }
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  /**
   * Session-wide rest — GM-only. The gateway orchestrates + broadcasts;
   * the actual domain rules live on the Character aggregate. For every
   * member: end-scene expires scene effects; end-day expires scene+day
   * effects AND restores PV/PM per the T20 rest rule. Healed vitals are
   * mirrored onto the live tracker entry so bars update in real time.
   */
  @SubscribeMessage('session-rest')
  async sessionRest(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody()
    body: SessionScopedBody & {
      scope: 'scene' | 'day';
      condition?: RestCondition;
    },
  ) {
    await this.assertSessionAccess(socket, body);
    this.assertGm(socket);
    const callerId = socket.data.user.id;
    const condition: RestCondition = body.condition ?? 'normal';
    const members = await this.prisma.campaignMember.findMany({
      where: { campaignId: body.campaignId },
      select: { characterId: true },
    });

    let healed = 0;
    for (const { characterId } of members) {
      if (body.scope === 'day') {
        await this.effects.endDay(callerId, characterId);
        const vitals = await this.effects.restVitals(
          callerId,
          characterId,
          condition,
        );
        this.mirrorVitalsToTracker(body.sessionId, characterId, vitals);
        healed++;
      } else {
        await this.effects.endScene(callerId, characterId);
      }
    }
    if (healed > 0) {
      this.emitSessionState(
        body.sessionId,
        this.state.getState(body.sessionId),
      );
    }

    this.server.to(sessionRoom(body.sessionId)).emit('session-rest', {
      sessionId: body.sessionId,
      scope: body.scope,
      condition: body.condition,
    });
    return { rested: body.scope, characters: members.length, healed };
  }

  /** Copy freshly-persisted PV/PM onto the matching live tracker entry
   * (if the character is in the current initiative) so the bars reflect
   * a rest without a reload. */
  private mirrorVitalsToTracker(
    sessionId: number,
    characterId: number,
    vitals: { hpCurrent: number; mpCurrent: number },
  ) {
    const entry = this.state
      .getState(sessionId)
      .initiative.find((e) => e.characterId === characterId);
    if (entry) this.state.patchVitals(sessionId, entry.id, vitals);
  }

  @SubscribeMessage('vitals-patch')
  async vitalsPatch(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody()
    body: SessionScopedBody & {
      entryId: string;
      patch: { hpCurrent?: number; mpCurrent?: number };
    },
  ) {
    await this.assertSessionAccess(socket, body);
    if (!body.entryId) throw new WsException('entryId is required');
    await this.assertVitalsEditable(socket, body.sessionId, body.entryId);
    const state = this.state.patchVitals(
      body.sessionId,
      body.entryId,
      body.patch ?? {},
    );
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  @SubscribeMessage('vitals-delta')
  async vitalsDelta(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody()
    body: SessionScopedBody & {
      entryId: string;
      hpDelta?: number;
      mpDelta?: number;
    },
  ) {
    await this.assertSessionAccess(socket, body);
    if (!body.entryId) throw new WsException('entryId is required');
    await this.assertVitalsEditable(socket, body.sessionId, body.entryId);
    const state = this.state.deltaVitals(body.sessionId, body.entryId, {
      hpDelta: body.hpDelta,
      mpDelta: body.mpDelta,
    });
    this.emitSessionState(body.sessionId, state);
    return state;
  }

  /**
   * Emits `session-state` to everyone in the room + kicks off a
   * fire-and-forget persist of the new state. Persistence P1b: this
   * is deliberately not awaited — WS latency stays low; a transient
   * DB failure logs and marks the session dirty so the next mutation
   * retries the write.
   *
   * `persistence-warning` broadcast: whenever the dirty flag flips
   * (either persist just failed for the first time, or the retry just
   * succeeded), we notify the room so clients can render a
   * "unsaved changes" badge. We track the last-emitted status per
   * session to avoid a spammy stream when the flag stays the same.
   */
  private readonly lastEmittedDirty = new Map<number, boolean>();

  private emitSessionState(sessionId: number, state: SessionRuntimeState) {
    this.server.to(sessionRoom(sessionId)).emit('session-state', state);
    this.state.persist(sessionId).then((dirty) => {
      const previous = this.lastEmittedDirty.get(sessionId) ?? false;
      if (previous === dirty) return;
      this.lastEmittedDirty.set(sessionId, dirty);
      this.server
        .to(sessionRoom(sessionId))
        .emit('persistence-warning', { sessionId, dirty });
    });
  }

  /**
   * Record the joining socket in the presence registry (after it joined
   * the room, so it receives its own roster) and broadcast the updated
   * roster so clients render "who's online" chips.
   */
  private trackPresence(socket: AuthedSocket, sessionId: number) {
    const roster = this.presence.join(sessionId, socket.id, {
      userId: socket.data.user.id,
      name: socket.data.user.name ?? socket.data.user.email,
      role: socket.data.role ?? 'player',
    });
    this.emitPresence(sessionId, roster);
  }

  private emitPresence(
    sessionId: number,
    users: { userId: number; name: string; role: 'gm' | 'player' }[],
  ) {
    this.server
      .to(sessionRoom(sessionId))
      .emit('presence', { sessionId, users });
  }

  private async assertSessionAccess(
    socket: AuthedSocket,
    body: SessionScopedBody,
  ) {
    if (
      !body ||
      !Number.isInteger(body.campaignId) ||
      !Number.isInteger(body.sessionId)
    ) {
      throw new WsException('campaignId and sessionId are required integers');
    }
    try {
      // Member-aware: GM or any player member may join + observe the
      // session. Per-action GM gating (turn advance, initiative edits)
      // reads the stashed role — vitals mutations additionally re-check
      // character ownership below.
      const { role } = await this.sessions.findOneForCaller(
        socket.data.user.id,
        body.campaignId,
        body.sessionId,
      );
      socket.data.role = role;
    } catch (err) {
      throw new WsException((err as Error).message);
    }
  }

  /**
   * GM-only gate for initiative control (add / update / remove / next
   * turn / reset). Relies on `socket.data.role` stashed by the
   * preceding `assertSessionAccess`. Players observe the tracker but
   * cannot drive combat flow — that's the GM's job.
   */
  private assertGm(socket: AuthedSocket) {
    if (socket.data.role !== 'gm') {
      throw new WsException('Only the campaign GM can control initiative');
    }
  }

  /**
   * Vitals mutations: the GM edits any combatant; a player may edit only
   * their own character's HP/PM. NPC entries (no `characterId`) are
   * GM-only. The realtime entry carries `characterId`; we resolve the
   * Character's owner to authorize.
   */
  private async assertVitalsEditable(
    socket: AuthedSocket,
    sessionId: number,
    entryId: string,
  ) {
    if (socket.data.role === 'gm') return;
    const entry = this.state
      .getState(sessionId)
      .initiative.find((e) => e.id === entryId);
    if (!entry) throw new WsException(`Entry ${entryId} not found`);
    if (entry.characterId === undefined) {
      throw new WsException('Only the GM can edit NPC vitals');
    }
    // Ownership is a Character-aggregate rule — delegate the check.
    try {
      await this.characters.assertOwner(socket.data.user.id, entry.characterId);
    } catch {
      throw new WsException("You can only edit your own character's vitals");
    }
  }
}
