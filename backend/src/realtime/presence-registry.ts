/** One connected participant in a session room. Multiple browser tabs of
 * the same user are separate socket entries; rosters dedupe by `userId`. */
export type PresenceUser = { userId: number; name: string; role: 'gm' | 'player' };

/**
 * In-memory "who's online" tracker for session rooms. Pure bookkeeping —
 * the gateway owns the socket.io broadcast; this owns the maps and the
 * dedupe-by-userId roster rule. Rebuilt from live sockets, so a server
 * restart starts empty and refills as clients reconnect.
 */
export class PresenceRegistry {
  /** sessionId → (socketId → presence). */
  private readonly presence = new Map<number, Map<string, PresenceUser>>();
  /** socketId → sessions it joined, so a disconnect cleans up every room
   * without relying on socket.io's post-disconnect room set. */
  private readonly socketSessions = new Map<string, Set<number>>();

  /** Record a socket in a session; returns the session's deduped roster. */
  join(sessionId: number, socketId: string, user: PresenceUser): PresenceUser[] {
    let room = this.presence.get(sessionId);
    if (!room) {
      room = new Map();
      this.presence.set(sessionId, room);
    }
    room.set(socketId, user);
    let sessions = this.socketSessions.get(socketId);
    if (!sessions) {
      sessions = new Set();
      this.socketSessions.set(socketId, sessions);
    }
    sessions.add(sessionId);
    return this.roster(sessionId);
  }

  /** Remove a socket from one session. Returns the new roster to broadcast,
   * or null if the socket wasn't present (nothing to announce). */
  leave(sessionId: number, socketId: string): PresenceUser[] | null {
    this.socketSessions.get(socketId)?.delete(sessionId);
    if (!this.drop(socketId, sessionId)) return null;
    return this.roster(sessionId);
  }

  /** Remove a socket from every session it joined. Returns one
   * `{ sessionId, roster }` per room that actually changed, so the caller
   * can broadcast each. */
  disconnect(socketId: string): { sessionId: number; roster: PresenceUser[] }[] {
    const sessions = this.socketSessions.get(socketId);
    if (!sessions) return [];
    const changed: { sessionId: number; roster: PresenceUser[] }[] = [];
    for (const sessionId of sessions) {
      if (this.drop(socketId, sessionId)) {
        changed.push({ sessionId, roster: this.roster(sessionId) });
      }
    }
    this.socketSessions.delete(socketId);
    return changed;
  }

  private drop(socketId: string, sessionId: number): boolean {
    const room = this.presence.get(sessionId);
    if (!room || !room.delete(socketId)) return false;
    if (room.size === 0) this.presence.delete(sessionId);
    return true;
  }

  /** Deduped by userId — multi-tab collapses to one chip; a user counts as
   * GM if any of their sockets is a GM. */
  private roster(sessionId: number): PresenceUser[] {
    const room = this.presence.get(sessionId);
    const byUser = new Map<number, PresenceUser>();
    for (const user of room?.values() ?? []) {
      const existing = byUser.get(user.userId);
      if (!existing || (existing.role !== 'gm' && user.role === 'gm')) {
        byUser.set(user.userId, user);
      }
    }
    return [...byUser.values()];
  }
}
