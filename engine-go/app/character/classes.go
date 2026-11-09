package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// EnterClass dá ao personagem o 1º nível numa classe nova (multiclasse, p35).
//
// Grava só a linha do nível, e é isso que a regra pede: PV e PM derivam dos níveis de
// classe, e "não ganha as perícias treinadas ou proficiências da nova classe". O PV a mais
// pode levantar quem estava caído (p236), pela mesma regra da queda de nível.
func (p Plays) EnterClass(ctx context.Context, row sqlcgen.Character, class string) error {
	_, classes, _ := book.CharacterCatalogs()
	if !slices.ContainsFunc(classes, func(c book.Class) bool { return c.Name == class }) {
		return fmt.Errorf("%s não é uma classe do livro", class)
	}
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return err
	}
	var total int64
	for _, c := range dto.Classes {
		if c.ClassName == class {
			return fmt.Errorf("o personagem já é %s: suba o nível dela", class)
		}
		total += c.Level
	}
	if total+1 > 20 {
		return fmt.Errorf("as classes somariam o nível %d, e o 20º é o último (p35)", total+1)
	}
	if err := sheet.EnteringClassKeepsTheDevotion(dto, class); err != nil {
		return err
	}
	err = p.inTx(ctx, "entrar em "+class, func(q *sqlcgen.Queries) error {
		if err := q.AddClassLevel(ctx, sqlcgen.AddClassLevelParams{Characterid: row.ID, Ref: class}); err != nil {
			return fmt.Errorf("gravar a classe %s: %w", class, err)
		}
		if err := q.SetCharacterLevel(ctx, sqlcgen.SetCharacterLevelParams{
			Level: total + 1, UpdatedAt: dbvalue.NowISO(), ID: row.ID,
		}); err != nil {
			return err
		}
		return syncLevelChoices(ctx, q, p.catalogs, row)
	})
	if err != nil {
		return err
	}
	projected := dto
	projected.Classes = append(slices.Clone(dto.Classes), sheet.ClassDTO{ClassName: class, Level: 1})
	return sheet.FollowMaximumChange(ctx, p.queries, dto, projected)
}

// LeaveClass tira da ficha uma classe que está no 1º nível. Quem diz se ela pode sair é o
// `sheet.ClassCanLeave`.
func (p Plays) LeaveClass(ctx context.Context, row sqlcgen.Character, class string) error {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return err
	}
	if err := sheet.ClassCanLeave(dto, class); err != nil {
		return err
	}
	var total int64
	remaining := make([]sheet.ClassDTO, 0, len(dto.Classes))
	for _, c := range dto.Classes {
		total += c.Level
		if c.ClassName != class {
			remaining = append(remaining, c)
		}
	}
	// A magia que só esta classe concede ficaria na ficha sem ninguém que a conjure.
	for _, learned := range dto.Spells {
		spell, known := catalog.LookupSpell(learned.CatalogSpellID)
		if known && len(sheet.CastingClasses(dto, remaining, spell.School, spell.Classes)) == 0 {
			return fmt.Errorf("não dá para tirar %s: a magia %s veio dela", class, spell.Name)
		}
	}
	return p.inTx(ctx, "tirar "+class, func(q *sqlcgen.Queries) error {
		if err := q.DeleteLastClassLevel(ctx, sqlcgen.DeleteLastClassLevelParams{Characterid: row.ID, Ref: class}); err != nil {
			return fmt.Errorf("apagar a classe %s: %w", class, err)
		}
		if err := q.SetCharacterLevel(ctx, sqlcgen.SetCharacterLevelParams{
			Level: total - 1, UpdatedAt: dbvalue.NowISO(), ID: row.ID,
		}); err != nil {
			return err
		}
		return syncLevelChoices(ctx, q, p.catalogs, row)
	})
}
