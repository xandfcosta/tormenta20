package api

import "t20engine/aovivo"

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
)

// expertiseDef is one builtin perícia (name + keying attribute). Ordered as
// t20-data EXPERTISES; every character is seeded with all of them on create.
type expertiseDef struct {
	name      string
	attribute string
}

var expertisesList = []expertiseDef{
	{"Acrobacia", "dexterity"}, {"Adestramento", "charisma"}, {"Atletismo", "strength"},
	{"Atuação", "charisma"}, {"Cavalgar", "dexterity"}, {"Conhecimento", "intelligence"},
	{"Cura", "wisdom"}, {"Diplomacia", "charisma"}, {"Enganação", "charisma"},
	{"Fortitude", "constitution"}, {"Furtividade", "dexterity"}, {"Guerra", "intelligence"},
	{"Iniciativa", "dexterity"}, {"Intimidação", "charisma"}, {"Intuição", "wisdom"},
	{"Investigação", "intelligence"}, {"Jogatina", "charisma"}, {"Ladinagem", "dexterity"},
	{"Luta", "strength"}, {"Misticismo", "intelligence"}, {"Nobreza", "intelligence"},
	{"Ofício", "intelligence"}, {"Percepção", "wisdom"}, {"Pilotagem", "dexterity"},
	{"Pontaria", "dexterity"}, {"Reflexos", "dexterity"}, {"Religião", "wisdom"},
	{"Sobrevivência", "wisdom"}, {"Vontade", "wisdom"},
}

// classProficiencies mirrors t20-data CLASS_PROFICIENCIES. armas-simples is
// granted to every class; a class with armaduras-pesadas also gets leves.
var classProficiencies = map[string][]string{
	"Arcanista": {}, "Bárbaro": {"armas-marciais", "escudos"}, "Bardo": {"armas-marciais"},
	"Bucaneiro": {"armas-marciais"}, "Caçador": {"armas-marciais", "escudos"},
	"Cavaleiro": {"armas-marciais", "armaduras-pesadas", "escudos"},
	"Clérigo":   {"armaduras-pesadas", "escudos"}, "Druida": {"escudos"},
	"Guerreiro": {"armas-marciais", "armaduras-pesadas", "escudos"},
	"Inventor":  {}, "Ladino": {}, "Lutador": {},
	"Nobre":    {"armas-marciais", "armaduras-pesadas", "escudos"},
	"Paladino": {"armas-marciais", "armaduras-pesadas", "escudos"},
}

// proficiencyOrder is PROFICIENCY_CATEGORIES — grantedProficiencies emits in it.
var proficiencyOrder = []string{
	"armas-simples", "armas-marciais", "armas-exoticas", "armas-de-fogo",
	"armaduras-leves", "armaduras-pesadas", "escudos",
}

// grantedProficiencies ports characterProficiencies(...).filter(granted): the
// default proficiency categories for a class set, in catalog order.
func grantedProficiencies(classNames []string) []string {
	granted := map[string]bool{"armas-simples": true}
	for _, cls := range classNames {
		for _, cat := range classProficiencies[cls] {
			granted[cat] = true
			if cat == "armaduras-pesadas" {
				granted["armaduras-leves"] = true
			}
		}
	}
	out := []string{}
	for _, cat := range proficiencyOrder {
		if granted[cat] {
			out = append(out, cat)
		}
	}
	return out
}

type classEntryBody struct {
	ClassName string `json:"className"`
	Level     int64  `json:"level"`
}

type startingItemBody struct {
	CatalogID *string  `json:"catalogId"`
	Name      *string  `json:"name"`
	Quantity  *int64   `json:"quantity"`
	Slots     *float64 `json:"slots"`
	Equipped  *string  `json:"equipped"`
}

