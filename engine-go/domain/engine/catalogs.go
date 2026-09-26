package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// A camada de leitura de catálogo de que o motor depende: a FORMA de cada
// catálogo (itens, raças, origens, poderes de classe e gerais, ids de poder da
// Tormenta) mais as consultas sobre eles.
//
// O dado é primado UMA vez pelo `PrimeEngineCatalogs` e entregue pelo receptor
// `Catalogs` — nunca lido de um cache no nível do pacote, para um teste poder
// primar o próprio sem tocar em estado global.

// ─── O formato que o despejo de catálogo traz ────────────────────────────────

// CatalogItem é a entrada de item: só os campos que esta camada lê são
// tipados.
type CatalogItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	// Equip, Slots e AppliesTo alimentam os validadores de mutação da API (eixo
	// de equipar, carga, compatibilidade de sobreposição); a derivação os
	// ignora.
	Equip     string       `json:"equip"`
	Slots     float64      `json:"slots"`
	AppliesTo []string     `json:"appliesTo,omitempty"`
	Weapon    *WeaponStats `json:"weapon,omitempty"`
	Modifiers []Modifier   `json:"modifiers"`
}

// Item devolve o item do catálogo pelo id, ou nulo quando ele não existe — é o
// acessor que os validadores de mutação de item usam.
func (c *Catalogs) Item(id string) *CatalogItem { return c.itemsByID[id] }

type WeaponStats struct {
	Purpose   string   `json:"purpose"` // 'melee' | 'thrown' | 'ranged'
	Hand      string   `json:"hand"`    // 'light' | 'one' | 'two' (finesse gating — ALE-31)
	Traits    []string `json:"traits"`
	Damage    string   `json:"damage"`    // dice, e.g. "1d8" (weapon-card display)
	CritRange int      `json:"critRange"` // lowest natural roll that threatens
	CritMult  int      `json:"critMult"`
	Finesse   bool     `json:"finesse"` // inherent Des-on-attack (Adaga) — ALE-31
}

// RaceDefinition é a raça do catálogo de habilidades, distinta da `Raca` de
// atributos mais abaixo.
type RaceDefinition struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	AttributeBonuses map[string]int `json:"attributeBonuses"`
	Abilities        []RaceAbility  `json:"abilities"`
	HasDeformity     bool           `json:"hasDeformidade"`
}

type RaceAbility struct {
	Modifiers []Modifier           `json:"modifiers"`
	Variants  []RaceAbilityVariant `json:"variants"`
}

type RaceAbilityVariant struct {
	ID        string     `json:"id"`
	Modifiers []Modifier `json:"modifiers"`
}

// OriginDefinition é a origem do catálogo de habilidades.
type OriginDefinition struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Benefits    []OriginBenefit `json:"benefits"`
	UniquePower OriginBenefit   `json:"poderUnico"`
}

type OriginBenefit struct {
	ID string `json:"id"`
	// PowerUid APONTA para o poder geral que este benefício concede, em vez de
	// repetir a regra dele (ALE-401). A origem descreve a ORIGEM; o poder mora
	// no `general-powers.json`, e o `uid` é a identidade que não muda.
	//
	// Quando ele está presente, `Modifiers` fica vazio: são as duas formas de
	// dizer a mesma coisa, e ter as duas foi o que deixou dezenove poderes com
	// duas regras diferentes.
	PowerUid  string     `json:"powerUid,omitempty"`
	Modifiers []Modifier `json:"modifiers"`
	PowerPick string     `json:"powerPick,omitempty"` // 'combate' | 'tormenta'
}

// ClassPower é o poder de classe, nos campos que esta camada lê.
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

// GeneralPower é o poder geral ou de combate.
type GeneralPower struct {
	ID string `json:"id"`
	// Uid é a identidade estável — ver o guia. Ela entrou quando a origem
	// passou a APONTAR para o poder em vez de copiá-lo (ALE-402).
	Uid       string     `json:"uid"`
	Name      string     `json:"name"`
	Modifiers []Modifier `json:"modifiers"`
}

