package character

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// Births cria heróis e mantém os poços deles de acordo com o motor.
type Births struct {
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
}

func NewBirths(db *sql.DB, q *sqlcgen.Queries, catalogs *engine.Catalogs) Births {
	return Births{db: db, queries: q, catalogs: catalogs}
}

// Create escreve a ficha e TODAS as relações dela numa transação só.
//
// O `defer Rollback` antes do primeiro `INSERT` é o que garante que uma falha no
// meio não deixe meia ficha: depois do `Commit` ele vira no-op, e antes dele
// desfaz tudo. Sem essa linha, um erro ao gravar o quinto item deixaria um herói
// sem perícias no banco — e ele abriria na tela, quebrado, sem nada acusando.
func (b Births) Create(
	ctx context.Context, ownerID int64, name string, body sheet.CreateBody,
	totalLevel int64, proficiencies []string, trained map[string]bool,
) (int64, error) {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("abrir a transação do nascimento: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := b.queries.WithTx(tx)
	now := dbvalue.NowISO()

	id, err := q.CreateCharacter(ctx, sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: body.Origin, God: dbvalue.NullString(body.God),
		GodPower: orElse(body.GodPower, ""), Tibar: orElseFloat(body.Tibar, 0), Level: totalLevel,
		Strength: body.Strength, Dexterity: body.Dexterity, Constitution: body.Constitution,
		Intelligence: body.Intelligence, Wisdom: body.Wisdom, Charisma: body.Charisma,
		Size: body.Size, Displacement: body.Displacement,
		Proficiencies:        sheet.MarshalStrings(&proficiencies),
		RaceAttributeChoices: compactOr(body.RaceAttributeChoices, "{}"),
		SecondaryRaceChoices: compactOr(body.SecondaryRaceChoices, "[]"),
		OriginChoices:        sheet.MarshalStrings(body.OriginChoices),
		ClassPowers:          sheet.MarshalStrings(body.ClassPowers),
		ClassChoices:         compactOr(body.ClassChoices, "{}"),
		PowerChoices:         compactOr(body.PowerChoices, "{}"),
		CreatedAt:            now, UpdatedAt: now,
	})
	if err != nil {
		return 0, fmt.Errorf("gravar a ficha de %q: %w", name, err)
	}
	for _, race := range body.Races {
		if err := q.CreateRace(ctx, sqlcgen.CreateRaceParams{Characterid: id, Race: race}); err != nil {
			return 0, fmt.Errorf("gravar a raça %q: %w", race, err)
		}
	}
	for _, c := range body.Classes {
		if err := q.CreateClass(ctx, sqlcgen.CreateClassParams{
			Characterid: id, Classname: c.ClassName, Level: c.Level,
		}); err != nil {
			return 0, fmt.Errorf("gravar a classe %q: %w", c.ClassName, err)
		}
	}
	// TODA ficha nasce com as vinte e nove perícias do livro, treinadas ou não:
	// a lista é fechada, e uma perícia que só aparecesse quando treinada faria a
	// tela ter de inventar as outras.
	for _, expertise := range sheet.BuiltinExpertises() {
		if _, err := q.CreateExpertise(ctx, sqlcgen.CreateExpertiseParams{
			Characterid: id, Name: expertise.Name, Attribute: expertise.Attribute,
			Trained: oneIfTrue(trained[expertise.Name]), Custom: 0,
		}); err != nil {
			return 0, fmt.Errorf("gravar a perícia %q: %w", expertise.Name, err)
		}
	}
	for _, item := range body.Items {
		if _, err := q.CreateItem(ctx, sqlcgen.CreateItemParams{
			Characterid: id, Catalogid: dbvalue.NullString(item.CatalogID),
			Name: orElse(item.Name, ""), Quantity: live.DerefOr(item.Quantity, 1),
			Slots: orElseFloat(item.Slots, 1), Equipped: dbvalue.NullString(item.Equipped),
			Improvements: "[]", Material: sql.NullString{}, Createdat: now,
		}); err != nil {
			return 0, fmt.Errorf("gravar um item da ficha %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("fechar a transação do nascimento: %w", err)
	}
	return id, nil
}

// compactOr reescreve o JSON sem espaço, ou devolve o padrão quando não veio —
// e também quando veio ILEGÍVEL: gravar o texto cru de um JSON quebrado faria a
// leitura seguinte falhar longe daqui, na tela de quem abrisse a ficha.
func compactOr(raw *json.RawMessage, standard string) string {
	if raw == nil {
		return standard
	}
	var anything any
	if json.Unmarshal(*raw, &anything) != nil {
		return standard
	}
	compact, err := json.Marshal(anything)
	if err != nil {
		return standard
	}
	return string(compact)
}

func orElse(p *string, standard string) string {
	if p == nil {
		return standard
	}
	return *p
}

func orElseFloat(p *float64, standard float64) float64 {
	if p == nil {
		return standard
	}
	return *p
}

// oneIfTrue traduz o `bool` do Go no INTEGER do SQLite, e é só isso.
func oneIfTrue(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