type createCharacterBody struct {
	Name                 string             `json:"name"`
	Races                []string           `json:"races"`
	Origin               string             `json:"origin"`
	Classes              []classEntryBody   `json:"classes"`
	God                  *string            `json:"god"`
	GodPower             *string            `json:"godPower"`
	Tibar                *float64           `json:"tibar"`
	Items                []startingItemBody `json:"items"`
	HpMax                int64              `json:"hpMax"`
	HpCurrent            int64              `json:"hpCurrent"`
	MpMax                int64              `json:"mpMax"`
	MpCurrent            int64              `json:"mpCurrent"`
	Strength             int64              `json:"strength"`
	Dexterity            int64              `json:"dexterity"`
	Constitution         int64              `json:"constitution"`
	Intelligence         int64              `json:"intelligence"`
	Wisdom               int64              `json:"wisdom"`
	Charisma             int64              `json:"charisma"`
	Size                 string             `json:"size"`
	Displacement         int64              `json:"displacement"`
	ClassPowers          *[]string          `json:"classPowers"`
	OriginChoices        *[]string          `json:"originChoices"`
	TrainedExpertises    []string           `json:"trainedExpertises"`
	ClassChoices         *json.RawMessage   `json:"classChoices"`
	PowerChoices         *json.RawMessage   `json:"powerChoices"`
	RaceAttributeChoices *json.RawMessage   `json:"raceAttributeChoices"`
	SecondaryRaceChoices *json.RawMessage   `json:"secondaryRaceChoices"`
}

// handleCreateCharacter validate (assertCharacterRules
// + presence), seed the aggregate (character + races + classes + all expertises +
// items) with the class-default proficiencies, then heal vitals from the engine.
// NOTE: catalog @IsIn checks (races/origin/god/size) + classChoices sanitize are
// deferred — the frontend pre-validates.
func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	var body createCharacterBody
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	user := currentUser(r)

	fields := plataforma.FieldErrorMap{}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		fields["name"] = []string{"name must be longer than or equal to 1 characters"}
	}
	if len(body.Races) == 0 {
		fields["races"] = []string{"races must contain at least 1 elements"}
	}
	if len(body.Classes) == 0 {
		fields["classes"] = []string{"classes must contain at least 1 elements"}
	}
	if body.HpCurrent > body.HpMax {
		fields["hpCurrent"] = []string{"HP current cannot exceed HP max"}
	}
	if body.MpCurrent > body.MpMax {
		fields["mpCurrent"] = []string{"MP current cannot exceed MP max"}
	}
	seen := map[string]bool{}
	for i, c := range body.Classes {
		if seen[c.ClassName] {
			fields[fmt.Sprintf("classes.%d.className", i)] = []string{
				fmt.Sprintf("Class %q already added — combine levels in one entry instead", c.ClassName)}
		}
		seen[c.ClassName] = true
	}
	if len(fields) > 0 {
		plataforma.WriteValidationError(w, fields)
		return
	}

	var totalLevel int64
	classNames := make([]string, len(body.Classes))
	for i, c := range body.Classes {
		totalLevel += c.Level
		classNames[i] = c.ClassName
	}
	granted := grantedProficiencies(classNames)
	trained := toStringSet(body.TrainedExpertises)

	id, err := s.insertCharacter(r, user.ID, name, body, totalLevel, granted, trained)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create character")
		return
	}

	row, err := s.queries.GetCharacter(r.Context(), id)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load character")
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load character")
		return
	}
	if err := s.healVitals(r, id, &dto); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not sync vitals")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, dto)
}

