package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/sheet"
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

// classProficiencies eram as proficiências de cada classe escritas à mão aqui.
// Elas saíram na ALE-272: a MESMA tabela já vinha do catálogo (`classes.json`,
// a linha "Proficiências." de p36–83), lida por `book.ProficienciesByClass` para
// o painel da ficha. Duas cópias da mesma transcrição não divergiram por sorte,
// e a que ficou é a que a validação de schema alcança.

// A FORJA é a primeira cena com adaptador PRÓPRIO (ALE-278, fatia 6).
//
// `forgeHost` cumpre a `forge.Deps` sem o `*Server` no meio: das seis
// assinaturas que a forja pede, quatro são do núcleo e duas — estas — precisam
// só de mais uma coisa, a transação. Por isso o adaptador é o núcleo mais um
// `*sql.DB`, e não o servidor inteiro.
//
// A escolha da forja como primeira não é gosto: a medição de acoplamento da
// ALE-278 dizia que ela **não vaza nenhum símbolo**, e por isso ela foi também
// a primeira cena a virar pacote. A mesma propriedade a faz a primeira a largar
// o servidor.
type forgeHost struct {
	sceneCore
	db *sql.DB
}

func (s *Server) forgeHost() forgeHost { return forgeHost{sceneCore: s.sceneCore(), db: s.db} }

// InsertCharacter writes the character + all relations in one transaction.
func (h forgeHost) InsertCharacter(r *http.Request, ownerID int64, name string, body sheet.CreateBody, totalLevel int64, granted []string, trained map[string]bool) (int64, error) {
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.queries.WithTx(tx)
	now := plataforma.NowISO()

	id, err := q.CreateCharacter(r.Context(), sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: body.Origin, God: plataforma.NullString(body.God),
		GodPower: derefStr(body.GodPower, ""), Tibar: derefF64(body.Tibar, 0), Level: totalLevel,
		HpMax: body.HpMax, HpCurrent: body.HpCurrent, MpMax: body.MpMax, MpCurrent: body.MpCurrent,
		Strength: body.Strength, Dexterity: body.Dexterity, Constitution: body.Constitution,
		Intelligence: body.Intelligence, Wisdom: body.Wisdom, Charisma: body.Charisma,
		Size: body.Size, Displacement: body.Displacement,
		Proficiencies:        sheet.MarshalStrings(&granted),
		RaceAttributeChoices: compactOrDefault(body.RaceAttributeChoices, "{}"),
		SecondaryRaceChoices: compactOrDefault(body.SecondaryRaceChoices, "[]"),
		OriginChoices:        sheet.MarshalStrings(body.OriginChoices),
		ClassPowers:          sheet.MarshalStrings(body.ClassPowers),
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
			Characterid: id, Catalogid: plataforma.NullString(it.CatalogID), Name: derefStr(it.Name, ""),
			Quantity: aovivo.DerefOr(it.Quantity, 1), Slots: derefF64(it.Slots, 1),
			Equipped: plataforma.NullString(it.Equipped), Improvements: "[]", Material: sql.NullString{}, Createdat: now,
		}); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// healVitals recomputes the pools (clamp-only, matching healVitalsFromEngine) and
// patches the aggregate + row.
func (h forgeHost) HealVitals(r *http.Request, id int64, dto *sheet.CharacterDTO) error {
	if h.catalogs == nil || len(dto.Classes) == 0 {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(*dto)
	if err != nil {
		return err
	}
	pools := h.catalogs.VitalsForCharacter(ec)
	stored := storedVitals{HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent}
	next := storedVitals{
		HpMax: int64(pools.PvMax), HpCurrent: clampCurrent(stored.HpCurrent, int64(pools.PvMax)),
		MpMax: int64(pools.PmMax), MpCurrent: clampCurrent(stored.MpCurrent, int64(pools.PmMax)),
	}
	if next == stored {
		return nil
	}
	if err := h.queries.SetCharacterVitals(r.Context(), sqlcgen.SetCharacterVitalsParams{
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
