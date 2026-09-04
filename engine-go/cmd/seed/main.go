// Command seed regenerates engine-go/seed.sql — a pure-SQL dump of the dev
// dataset (3 accounts, the diverse test roster, demo chronicles) that applies
// instantly with `sqlite3 data/t20-dev.db < seed.sql`, no API server (ALE-57).
//
// Ele monta o dado chamando as REGRAS do app num banco migrado descartável — os
// hashes de bcrypt, os vitais computados pelo motor e o leque normalizado vêm do
// mesmo código que o servidor roda, nunca mantidos à mão — e despeja o banco em
// SQL. O elenco mora no `seed-data.json` embutido, que é a fonte legível.
//
// Ele dirigia os MANIPULADORES HTTP em processo até a ALE-287, e as sete rotas
// que ele usava foram apagadas na ALE-277 por não terem consumidor: o gerador
// parou de rodar sem que nada acusasse, e a varredura de órfãs não o viu porque
// ele chamava por CAMINHO EM STRING. Hoje ele pede pela `api.Seeder`, a porta
// declarada logo abaixo — a mesma forma das onze cenas.
//
// Regenerate after roster/rule/chronicle changes:
//
//	go run ./cmd/seed            # writes ./seed.sql (from the engine-go dir)
package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"t20engine/api"
	"t20engine/catalog"
	"t20engine/db"
	"t20engine/engine"
	"t20engine/plataforma"
	"t20engine/sheet"
)

//go:embed seed-data.json
var seedData []byte

type seedFile struct {
	Password string     `json:"password"`
	Users    []seedUser `json:"users"`
}

type seedUser struct {
	Email      string          `json:"email"`
	Name       string          `json:"name"`
	Characters []seedCharacter `json:"characters"`
}

type seedCharacter struct {
	Create      json.RawMessage `json:"create"`
	Spells      []seedSpell     `json:"spells"`
	Simple      bool            `json:"simple"`
	SceneEffect bool            `json:"sceneEffect"`
	HpFraction  *float64        `json:"hpFraction"`
}

type seedSpell struct {
	ID       string `json:"id"`
	Prepared bool   `json:"prepared"`
}

// casaDaSeed é o que este gerador pede do app.
//
// Declarada AQUI e não no `api`, como as portas das cenas: quem escolhe o que
// atravessa a fronteira é o consumidor. O `api.Seeder` a cumpre, e é na linha
// que monta (`srv.Seeder()`) que o compilador cobra quando ela deixa de ser
// cumprida.
type casaDaSeed interface {
	CreateAccount(ctx context.Context, email, nome, senha string) error
	CreateCharacter(ctx context.Context, donoID int64, corpo sheet.CreateBody) (int64, error)
	Character(ctx context.Context, id int64) (sheet.CharacterDTO, error)
	LearnSpell(ctx context.Context, id int64, catalogo string, preparada bool) error
	SetHp(ctx context.Context, id, atual int64) error
	ConsumeItem(ctx context.Context, id, itemID int64) error
}

// standardTrained is TRAINED_EXPERTISES — every non-simple
// character trains these, giving the skill list real totals. Simple PCs train none.
var standardTrained = []string{
	"Luta", "Atletismo", "Pontaria", "Reflexos", "Fortitude",
	"Vontade", "Percepção", "Intimidação", "Investigação", "Misticismo",
}

// sceneConsumable is the item whose scene effect a sceneEffect character carries.
const sceneConsumable = "cosmetico"

func main() {
	out := "seed.sql"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		log.Fatalf("seed-data.json: %v", err)
	}
	casa, database, cleanup := freshServer(seedEmails(sf))
	defer cleanup()
	total, seeded := 0, 0
	for _, u := range sf.Users {
		total += len(u.Characters)
		seeded += seedUserCharacters(casa, database, sf.Password, u)
	}
	if err := seedChronicles(database); err != nil {
		log.Fatalf("chronicles: %v", err)
	}
	script, err := dump(database)
	if err != nil {
		log.Fatalf("dump: %v", err)
	}
	if err := os.WriteFile(out, []byte(script), 0o644); err != nil {
		log.Fatalf("write %s: %v", out, err)
	}
	log.Printf("wrote %s — %d/%d characters across %d users", out, seeded, total, len(sf.Users))
}

// seedEmails lists the accounts this run creates. They go in as ADMIN_EMAILS so
// registration works: since ALE-120 /auth/register demands an invite, and the
// first account of an empty database has nobody to have invited it — the
// generator is its own admin. Nothing of the role reaches seed.sql: it is
// derived from the environment at request time and has no column.
func seedEmails(sf seedFile) []string {
	emails := make([]string, 0, len(sf.Users))
	for _, u := range sf.Users {
		emails = append(emails, u.Email)
	}
	return emails
}

