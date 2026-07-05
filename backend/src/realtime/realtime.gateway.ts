import { Logger } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import {
  ConnectedSocket,
  MessageBody,
  OnGatewayConnection,
  OnGatewayDisconnect,
  SubscribeMessage,
  WebSocketGateway,
  WsException,
} from '@nestjs/websockets';
import type { Server, Socket } from 'socket.io';
import { PrismaService } from '../prisma/prisma.service';
import { SessionsService } from '../sessions/sessions.service';
import type { AuthUser } from '../auth/auth.service';
import { verifyHandshake } from './ws-auth';

type AuthedSocket = Socket & { data: { user: AuthUser } };

export function sessionRoom(sessionId: number): string {
  return `session:${sessionId}`;
}

/**
 * WebSocket gateway root for realtime session play. Fase C1 ships the
 * skeleton:
 *   - JWT handshake auth (drops unauthenticated sockets)
 *   - `join-session` message: validates that the user owns the
 *     underlying campaign, adds the socket to the session room
 *   - `leave-session` message
 *   - Broadcast primitives (state/events) plug in on Fase C2+
 */
@WebSocketGateway({ cors: { origin: true, credentials: true } })
export class RealtimeGateway implements OnGatewayConnection, OnGatewayDisconnect {
  private readonly logger = new Logger(RealtimeGateway.name);

  constructor(
    private readonly jwt: JwtService,
    private readonly prisma: PrismaService,
    private readonly sessions: SessionsService,
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

  /**
   * Join a session room. The gateway delegates ownership enforcement
   * to `SessionsService.findOne`, which already throws NotFound /
   * Forbidden — we translate those into WsException so the socket
   * client sees a structured error instead of a stack trace.
   */
  @SubscribeMessage('join-session')
  async joinSession(
    @ConnectedSocket() socket: AuthedSocket,
    @MessageBody() body: { campaignId: number; sessionId: number },
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

  /**
   * Broadcast a payload to every socket currently subscribed to the
   * session room. Fase C2 will call this from the state service; kept
   * public so the tests can drive it directly.
   */
  broadcastToSession(
    server: Server,
    sessionId: number,
    event: string,
    payload: unknown,
  ) {
    server.to(sessionRoom(sessionId)).emit(event, payload);
  }
}
