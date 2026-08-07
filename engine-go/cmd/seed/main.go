// Command seed populates a running Go API with the test table by driving its
// HTTP endpoints (register → create → learn/prepare spells → damage/consume) —
// reusing every handler's validation + engine integration, no logic duplicated.
// The roster lives in the embedded seed-data.json and mirrors the Nest seed
// (backend/src/seed.ts): 3 accounts, 15 diverse characters. Run against a live
// server:
//
//	PORT=3001 JWT_SECRET=dev go run ./cmd/api        # in one shell
//	SEED_API_URL=http://localhost:3001 go run ./cmd/seed
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"os"

	"t20engine/catalog"
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

// standardTrained is TRAINED_EXPERTISES from the Nest seed — every non-simple
// character trains these, giving the skill list real totals (treino + ½ nível +
// atributo). Simple starter PCs train nothing.
var standardTrained = []string{
	"Luta", "Atletismo", "Pontaria", "Reflexos", "Fortitude",
	"Vontade", "Percepção", "Intimidação", "Investigação", "Misticismo",
}

// sceneConsumable is the item whose scene effect a sceneEffect character carries.
const sceneConsumable = "cosmetico"

func main() {
	base := env("SEED_API_URL", "http://localhost:3001")
	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		log.Fatalf("seed-data.json: %v", err)
	}
	total, seeded := 0, 0
	for _, u := range sf.Users {
		total += len(u.Characters)
		seeded += seedUserCharacters(base, sf.Password, u)
	}
	log.Printf("done: %d/%d characters across %d users", seeded, total, len(sf.Users))
}

// seedUserCharacters authenticates one account and seeds its roster, returning
// how many characters were created.
func seedUserCharacters(base, password string, u seedUser) int {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	if err := authenticate(client, base, u.Email, u.Name, password); err != nil {
		log.Printf("auth %s: %v", u.Email, err)
		return 0
	}
	log.Printf("authenticated as %s (%d characters)", u.Email, len(u.Characters))

	seeded := 0
	for _, ch := range u.Characters {
		if err := seedCharacterRow(client, base, ch); err != nil {
			log.Printf("%s: %v", u.Email, err)
			continue
		}
		seeded++
	}
	return seeded
}

// seedCharacterRow creates one character then applies its spells, damaged HP,
// and scene effect — mirroring enrichCharacter in the Nest seed.
func seedCharacterRow(c *http.Client, base string, ch seedCharacter) error {
	body, err := enrichCreate(ch)
	if err != nil {
		return err
	}
	id, err := createCharacter(c, base, body)
	if err != nil {
		return err
	}
	for _, sp := range ch.Spells {
		if err := learnSpell(c, base, id, sp); err != nil {
			log.Printf("character %d spell %q: %v", id, sp.ID, err)
		}
	}
	if ch.HpFraction != nil || ch.SceneEffect {
		if err := enrichLiveState(c, base, id, ch); err != nil {
			log.Printf("character %d live-state: %v", id, err)
		}
	}
	log.Printf("seeded character %d", id)
	return nil
}

// enrichCreate fills the create body with data derived from the catalog + roster
// flags: vitals the engine will heal, the standard trained perícias (non-simple),
// and each item's catalog name + slot cost. Keeps seed-data.json free of data
// the catalog already owns.
func enrichCreate(ch seedCharacter) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(ch.Create, &obj); err != nil {
		return nil, fmt.Errorf("create body: %w", err)
	}
	// healVitals recomputes the real maxes from the engine, so pass a value it
	// can only clamp down to full. Damaged bars are set afterwards via /vitals.
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

// resolveItemMetadata fills each item's name + slots from the catalog (source of
// truth) so the roster references items by catalogId + quantity + equipped alone.
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

