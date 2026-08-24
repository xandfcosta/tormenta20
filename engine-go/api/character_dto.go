package api

import (
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// CharacterDTO is the character aggregate the frontend consumes (shared/api/api.ts
// `Character`). sqlc lowercases column identifiers, so the DB structs can't carry
// the camelCase JSON contract — this hand-written shape does, and the mappers
// below convert (bool from INTEGER, *string from NULL).
type CharacterDTO struct {
	ID                   int64   `json:"id"`
	OwnerID              int64   `json:"ownerId"`
	Name                 string  `json:"name"`
	Origin               string  `json:"origin"`
	God                  *string `json:"god"`
	GodPower             string  `json:"godPower"`
	Tibar                float64 `json:"tibar"`
	Level                int64   `json:"level"`
	HpMax                int64   `json:"hpMax"`
	HpCurrent            int64   `json:"hpCurrent"`
	MpMax                int64   `json:"mpMax"`
	MpCurrent            int64   `json:"mpCurrent"`
	Strength             int64   `json:"strength"`
	Dexterity            int64   `json:"dexterity"`
	Constitution         int64   `json:"constitution"`
	Intelligence         int64   `json:"intelligence"`
	Wisdom               int64   `json:"wisdom"`
	Charisma             int64   `json:"charisma"`
	Size                 string  `json:"size"`
	Displacement         int64   `json:"displacement"`
	Proficiencies        string  `json:"proficiencies"`
	RaceAbilityChoices   string  `json:"raceAbilityChoices"`
	RaceAttributeChoices string  `json:"raceAttributeChoices"`
	SecondaryRaceChoices string  `json:"secondaryRaceChoices"`
	OriginChoices        string  `json:"originChoices"`
	ClassPowers          string  `json:"classPowers"`
	ClassChoices         string  `json:"classChoices"`
	PowerChoices         string  `json:"powerChoices"`
	ActiveConditions     string  `json:"activeConditions"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
	// IgnoredRules são as regras opcionais desligadas para ESTA ficha (ALE-221).
	// Não é campo do personagem: é a mesa dele, resolvida em `loadCharacter` e
	// carimbada aqui para atravessar até o motor pelo `engineCharacterFrom`, que
	// é um round-trip de JSON. Assim a ficha do servidor e a do navegador
	// calculam com as mesmas regras sem nenhuma assinatura mudar.
	IgnoredRules  engine.IgnoredRules `json:"ignoredRules"`
	Races         []RaceDTO           `json:"races"`
	Classes       []ClassDTO          `json:"classes"`
	Expertises    []ExpertiseDTO      `json:"expertises"`
	Items         []ItemDTO           `json:"items"`
	ActiveEffects []EffectDTO         `json:"activeEffects"`
	Spells        []SpellDTO          `json:"spells"`
	// O estado de JOGO da ficha (ALE-222) — situacionais ligados, usos gastos e
	// o preco pago pelas posturas. Viaja com a ficha porque a tela precisa dos
	// tres para desenhar o primeiro quadro.
	Conditionals []string      `json:"conditionals"`
	PowerUses    []PowerUseDTO `json:"powerUses"`
	Stances      []StanceDTO   `json:"stances"`
}

type RaceDTO struct {
	Race string `json:"race"`
}

type ClassDTO struct {
	ClassName string `json:"className"`
	Level     int64  `json:"level"`
}

type ExpertiseDTO struct {
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
	Trained   bool   `json:"trained"`
	Custom    bool   `json:"custom"`
}

type ItemDTO struct {
	ID           int64   `json:"id"`
	CatalogID    *string `json:"catalogId"`
	Name         string  `json:"name"`
	Quantity     int64   `json:"quantity"`
	Slots        float64 `json:"slots"`
	Equipped     *string `json:"equipped"`
	Improvements string  `json:"improvements"`
	Material     *string `json:"material"`
}

type EffectDTO struct {
	ID        int64  `json:"id"`
	CatalogID string `json:"catalogId"`
	Scope     string `json:"scope"`
	Modifiers string `json:"modifiers"`
	CreatedAt string `json:"createdAt"`
}

type SpellDTO struct {
	ID             int64  `json:"id"`
	CatalogSpellID string `json:"catalogSpellId"`
	Prepared       bool   `json:"prepared"`
	LearnedAt      string `json:"learnedAt"`
}

// characterScalarsFrom maps the flat DB row; relations are attached by the loader.
func characterScalarsFrom(c sqlcgen.Character) CharacterDTO {
	return CharacterDTO{
		ID:                   c.ID,
		OwnerID:              c.Ownerid,
		Name:                 c.Name,
		Origin:               c.Origin,
		God:                  plataforma.NullToPtr(c.God),
		GodPower:             c.Godpower,
		Tibar:                c.Tibar,
		Level:                c.Level,
		HpMax:                c.Hpmax,
		HpCurrent:            c.Hpcurrent,
		MpMax:                c.Mpmax,
		MpCurrent:            c.Mpcurrent,
		Strength:             c.Strength,
		Dexterity:            c.Dexterity,
		Constitution:         c.Constitution,
		Intelligence:         c.Intelligence,
		Wisdom:               c.Wisdom,
		Charisma:             c.Charisma,
		Size:                 c.Size,
		Displacement:         c.Displacement,
		Proficiencies:        c.Proficiencies,
		RaceAbilityChoices:   c.Raceabilitychoices,
		RaceAttributeChoices: c.Raceattributechoices,
		SecondaryRaceChoices: c.Secondaryracechoices,
		OriginChoices:        c.Originchoices,
		ClassPowers:          c.Classpowers,
		ClassChoices:         c.Classchoices,
		PowerChoices:         c.Powerchoices,
		ActiveConditions:     c.Activeconditions,
		CreatedAt:            c.Createdat,
		UpdatedAt:            c.Updatedat,
		// Relations default to empty slices (never null) — matches the Prisma include.
		Races:         []RaceDTO{},
		Classes:       []ClassDTO{},
		Expertises:    []ExpertiseDTO{},
		Items:         []ItemDTO{},
		ActiveEffects: []EffectDTO{},
		Spells:        []SpellDTO{},
		Conditionals:  []string{},
		PowerUses:     []PowerUseDTO{},
		Stances:       []StanceDTO{},
	}
}
