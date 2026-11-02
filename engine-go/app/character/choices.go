package character

import (
	"context"
	"fmt"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ChoiceWrite são as cinco colunas de ESCOLHA da ficha, e NULO quer dizer "não
// mexa nesta".
//
// Dar um nome a elas aqui é o que tira o nome da COLUNA da cena: ela declara o
// que mudou, este pacote sabe onde isso mora. Ela nasceu no `sheetui` por não
// haver camada; mora aqui porque quem grava é o caso de uso.
type ChoiceWrite struct {
	ClassPowers          *string
	OriginChoices        *string
	ClassChoices         *string
	RaceAbilityChoices   *string
	RaceAttributeChoices *string
	PowerChoices         *string
	SecondaryRaceChoices *string
	God                  *string
	GodPower             *string
}

// SaveChoices grava só as escolhas que MUDARAM.
//
// # O `COALESCE` é o que substitui a montagem dinâmica do `SET`
//
// "Só as que mudaram" pedia, até a ALE-347, um construtor que concatenava nomes
// de coluna — e ele existia para uma supressão de `gosec` ser afirmada uma vez
// só, porque a promessa "isto é literal, não entrada do usuário" repetida em
// cada sítio é a promessa que o próximo sítio quebra em silêncio.
//
// `COALESCE(?, coluna)` diz a MESMA coisa numa instrução FIXA: o parâmetro nulo
// devolve o valor que já está lá. Não há concatenação, não há supressão, e as
// colunas estão escritas à vista. Todas são `TEXT NOT NULL`, então nenhuma delas
// pode ser nula no banco e o `COALESCE` nunca cai num terceiro caso. Os poderes e
// as escolhas deles vão para as linhas por nível (`saveChoiceBlobs`).
//
// # E um pedido VAZIO não vira `UPDATE`
//
// Não é economia: gravar só o carimbo diria que a ficha mudou quando ela não
// mudou, e o `updatedAt` é o que a Mesa lê para repedir a ficha de quem está à
// mesa.
func (p Plays) SaveChoices(ctx context.Context, id int64, choices ChoiceWrite) error {
	if choices == (ChoiceWrite{}) {
		return nil
	}
	return p.inTx(ctx, "gravar as escolhas", func(tx *sqlcgen.Queries) error {
		row, err := tx.GetCharacter(ctx, id)
		if err != nil {
			return fmt.Errorf("ler a ficha %d: %w", id, err)
		}
		if err := tx.SetCharacterChoices(ctx, sqlcgen.SetCharacterChoicesParams{
			OriginChoices:        dbvalue.NullString(choices.OriginChoices),
			ClassChoices:         dbvalue.NullString(choices.ClassChoices),
			RaceAbilityChoices:   dbvalue.NullString(choices.RaceAbilityChoices),
			RaceAttributeChoices: dbvalue.NullString(choices.RaceAttributeChoices),
			SecondaryRaceChoices: dbvalue.NullString(choices.SecondaryRaceChoices),
			God:                  dbvalue.NullString(choices.God),
			GodPower:             dbvalue.NullString(choices.GodPower),
			UpdatedAt:            dbvalue.NowISO(), ID: id,
		}); err != nil {
			return fmt.Errorf("gravar as escolhas da ficha %d: %w", id, err)
		}
		return saveChoiceBlobs(ctx, tx, p.catalogs, row, choices.ClassPowers, choices.PowerChoices)
	})
}
