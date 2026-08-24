// Command seed regenerates engine-go/seed.sql — a pure-SQL dump of the dev
// dataset (3 accounts, the diverse test roster, demo chronicles) that applies
// instantly with `sqlite3 data/t20-dev.db < seed.sql`, no API server (ALE-57).
//
// It builds the data by driving the REAL HTTP handlers IN-PROCESS (httptest, no
// network) into a throwaway migrated DB — so bcrypt hashes, engine-computed
// vitals and the normalized fan-out come from the same code the API runs, never
// hand-maintained — then dumps that DB to SQL. The roster lives in the embedded
// seed-data.json (readable source of truth). Regenerate after roster/rule/
// chronicle changes:
//
//	go run ./cmd/seed            # writes ./seed.sql (from the engine-go dir)
package main

import (
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"t20engine/api"
	"t20engine/catalog"
	"t20engine/db"
	"t20engine/engine"
	"t20engine/plataforma"
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
	handler, database, cleanup := freshServer(seedEmails(sf))
	defer cleanup()
	total, seeded := 0, 0
	for _, u := range sf.Users {
		total += len(u.Characters)
		seeded += seedUserCharacters(handler, sf.Password, u)
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

// freshServer boots the real API against a throwaway migrated SQLite DB and
// returns its in-process handler plus the DB (for the chronicle seed + dump).
func freshServer(adminEmails []string) (http.Handler, *sql.DB, func()) {
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
	return srv.Router(), database, cleanup
}

// ── in-process HTTP client (no network) ──────────────────────────────────────

// client drives the API handler directly via httptest, tracking the session
// cookie between calls so an authenticated flow works exactly as over the wire.
type client struct {
	h       http.Handler
	cookies map[string]*http.Cookie
}

func newClient(h http.Handler) *client {
	return &client{h: h, cookies: map[string]*http.Cookie{}}
}

func (c *client) do(method, path string, body []byte) (int, []byte) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	for _, ck := range rec.Result().Cookies() {
		c.cookies[ck.Name] = ck
	}
	return rec.Code, rec.Body.Bytes()
}

// ── seeding via the real handlers ────────────────────────────────────────────

func seedUserCharacters(h http.Handler, password string, u seedUser) int {
	c := newClient(h)
	if err := authenticate(c, u.Email, u.Name, password); err != nil {
		log.Printf("auth %s: %v", u.Email, err)
		return 0
	}
	seeded := 0
	for _, ch := range u.Characters {
		if err := seedCharacterRow(c, ch); err != nil {
			log.Printf("%s: %v", u.Email, err)
			continue
		}
		seeded++
	}
	return seeded
}

func seedCharacterRow(c *client, ch seedCharacter) error {
	body, err := enrichCreate(ch)
	if err != nil {
		return err
	}
	id, err := createCharacter(c, body)
	if err != nil {
		return err
	}
	for _, sp := range ch.Spells {
		if err := learnSpell(c, id, sp); err != nil {
			log.Printf("character %d spell %q: %v", id, sp.ID, err)
		}
	}
	if ch.HpFraction != nil || ch.SceneEffect {
		if err := enrichLiveState(c, id, ch); err != nil {
			log.Printf("character %d live-state: %v", id, err)
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

func enrichLiveState(c *client, id int64, ch seedCharacter) error {
	char, err := getCharacter(c, id)
	if err != nil {
		return err
	}
	if ch.HpFraction != nil {
		hp := int64(float64(char.HpMax)**ch.HpFraction + 0.5)
		if err := patchVitals(c, id, hp); err != nil {
			return err
		}
	}
	if ch.SceneEffect {
		return applySceneEffect(c, id, char.Items)
	}
	return nil
}

type charItem struct {
	ID        int64   `json:"id"`
	CatalogID *string `json:"catalogId"`
}

type characterState struct {
	HpMax int64      `json:"hpMax"`
	Items []charItem `json:"items"`
}

func getCharacter(c *client, id int64) (characterState, error) {
	var out characterState
	status, body := c.do(http.MethodGet, fmt.Sprintf("/characters/%d", id), nil)
	if status != http.StatusOK {
		return out, fmt.Errorf("get character status %d: %s", status, body)
	}
	return out, json.Unmarshal(body, &out)
}

func patchVitals(c *client, id, hpCurrent int64) error {
	status, body := c.do(http.MethodPatch, fmt.Sprintf("/characters/%d/vitals", id), mustJSON(map[string]int64{"hpCurrent": hpCurrent}))
	if status != http.StatusOK {
		return fmt.Errorf("patch vitals status %d: %s", status, body)
	}
	return nil
}

func applySceneEffect(c *client, id int64, items []charItem) error {
	for _, it := range items {
		if it.CatalogID == nil || *it.CatalogID != sceneConsumable {
			continue
		}
		status, body := c.do(http.MethodPost, fmt.Sprintf("/characters/%d/items/%d/consume", id, it.ID), []byte("{}"))
		if status != http.StatusOK {
			return fmt.Errorf("consume status %d: %s", status, body)
		}
		return nil
	}
	return nil
}

func authenticate(c *client, email, name, password string) error {
	status, body := c.do(http.MethodPost, "/auth/register", mustJSON(map[string]string{"email": email, "password": password, "name": name}))
	if status == http.StatusCreated {
		return nil
	}
	return fmt.Errorf("register status %d: %s", status, body)
}

func createCharacter(c *client, createBody json.RawMessage) (int64, error) {
	status, body := c.do(http.MethodPost, "/characters", createBody)
	if status != http.StatusCreated {
		return 0, fmt.Errorf("create status %d: %s", status, body)
	}
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

func learnSpell(c *client, id int64, sp seedSpell) error {
	if s, b := c.do(http.MethodPost, fmt.Sprintf("/characters/%d/spells", id), mustJSON(map[string]string{"catalogSpellId": sp.ID})); s != http.StatusCreated && s != http.StatusConflict {
		return fmt.Errorf("learn status %d: %s", s, b)
	}
	if !sp.Prepared {
		return nil
	}
	if s, b := c.do(http.MethodPatch, fmt.Sprintf("/characters/%d/spells/%s/prepared", id, sp.ID), mustJSON(map[string]bool{"prepared": true})); s != http.StatusOK {
		return fmt.Errorf("prepare status %d: %s", s, b)
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
