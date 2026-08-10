package main

import (
	"database/sql"
	"fmt"
	"time"
)

// Demo chronicles (kept in the seed and grown over time). Members reference a
// character by its owner's email + name (unique within a user). One campaign
// carries a live session so the roster's "ao vivo" state is reproducible.
type demoMember struct {
	ownerEmail string
	charName   string
	role       string
}

type demoCampaign struct {
	name          string
	description   string
	ownerEmail    string
	activeSession bool
	members       []demoMember
}

var demoCampaigns = []demoCampaign{
	{
		name:          "Snapshot Test ALE-33",
		ownerEmail:    "mestre@t20.local",
		activeSession: true,
		members:       []demoMember{{"jogador@t20.local", "Guerreiro Veterano Nv8", "player"}},
	},
	{
		name:        "A Queda de Tauron",
		ownerEmail:  "mestre@t20.local",
		description: "Uma metrópole à beira do colapso: cultos de Aharadak infiltram o conselho enquanto a Tormenta avança pelo norte.",
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
		members:     []demoMember{{"mestre@t20.local", "Tanque Placas Nv10", "player"}},
	},
	{
		name:        "Caçadores de Deheon",
		ownerEmail:  "jogador@t20.local",
		description: "Fronteira selvagem de Deheon: contratos, bestas e política nobre.",
		members:     []demoMember{{"mestre@t20.local", "Bardo Versátil Nv7", "player"}},
	},
}

// seedChronicles inserts the demo campaigns, memberships and live session
// directly (no engine needed — plain rows referencing the seeded users/chars).
func seedChronicles(database *sql.DB) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	for i, dc := range demoCampaigns {
		ownerID, err := userID(database, dc.ownerEmail)
		if err != nil {
			return err
		}
		res, err := database.Exec(
			`INSERT INTO campaigns (ownerId, name, description, inviteToken, createdAt, updatedAt)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			ownerID, dc.name, nullable(dc.description), fmt.Sprintf("seedtoken-%02d", i+1), now, now)
		if err != nil {
			return fmt.Errorf("campaign %q: %w", dc.name, err)
		}
		campID, _ := res.LastInsertId()
		for _, m := range dc.members {
			chID, err := characterID(database, m.ownerEmail, m.charName)
			if err != nil {
				return fmt.Errorf("campaign %q member: %w", dc.name, err)
			}
			if _, err := database.Exec(
				`INSERT INTO campaign_members (campaignId, characterId, role, addedAt) VALUES (?, ?, ?, ?)`,
				campID, chID, m.role, now); err != nil {
				return fmt.Errorf("campaign %q membership: %w", dc.name, err)
			}
		}
		if dc.activeSession {
			if _, err := database.Exec(
				`INSERT INTO sessions (campaignId, sessionNumber, status, startedAt, createdAt, updatedAt)
				 VALUES (?, 1, 'active', ?, ?, ?)`,
				campID, now, now, now); err != nil {
				return fmt.Errorf("campaign %q session: %w", dc.name, err)
			}
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
