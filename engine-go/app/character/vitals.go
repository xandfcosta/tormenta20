package character

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// syncVitals pergunta os máximos ao motor e grava o que a regra decidir para os
// atuais, remendando o agregado junto.
//
// Ela é função do PACOTE e não método de um dos dois tipos porque os dois
// gestos a usam: o nascimento e o passo de atributo (`Births`), e o degrau de
// nível (`Plays`). Enquanto era método do `Births`, o degrau de nível carregava
// no `serve/api` uma SEGUNDA cópia da mesma conta, linha a linha igual ao
// `sheet.ShiftedByNewMax` — ela deixou de existir com esta fatia (ALE-347).
//
// Motor ausente ou ficha SEM CLASSE devolve sem escrever: os poços do livro
// dependem da classe, e gravar 0/0 apagaria os números que a pessoa digitou.
func syncVitals(
	ctx context.Context, q *sqlcgen.Queries, catalogs *engine.Catalogs,
	id int64, dto *sheet.CharacterDTO,
	regra func(sheet.Vitals, int, int) (sheet.Vitals, bool),
) error {
	if catalogs == nil || len(dto.Classes) == 0 {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(*dto)
	if err != nil {
		return fmt.Errorf("montar o personagem do motor (%d): %w", id, err)
	}
	pocos := catalogs.VitalsForCharacter(ec)
	atuais := sheet.Vitals{
		HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent,
	}
	novos, mudou := regra(atuais, pocos.PvMax, pocos.PmMax)
	if !mudou {
		return nil
	}
	if err := q.SetCharacterVitals(ctx, sqlcgen.SetCharacterVitalsParams{
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
