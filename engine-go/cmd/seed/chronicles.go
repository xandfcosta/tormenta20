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

// demoMember é quem senta à mesa.
//
// O `role` saiu na ALE-287 com a coluna: quem mestra é o DONO da campanha, e
// isso já está dito pelo `ownerEmail` do personagem casado com o dono da mesa —
// escrever o papel de novo seria uma segunda fonte para o mesmo fato, livre
// para divergir dela.
type demoMember struct {
	ownerEmail string
	charName   string
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

// demoPlace é uma cena GUARDADA no acervo da campanha (ALE-292).
//
// A cena vai como JSON cru porque é isso que a coluna guarda: o `BoardState`
// serializado. Montá-la com o tipo do `tabuleiro` obrigaria este gerador a
// importar o domínio ao vivo para escrever seis peças — e o que se quer aqui é
// uma cena FIXA, não uma cena computada.
type demoPlace struct {
	name string
	// state é o `BoardState` serializado. Ver `tabuleiro/board_state.go` para os
	// nomes dos campos; eles são a fronteira e não mudam por capricho.
	state string
}

type demoCampaign struct {
	name        string
	description string
	ownerEmail  string
	members     []demoMember
	sessions    []demoSession
	places      []demoPlace
}

var demoCampaigns = []demoCampaign{
	{
		name:        "Snapshot Test ALE-33",
		ownerEmail:  "mestre@t20.local",
		description: "Mesa-vitrine: um grupo veterano segurando a linha contra a Tormenta enquanto o conselho de Tauron desmorona por dentro.",
		members: []demoMember{
			{"mestre@t20.local", "Tanque Placas Nv10"},
			{"jogador@t20.local", "Guerreiro Veterano Nv8"},
			{"jogador@t20.local", "Arcanista Erudito Nv9"},
			{"jogador@t20.local", "Paladino Sagrado Nv10"},
			{"jogador@t20.local", "Recruta Nv1 Simples"},
		},
		sessions: []demoSession{
			{1, "A emboscada na ponte", "ended", "2026-07-11T19:00:00.000Z", "2026-07-11T23:10:00.000Z", ""},
			{2, "O covil do necromante", "ended", "2026-07-25T19:00:00.000Z", "2026-07-25T23:40:00.000Z", ""},
			{3, "A queda do conselho", "ended", "2026-08-08T19:00:00.000Z", "2026-08-08T23:30:00.000Z", ""},
			{4, "", "active", "2026-08-10T19:00:00.000Z", "", ""},
			{5, "", "planned", "", "", "2026-08-22T19:00:00.000Z"},
		},
		// O ACERVO da mesa-vitrine (ALE-271), e ele não é enfeite: o botão
		// "Lugares da campanha · N" é o ÚNICO item do painel de verbos com texto
		// e contagem, e era ele que empurrava dois botões para fora da janela a
		// 390px. Sem acervo na seed, o painel que o e2e mede não é o painel que a
		// mesa vê — a cena estava na lista dos guardas e o ESTADO não.
		//
		// TRÊS e não cento e quarenta e oito: o que muda a largura é o botão
		// EXISTIR e a contagem ter dois dígitos, e três lugares já dão isso. Um
		// acervo grande na seed custaria o dobro do arquivo para medir a mesma
		// coisa.
		places: []demoPlace{
			{"Taverna do Javali", tavernaDoJavali},
			{"Ponte de Vectora", pontDeVectora},
			{"Ruínas de Lenoria", ruinasDeLenoria},
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
		places:      []demoPlace{{"Cripta de Thwor", criptaDeThwor}},
	},
	{
		name:        "A Lâmina de Arton",
		ownerEmail:  "jogador@t20.local",
		description: "Mercenários caçando uma relíquia de Tenebra pelos Reinados. Mesa quinzenal.",
		members: []demoMember{
			{"mestre@t20.local", "Tanque Placas Nv10"},
			{"mestre@t20.local", "Curandeira Divina Nv8"},
			{"mestre@t20.local", "Necromante Nv12 Magias"},
		},
		sessions: []demoSession{
			{1, "O contrato de Tenebra", "ended", "2026-07-05T20:00:00.000Z", "2026-07-05T23:20:00.000Z", ""},
		},
	},
	{
		name:        "Caçadores de Deheon",
		ownerEmail:  "jogador@t20.local",
		description: "Fronteira selvagem de Deheon: contratos, bestas e política nobre.",
		members:     []demoMember{{"mestre@t20.local", "Bardo Versátil Nv7"}},
	},
}

// A cena guardada de "O Chamado de Valkaria" (ALE-292).
//
// Ela vive nessa campanha DE PROPÓSITO. As mesas do e2e são todas da campanha 1,
// e um lugar a mais no acervo dela mudaria a contagem do botão "Lugares da
// campanha · N" debaixo de specs que não falam disto.
//
// A cena tem peça, peça ESCONDIDA, marcador escondido e terreno difícil porque o
// guarda de contraste mede o que a tela DESENHA: um lugar vazio seria uma grade
// em branco, e a medição passaria por cima da tinta que ela existe para
// conferir. Foi assim que ela achou, na estreia, um marcador carmim a 4,11:1.
// A TAVERNA é a cena de interpretação: gente e mobília, sem terreno difícil.
const tavernaDoJavali = `{"id":"semente-taverna","version":5,"place":"Taverna do Javali","terrain":"taverna",` +
	`"tokens":[` +
	`{"id":"semente-taverna-1","label":"Taverneiro","kind":"npc","x":0,"y":-2,"footprint":1},` +
	`{"id":"semente-taverna-2","label":"Balcão","kind":"object","x":1,"y":-2,"footprint":2},` +
	`{"id":"semente-taverna-3","label":"Mesa redonda","kind":"object","x":-2,"y":1,"footprint":2}]}`

// A PONTE é a cena da primeira sessão da mesa-vitrine ("A emboscada na ponte"),
// com terreno difícil dos dois lados.
const pontDeVectora = `{"id":"semente-ponte","version":4,"place":"Ponte de Vectora","terrain":"pedra",` +
	`"tokens":[{"id":"semente-ponte-1","label":"Salteador","kind":"npc","x":2,"y":0,"footprint":1,"hidden":true}],` +
	`"difficult":[{"x":-1,"y":0},{"x":-1,"y":1},{"x":5,"y":0},{"x":5,"y":1}]}`

// A cena VAZIA existe de propósito: é o lugar aberto e abandonado, o caso que a
// linha "cena vazia" do acervo descreve e que o mestre abre o acervo para
// limpar.
const ruinasDeLenoria = `{"id":"semente-ruinas","version":1,"place":"Ruínas de Lenoria","terrain":"ermo","tokens":[]}`

const criptaDeThwor = `{"id":"semente-cripta","version":3,"place":"Cripta de Thwor","terrain":"cripta",` +
	`"tokens":[` +
	`{"id":"semente-peca-1","label":"Porta selada","kind":"object","x":-2,"y":1,"footprint":1},` +
	`{"id":"semente-peca-2","label":"Guardião de Thwor","kind":"npc","x":3,"y":-1,"footprint":2,"hidden":true}],` +
	`"markers":[{"id":"semente-marca-1","text":"A","color":"carmim","x":0,"y":0,"hidden":true}],` +
	`"difficult":[{"x":1,"y":1},{"x":2,"y":1}]}`

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
		if err := seedPlaces(database, campID, dc); err != nil {
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
			`INSERT INTO campaign_members (campaignId, characterId, addedAt) VALUES (?, ?, ?)`,
			campID, chID, seedChronicleDate); err != nil {
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

func seedPlaces(database *sql.DB, campID int64, dc demoCampaign) error {
	for _, l := range dc.places {
		if _, err := database.Exec(
			`INSERT INTO campaign_places (campaignId, name, state, createdAt, updatedAt)
			 VALUES (?, ?, ?, ?, ?)`,
			campID, l.name, l.state, seedChronicleDate, seedChronicleDate); err != nil {
			return fmt.Errorf("campaign %q place %q: %w", dc.name, l.name, err)
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