// enrichLiveState GETs the created character once, then damages it to hpFraction
// and/or consumes a scene catalisador so its sheet carries a live ActiveEffect.
func enrichLiveState(c *http.Client, base string, id int64, ch seedCharacter) error {
	char, err := getCharacter(c, base, id)
	if err != nil {
		return err
	}
	if ch.HpFraction != nil {
		hp := int64(math.Round(float64(char.HpMax) * *ch.HpFraction))
		if err := patchVitals(c, base, id, hp); err != nil {
			return err
		}
	}
	if ch.SceneEffect {
		if err := applySceneEffect(c, base, id, char.Items); err != nil {
			return err
		}
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

func getCharacter(c *http.Client, base string, id int64) (characterState, error) {
	var out characterState
	status, respBody, err := do(c, http.MethodGet, fmt.Sprintf("%s/characters/%d", base, id), nil)
	if err != nil {
		return out, err
	}
	if status != http.StatusOK {
		return out, fmt.Errorf("get character status %d: %s", status, respBody)
	}
	return out, json.Unmarshal(respBody, &out)
}

func patchVitals(c *http.Client, base string, id, hpCurrent int64) error {
	url := fmt.Sprintf("%s/characters/%d/vitals", base, id)
	if s, b, err := do(c, http.MethodPatch, url, mustJSON(map[string]int64{"hpCurrent": hpCurrent})); err != nil {
		return err
	} else if s != http.StatusOK {
		return fmt.Errorf("patch vitals status %d: %s", s, b)
	}
	return nil
}

// applySceneEffect consumes the first scene catalisador on hand, matching the
// Nest seed's applySceneEffect (idempotent no-op when the potion is absent).
func applySceneEffect(c *http.Client, base string, id int64, items []charItem) error {
	for _, it := range items {
		if it.CatalogID == nil || *it.CatalogID != sceneConsumable {
			continue
		}
		url := fmt.Sprintf("%s/characters/%d/items/%d/consume", base, id, it.ID)
		if s, b, err := do(c, http.MethodPost, url, []byte("{}")); err != nil {
			return err
		} else if s != http.StatusOK {
			return fmt.Errorf("consume status %d: %s", s, b)
		}
		return nil
	}
	return nil
}

// authenticate registers the seed user, falling back to login if it exists.
func authenticate(c *http.Client, base, email, name, password string) error {
	body := map[string]string{"email": email, "password": password, "name": name}
	status, respBody, err := do(c, http.MethodPost, base+"/auth/register", mustJSON(body))
	if err != nil {
		return err
	}
	switch status {
	case http.StatusCreated:
		return nil
	case http.StatusConflict:
		creds := map[string]string{"email": email, "password": password}
		s, b, err := do(c, http.MethodPost, base+"/auth/login", mustJSON(creds))
		if err != nil {
			return err
		}
		if s != http.StatusOK {
			return fmt.Errorf("login status %d: %s", s, b)
		}
		return nil
	default:
		return fmt.Errorf("register status %d: %s", status, respBody)
	}
}

func createCharacter(c *http.Client, base string, createBody json.RawMessage) (int64, error) {
	status, respBody, err := do(c, http.MethodPost, base+"/characters", createBody)
	if err != nil {
		return 0, err
	}
	if status != http.StatusCreated {
		return 0, fmt.Errorf("create status %d: %s", status, respBody)
	}
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return 0, err
	}
	return out.ID, nil
}

func learnSpell(c *http.Client, base string, id int64, sp seedSpell) error {
	learnURL := fmt.Sprintf("%s/characters/%d/spells", base, id)
	if s, _, err := do(c, http.MethodPost, learnURL, mustJSON(map[string]string{"catalogSpellId": sp.ID})); err != nil {
		return err
	} else if s != http.StatusCreated && s != http.StatusConflict {
		return fmt.Errorf("learn status %d", s)
	}
	if !sp.Prepared {
		return nil
	}
	prepURL := fmt.Sprintf("%s/characters/%d/spells/%s/prepared", base, id, sp.ID)
	s, _, err := do(c, http.MethodPatch, prepURL, mustJSON(map[string]bool{"prepared": true}))
	if err != nil {
		return err
	}
	if s != http.StatusOK {
		return fmt.Errorf("prepare status %d", s)
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func do(c *http.Client, method, url string, body []byte) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
