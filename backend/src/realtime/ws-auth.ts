import { JwtService } from '@nestjs/jwt';
import type { Socket } from 'socket.io';
import type { AuthService } from '../auth/auth.service';
import type { AuthUser, JwtPayload } from '../auth/auth-user.type';

/**
 * WebSocket handshake authentication. Client passes the JWT in the
 * handshake `auth` object (`io({ auth: { token } })`) OR via the
 * `Authorization: Bearer …` header. The token is verified using the
 * same secret as the HTTP layer; the resolved user id is looked up
 * against the DB so revoked users can't linger on stale tokens.
 *
 * Returns the resolved AuthUser or throws — callers wrap the throw in a
 * `WsException` before rejecting the connection.
 */
export async function verifyHandshake(
  socket: Socket,
  jwt: JwtService,
  auth: AuthService,
): Promise<AuthUser> {
  const token = extractToken(socket);
  if (!token) throw new Error('Missing auth token');
  const payload = jwt.verify<JwtPayload>(token);
  // Identity resolution (incl. the "revoked user can't linger" rule) is
  // the auth context's concern — delegate instead of re-querying users.
  const user = await auth.findById(payload.sub);
  if (!user) throw new Error('User no longer exists');
  return user;
}

const COOKIE_NAME = process.env.COOKIE_NAME ?? 't20_session';

function extractToken(socket: Socket): string | null {
  const auth = socket.handshake.auth as { token?: string } | undefined;
  if (auth?.token) return auth.token;
  const header = socket.handshake.headers.authorization;
  if (header?.startsWith('Bearer ')) return header.slice(7);
  const cookie = socket.handshake.headers.cookie;
  if (cookie) {
    const match = new RegExp(`(?:^|;\\s*)${COOKIE_NAME}=([^;]+)`).exec(cookie);
    if (match?.[1]) return decodeURIComponent(match[1]);
  }
  return null;
}
