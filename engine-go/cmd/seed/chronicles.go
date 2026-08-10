package main

import (
	"database/sql"
	"fmt"
)

// Demo chronicles (kept in the seed and grown over time). Members reference a
// character by its owner's email + name (unique within a user); sessions carry
// a small history so the detail tome's roster + session log validate with real
// shape. Dates are fixed constants → the dump stays deterministic without a
// clock (these `*At` columns are excluded from the dump's timestamp normalizer).
const seedChronicleDate = "2026-07-01T12:00:00.000Z"

type demoMember struct {
	ownerEmail string
	charName   string
	role       string // gm | player
}

type demoSession struct {
	number      int
	title       string
	status      string // planned | active | ended
	startedAt   string // "" when planned
	endedAt     string // "" unless ended
	scheduledAt string // planned-session date; only used to timestamp when startedAt is ""
}

// sessionDate is the row's createdAt/updatedAt — what the session log timeline
// shows. Prefer the real start; a planned session has none, so fall back to its
// scheduled date (keeps the timeline chronological instead of all-same-day).
func (s demoSession) sessionDate() string {
	if s.startedAt != "" {
		return s.startedAt
	}
	if s.scheduledAt != "" {
		return s.scheduledAt
	}
	return seedChronicleDate
}

type demoCampaign struct {
	name        string
	description string
	ownerEmail  string
	members     []demoMember
	sessions    []demoSession
}

var demoCampaigns = []demoCampaign{
	{
		name:        "Snapshot Test ALE-33",
		ownerEmail:  "mestre@t20.local",
		description: "Mesa-vitrine: um grupo veterano segurando a linha contra a Tormenta enquanto o conselho de Tauron desmorona por dentro.",
		members: []demoMember{
			{"mestre@t20.local", "Tanque Placas Nv10", "gm"},
			{"jogador@t20.local", "Guerreiro Veterano Nv8", "player"},
			{"jogador@t20.local", "Arcanista Erudito Nv9", "player"},
			{"jogador@t20.local", "Paladino Sagrado Nv10", "player"},
			{"jogador@t20.local", "Recruta Nv1 Simples", "player"},
		},
		sessions: []demoSession{
			{1, "A emboscada na ponte", "ended", "2026-07-11T19:00:00.000Z", "2026-07-11T23:10:00.000Z", ""},
			{2, "O covil do necromante", "ended", "2026-07-25T19:00:00.000Z", "2026-07-25T23:40:00.000Z", ""},
			{3, "A queda do conselho", "ended", "2026-08-08T19:00:00.000Z", "2026-08-08T23:30:00.000Z", ""},
			{4, "", "active", "2026-08-10T19:00:00.000Z", "", ""},
			{5, "", "planned", "", "", "2026-08-22T19:00:00.000Z"},
		},
	},
	{
		name:        "A Queda de Tauron",
		ownerEmail:  "mestre@t20.local",
		description: "Uma metrópole à beira do colapso: cultos de Aharadak infiltram o conselho enquanto a Tormenta avança pelo norte.",
		sessions: []demoSession{
			{1, "Chegada a Tauron", "ended", "2026-06-20T19:00:00.000Z", "2026-06-20T22:40:00.000Z", ""},
			{2, "", "planned", "", "", "2026-08-15T19:00:00.000Z"},
		},
	},
	{
		name:        "Segredos de Wynlla",
		ownerEmail:  "mestre@t20.local",
		description: "Campanha de intriga arcana na Academia Arcana de Wynlla — segredos proibidos e um necromante à espreita.",
	},
	{
		name:        "O Chamado de Valkaria",
		ownerEmail:  "mestre@t20.local",
		description: "Heróis reunidos pela deusa da ambição para deter uma invasão de lefeu nas Montanhas Uivantes.",
	},
	{
		name:        "A Lâmina de Arton",
		ownerEmail:  "jogador@t20.local",
		description: "Mercenários caçando uma relíquia de Tenebra pelos Reinados. Mesa quinzenal.",
		members: []demoMember{
			{"mestre@t20.local", "Tanque Placas Nv10", "player"},
			{"mestre@t20.local", "Curandeira Divina Nv8", "player"},
			{"mestre@t20.local", "Necromante Nv12 Magias", "player"},
		},
		sessions: []demoSession{
			{1, "O contrato de Tenebra", "ended", "2026-07-05T20:00:00.000Z", "2026-07-05T23:20:00.000Z", ""},
		},
	},
	{
		name:        "Caçadores de Deheon",
		ownerEmail:  "jogador@t20.local",
		description: "Fronteira selvagem de Deheon: contratos, bestas e política nobre.",
		members:     []demoMember{{"mestre@t20.local", "Bardo Versátil Nv7", "player"}},
	},
}

// seedChronicles inserts the demo campaigns, memberships and sessions directly
// (no engine needed — plain rows referencing the seeded users/chars).
func seedChronicles(database *sql.DB) error {
	for i, dc := range demoCampaigns {
		ownerID, err := userID(database, dc.ownerEmail)
		if err != nil {
			return err
		}
		res, err := database.Exec(
			`INSERT INTO campaigns (ownerId, name, description, inviteToken, createdAt, updatedAt)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			ownerID, dc.name, nullable(dc.description), fmt.Sprintf("seedtoken-%02d", i+1), seedChronicleDate, seedChronicleDate)
		if err != nil {
			return fmt.Errorf("campaign %q: %w", dc.name, err)
		}
		campID, _ := res.LastInsertId()
		if err := seedMembers(database, campID, dc); err != nil {
			return err
		}
		if err := seedSessions(database, campID, dc); err != nil {
			return err
		}
	}
	return nil
}

func seedMembers(database *sql.DB, campID int64, dc demoCampaign) error {
	for _, m := range dc.members {
		chID, err := characterID(database, m.ownerEmail, m.charName)
		if err != nil {
			return fmt.Errorf("campaign %q member: %w", dc.name, err)
		}
		if _, err := database.Exec(
			`INSERT INTO campaign_members (campaignId, characterId, role, addedAt) VALUES (?, ?, ?, ?)`,
			campID, chID, m.role, seedChronicleDate); err != nil {
			return fmt.Errorf("campaign %q membership: %w", dc.name, err)
		}
	}
	return nil
}

func seedSessions(database *sql.DB, campID int64, dc demoCampaign) error {
	for _, s := range dc.sessions {
		at := s.sessionDate()
		if _, err := database.Exec(
			`INSERT INTO sessions (campaignId, sessionNumber, title, status, startedAt, endedAt, createdAt, updatedAt)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			campID, s.number, nullable(s.title), s.status, nullable(s.startedAt),
			nullable(s.endedAt), at, at); err != nil {
			return fmt.Errorf("campaign %q session %d: %w", dc.name, s.number, err)
		}
	}
	return nil
}

func userID(database *sql.DB, email string) (int64, error) {
	var id int64
	if err := database.QueryRow(`SELECT id FROM users WHERE email = ?`, email).Scan(&id); err != nil {
		return 0, fmt.Errorf("user %q: %w", email, err)
	}
	return id, nil
}

func characterID(database *sql.DB, email, name string) (int64, error) {
	var id int64
	err := database.QueryRow(
		`SELECT c.id FROM characters c JOIN users u ON u.id = c.ownerId WHERE u.email = ? AND c.name = ?`,
		email, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("character %q/%q: %w", email, name, err)
	}
	return id, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
