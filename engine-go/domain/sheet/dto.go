package sheet

import (
	"encoding/json"
	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// CharacterDTO é o agregado do personagem que a fronteira JSON entrega. O sqlc
// minuscula identificador de coluna, então as structs do banco não podem
// carregar o contrato camelCase — esta forma escrita à mão carrega, e os
// mapeadores abaixo convertem (bool vindo de INTEGER, *string vindo de NULL).
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
	// IgnoredRules são as regras opcionais desligadas para ESTA ficha. Não é
	// campo do personagem: é a mesa dele, resolvida em `loadCharacter` e
	// carimbada aqui para atravessar até o motor pelo `engineCharacterFrom`,
	// que é um round-trip de JSON — e assim as duas pontas calculam com as
	// mesmas regras sem nenhuma assinatura mudar.
	IgnoredRules  engine.IgnoredRules `json:"ignoredRules"`
	Races         []RaceDTO           `json:"races"`
	Classes       []ClassDTO          `json:"classes"`
	Expertises    []ExpertiseDTO      `json:"expertises"`
	Items         []ItemDTO           `json:"items"`
	ActiveEffects []EffectDTO         `json:"activeEffects"`
	Spells        []SpellDTO          `json:"spells"`
	// O estado de JOGO da ficha — situacionais ligados, usos gastos e o preço
	// pago pelas posturas. Viaja com a ficha porque a tela precisa dos três
	// para desenhar o primeiro quadro.
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

// characterScalarsFrom mapeia a linha achatada do banco; as relações são
// penduradas pelo carregador.
func CharacterScalarsFrom(c sqlcgen.Character) CharacterDTO {
	return CharacterDTO{
		ID:                   c.ID,
		OwnerID:              c.Ownerid,
		Name:                 c.Name,
		Origin:               c.Origin,
		God:                  dbvalue.NullToPtr(c.God),
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
		// Relação nasce em fatia VAZIA, nunca nula.
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

// MarshalStrings codifica uma lista de strings em JSON, normalizando o NULO
// (ausente, ou `null` no JSON) para `"[]"` em vez do `"null"` do Go.
//
// A normalização é o ponto: a coluna guarda o que o `JSON.stringify` do
// navegador produziria, e `null` ali faria o cliente ler ausência onde há lista
// vazia. Ela mora aqui porque é a forma do DADO — a cena da ficha e o
// hospedeiro gravam a mesma coluna.
func MarshalStrings(p *[]string) string {
	if p == nil {
		return "[]"
	}
	b, err := json.Marshal(*p)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// UnmarshalStrings é o caminho de VOLTA do `MarshalStrings`: a coluna de blob
// virando a lista de ids.
//
// Blob TORTO vira lista vazia, e não erro: essas colunas são `string[]` cru, e a
// ficha inteira não pode deixar de abrir porque uma linha do banco está errada.
// Sem nada escolhido é um estado legítimo — um arcanista de nível 1 chega perto
// disso —, então a degradação é para um estado que a tela sabe desenhar.
func UnmarshalStrings(blob string) []string {
	var ids []string
	if json.Unmarshal([]byte(blob), &ids) != nil {
		return nil
	}
	return ids
}

// AugmentPick é um aprimoramento escolhido ao conjurar, e quantas vezes.
//
// Ele atravessa a fronteira porque a CENA o lê dos sinais do Datastar e o
// HOSPEDEIRO o consome ao cobrar o PM — a mesma forma nos dois lados, e uma
// segunda declaração seria a cópia que diverge.
type AugmentPick struct {
	AugmentIndex int `json:"augmentIndex"`
	Stacks       int `json:"stacks"`
}