// freshServer sobe o app de verdade sobre um SQLite migrado descartável e
// devolve a PORTA dele mais o banco (para as crônicas e para o despejo).
func freshServer(adminEmails []string) (casaDaSeed, *sql.DB, func()) {
	dir, err := os.MkdirTemp("", "seedgen")
	if err != nil {
		log.Fatalf("tempdir: %v", err)
	}
	dbPath := filepath.Join(dir, "seed.db")
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db.Open: %v", err)
	}
	cfg, err := plataforma.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	cfg.DatabasePath = dbPath
	cfg.AdminEmails = adminEmails
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "seedgen"
	}
	raw, err := os.ReadFile(cfg.CatalogPath)
	if err != nil {
		log.Fatalf("catalogs %q: %v", cfg.CatalogPath, err)
	}
	catalogs, err := engine.PrimeEngineCatalogs(raw)
	if err != nil {
		log.Fatalf("prime catalogs: %v", err)
	}
	srv := api.NewServer(cfg, database, catalogs)
	cleanup := func() {
		_ = database.Close()
		_ = os.RemoveAll(dir)
	}
	return srv.Seeder(), database, cleanup
}

// ── semeando pelas REGRAS ──────────────────────────────────────────────────────

func seedUserCharacters(casa casaDaSeed, database *sql.DB, password string, u seedUser) int {
	ctx := context.Background()
	if err := casa.CreateAccount(ctx, u.Email, u.Name, password); err != nil {
		log.Printf("conta %s: %v", u.Email, err)
		return 0
	}
	donoID, err := userID(database, u.Email)
	if err != nil {
		log.Printf("conta %s: %v", u.Email, err)
		return 0
	}
	seeded := 0
	for _, ch := range u.Characters {
		if err := seedCharacterRow(ctx, casa, donoID, ch); err != nil {
			log.Printf("%s: %v", u.Email, err)
			continue
		}
		seeded++
	}
	return seeded
}

func seedCharacterRow(ctx context.Context, casa casaDaSeed, donoID int64, ch seedCharacter) error {
	bruto, err := enrichCreate(ch)
	if err != nil {
		return err
	}
	var corpo sheet.CreateBody
	if err := json.Unmarshal(bruto, &corpo); err != nil {
		return fmt.Errorf("corpo de criação: %w", err)
	}
	id, err := casa.CreateCharacter(ctx, donoID, corpo)
	if err != nil {
		return err
	}
	for _, sp := range ch.Spells {
		if err := casa.LearnSpell(ctx, id, sp.ID, sp.Prepared); err != nil {
			log.Printf("personagem %d, magia %q: %v", id, sp.ID, err)
		}
	}
	if ch.HpFraction != nil || ch.SceneEffect {
		if err := enrichLiveState(ctx, casa, id, ch); err != nil {
			log.Printf("personagem %d, estado de jogo: %v", id, err)
		}
	}
	return nil
}

// enrichCreate fills the create body with data derived from the catalog + roster
// flags: vitals the engine will heal, the standard trained perícias (non-simple),
// and each item's catalog name + slot cost.
func enrichCreate(ch seedCharacter) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(ch.Create, &obj); err != nil {
		return nil, fmt.Errorf("create body: %w", err)
	}
	// healVitals recomputes the real maxes from the engine, so pass a value it can
	// only clamp down to full. Damaged bars are set afterwards via /vitals.
	for _, field := range []string{"hpMax", "hpCurrent", "mpMax", "mpCurrent"} {
		obj[field] = json.RawMessage("9999")
	}
	if !ch.Simple {
		if _, ok := obj["trainedExpertises"]; !ok {
			trained, _ := json.Marshal(standardTrained)
			obj["trainedExpertises"] = trained
		}
	}
	if err := resolveItemMetadata(obj); err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

// resolveItemMetadata fills each item's name + slots from the catalog so the
// roster references items by catalogId + quantity + equipped alone.
func resolveItemMetadata(obj map[string]json.RawMessage) error {
	raw, ok := obj["items"]
	if !ok {
		return nil
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("items: %w", err)
	}
	for _, it := range items {
		id, _ := it["catalogId"].(string)
		meta, known := catalog.LookupItem(id)
		if !known {
			return fmt.Errorf("unknown catalog item %q", id)
		}
		it["name"] = meta.Name
		it["slots"] = meta.Slots
	}
	enriched, err := json.Marshal(items)
	if err != nil {
		return err
	}
	obj["items"] = enriched
	return nil
}

func enrichLiveState(ctx context.Context, casa casaDaSeed, id int64, ch seedCharacter) error {
	ficha, err := casa.Character(ctx, id)
	if err != nil {
		return err
	}
	if ch.HpFraction != nil {
		// O PV MÁXIMO é o que o motor calculou, e por isso ele é lido de volta:
		// o corpo de criação manda 9999 nos quatro vitais justamente para a cura
		// aparar para o número certo.
		pv := int64(float64(ficha.HpMax)**ch.HpFraction + 0.5)
		if err := casa.SetHp(ctx, id, pv); err != nil {
			return err
		}
	}
	if ch.SceneEffect {
		return applySceneEffect(ctx, casa, id, ficha.Items)
	}
	return nil
}

func applySceneEffect(ctx context.Context, casa casaDaSeed, id int64, itens []sheet.ItemDTO) error {
	for _, it := range itens {
		if it.CatalogID == nil || *it.CatalogID != sceneConsumable {
			continue
		}
		return casa.ConsumeItem(ctx, id, it.ID)
	}
	return nil
}
