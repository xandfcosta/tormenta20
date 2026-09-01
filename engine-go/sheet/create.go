package sheet

import "encoding/json"

// A CARGA DE CRIAÇÃO (ALE-278): o herói que vai nascer, como quem o pede o
// descreve.
//
// Ela morava no `api/character_create.go`, junto do handler que a recebe, e saiu
// porque a FORJA precisa montá-la — e a forja não pode importar o `api`, que a
// importa de volta para montar rota. Os três tipos não têm dependência nenhuma:
// nem banco, nem motor, nem HTTP.

type ClassEntry struct {
	ClassName string `json:"className"`
	Level     int64  `json:"level"`
}

type StartingItem struct {
	CatalogID *string  `json:"catalogId"`
	Name      *string  `json:"name"`
	Quantity  *int64   `json:"quantity"`
	Slots     *float64 `json:"slots"`
	Equipped  *string  `json:"equipped"`
}

type CreateBody struct {
	Name                 string           `json:"name"`
	Races                []string         `json:"races"`
	Origin               string           `json:"origin"`
	Classes              []ClassEntry     `json:"classes"`
	God                  *string          `json:"god"`
	GodPower             *string          `json:"godPower"`
	Tibar                *float64         `json:"tibar"`
	Items                []StartingItem   `json:"items"`
	HpMax                int64            `json:"hpMax"`
	HpCurrent            int64            `json:"hpCurrent"`
	MpMax                int64            `json:"mpMax"`
	MpCurrent            int64            `json:"mpCurrent"`
	Strength             int64            `json:"strength"`
	Dexterity            int64            `json:"dexterity"`
	Constitution         int64            `json:"constitution"`
	Intelligence         int64            `json:"intelligence"`
	Wisdom               int64            `json:"wisdom"`
	Charisma             int64            `json:"charisma"`
	Size                 string           `json:"size"`
	Displacement         int64            `json:"displacement"`
	ClassPowers          *[]string        `json:"classPowers"`
	OriginChoices        *[]string        `json:"originChoices"`
	TrainedExpertises    []string         `json:"trainedExpertises"`
	ClassChoices         *json.RawMessage `json:"classChoices"`
	PowerChoices         *json.RawMessage `json:"powerChoices"`
	RaceAttributeChoices *json.RawMessage `json:"raceAttributeChoices"`
	SecondaryRaceChoices *json.RawMessage `json:"secondaryRaceChoices"`
}