// GrantedPower é o poder concedido do deus — só o nome e os modificadores são
// lidos (Bênção do Mana → maxPm). Indexado por NOME porque é o nome que o
// `Character.godPower` guarda.
type GrantedPower struct {
	// ID já existia no catálogo e ninguém o lia. Ele entrou quando a devoção
	// virou fonte do coletor (ALE-397): toda fonte precisa de um `SourceID`
	// estável, e derivá-lo do nome criaria uma segunda grafia do mesmo id.
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Modifiers []Modifier `json:"modifiers"`
}

// RaceAttributeEntry é a entrada de raça do `races.json` — o MESMO arquivo que o
// forge lê para montar a carta de raça.
//
// `Size` e `Speed` entraram na ALE-383, e o que importa é de ONDE: eles já
// chegavam no payload e ninguém os lia. Transcrevê-los para o `race-defs.json`
// teria criado uma TERCEIRA cópia do mesmo número do livro — que é exatamente
// como `characters.displacement` passou a discordar do catálogo.
type RaceAttributeEntry struct {
	Name         string       `json:"name"`
	AttributeMod AttributeMod `json:"atributoMod"`
	// Size é categórico ("Médio", "Pequeno", "Minúsculo") e NÃO determina o
	// deslocamento: o goblin é Pequeno e anda 9m. São dois fatos do verbete.
	Size  string `json:"tamanho"`
	Speed int    `json:"deslocamento"`
}

// AttributeMod é a união de modificadores de raça, achatada por `kind`. `Mods` e
// `Variants` usam `orderedInts` porque um modificador sai por entrada NA ORDEM
// em que o catálogo as traz — um mapa comum embaralharia.
type AttributeMod struct {
	Kind     string                 `json:"kind"` // fixed | floating | subraca-gated
	Mods     orderedInts            `json:"mods"`
	Count    int                    `json:"count"`
	Value    int                    `json:"value"`
	Exclude  string                 `json:"exclude"`
	Penalty  *AttributePenalty      `json:"penalty"`
	Variants map[string]orderedInts `json:"variants"`
}

type AttributePenalty struct {
	Attribute string `json:"attribute"`
	Value     int    `json:"value"`
}

// attrDelta é um par atributo→quantidade, guardado em fatia ordenada.
type attrDelta struct {
	attr   string
	amount int
}

// orderedInts é um objeto JSON de inteiros que PRESERVA a ordem das chaves na
// decodificação — mapa de Go não preserva. Usado nos `mods`/`variants` do
// `atributoMod`, para os modificadores derivados saírem na ordem do catálogo.
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

// ─── O porta-catálogos e a primagem ──────────────────────────────────────────

// Catalogs guarda todo catálogo primado que esta camada lê. Injetado no
// `ActiveItemsFor` em vez de virar cache no nível do pacote — dependência por
// parâmetro. Estático e primado uma vez, então não precisa de reatividade.
type Catalogs struct {
	itemsByID     map[string]*CatalogItem
	racesByID     map[string]*RaceDefinition
	origins       []*OriginDefinition
	classPowers   []*ClassPower
	generalByID   map[string]*GeneralPower
	generalByUid  map[string]*GeneralPower
	grantedByName map[string]*GrantedPower
	racesByName   map[string]*RaceAttributeEntry
	tormentaIDs   map[string]bool
}

// enginePayload é a forma JSON que o `cmd/genoracle` despeja em
// `engine-go/parity/_catalogs.json`.
type enginePayload struct {
	Items         []CatalogItem                 `json:"items"`
	Races         []RaceDefinition              `json:"races"`
	Origins       []OriginDefinition            `json:"origins"`
	ClassPowers   []ClassPower                  `json:"classPowers"`
	GeneralPowers []GeneralPower                `json:"generalPowers"`
	GrantedPowers []GrantedPower                `json:"grantedPowers"`
	Ancestries    map[string]RaceAttributeEntry `json:"racas"`
	TormentaIDs   []string                      `json:"tormentaPowerIds"`
}

