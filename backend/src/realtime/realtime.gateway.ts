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
import type { AuthUser } from '../auth/auth.service';
import {
  SessionStateService,
  type AddEntryInput,
  type SessionRuntimeState,
  type UpdateEntryInput,
} from './session-state.service';
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

  constructor(
    private readonly jwt: JwtService,
    private readonly prisma: PrismaService,
    private readonly sessions: SessionsService,
    private readonly state: SessionStateService,
  ) {}

  async handleConnection(socket: Socket) {
    try {
      const user = await verifyHandshake(socket, this.jwt, this.prisma);
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
  }

  @SubscribeMessage('join-session')
  async joinSession(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody,
  ) {
    await this.assertSessionAccess(socket, body);
    const room = sessionRoom(body.sessionId);
    await socket.join(room);
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
    const state = this.state.resetInitiative(body.sessionId);
    this.emitSessionState(body.sessionId, state);
    return state;
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
}
