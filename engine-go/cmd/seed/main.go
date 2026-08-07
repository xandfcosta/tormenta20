// Command seed populates a running Go API with test data by driving its HTTP
// endpoints (register → create → learn/prepare spells) — reusing every handler's
// validation + engine integration, no logic duplicated. Data lives in the
// embedded seed-data.json; append characters there to grow the seed toward the
// full 16. Run against a live server:
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
	"net/http"
	"net/http/cookiejar"
	"os"
)

//go:embed seed-data.json
var seedData []byte

type seedFile struct {
	Email      string          `json:"email"`
	Name       string          `json:"name"`
	Password   string          `json:"password"`
	Characters []seedCharacter `json:"characters"`
}

type seedCharacter struct {
	Create json.RawMessage `json:"create"`
	Spells []seedSpell     `json:"spells"`
}

type seedSpell struct {
	ID       string `json:"id"`
	Prepared bool   `json:"prepared"`
}

func main() {
	base := env("SEED_API_URL", "http://localhost:3001")
	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		log.Fatalf("seed-data.json: %v", err)
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	if err := authenticate(client, base, sf); err != nil {
		log.Fatalf("auth: %v", err)
	}
	log.Printf("authenticated as %s", sf.Email)

	seeded := 0
	for _, ch := range sf.Characters {
		id, err := createCharacter(client, base, ch.Create)
		if err != nil {
			log.Printf("create failed: %v", err)
			continue
		}
		for _, sp := range ch.Spells {
			if err := learnSpell(client, base, id, sp); err != nil {
				log.Printf("character %d spell %q: %v", id, sp.ID, err)
			}
		}
		seeded++
		log.Printf("seeded character %d", id)
	}
	log.Printf("done: %d/%d characters", seeded, len(sf.Characters))
}

// authenticate registers the seed user, falling back to login if it exists.
func authenticate(c *http.Client, base string, sf seedFile) error {
	body := map[string]string{"email": sf.Email, "password": sf.Password, "name": sf.Name}
	status, respBody, err := post(c, base+"/auth/register", body)
	if err != nil {
		return err
	}
	switch status {
	case http.StatusCreated:
		return nil
	case http.StatusConflict:
		s, b, err := post(c, base+"/auth/login", map[string]string{"email": sf.Email, "password": sf.Password})
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
	status, respBody, err := postRaw(c, base+"/characters", createBody)
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
	if s, _, err := post(c, fmt.Sprintf("%s/characters/%d/spells", base, id), map[string]string{"catalogSpellId": sp.ID}); err != nil {
		return err
	} else if s != http.StatusCreated && s != http.StatusConflict {
		return fmt.Errorf("learn status %d", s)
	}
	if !sp.Prepared {
		return nil
	}
	s, _, err := patch(c, fmt.Sprintf("%s/characters/%d/spells/%s/prepared", base, id, sp.ID), map[string]bool{"prepared": true})
	if err != nil {
		return err
	}
	if s != http.StatusOK {
		return fmt.Errorf("prepare status %d", s)
	}
	return nil
}

func post(c *http.Client, url string, body any) (int, []byte, error) {
	b, _ := json.Marshal(body)
	return postRaw(c, url, b)
}

func patch(c *http.Client, url string, body any) (int, []byte, error) {
	b, _ := json.Marshal(body)
	return do(c, http.MethodPatch, url, b)
}

func postRaw(c *http.Client, url string, body []byte) (int, []byte, error) {
	return do(c, http.MethodPost, url, body)
}

func do(c *http.Client, method, url string, body []byte) (int, []byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
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
