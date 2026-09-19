// Command seed regenera o `engine-go/seed.sql` — um despejo em SQL puro do
// conjunto de desenvolvimento (3 contas, o elenco variado de teste, crônicas de
// demonstração) que se aplica na hora com `sqlite3 data/t20-dev.db < seed.sql`,
// sem servidor nenhum.
//
// Ele monta o dado chamando as REGRAS do app num banco migrado descartável — os
// hashes de bcrypt, os vitais computados pelo motor e o leque normalizado vêm do
// mesmo código que o servidor roda, nunca mantidos à mão — e despeja o banco em
// SQL. O elenco mora no `seed-data.json` embutido, que é a fonte legível.
//
// Ele pede o que precisa pela `api.Seeder`, a porta declarada logo abaixo. Pedir
// por CAMINHO EM STRING já fez este gerador parar de rodar sem que nada
// acusasse, no dia em que as rotas que ele dirigia foram apagadas por não terem
// consumidor.
//
// Regenerar depois de mudar elenco, regra ou crônica:
//
//	go run ./cmd/seed            # escreve ./seed.sql (a partir de engine-go/)
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

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/config"
	"t20engine/infra/db"
	"t20engine/serve/api"
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

// standardTrained são as perícias que todo personagem não-simples treina, para a
// lista de perícias ter totais de verdade. Personagem simples não treina nenhuma.
var standardTrained = []string{
	"Luta", "Atletismo", "Pontaria", "Reflexos", "Fortitude",
	"Vontade", "Percepção", "Intimidação", "Investigação", "Misticismo",
}

// sceneConsumable é o item cujo efeito de cena um personagem `sceneEffect` leva.
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
	// ANTES DE O SERVIDOR SUBIR: nenhuma linha vai para o banco com referência
	// quebrada. Um id de catálogo errado não quebra nada — ele produz um
	// personagem QUASE certo, e o e2e roda contra a seed.
	if err := validateCatalogRefs(sf); err != nil {
		log.Fatalf("%v", err)
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

// seedEmails são as contas que esta rodada cria. Elas entram como ADMIN_EMAILS
// para o registro funcionar: o registro exige convite, e a primeira conta de um
// banco vazio não tem quem a tivesse convidado — o gerador é o próprio admin dele.
// Nada do papel chega ao seed.sql: ele sai do ambiente na hora do pedido e não
// tem coluna.
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
	cfg, err := config.LoadConfig()
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

// enrichCreate completa o corpo de criação com o que sai do catálogo e das
// marcas do elenco: os vitais que o motor vai curar, as perícias treinadas
// padrão (não-simples) e o nome e o custo de espaço de cada item.
func enrichCreate(ch seedCharacter) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(ch.Create, &obj); err != nil {
		return nil, fmt.Errorf("create body: %w", err)
	}
	// Aqui moravam quatro vitais escritos com 9999, para uma cura que os aparava
	// para baixo. Os dois sumiram: o poço é derivado do catálogo a cada leitura e
	// o corpo de criação nem tem mais esses campos. Barra danificada vem depois,
	// pelo `HpFraction` (ALE-355).
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

// resolveItemMetadata completa nome e espaços de cada item a partir do catálogo,
// para o elenco referenciar item só por catalogId, quantidade e equipado.
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
