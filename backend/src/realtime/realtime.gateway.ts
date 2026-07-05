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

type AuthedSocket = Socket & { data: { user: AuthUser } };

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
    return this.state.getState(body.sessionId);
  }

  @SubscribeMessage('initiative-add')
  async initiativeAdd(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: SessionScopedBody & { entry: AddEntryInput },
  ) {
    await this.assertSessionAccess(socket, body);
    if (!body.entry || typeof body.entry.label !== 'string') {
      throw new WsException('entry.label is required');
    }
    const state = this.state.addEntry(body.sessionId, body.entry);
    this.emitSessionState(body.sessionId, state);
    return state;
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

  private emitSessionState(sessionId: number, state: SessionRuntimeState) {
    this.server.to(sessionRoom(sessionId)).emit('session-state', state);
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
      await this.sessions.findOne(
        socket.data.user.id,
        body.campaignId,
        body.sessionId,
      );
    } catch (err) {
      throw new WsException((err as Error).message);
    }
  }
}