// PrimeEngineCatalogs ingere o JSON dos catálogos num `Catalogs` indexado.
// Devolve erro NOMEANDO a forma ofensora em vez de entrar em pânico, para quem
// chama poder mostrá-la.
func PrimeEngineCatalogs(raw []byte) (*Catalogs, error) {
	var p enginePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("PrimeEngineCatalogs: bad catalog JSON: %w", err)
	}
	c := &Catalogs{
		itemsByID:     make(map[string]*CatalogItem, len(p.Items)),
		racesByID:     make(map[string]*RaceDefinition, len(p.Races)),
		generalByID:   make(map[string]*GeneralPower, len(p.GeneralPowers)),
		generalByUid:  make(map[string]*GeneralPower, len(p.GeneralPowers)),
		grantedByName: make(map[string]*GrantedPower, len(p.GrantedPowers)),
		racesByName:   make(map[string]*RaceAttributeEntry, len(p.Ancestries)),
		tormentaIDs:   make(map[string]bool, len(p.TormentaIDs)),
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
	}
	for i := range p.GeneralPowers {
		c.generalByID[p.GeneralPowers[i].ID] = &p.GeneralPowers[i]
		if uid := p.GeneralPowers[i].Uid; uid != "" {
			c.generalByUid[uid] = &p.GeneralPowers[i]
		}
	}
	for i := range p.GrantedPowers {
		c.grantedByName[p.GrantedPowers[i].Name] = &p.GrantedPowers[i]
	}
	for id := range p.Ancestries {
		r := p.Ancestries[id]
		c.racesByName[r.Name] = &r
	}
	for _, id := range p.TormentaIDs {
		c.tormentaIDs[id] = true
	}
	return c, nil
}

// ─── As consultas ────────────────────────────────────────────────────────────

func (c *Catalogs) getRace(id string) *RaceDefinition { return c.racesByID[id] }

func (c *Catalogs) getGeneralPower(id string) *GeneralPower { return c.generalByID[id] }

// generalPowerByUid resolve o apontamento de um benefício de origem. Por UID e
// não por id: o id é o nome em kebab-case e muda quando o nome muda, e um
// apontamento que se quebra num renome não é apontamento.
func (c *Catalogs) generalPowerByUid(uid string) *GeneralPower { return c.generalByUid[uid] }

// grantedPowerByName é o poder concedido do deus indexado pelo nome do livro,
// que é o que o `Character.godPower` guarda.
func (c *Catalogs) grantedPowerByName(name string) *GrantedPower { return c.grantedByName[name] }

// getOrigin acha uma origem pelo id.
func (c *Catalogs) getOrigin(id string) *OriginDefinition {
	for _, o := range c.origins {
		if o.ID == id {
			return o
		}
	}
	return nil
}

// getOriginBenefit acha um benefício em TODAS as origens, inclusive o poder
// único.
func (c *Catalogs) getOriginBenefit(benefitID string) *OriginBenefit {
	for _, o := range c.origins {
		for i := range o.Benefits {
			if o.Benefits[i].ID == benefitID {
				return &o.Benefits[i]
			}
		}
		if o.UniquePower.ID == benefitID {
			return &o.UniquePower
		}
	}
	return nil
}

// raceEntryByName acha a entrada de atributo de uma raça pelo nome. O mapa é
// montado uma vez, quando os catálogos são primados.
func (c *Catalogs) raceEntryByName(name string) *RaceAttributeEntry { return c.racesByName[name] }

// raceWithDeformidade devolve o primeiro nome que tem Deformidade (Lefou p23).
func (c *Catalogs) raceWithDeformidade(names ...string) string {
	owners := map[string]bool{}
	for _, r := range c.racesByID {
		if r.HasDeformity {
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

// isTormentaPower diz se o id é de um poder da Tormenta.
func (c *Catalogs) isTormentaPower(id string) bool { return c.tormentaIDs[id] }

// ownedClassPowers são todos os poderes de classe que o personagem tem para uma
// classe, nível, ids escolhidos e escolhas — em ordem de CATÁLOGO, para a ordem
// dos `ActiveItem` lá na frente ser determinística.
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
