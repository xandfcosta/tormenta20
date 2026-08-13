// Command genoracle escreve os oráculos de paridade a partir do motor Go.
//
// Uso: cd engine-go && go run ./cmd/genoracle
//
// Substitui o harness `GEN_ORACLE=1` do front, que os gerava pela implementação
// TS de referência. Com o `t20-data` aposentado não há segunda implementação, e
// o oráculo passa a ser o Go descrevendo o Go — por isso a regra de processo do
// CLAUDE.md deste pacote vale mais do que nunca:
//
//	O diff de um oráculo é revisado contra o LIVRO, nunca aceito porque "o
//	teste ficou verde".
//
// O que o oráculo protege continua sendo enorme: é a ficha inteira de 18
// personagens, ponta a ponta, e ele acusa qualquer mudança de número que não
// tenha sido pedida. O que ele deixa de provar é que DOIS motores concordam,
// porque só existe um.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"t20engine/engine"
)

type fixture struct {
	Slug string `json:"slug"`
	// Raw é o personagem VERBATIM. O `engine.Character` é um espelho PARCIAL de
	// propósito ("only the fields the collection layer touches are typed"), então
	// passar a fixture por ele e re-serializar DESCARTA campos que o oráculo
	// guardava — ownerId, size, mpMax, updatedAt, os timestamps dos efeitos. O
	// bloco `char` sai como entrou; só o que é CALCULADO é recalculado.
	Raw  json.RawMessage  `json:"char"`
	Char engine.Character `json:"-"`
}

// oraclePayload é a mesma forma que o harness do front escrevia — a ordem dos
// campos segue a do JSON commitado para o diff continuar legível.
type oraclePayload struct {
	Slug                        string   `json:"slug"`
	Char                        any      `json:"char"`
	ActiveItems                 any      `json:"activeItems"`
	ItemEffects                 any      `json:"itemEffects"`
	SheetV2                     any      `json:"sheetV2"`
	ActiveConditionals          []string `json:"activeConditionals"`
	SheetV2WithConditionals     any      `json:"sheetV2WithConditionals"`
	WeaponCardsWithConditionals any      `json:"weaponCardsWithConditionals"`
	Vitals                      any      `json:"vitals"`
	EquippedFlags               any      `json:"equippedFlags"`
	WeaponCards                 any      `json:"weaponCards"`
}

func main() {
	dir, err := parityDir()
	if err != nil {
		fail(err)
	}
	catalogs, err := primeCatalogs(dir)
	if err != nil {
		fail(err)
	}
	fixtures, err := readFixtures(dir)
	if err != nil {
		fail(err)
	}

	for _, f := range fixtures {
		payload, err := buildPayload(catalogs, f)
		if err != nil {
			fail(fmt.Errorf("%s: %w", f.Slug, err))
		}
		body, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fail(fmt.Errorf("%s: serializar: %w", f.Slug, err))
		}
		out := filepath.Join(dir, f.Slug+".json")
		if err := os.WriteFile(out, append(body, '\n'), 0o644); err != nil {
			fail(fmt.Errorf("escrever %s: %w", out, err))
		}
	}
	fmt.Printf("%d oráculos escritos em %s\n", len(fixtures), dir)
	fmt.Println("REVISE O DIFF CONTRA O LIVRO — verde não é prova de que o número está certo.")
}

// buildPayload calcula tudo o que o oráculo guarda para um personagem.
func buildPayload(c *engine.Catalogs, f fixture) (oraclePayload, error) {
	none := map[string]bool{}
	effects := engine.ComputeItemEffects(c.ActiveItemsFor(f.Char))

	// Todo opt-in que este personagem PODERIA ligar, em ordem estável — é a
	// segunda passada que exercita a dobra dos condicionais (ALE-106).
	ids := make([]string, 0, len(effects.Conditional))
	for _, cond := range effects.Conditional {
		ids = append(ids, engine.ConditionalID(cond))
	}
	sort.Strings(ids)
	on := map[string]bool{}
	for _, id := range ids {
		on[id] = true
	}

	return oraclePayload{
		Slug:                        f.Slug,
		Char:                        f.Raw,
		ActiveItems:                 c.ActiveItemsFor(f.Char),
		ItemEffects:                 effects,
		SheetV2:                     c.ComputeSheetV2(f.Char, none),
		ActiveConditionals:          ids,
		SheetV2WithConditionals:     c.ComputeSheetV2(f.Char, on),
		WeaponCardsWithConditionals: c.ComputeWeaponCards(f.Char, on),
		Vitals:                      c.ComputeVitals(c.VitalContextFor(f.Char)),
		EquippedFlags:               c.ComputeEquippedFlags(f.Char.Items),
		WeaponCards:                 c.ComputeWeaponCards(f.Char, none),
	}, nil
}

func readFixtures(dir string) ([]fixture, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "_fixtures.json"))
	if err != nil {
		return nil, fmt.Errorf("ler as fixtures: %w", err)
	}
	var out []fixture
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("fixtures ilegíveis: %w", err)
	}
	for i := range out {
		if err := json.Unmarshal(out[i].Raw, &out[i].Char); err != nil {
			return nil, fmt.Errorf("%s: personagem ilegível: %w", out[i].Slug, err)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("_fixtures.json não trouxe personagem nenhum")
	}
	return out, nil
}

func primeCatalogs(dir string) (*engine.Catalogs, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "_catalogs.json"))
	if err != nil {
		return nil, fmt.Errorf("ler os catálogos: %w", err)
	}
	return engine.PrimeEngineCatalogs(raw)
}

func parityDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; {
		candidate := filepath.Join(dir, "engine-go", "parity")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("genoracle: não achei engine-go/parity subindo de %q", cwd)
		}
		dir = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
