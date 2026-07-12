import { PresenceRegistry } from './presence-registry';

describe('PresenceRegistry', () => {
  const gm = { userId: 7, name: 'GM', role: 'gm' as const };

  it('join returns a roster that includes the joiner', () => {
    const r = new PresenceRegistry();
    expect(r.join(5, 'a', gm)).toEqual([gm]);
  });

  it('dedupes multiple tabs of one user into a single entry (GM wins)', () => {
    const r = new PresenceRegistry();
    r.join(5, 'a', { userId: 7, name: 'X', role: 'player' });
    const roster = r.join(5, 'b', { userId: 7, name: 'X', role: 'gm' });
    expect(roster).toEqual([{ userId: 7, name: 'X', role: 'gm' }]);
  });

  it('leave removes the socket and returns the new roster', () => {
    const r = new PresenceRegistry();
    r.join(5, 'a', gm);
    r.join(5, 'b', { userId: 8, name: 'P', role: 'player' });
    expect(r.leave(5, 'a')).toEqual([
      { userId: 8, name: 'P', role: 'player' },
    ]);
  });

  it('leave returns null when the socket was not present', () => {
    const r = new PresenceRegistry();
    expect(r.leave(5, 'ghost')).toBeNull();
  });

  it('disconnect drops the socket from every session it joined', () => {
    const r = new PresenceRegistry();
    r.join(5, 'a', gm);
    r.join(6, 'a', gm);
    const changed = r.disconnect('a');
    expect(changed.map((c) => c.sessionId).sort()).toEqual([5, 6]);
    expect(changed.every((c) => c.roster.length === 0)).toBe(true);
  });
});
