package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// This file ports the catalog-reading layer the collection engine depends on:
// the SHAPES of the fetched catalogs (items, races, origins, class/general
// powers, racas, tormenta-power ids) plus the sync lookups derived.ts reaches
// through the frontend *-cache modules (getCatalogItem, getRace, getOrigin,
// getOriginBenefit, getClassPower, getGeneralPower, ownedClassPowers,
// raceWithDeformidade, racaByName). Data comes from the same JSON the front
// fetches — primed ONCE via PrimeEngineCatalogs. See PORT-PLAN.md §2.

// ─── Catalog shapes (mirror the TS catalog types) ─────────────────────

// CatalogItem mirrors items/types.ts CatalogItem — only the fields the
// collection layer reads are typed.
type CatalogItem struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Category  string       `json:"category"`
	Weapon    *WeaponStats `json:"weapon,omitempty"`
	Modifiers []Modifier   `json:"modifiers"`
}

type WeaponStats struct {
	Purpose string   `json:"purpose"` // 'melee' | 'thrown' | 'ranged'
	Traits  []string `json:"traits"`
}

// RaceDefinition mirrors abilities/types.ts RaceDefinition (the abilities
// catalog race, distinct from the racas.ts Raca below).
type RaceDefinition struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	AttributeBonuses map[string]int `json:"attributeBonuses"`
	Abilities        []RaceAbility  `json:"abilities"`
	HasDeformidade   bool           `json:"hasDeformidade"`
}

type RaceAbility struct {
	Modifiers []Modifier           `json:"modifiers"`
	Variants  []RaceAbilityVariant `json:"variants"`
}

type RaceAbilityVariant struct {
	ID        string     `json:"id"`
	Modifiers []Modifier `json:"modifiers"`
}

// OriginDefinition mirrors abilities/types.ts OriginDefinition.
type OriginDefinition struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Benefits   []OriginBenefit `json:"benefits"`
	PoderUnico OriginBenefit   `json:"poderUnico"`
}

type OriginBenefit struct {
	ID        string     `json:"id"`
	Modifiers []Modifier `json:"modifiers"`
	PowerPick string     `json:"powerPick,omitempty"` // 'combate' | 'tormenta'
}

// ClassPower mirrors abilities/types.ts ClassPower (collection-relevant fields).
type ClassPower struct {
	ID              string           `json:"id"`
	ClassName       string           `json:"className"`
	Name            string           `json:"name"`
	GrantedAtLevel  *int             `json:"grantedAtLevel"`
	GrantedByChoice *GrantedByChoice `json:"grantedByChoice"`
	Modifiers       []Modifier       `json:"modifiers"`
}

type GrantedByChoice struct {
	Field string `json:"field"` // 'devoto' | 'caminho'
	Value string `json:"value"`
}

// GeneralPower mirrors abilities/general-powers.ts GeneralPower.
type GeneralPower struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Modifiers []Modifier `json:"modifiers"`
}

// Raca mirrors racas.ts Raca (only name + atributoMod are read here).
type Raca struct {
	Name        string      `json:"name"`
	AtributoMod AtributoMod `json:"atributoMod"`
}

// AtributoMod is the racas.ts AtributoMod union flattened by `kind`. `Mods` and
// `Variants` use orderedInts because raceAttributeMods emits one modifier per
// entry in the SOURCE object's key order (Object.entries) — a plain map would
// scramble it and break activeItems parity.
type AtributoMod struct {
	Kind     string                 `json:"kind"` // fixed | floating | subraca-gated
	Mods     orderedInts            `json:"mods"`
	Count    int                    `json:"count"`
	Value    int                    `json:"value"`
	Exclude  string                 `json:"exclude"`
	Penalty  *AtributoPenalty       `json:"penalty"`
	Variants map[string]orderedInts `json:"variants"`
}

type AtributoPenalty struct {
	Attribute string `json:"attribute"`
	Value     int    `json:"value"`
}

// attrDelta is one attribute→amount entry, kept in an ordered slice.
type attrDelta struct {
	attr   string
	amount int
}

// orderedInts is a JSON object of int values that preserves key order on decode
// (Go maps don't). Used for atributoMod's mods/variants so the derived attribute
// modifiers land in the same order as the TS Object.entries iteration.
type orderedInts struct {
	pairs []attrDelta
}

func (o *orderedInts) UnmarshalJSON(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil { // opening '{'
		return err
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return err
		}
		var v int
		if err := dec.Decode(&v); err != nil {
			return err
		}
		o.pairs = append(o.pairs, attrDelta{attr: key.(string), amount: v})
	}
	_, err := dec.Token() // closing '}'
	return err
}

// ─── Catalogs holder + priming ────────────────────────────────────────

// Catalogs holds every primed catalog the collection layer reads. Injected into
// ActiveItemsFor instead of the TS module-level caches (CLAUDE.md: deps by
// parameter). Static + primed once, so no reactivity is needed.
type Catalogs struct {
	itemsByID      map[string]*CatalogItem
	racesByID      map[string]*RaceDefinition
	origins        []*OriginDefinition
	classPowers    []*ClassPower
	classPowerByID map[string]*ClassPower
	generalByID    map[string]*GeneralPower
	racasByName    map[string]*Raca
	tormentaIDs    map[string]bool
}