// insertCharacter writes the character + all relations in one transaction.
func (s *Server) insertCharacter(r *http.Request, ownerID int64, name string, body createCharacterBody, totalLevel int64, granted []string, trained map[string]bool) (int64, error) {
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	q := s.queries.WithTx(tx)
	now := plataforma.NowISO()

	id, err := q.CreateCharacter(r.Context(), sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: body.Origin, God: nullString(body.God),
		GodPower: derefStr(body.GodPower, ""), Tibar: derefF64(body.Tibar, 0), Level: totalLevel,
		HpMax: body.HpMax, HpCurrent: body.HpCurrent, MpMax: body.MpMax, MpCurrent: body.MpCurrent,
		Strength: body.Strength, Dexterity: body.Dexterity, Constitution: body.Constitution,
		Intelligence: body.Intelligence, Wisdom: body.Wisdom, Charisma: body.Charisma,
		Size: body.Size, Displacement: body.Displacement,
		Proficiencies:        marshalStrings(&granted),
		RaceAttributeChoices: compactOrDefault(body.RaceAttributeChoices, "{}"),
		SecondaryRaceChoices: compactOrDefault(body.SecondaryRaceChoices, "[]"),
		OriginChoices:        marshalStrings(body.OriginChoices),
		ClassPowers:          marshalStrings(body.ClassPowers),
		ClassChoices:         compactOrDefault(body.ClassChoices, "{}"),
		PowerChoices:         compactOrDefault(body.PowerChoices, "{}"),
		CreatedAt:            now, UpdatedAt: now,
	})
	if err != nil {
		return 0, err
	}
	for _, race := range body.Races {
		if err := q.CreateRace(r.Context(), sqlcgen.CreateRaceParams{Characterid: id, Race: race}); err != nil {
			return 0, err
		}
	}
	for _, c := range body.Classes {
		if err := q.CreateClass(r.Context(), sqlcgen.CreateClassParams{Characterid: id, Classname: c.ClassName, Level: c.Level}); err != nil {
			return 0, err
		}
	}
	for _, ex := range expertisesList {
		if _, err := q.CreateExpertise(r.Context(), sqlcgen.CreateExpertiseParams{
			Characterid: id, Name: ex.name, Attribute: ex.attribute, Trained: boolToInt(trained[ex.name]), Custom: 0,
		}); err != nil {
			return 0, err
		}
	}
	for _, it := range body.Items {
		if _, err := q.CreateItem(r.Context(), sqlcgen.CreateItemParams{
			Characterid: id, Catalogid: nullString(it.CatalogID), Name: derefStr(it.Name, ""),
			Quantity: aovivo.DerefOr(it.Quantity, 1), Slots: derefF64(it.Slots, 1),
			Equipped: nullString(it.Equipped), Improvements: "[]", Material: sql.NullString{}, Createdat: now,
		}); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// healVitals recomputes the pools (clamp-only, matching healVitalsFromEngine) and
// patches the aggregate + row.
func (s *Server) healVitals(r *http.Request, id int64, dto *CharacterDTO) error {
	if s.catalogs == nil || len(dto.Classes) == 0 {
		return nil
	}
	ec, err := engineCharacterFrom(*dto)
	if err != nil {
		return err
	}
	pools := s.catalogs.VitalsForCharacter(ec)
	stored := storedVitals{HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent}
	next := storedVitals{
		HpMax: int64(pools.PvMax), HpCurrent: clampCurrent(stored.HpCurrent, int64(pools.PvMax)),
		MpMax: int64(pools.PmMax), MpCurrent: clampCurrent(stored.MpCurrent, int64(pools.PmMax)),
	}
	if next == stored {
		return nil
	}
	if err := s.queries.SetCharacterVitals(r.Context(), sqlcgen.SetCharacterVitalsParams{
		HpMax: next.HpMax, HpCurrent: next.HpCurrent, MpMax: next.MpMax, MpCurrent: next.MpCurrent,
		UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		return err
	}
	dto.HpMax, dto.HpCurrent, dto.MpMax, dto.MpCurrent = next.HpMax, next.HpCurrent, next.MpMax, next.MpCurrent
	return nil
}

func compactOrDefault(raw *json.RawMessage, def string) string {
	if raw == nil {
		return def
	}
	return compactJSON(*raw)
}

func derefStr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func derefF64(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}
