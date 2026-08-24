package aovivo

import (
	"sync"
	"testing"
)

func user(id int64, role string) PresenceUser {
	return PresenceUser{UserID: id, Name: "u", Role: role}
}

func TestPresenceDedupByUser(t *testing.T) {
	p := NewPresenceRegistry()
	// Same user, two tabs (two sockets) in the same session → one roster entry.
	p.Join(1, "sockA", user(10, "player"))
	roster := p.Join(1, "sockB", user(10, "player"))
	if len(roster) != 1 || roster[0].UserID != 10 {
		t.Fatalf("roster=%+v, want a single user 10", roster)
	}
	// A second distinct user shows up too, sorted by userId.
	roster = p.Join(1, "sockC", user(4, "gm"))
	if len(roster) != 2 || roster[0].UserID != 4 || roster[1].UserID != 10 {
		t.Errorf("roster=%+v, want [4,10]", roster)
	}
}

func TestPresenceGmWins(t *testing.T) {
	p := NewPresenceRegistry()
	p.Join(1, "player-tab", user(7, "player"))
	roster := p.Join(1, "gm-tab", user(7, "gm")) // same user, one socket is GM
	if len(roster) != 1 || roster[0].Role != "gm" {
		t.Errorf("roster=%+v, want single user as gm", roster)
	}
}

func TestPresenceLeave(t *testing.T) {
	p := NewPresenceRegistry()
	p.Join(1, "a", user(10, "player"))
	p.Join(1, "b", user(20, "player"))

	roster, ok := p.Leave(1, "a")
	if !ok || len(roster) != 1 || roster[0].UserID != 20 {
		t.Errorf("after Leave: ok=%v roster=%+v, want [20]", ok, roster)
	}
	// Leaving a socket that isn't present announces nothing.
	if _, ok := p.Leave(1, "ghost"); ok {
		t.Error("leaving an absent socket should return ok=false")
	}
	// Last one leaves → empty roster, room cleaned up.
	roster, ok = p.Leave(1, "b")
	if !ok || len(roster) != 0 {
		t.Errorf("last Leave: ok=%v roster=%+v, want empty", ok, roster)
	}
}

func TestPresenceDisconnectAcrossSessions(t *testing.T) {
	p := NewPresenceRegistry()
	// One socket present in two sessions, plus another user in session 1.
	p.Join(1, "sock", user(10, "player"))
	p.Join(2, "sock", user(10, "player"))
	p.Join(1, "other", user(20, "gm"))

	changed := p.disconnect("sock")
	if len(changed) != 2 {
		t.Fatalf("disconnect touched %d rooms, want 2", len(changed))
	}
	// Session 1 (sorted first) still has user 20; session 2 is now empty.
	if changed[0].sessionID != 1 || len(changed[0].roster) != 1 || changed[0].roster[0].UserID != 20 {
		t.Errorf("session 1 roster=%+v, want [20]", changed[0].roster)
	}
	if changed[1].sessionID != 2 || len(changed[1].roster) != 0 {
		t.Errorf("session 2 roster=%+v, want empty", changed[1].roster)
	}
	// The socket is fully forgotten — a second disconnect is a no-op.
	if again := p.disconnect("sock"); again != nil {
		t.Errorf("second disconnect=%+v, want nil", again)
	}
}

// Concurrent joins/disconnects must not race (run with -race).
func TestPresenceConcurrent(t *testing.T) {
	p := NewPresenceRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sock := "s" + itoa(n)
			p.Join(1, sock, user(int64(n), "player"))
			p.disconnect(sock)
		}(i)
	}
	wg.Wait()
	if roster := p.Join(1, "final", user(999, "gm")); len(roster) != 1 {
		t.Errorf("after churn, roster=%+v, want just the final user", roster)
	}
}