// enginePayload is the JSON shape the frontend GEN_ORACLE harness dumps to
// engine-go/parity/_catalogs.json (mirrors ensureCatalogs' priming inputs).
type enginePayload struct {
	Items         []CatalogItem      `json:"items"`
	Races         []RaceDefinition   `json:"races"`
	Origins       []OriginDefinition `json:"origins"`
	ClassPowers   []ClassPower       `json:"classPowers"`
	GeneralPowers []GeneralPower     `json:"generalPowers"`
	Racas         map[string]Raca    `json:"racas"`
	TormentaIDs   []string           `json:"tormentaPowerIds"`
}

// PrimeEngineCatalogs ingests the fetched-catalog JSON (the same data the front
// primes via ensureCatalogs) into an indexed Catalogs. Returns an error with the
// offending shape rather than panicking, so the WASM boundary can surface it.
func PrimeEngineCatalogs(raw []byte) (*Catalogs, error) {
	var p enginePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("PrimeEngineCatalogs: bad catalog JSON: %w", err)
	}
	c := &Catalogs{
		itemsByID:      make(map[string]*CatalogItem, len(p.Items)),
		racesByID:      make(map[string]*RaceDefinition, len(p.Races)),
		classPowerByID: make(map[string]*ClassPower, len(p.ClassPowers)),
		generalByID:    make(map[string]*GeneralPower, len(p.GeneralPowers)),
		racasByName:    make(map[string]*Raca, len(p.Racas)),
		tormentaIDs:    make(map[string]bool, len(p.TormentaIDs)),
	}
	for i := range p.Items {
		c.itemsByID[p.Items[i].ID] = &p.Items[i]
	}
	for i := range p.Races {
		c.racesByID[p.Races[i].ID] = &p.Races[i]
	}
	for i := range p.Origins {
		c.origins = append(c.origins, &p.Origins[i])
	}
	for i := range p.ClassPowers {
		c.classPowers = append(c.classPowers, &p.ClassPowers[i])
		c.classPowerByID[p.ClassPowers[i].ID] = &p.ClassPowers[i]
	}
	for i := range p.GeneralPowers {
		c.generalByID[p.GeneralPowers[i].ID] = &p.GeneralPowers[i]
	}
	for id := range p.Racas {
		r := p.Racas[id]
		c.racasByName[r.Name] = &r
	}
	for _, id := range p.TormentaIDs {
		c.tormentaIDs[id] = true
	}
	return c, nil
}

// ─── Lookups (mirror the frontend *-cache accessors) ──────────────────

func (c *Catalogs) getCatalogItem(id string) *CatalogItem { return c.itemsByID[id] }

func (c *Catalogs) getRace(id string) *RaceDefinition { return c.racesByID[id] }

func (c *Catalogs) getClassPower(id string) *ClassPower { return c.classPowerByID[id] }

func (c *Catalogs) getGeneralPower(id string) *GeneralPower { return c.generalByID[id] }

// getOrigin looks up an origin by id (abilities-cache getOrigin).
func (c *Catalogs) getOrigin(id string) *OriginDefinition {
	for _, o := range c.origins {
		if o.ID == id {
			return o
		}
	}
	return nil
}

// getOriginBenefit finds a benefit across all origins, including the poder único
// — mirrors abilities-cache.getOriginBenefit.
func (c *Catalogs) getOriginBenefit(benefitID string) *OriginBenefit {
	for _, o := range c.origins {
		for i := range o.Benefits {
			if o.Benefits[i].ID == benefitID {
				return &o.Benefits[i]
			}
		}
		if o.PoderUnico.ID == benefitID {
			return &o.PoderUnico
		}
	}
	return nil
}

// racaByName finds a racas.ts Raca by name (derived.ts racaByName over
// racasList) — 17 racas, linear scan is cheap.
func (c *Catalogs) racaByName(name string) *Raca { return c.racasByName[name] }

// raceWithDeformidade returns the first name that owns Deformidade (Lefou p23) —
// mirrors abilities-cache.raceWithDeformidade.
func (c *Catalogs) raceWithDeformidade(names ...string) string {
	owners := map[string]bool{}
	for _, r := range c.racesByID {
		if r.HasDeformidade {
			owners[r.Name] = true
		}
	}
	for _, n := range names {
		if owners[n] {
			return n
		}
	}
	return ""
}

// isTormentaPower mirrors the derived.ts `id in tormentaPowersRecord()` check.
func (c *Catalogs) isTormentaPower(id string) bool { return c.tormentaIDs[id] }

// ownedClassPowers mirrors abilities-cache.ownedClassPowers → ownedClassPowersIn:
// every class power owned for a class + level + chosen ids + choices, in catalog
// order (so downstream ActiveItem order is deterministic).
func (c *Catalogs) ownedClassPowers(
	className string,
	classLevel int,
	chosen map[string]bool,
	choice ClassChoiceSelections,
) []*ClassPower {
	out := []*ClassPower{}
	for _, power := range c.classPowers {
		if power.ClassName != className {
			continue
		}
		if ownsClassPower(power, classLevel, chosen, choice) {
			out = append(out, power)
		}
	}
	return out
}
