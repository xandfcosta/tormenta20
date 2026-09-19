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
	ctx context.Context, ownerID int64, nome string, corpo sheet.CreateBody,
	nivelTotal int64, proficiencias []string, treinadas map[string]bool,
) (int64, error) {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("abrir a transação do nascimento: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := b.queries.WithTx(tx)
	agora := dbvalue.NowISO()

	id, err := q.CreateCharacter(ctx, sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: nome, Origin: corpo.Origin, God: dbvalue.NullString(corpo.God),
		GodPower: orElse(corpo.GodPower, ""), Tibar: orElseFloat(corpo.Tibar, 0), Level: nivelTotal,
		HpMax: corpo.HpMax, HpCurrent: corpo.HpCurrent, MpMax: corpo.MpMax, MpCurrent: corpo.MpCurrent,
		Strength: corpo.Strength, Dexterity: corpo.Dexterity, Constitution: corpo.Constitution,
		Intelligence: corpo.Intelligence, Wisdom: corpo.Wisdom, Charisma: corpo.Charisma,
		Size: corpo.Size, Displacement: corpo.Displacement,
		Proficiencies:        sheet.MarshalStrings(&proficiencias),
		RaceAttributeChoices: compactOr(corpo.RaceAttributeChoices, "{}"),
		SecondaryRaceChoices: compactOr(corpo.SecondaryRaceChoices, "[]"),
		OriginChoices:        sheet.MarshalStrings(corpo.OriginChoices),
		ClassPowers:          sheet.MarshalStrings(corpo.ClassPowers),
		ClassChoices:         compactOr(corpo.ClassChoices, "{}"),
		PowerChoices:         compactOr(corpo.PowerChoices, "{}"),
		CreatedAt:            agora, UpdatedAt: agora,
	})
	if err != nil {
		return 0, fmt.Errorf("gravar a ficha de %q: %w", nome, err)
	}
	for _, raca := range corpo.Races {
		if err := q.CreateRace(ctx, sqlcgen.CreateRaceParams{Characterid: id, Race: raca}); err != nil {
			return 0, fmt.Errorf("gravar a raça %q: %w", raca, err)
		}
	}
	for _, c := range corpo.Classes {
		if err := q.CreateClass(ctx, sqlcgen.CreateClassParams{
			Characterid: id, Classname: c.ClassName, Level: c.Level,
		}); err != nil {
			return 0, fmt.Errorf("gravar a classe %q: %w", c.ClassName, err)
		}
	}
	// TODA ficha nasce com as vinte e nove perícias do livro, treinadas ou não:
	// a lista é fechada, e uma perícia que só aparecesse quando treinada faria a
	// tela ter de inventar as outras.
	for _, pericia := range sheet.BuiltinExpertises() {
		if _, err := q.CreateExpertise(ctx, sqlcgen.CreateExpertiseParams{
			Characterid: id, Name: pericia.Name, Attribute: pericia.Attribute,
			Trained: oneIfTrue(treinadas[pericia.Name]), Custom: 0,
		}); err != nil {
			return 0, fmt.Errorf("gravar a perícia %q: %w", pericia.Name, err)
		}
	}
	for _, item := range corpo.Items {
		if _, err := q.CreateItem(ctx, sqlcgen.CreateItemParams{
			Characterid: id, Catalogid: dbvalue.NullString(item.CatalogID),
			Name: orElse(item.Name, ""), Quantity: live.DerefOr(item.Quantity, 1),
			Slots: orElseFloat(item.Slots, 1), Equipped: dbvalue.NullString(item.Equipped),
			Improvements: "[]", Material: sql.NullString{}, Createdat: agora,
		}); err != nil {
			return 0, fmt.Errorf("gravar um item da ficha %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("fechar a transação do nascimento: %w", err)
	}
	return id, nil
}

// HealVitals recomputa os poços e PRENDE o atual na faixa — a ficha que nasce
// nasce cheia, e o que se evita é um atual maior que o máximo.
func (b Births) HealVitals(ctx context.Context, id int64, dto *sheet.CharacterDTO) error {
	return b.recompute(ctx, id, dto, sheet.ClampedToNewMax)
}

// ShiftVitalsToNewMax recomputa os poços e faz os ATUAIS acompanharem o delta,
// que é o que um passo de atributo faz com os poços de um herói que já apanhou.
func (b Births) ShiftVitalsToNewMax(ctx context.Context, id int64, dto *sheet.CharacterDTO) error {
	return b.recompute(ctx, id, dto, sheet.ShiftedByNewMax)
}

// recompute pergunta os máximos ao motor e grava o que a regra decidir para os
// atuais, remendando o agregado junto.
//
// Motor ausente ou ficha SEM CLASSE devolve sem escrever: os poços do livro
// dependem da classe, e gravar 0/0 apagaria os números que a pessoa digitou.
func (b Births) recompute(
	ctx context.Context, id int64, dto *sheet.CharacterDTO,
	regra func(sheet.Vitals, int, int) (sheet.Vitals, bool),
) error {
	if b.catalogs == nil || len(dto.Classes) == 0 {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(*dto)
	if err != nil {
		return fmt.Errorf("montar o personagem do motor (%d): %w", id, err)
	}
	pocos := b.catalogs.VitalsForCharacter(ec)
	atuais := sheet.Vitals{
		HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent,
	}
	novos, mudou := regra(atuais, pocos.PvMax, pocos.PmMax)
	if !mudou {
		return nil
	}
	if err := b.queries.SetCharacterVitals(ctx, sqlcgen.SetCharacterVitalsParams{
		HpMax: novos.HpMax, HpCurrent: novos.HpCurrent,
		MpMax: novos.MpMax, MpCurrent: novos.MpCurrent,
		UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		return fmt.Errorf("gravar os vitais da ficha %d: %w", id, err)
	}
	dto.HpMax, dto.HpCurrent = novos.HpMax, novos.HpCurrent
	dto.MpMax, dto.MpCurrent = novos.MpMax, novos.MpCurrent
	return nil
}

// compactOr reescreve o JSON sem espaço, ou devolve o padrão quando não veio —
// e também quando veio ILEGÍVEL: gravar o texto cru de um JSON quebrado faria a
// leitura seguinte falhar longe daqui, na tela de quem abrisse a ficha.
func compactOr(bruto *json.RawMessage, padrao string) string {
	if bruto == nil {
		return padrao
	}
	var qualquer any
	if json.Unmarshal(*bruto, &qualquer) != nil {
		return padrao
	}
	compacto, err := json.Marshal(qualquer)
	if err != nil {
		return padrao
	}
	return string(compacto)
}

func orElse(p *string, padrao string) string {
	if p == nil {
		return padrao
	}
	return *p
}

func orElseFloat(p *float64, padrao float64) float64 {
	if p == nil {
		return padrao
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
